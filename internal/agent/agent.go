package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	defaultOllamaBaseURL = "http://127.0.0.1:11434"
	defaultOllamaModel   = "deepseek-coder-v2:latest"
	defaultOpenAIBaseURL = "https://api.openai.com/v1"
	defaultOpenAIModel   = "gpt-4o-mini"
	maxRetries           = 2
)

var (
	retryDelay       = 2 * time.Second
	sharedHTTPClient = &http.Client{Timeout: 60 * time.Second}

	configOnce  sync.Once
	configCached userConfig
	configErr    error
)

type userConfig struct {
	Ollama ollamaConfig `json:"ollama"`
	OpenAI openAIConfig `json:"openai"`
}

type ollamaConfig struct {
	BaseURL string `json:"base_url"`
	Model   string `json:"model"`
}

type openAIConfig struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
	Model   string `json:"model"`
}

type AskOptions struct {
	Provider     string
	Model        string
	BaseURL      string
	Temperature  *float64
	MaxTokens    int
	JSONMode     bool
	SystemPrompt string
}

type AskResult struct {
	Text     string
	Provider string
	Model    string
}

type SessionProvider struct {
	Provider string
	Model    string
	Options  AskOptions
}

type DecisionResult struct {
	Action              string
	Answer              string
	Plugin              string
	PluginArgs          map[string]string
	Tool                string
	ToolArgs            map[string]string
	Args                []string
	Reason              string
	FunctionDescription string
	Provider            string
	Model               string
}

func AskWithOptions(prompt string, opts AskOptions) (AskResult, error) {
	text := strings.TrimSpace(prompt)
	if text == "" {
		return AskResult{}, fmt.Errorf("prompt is required")
	}

	cfg, cfgErr := cachedUserConfig()
	if cfgErr != nil {
		fmt.Fprintln(os.Stderr, "Warning: failed to load config:", cfgErr)
	}

	provider := strings.ToLower(strings.TrimSpace(opts.Provider))
	if provider == "" {
		provider = "openai"
	}

	switch provider {
	case "ollama":
		applyOllamaOverrides(&cfg, opts)
		answer, model, err := askOllama(text, cfg.Ollama, opts)
		if err != nil {
			return AskResult{}, err
		}
		return AskResult{Text: answer, Provider: "ollama", Model: model}, nil
	case "openai":
		applyOpenAIOverrides(&cfg, opts)
		answer, model, err := askOpenAI(text, cfg.OpenAI, opts)
		if err != nil {
			return AskResult{}, err
		}
		return AskResult{Text: answer, Provider: "openai", Model: model}, nil
	case "auto":
		applyOllamaOverrides(&cfg, opts)
		if answer, model, err := askOllama(text, cfg.Ollama, opts); err == nil {
			return AskResult{Text: answer, Provider: "ollama", Model: model}, nil
		}
		applyOpenAIOverrides(&cfg, opts)
		answer, model, err := askOpenAI(text, cfg.OpenAI, opts)
		if err != nil {
			return AskResult{}, fmt.Errorf("ollama unavailable and openai fallback failed: %w", err)
		}
		return AskResult{Text: answer, Provider: "openai", Model: model}, nil
	default:
		return AskResult{}, fmt.Errorf("invalid provider %q (use auto|ollama|openai)", opts.Provider)
	}
}

func ResolveSessionProvider(opts AskOptions) (SessionProvider, error) {
	cfg, cfgErr := cachedUserConfig()
	if cfgErr != nil {
		fmt.Fprintln(os.Stderr, "Warning: failed to load config:", cfgErr)
	}
	applyOllamaOverrides(&cfg, opts)
	applyOpenAIOverrides(&cfg, opts)

	ollamaBase, ollamaModel := resolvedOllama(cfg)
	openAIBase, openAIModel, openAIKey := resolvedOpenAI(cfg)

	if err := validateBaseURL(ollamaBase, "ollama"); err != nil {
		fmt.Fprintln(os.Stderr, "Warning:", err)
	}
	if err := validateBaseURL(openAIBase, "openai"); err != nil {
		fmt.Fprintln(os.Stderr, "Warning:", err)
	}

	reqProvider := strings.ToLower(strings.TrimSpace(opts.Provider))
	if reqProvider == "" {
		reqProvider = "openai"
	}

	switch reqProvider {
	case "ollama":
		if err := pingOllama(ollamaBase); err != nil {
			return SessionProvider{}, fmt.Errorf("ollama unavailable: %w\n  Hint: run 'dm doctor' for diagnostics", err)
		}
		return newSessionProvider("ollama", ollamaModel, ollamaBase), nil
	case "openai":
		if strings.TrimSpace(openAIKey) == "" {
			return SessionProvider{}, fmt.Errorf("missing OpenAI API key (set in %s or OPENAI_API_KEY)\n  Hint: run 'dm doctor' for diagnostics", configPath())
		}
		return newSessionProvider("openai", openAIModel, openAIBase), nil
	case "auto":
		if err := pingOllama(ollamaBase); err == nil {
			return newSessionProvider("ollama", ollamaModel, ollamaBase), nil
		}
		if strings.TrimSpace(openAIKey) == "" {
			return SessionProvider{}, fmt.Errorf("ollama unavailable and OpenAI API key is missing\n  Hint: run 'dm doctor' for diagnostics")
		}
		return newSessionProvider("openai", openAIModel, openAIBase), nil
	default:
		return SessionProvider{}, fmt.Errorf("invalid provider %q (use auto|ollama|openai)", opts.Provider)
	}
}

