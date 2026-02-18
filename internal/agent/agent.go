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
	Action   string
	Answer   string
	Plugin   string
	Tool     string
	ToolArgs map[string]string
	Args     []string
	Reason   string
	Provider string
	Model    string
}

func Ask(prompt string) (string, error) {
	res, err := AskWithOptions(prompt, AskOptions{})
	if err != nil {
		return "", err
	}
	return res.Text, nil
}

func AskWithOptions(prompt string, opts AskOptions) (AskResult, error) {
	text := strings.TrimSpace(prompt)
	if text == "" {
		return AskResult{}, fmt.Errorf("prompt is required")
	}

	cfg, _ := loadUserConfig()

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
	cfg, _ := loadUserConfig()
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
		return SessionProvider{
			Provider: "ollama",
			Model:    ollamaModel,
			Options: AskOptions{
				Provider: "ollama",
				Model:    ollamaModel,
				BaseURL:  ollamaBase,
			},
		}, nil
	case "openai":
		if strings.TrimSpace(openAIKey) == "" {
			return SessionProvider{}, fmt.Errorf("missing OpenAI API key (set in %s or OPENAI_API_KEY)", configPath())
		}
		return SessionProvider{
			Provider: "openai",
			Model:    openAIModel,
			Options: AskOptions{
				Provider: "openai",
				Model:    openAIModel,
				BaseURL:  openAIBase,
			},
		}, nil
	case "auto":
		if err := pingOllama(ollamaBase); err == nil {
			return SessionProvider{
				Provider: "ollama",
				Model:    ollamaModel,
				Options: AskOptions{
					Provider: "ollama",
					Model:    ollamaModel,
					BaseURL:  ollamaBase,
				},
			}, nil
		}
		if strings.TrimSpace(openAIKey) == "" {
			return SessionProvider{}, fmt.Errorf("ollama unavailable and OpenAI API key is missing")
		}
		return SessionProvider{
			Provider: "openai",
			Model:    openAIModel,
			Options: AskOptions{
				Provider: "openai",
				Model:    openAIModel,
				BaseURL:  openAIBase,
			},
		}, nil
	default:
		return SessionProvider{}, fmt.Errorf("invalid provider %q (use auto|ollama|openai)", opts.Provider)
	}
}

func DecideWithPlugins(userPrompt string, pluginCatalog string, toolCatalog string, opts AskOptions) (DecisionResult, error) {
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
	decisionPrompt := strings.Join([]string{
		"You are an execution planner for a CLI assistant.",
		"You can either answer directly, or request running one plugin/function, or request running one tool.",
		"Available plugins/functions:",
		pluginCatalog,
		"",
		"Available tools:",
		toolCatalog,
		"",
		"Return ONLY valid JSON with this schema:",
		`{"action":"answer","answer":"text"}`,
		`or {"action":"run_plugin","plugin":"name","args":["arg1"],"reason":"why","answer":"optional text before/after run"}`,
		`or {"action":"run_tool","tool":"name","tool_args":{"key":"value"},"reason":"why","answer":"optional text before/after run"}`,
		"",
		"Rules:",
		"- action must be answer, run_plugin, or run_tool",
		"- if run_plugin, plugin must be one of the available plugin names",
		"- if run_tool, tool must be one of the available tools",
		"- do not invent plugin names",
		"- do not invent tool names",
		"- for search tool prefer tool_args keys: base, ext, name, sort, limit, offset",
		"- for rename tool prefer tool_args keys: base, from, to, name, case_sensitive",
		"- for recent tool prefer tool_args keys: base, limit, offset",
		"- for clean tool prefer tool_args keys: base, apply",
		"",
		"User request:",
		p,
	}, "\n")

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
				if parsed2.Action != "run_plugin" && parsed2.Action != "run_tool" {
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
	if parsed.Action != "run_plugin" && parsed.Action != "run_tool" {
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
		`{"action":"run_plugin","plugin":"name","args":["arg1"],"reason":"why","answer":"optional text before/after run"}`,
		`{"action":"run_tool","tool":"name","tool_args":{"key":"value"},"reason":"why","answer":"optional text before/after run"}`,
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
		Action   string         `json:"action"`
		Answer   string         `json:"answer"`
		Plugin   string         `json:"plugin"`
		Tool     string         `json:"tool"`
		ToolArgs map[string]any `json:"tool_args"`
		Args     []string       `json:"args"`
		Reason   string         `json:"reason"`
	}
	if err := json.Unmarshal([]byte(payload), &obj); err != nil {
		return DecisionResult{}, err
	}
	toolArgs := map[string]string{}
	for k, v := range obj.ToolArgs {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		if v == nil {
			continue
		}
		val := strings.TrimSpace(fmt.Sprint(v))
		lc := strings.ToLower(val)
		if val == "" || lc == "<nil>" || lc == "null" {
			continue
		}
		toolArgs[key] = val
	}
	return DecisionResult{
		Action:   strings.ToLower(strings.TrimSpace(obj.Action)),
		Answer:   strings.TrimSpace(obj.Answer),
		Plugin:   strings.TrimSpace(obj.Plugin),
		Tool:     strings.TrimSpace(obj.Tool),
		ToolArgs: toolArgs,
		Args:     obj.Args,
		Reason:   strings.TrimSpace(obj.Reason),
	}, nil
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
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/generate", bytes.NewReader(raw))
	if err != nil {
		return "", model, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 45 * time.Second}
	res, err := client.Do(req)
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
	req, err := http.NewRequest(http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return "", model, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	client := &http.Client{Timeout: 60 * time.Second}
	res, err := client.Do(req)
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
