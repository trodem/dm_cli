package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
	Provider string
	Model    string
	BaseURL  string
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

	cfg, cfgErr := loadUserConfig()
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
		answer, model, err := askOllama(text, cfg.Ollama)
		if err != nil {
			return AskResult{}, err
		}
		return AskResult{Text: answer, Provider: "ollama", Model: model}, nil
	case "openai":
		applyOpenAIOverrides(&cfg, opts)
		answer, model, err := askOpenAI(text, cfg.OpenAI)
		if err != nil {
			return AskResult{}, err
		}
		return AskResult{Text: answer, Provider: "openai", Model: model}, nil
	case "auto":
		applyOllamaOverrides(&cfg, opts)
		if answer, model, err := askOllama(text, cfg.Ollama); err == nil {
			return AskResult{Text: answer, Provider: "ollama", Model: model}, nil
		}
		answer, model, err := askOpenAI(text, cfg.OpenAI)
		if err != nil {
			return AskResult{}, fmt.Errorf("ollama unavailable and openai fallback failed: %w", err)
		}
		return AskResult{Text: answer, Provider: "openai", Model: model}, nil
	default:
		return AskResult{}, fmt.Errorf("invalid provider %q (use auto|ollama|openai)", opts.Provider)
	}
}

func ResolveSessionProvider(opts AskOptions) (SessionProvider, error) {
	cfg, cfgErr := loadUserConfig()
	if cfgErr != nil {
		fmt.Fprintln(os.Stderr, "Warning: failed to load config:", cfgErr)
	}
	applyOllamaOverrides(&cfg, opts)
	applyOpenAIOverrides(&cfg, opts)

	ollamaBase, ollamaModel := resolvedOllama(cfg)
	openAIBase, openAIModel, openAIKey := resolvedOpenAI(cfg)
	reqProvider := strings.ToLower(strings.TrimSpace(opts.Provider))
	if reqProvider == "" {
		reqProvider = "openai"
	}

	switch reqProvider {
	case "ollama":
		if err := pingOllama(ollamaBase); err != nil {
			return SessionProvider{}, fmt.Errorf("ollama unavailable: %w", err)
		}
		return newSessionProvider("ollama", ollamaModel, ollamaBase), nil
	case "openai":
		if strings.TrimSpace(openAIKey) == "" {
			return SessionProvider{}, fmt.Errorf("missing OpenAI API key (set in %s or OPENAI_API_KEY)", configPath())
		}
		return newSessionProvider("openai", openAIModel, openAIBase), nil
	case "auto":
		if err := pingOllama(ollamaBase); err == nil {
			return newSessionProvider("ollama", ollamaModel, ollamaBase), nil
		}
		if strings.TrimSpace(openAIKey) == "" {
			return SessionProvider{}, fmt.Errorf("ollama unavailable and OpenAI API key is missing")
		}
		return newSessionProvider("openai", openAIModel, openAIBase), nil
	default:
		return SessionProvider{}, fmt.Errorf("invalid provider %q (use auto|ollama|openai)", opts.Provider)
	}
}

func newSessionProvider(provider, model, baseURL string) SessionProvider {
	return SessionProvider{
		Provider: provider,
		Model:    model,
		Options:  AskOptions{Provider: provider, Model: model, BaseURL: baseURL},
	}
}

func DecideWithPlugins(userPrompt string, pluginCatalog string, toolCatalog string, opts AskOptions, envContext string) (DecisionResult, error) {
	p := strings.TrimSpace(userPrompt)
	if p == "" {
		return DecisionResult{}, fmt.Errorf("prompt is required")
	}
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
		"Plugin argument rules:",
		"- Use plugin_args (object) for named PowerShell parameters, NOT the args array.",
		"- Keys are parameter names WITHOUT the leading dash (e.g. \"Host\" not \"-Host\").",
		"- For switch parameters (flags like -Force, -Confirm), set the value to \"true\".",
		"- If a mandatory parameter is missing from the user request, do NOT guess.",
		"  Instead return action=answer and ask the user to provide the missing value.",
		"- Only include parameters the user explicitly mentioned or that have obvious defaults.",
		"",
		"General rules:",
		"- action must be answer, run_plugin, run_tool, or create_function.",
		"- Do not invent plugin or tool names; use only the catalog above.",
		"- If the user request requires an operation that no existing plugin or tool can handle, return action=create_function.",
		"- Only use create_function for tasks that genuinely need a new automation capability, not for general knowledge questions.",
		"- If a plugin requires confirmation or is destructive, mention it in the answer.",
		"- For search tool use tool_args keys: base, ext, name, sort, limit, offset.",
		"- For rename tool use tool_args keys: base, from, to, name, case_sensitive.",
		"- For recent tool use tool_args keys: base, limit, offset.",
		"- For clean tool use tool_args keys: base, apply (true for delete, otherwise preview).",
	}
	if strings.TrimSpace(envContext) != "" {
		parts = append(parts, "", "Environment context:", envContext)
	}
	parts = append(parts, "", "User request:", p)
	decisionPrompt := strings.Join(parts, "\n")

	raw, err := AskWithOptions(decisionPrompt, opts)
	if err != nil {
		return DecisionResult{}, err
	}
	parsed, err := parseDecisionJSON(raw.Text)
	if err != nil {
		repaired, repErr := askDecisionJSONRepair(raw.Text, opts)
		if repErr == nil {
			if parsed2, p2Err := parseDecisionJSON(repaired.Text); p2Err == nil {
				parsed2.Provider = repaired.Provider
				parsed2.Model = repaired.Model
				if parsed2.Action != "run_plugin" && parsed2.Action != "run_tool" && parsed2.Action != "create_function" {
					parsed2.Action = "answer"
				}
				return parsed2, nil
			}
		}
		// fallback to plain answer if model never returned valid JSON
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

var jsonBlockRe = regexp.MustCompile("(?s)\\{.*\\}")

func parseDecisionJSON(text string) (DecisionResult, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return DecisionResult{}, fmt.Errorf("empty decision")
	}
	payload := trimmed
	if !strings.HasPrefix(payload, "{") {
		m := jsonBlockRe.FindString(trimmed)
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

func sanitizeAnyMap(m map[string]any) map[string]string {
	out := map[string]string{}
	for k, v := range m {
		key := strings.TrimSpace(k)
		if key == "" || v == nil {
			continue
		}
		val := strings.TrimSpace(fmt.Sprint(v))
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

func askOllama(prompt string, cfg ollamaConfig) (string, string, error) {
	baseURL, model := normalizedOllamaValues(cfg)

	reqBody := map[string]any{
		"model":  model,
		"prompt": prompt,
		"stream": false,
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return "", model, err
	}
	res, err := doWithRetry(func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, baseURL+"/api/generate", bytes.NewReader(raw))
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
		Response string `json:"response"`
	}
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return "", model, err
	}
	answer := strings.TrimSpace(parsed.Response)
	if answer == "" {
		return "", model, fmt.Errorf("empty ollama response")
	}
	return answer, model, nil
}

func askOpenAI(prompt string, cfg openAIConfig) (string, string, error) {
	baseURL, model, apiKey := normalizedOpenAIValues(cfg)
	if apiKey == "" {
		return "", "", fmt.Errorf("missing OpenAI API key (set in %s or OPENAI_API_KEY)", configPath())
	}

	reqBody := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a pragmatic coding assistant."},
			{"role": "user", "content": prompt},
		},
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