func validateBaseURL(u, label string) error {
	if strings.TrimSpace(u) == "" {
		return nil
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		return fmt.Errorf("%s base_url %q has no http(s) scheme", label, u)
	}
	return nil
}

func newSessionProvider(provider, model, baseURL string) SessionProvider {
	return SessionProvider{
		Provider: provider,
		Model:    model,
		Options:  AskOptions{Provider: provider, Model: model, BaseURL: baseURL},
	}
}

const decisionTemperature = 0.2
const decisionMaxTokens = 1024

func buildDecisionSystemPrompt(pluginCatalog, toolCatalog string) string {
	if strings.TrimSpace(pluginCatalog) == "" {
		pluginCatalog = "(none)"
	}
	if strings.TrimSpace(toolCatalog) == "" {
		toolCatalog = "(none)"
	}
	parts := []string{
		"You are an execution planner for a CLI assistant.",
		"You can either answer directly, run a plugin (PowerShell function), run a built-in tool, or propose creating a new function.",
		"",
		"Available plugins (PowerShell functions):",
		pluginCatalog,
		"",
		"Available tools:",
		toolCatalog,
		"",
		"Return ONLY valid JSON. Use one of these schemas:",
		`{"action":"answer","answer":"text"}`,
		`{"action":"run_plugin","plugin":"name","plugin_args":{"ParamName":"value","SwitchParam":"true"},"reason":"why","answer":"optional text"}`,
		`{"action":"run_tool","tool":"name","tool_args":{"key":"value"},"reason":"why","answer":"optional text"}`,
		`{"action":"create_function","function_description":"detailed description of what the function should do, its inputs and outputs","reason":"why no existing plugin fits"}`,
		"",
		"Catalog notation: Name* = required, Flag? = switch, Param=val = default value, Param=a|b|c = allowed values.",
		"",
		"Plugin argument rules:",
		"- Use plugin_args (object) for named PowerShell parameters, NOT the args array.",
		"- Keys are parameter names WITHOUT the leading dash (e.g. \"Host\" not \"-Host\").",
		"- For switch parameters (marked ? in catalog), set the value to \"true\".",
		"- ALWAYS include ALL required parameters (marked * in catalog) for the chosen plugin.",
		"- Map values from the user request to the correct parameter names in the catalog.",
		"  Example: user says 'search for mario in user table' with params Table*, Value*, Limit=20:",
		`  => plugin_args: {"Table":"user","Value":"mario"}`,
		"- If a required parameter cannot be inferred from the user request at all, return action=answer and ask the user.",
		"- If a previous step failed with 'missing mandatory parameters', the NEXT attempt MUST include those parameters.",
		"",
		"Decision process (follow in order):",
		"1. Identify the user's INTENT: what do they want to accomplish?",
		"2. Find the best matching toolkit group [Name] in the catalog for that domain.",
		"3. Pick the specific function whose name and synopsis best match the intent.",
		"4. Check required params (*): can ALL of them be inferred from the user request? If not, action=answer and ask.",
		"5. Map user values to the correct parameter names. Use defaults when the user did not specify optional params.",
		"6. If no plugin or tool matches, consider action=answer for knowledge questions or action=create_function for new automation.",
		"7. Put your reasoning in the \"reason\" field.",
		"",
		"General rules:",
		"- If answering the request requires live data you do not have (git changes, file contents, system state, etc.), run the appropriate tool FIRST; your output will appear in the action history so the next step can use it.",
		"- When asked to write a commit message: output ONLY one subject line in English, imperative mood, <=72 chars, no trailing period; summarize the main change and motivation (component + intent), avoid vague wording and avoid file names unless essential. Example: 'Add retry logic to HTTP client for transient failures'.",
		"- action must be answer, run_plugin, run_tool, or create_function.",
		"- Do not invent plugin or tool names; use only the catalog above.",
		"- If the user request requires an operation that no existing plugin or tool can handle, return action=create_function.",
		"- Only use create_function for tasks that genuinely need a new automation capability, not for general knowledge questions.",
		"- If a plugin requires confirmation or is destructive, mention it in the answer.",
		"- Tool arguments are already listed in the catalog after 'tool_args:'. Use those exact keys.",
	}
	return strings.Join(parts, "\n")
}

func buildDecisionUserPrompt(userPrompt, envContext string) string {
	parts := []string{}
	if strings.TrimSpace(envContext) != "" {
		parts = append(parts, "Environment context:", envContext, "")
	}
	parts = append(parts, "User request:", strings.TrimSpace(userPrompt))
	return strings.Join(parts, "\n")
}

func decisionOpts(base AskOptions, systemPrompt string) AskOptions {
	temp := decisionTemperature
	return AskOptions{
		Provider:     base.Provider,
		Model:        base.Model,
		BaseURL:      base.BaseURL,
		Temperature:  &temp,
		MaxTokens:    decisionMaxTokens,
		JSONMode:     true,
		SystemPrompt: systemPrompt,
	}
}

func DecideWithPlugins(userPrompt string, pluginCatalog string, toolCatalog string, opts AskOptions, envContext string) (DecisionResult, error) {
	p := strings.TrimSpace(userPrompt)
	if p == "" {
		return DecisionResult{}, fmt.Errorf("prompt is required")
	}

	systemPrompt := buildDecisionSystemPrompt(pluginCatalog, toolCatalog)
	userMsg := buildDecisionUserPrompt(p, envContext)
	dOpts := decisionOpts(opts, systemPrompt)

	raw, err := AskWithOptions(userMsg, dOpts)
	if err != nil {
		return DecisionResult{}, err
	}
	parsed, err := parseDecisionJSON(raw.Text)
	if err != nil {
		slog.Warn("JSON parse failed, attempting repair", "error", err)
		slog.Debug("raw LLM output for repair", "text", truncateLog(raw.Text, 300))
		repaired, repErr := askDecisionJSONRepair(raw.Text, dOpts)
		if repErr == nil {
			if parsed2, p2Err := parseDecisionJSON(repaired.Text); p2Err == nil {
				slog.Warn("JSON repair succeeded", "action", parsed2.Action)
				parsed2.Provider = repaired.Provider
				parsed2.Model = repaired.Model
				if parsed2.Action != "run_plugin" && parsed2.Action != "run_tool" && parsed2.Action != "create_function" {
					parsed2.Action = "answer"
				}
				return parsed2, nil
			}
		}
		slog.Warn("JSON repair failed, falling back to raw answer")
		return DecisionResult{
			Action:   "answer",
			Answer:   raw.Text,
			Provider: raw.Provider,
			Model:    raw.Model,
		}, nil
	}
	parsed.Provider = raw.Provider
	parsed.Model = raw.Model
	if parsed.Action != "run_plugin" && parsed.Action != "run_tool" && parsed.Action != "create_function" {
		parsed.Action = "answer"
	}
	return parsed, nil
}

func askDecisionJSONRepair(rawText string, opts AskOptions) (AskResult, error) {
	repairPrompt := strings.Join([]string{
		"Convert the following text to valid JSON only.",
		"Do not add markdown fences.",
		"Use exactly one of these schemas:",
		`{"action":"answer","answer":"text"}`,
		`{"action":"run_plugin","plugin":"name","plugin_args":{"ParamName":"value"},"reason":"why","answer":"optional text"}`,
		`{"action":"run_tool","tool":"name","tool_args":{"key":"value"},"reason":"why","answer":"optional text"}`,
		`{"action":"create_function","function_description":"description","reason":"why"}`,
		"",
		"Text:",
		strings.TrimSpace(rawText),
	}, "\n")
	return AskWithOptions(repairPrompt, opts)
}

func findFirstJSONObject(text string) string {
	start := strings.Index(text, "{")
	if start < 0 {
		return ""
	}
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(text); i++ {
		ch := text[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inString {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if ch == '{' {
			depth++
		} else if ch == '}' {
			depth--
			if depth == 0 {
				return text[start : i+1]
			}
		}
	}
	return ""
}

func parseDecisionJSON(text string) (DecisionResult, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return DecisionResult{}, fmt.Errorf("empty decision")
	}
	payload := trimmed
	if !strings.HasPrefix(payload, "{") {
		m := findFirstJSONObject(trimmed)
		if m == "" {
			return DecisionResult{}, fmt.Errorf("no json object found")
		}
		payload = m
	}
	var obj struct {
		Action              string         `json:"action"`
		Answer              string         `json:"answer"`
		Plugin              string         `json:"plugin"`
		PluginArgs          map[string]any `json:"plugin_args"`
		Tool                string         `json:"tool"`
		ToolArgs            map[string]any `json:"tool_args"`
		Args                []string       `json:"args"`
		Reason              string         `json:"reason"`
		FunctionDescription string         `json:"function_description"`
	}
	if err := json.Unmarshal([]byte(payload), &obj); err != nil {
		return DecisionResult{}, err
	}
	pluginArgs := sanitizeAnyMap(obj.PluginArgs)
	toolArgs := sanitizeAnyMap(obj.ToolArgs)
	return DecisionResult{
		Action:              strings.ToLower(strings.TrimSpace(obj.Action)),
		Answer:              strings.TrimSpace(obj.Answer),
		Plugin:              strings.TrimSpace(obj.Plugin),
		PluginArgs:          pluginArgs,
		Tool:                strings.TrimSpace(obj.Tool),
		ToolArgs:            toolArgs,
		Args:                obj.Args,
		Reason:              strings.TrimSpace(obj.Reason),
		FunctionDescription: strings.TrimSpace(obj.FunctionDescription),
	}, nil
}

func truncateLog(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func sanitizeAnyMap(m map[string]any) map[string]string {
	out := map[string]string{}
	for k, v := range m {
		key := strings.TrimSpace(k)
		if key == "" || v == nil {
			continue
		}
		var val string
		switch tv := v.(type) {
		case string:
			val = tv
		case float64:
			if tv == float64(int64(tv)) {
				val = fmt.Sprintf("%d", int64(tv))
			} else {
				val = fmt.Sprintf("%g", tv)
			}
		case bool:
			val = fmt.Sprintf("%t", tv)
		default:
			raw, err := json.Marshal(tv)
			if err != nil {
				val = fmt.Sprint(tv)
			} else {
				val = string(raw)
			}
		}
		val = strings.TrimSpace(val)
		lc := strings.ToLower(val)
		if val == "" || lc == "<nil>" || lc == "null" {
			continue
		}
		out[key] = val
	}
	return out
}

func applyOllamaOverrides(cfg *userConfig, opts AskOptions) {
	if strings.TrimSpace(opts.Model) != "" {
		cfg.Ollama.Model = strings.TrimSpace(opts.Model)
	}
	if strings.TrimSpace(opts.BaseURL) != "" {
		cfg.Ollama.BaseURL = strings.TrimSpace(opts.BaseURL)
	}
}

func applyOpenAIOverrides(cfg *userConfig, opts AskOptions) {
	if strings.TrimSpace(opts.Model) != "" {
		cfg.OpenAI.Model = strings.TrimSpace(opts.Model)
	}
	if strings.TrimSpace(opts.BaseURL) != "" {
		cfg.OpenAI.BaseURL = strings.TrimSpace(opts.BaseURL)
	}
}

func loadUserConfig() (userConfig, error) {
	for _, path := range configPaths() {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return userConfig{}, err
		}
		var cfg userConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			return userConfig{}, err
		}
		return cfg, nil
	}
	return userConfig{}, nil
}

func cachedUserConfig() (userConfig, error) {
	configOnce.Do(func() {
		configCached, configErr = loadUserConfig()
	})
	return configCached, configErr
}

func configPath() string {
	paths := configPaths()
	if len(paths) == 0 {
		return "dm.agent.json"
	}
	return paths[0]
}

func configPaths() []string {
	if p := strings.TrimSpace(os.Getenv("DM_AGENT_CONFIG")); p != "" {
		return []string{p}
	}
	if p := configPathNearExecutable(); p != "" {
		return []string{p}
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return []string{".config/dm/agent.json"}
	}
	return []string{filepath.Join(home, ".config", "dm", "agent.json")}
}

func configPathNearExecutable() string {
	exe, err := os.Executable()
	if err != nil || strings.TrimSpace(exe) == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(exe), "dm.agent.json")
}

func doWithRetry(buildReq func() (*http.Request, error)) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := retryDelay * time.Duration(1<<(attempt-1))
			time.Sleep(delay)
		}
		req, err := buildReq()
		if err != nil {
			return nil, err
		}
		res, err := sharedHTTPClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		if res.StatusCode == 429 || res.StatusCode >= 500 {
			_ = res.Body.Close()
			lastErr = fmt.Errorf("server error: %s", res.Status)
			continue
		}
		return res, nil
	}
	return nil, lastErr
}

func askOllama(prompt string, cfg ollamaConfig, opts AskOptions) (string, string, error) {
	baseURL, model := normalizedOllamaValues(cfg)
	slog.Debug("LLM request", "provider", "ollama", "model", model, "prompt_chars", len(prompt))

	messages := []map[string]string{}
	systemMsg := "You are a pragmatic coding assistant."
	if strings.TrimSpace(opts.SystemPrompt) != "" {
		systemMsg = opts.SystemPrompt
	}
	messages = append(messages, map[string]string{"role": "system", "content": systemMsg})
	messages = append(messages, map[string]string{"role": "user", "content": prompt})

	reqBody := map[string]any{
		"model":    model,
		"messages": messages,
		"stream":   false,
	}
	if opts.JSONMode {
		reqBody["format"] = "json"
	}
	ollamaOpts := map[string]any{}
	if opts.Temperature != nil {
		ollamaOpts["temperature"] = *opts.Temperature
	}
	if opts.MaxTokens > 0 {
		ollamaOpts["num_predict"] = opts.MaxTokens
	}
	if len(ollamaOpts) > 0 {
		reqBody["options"] = ollamaOpts
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return "", model, err
	}
	res, err := doWithRetry(func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, baseURL+"/api/chat", bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		return req, nil
	})
	if err != nil {
		return "", model, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", model, fmt.Errorf("ollama status: %s", res.Status)
	}
	var parsed struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return "", model, err
	}
	answer := strings.TrimSpace(parsed.Message.Content)
	if answer == "" {
		return "", model, fmt.Errorf("empty ollama response")
	}
	return answer, model, nil
}

func askOpenAI(prompt string, cfg openAIConfig, opts AskOptions) (string, string, error) {
	baseURL, model, apiKey := normalizedOpenAIValues(cfg)
	if apiKey == "" {
		return "", "", fmt.Errorf("missing OpenAI API key (set in %s or OPENAI_API_KEY)", configPath())
	}
	slog.Debug("LLM request", "provider", "openai", "model", model, "prompt_chars", len(prompt))

	systemMsg := "You are a pragmatic coding assistant."
	if strings.TrimSpace(opts.SystemPrompt) != "" {
		systemMsg = opts.SystemPrompt
	}

	reqBody := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": systemMsg},
			{"role": "user", "content": prompt},
		},
	}
	if opts.Temperature != nil {
		reqBody["temperature"] = *opts.Temperature
	}
	if opts.MaxTokens > 0 {
		reqBody["max_tokens"] = opts.MaxTokens
	}
	if opts.JSONMode {
		reqBody["response_format"] = map[string]string{"type": "json_object"}
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return "", model, err
	}
	res, err := doWithRetry(func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)
		return req, nil
	})
	if err != nil {
		return "", model, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", model, fmt.Errorf("openai status: %s", res.Status)
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return "", model, err
	}
	if len(parsed.Choices) == 0 {
		return "", model, fmt.Errorf("empty openai response")
	}
	answer := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if answer == "" {
		return "", model, fmt.Errorf("empty openai content")
	}
	return answer, model, nil
}

func resolvedOllama(cfg userConfig) (string, string) {
	return normalizedOllamaValues(cfg.Ollama)
}

func resolvedOpenAI(cfg userConfig) (string, string, string) {
	return normalizedOpenAIValues(cfg.OpenAI)
}

func normalizedOllamaValues(cfg ollamaConfig) (string, string) {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultOllamaBaseURL
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = defaultOllamaModel
	}
	return baseURL, model
}

func normalizedOpenAIValues(cfg openAIConfig) (string, string, string) {
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	}
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = defaultOpenAIModel
	}
	return baseURL, model, apiKey
}

func pingOllama(baseURL string) error {
	u := strings.TrimRight(strings.TrimSpace(baseURL), "/") + "/api/tags"
	client := &http.Client{Timeout: 3 * time.Second}
	res, err := client.Get(u)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("status %s", res.Status)
	}
	return nil
}
