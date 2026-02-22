package agent

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestParseDecisionJSON_Answer(t *testing.T) {
	raw := `{"action":"answer","answer":"hello world"}`
	d, err := parseDecisionJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if d.Action != "answer" {
		t.Fatalf("expected action=answer, got %q", d.Action)
	}
	if d.Answer != "hello world" {
		t.Fatalf("expected answer='hello world', got %q", d.Answer)
	}
}

func TestParseDecisionJSON_RunPlugin(t *testing.T) {
	raw := `{"action":"run_plugin","plugin":"restart_backend","args":["-Force"],"reason":"user asked"}`
	d, err := parseDecisionJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if d.Action != "run_plugin" {
		t.Fatalf("expected action=run_plugin, got %q", d.Action)
	}
	if d.Plugin != "restart_backend" {
		t.Fatalf("expected plugin='restart_backend', got %q", d.Plugin)
	}
	if len(d.Args) != 1 || d.Args[0] != "-Force" {
		t.Fatalf("expected args=[\"-Force\"], got %v", d.Args)
	}
	if d.Reason != "user asked" {
		t.Fatalf("expected reason='user asked', got %q", d.Reason)
	}
}

func TestParseDecisionJSON_RunTool(t *testing.T) {
	raw := `{"action":"run_tool","tool":"search","tool_args":{"base":"C:\\Users","ext":"pdf"},"reason":"find pdfs"}`
	d, err := parseDecisionJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if d.Action != "run_tool" {
		t.Fatalf("expected action=run_tool, got %q", d.Action)
	}
	if d.Tool != "search" {
		t.Fatalf("expected tool='search', got %q", d.Tool)
	}
	if d.ToolArgs["base"] != "C:\\Users" {
		t.Fatalf("expected base='C:\\Users', got %q", d.ToolArgs["base"])
	}
	if d.ToolArgs["ext"] != "pdf" {
		t.Fatalf("expected ext='pdf', got %q", d.ToolArgs["ext"])
	}
}

func TestParseDecisionJSON_MarkdownFence(t *testing.T) {
	raw := "```json\n{\"action\":\"answer\",\"answer\":\"wrapped in fences\"}\n```"
	d, err := parseDecisionJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if d.Action != "answer" {
		t.Fatalf("expected action=answer, got %q", d.Action)
	}
	if d.Answer != "wrapped in fences" {
		t.Fatalf("expected answer='wrapped in fences', got %q", d.Answer)
	}
}

func TestParseDecisionJSON_PrefixText(t *testing.T) {
	raw := "Here is my response:\n{\"action\":\"answer\",\"answer\":\"extracted\"}"
	d, err := parseDecisionJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if d.Answer != "extracted" {
		t.Fatalf("expected answer='extracted', got %q", d.Answer)
	}
}

func TestParseDecisionJSON_Empty(t *testing.T) {
	_, err := parseDecisionJSON("")
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestParseDecisionJSON_InvalidJSON(t *testing.T) {
	_, err := parseDecisionJSON("not json at all with no braces")
	if err == nil {
		t.Fatal("expected error for invalid json")
	}
}

func TestParseDecisionJSON_RunPluginWithPluginArgs(t *testing.T) {
	raw := `{"action":"run_plugin","plugin":"sys_uptime","plugin_args":{"Host":"server1","Force":"true"},"reason":"check uptime"}`
	d, err := parseDecisionJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if d.Action != "run_plugin" {
		t.Fatalf("expected action=run_plugin, got %q", d.Action)
	}
	if d.Plugin != "sys_uptime" {
		t.Fatalf("expected plugin='sys_uptime', got %q", d.Plugin)
	}
	if d.PluginArgs["Host"] != "server1" {
		t.Fatalf("expected PluginArgs[Host]='server1', got %q", d.PluginArgs["Host"])
	}
	if d.PluginArgs["Force"] != "true" {
		t.Fatalf("expected PluginArgs[Force]='true', got %q", d.PluginArgs["Force"])
	}
}

func TestParseDecisionJSON_PluginArgsFalseSwitch(t *testing.T) {
	raw := `{"action":"run_plugin","plugin":"test","plugin_args":{"Name":"val","Skip":"false","Empty":null}}`
	d, err := parseDecisionJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if d.PluginArgs["Name"] != "val" {
		t.Fatalf("expected Name='val', got %q", d.PluginArgs["Name"])
	}
	if d.PluginArgs["Skip"] != "false" {
		t.Fatalf("expected Skip='false', got %q", d.PluginArgs["Skip"])
	}
	if _, ok := d.PluginArgs["Empty"]; ok {
		t.Fatal("expected null to be filtered out")
	}
}

func TestSanitizeAnyMap(t *testing.T) {
	m := map[string]any{
		"good":    "value",
		"null":    nil,
		"empty":   "",
		"  trim ": " padded ",
	}
	out := sanitizeAnyMap(m)
	if out["good"] != "value" {
		t.Fatalf("expected good='value', got %q", out["good"])
	}
	if _, ok := out["null"]; ok {
		t.Fatal("expected null to be filtered")
	}
	if _, ok := out["empty"]; ok {
		t.Fatal("expected empty to be filtered")
	}
	if out["trim"] != "padded" {
		t.Fatalf("expected trim='padded', got %q", out["trim"])
	}
}

func TestParseDecisionJSON_NullToolArgs(t *testing.T) {
	raw := `{"action":"run_tool","tool":"search","tool_args":{"base":"C:\\","ext":null,"name":""}}`
	d, err := parseDecisionJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := d.ToolArgs["ext"]; ok {
		t.Fatal("expected null ext to be filtered out")
	}
	if _, ok := d.ToolArgs["name"]; ok {
		t.Fatal("expected empty name to be filtered out")
	}
	if d.ToolArgs["base"] != "C:\\" {
		t.Fatalf("expected base='C:\\', got %q", d.ToolArgs["base"])
	}
}

func TestNormalizedOllamaValues_Defaults(t *testing.T) {
	base, model := normalizedOllamaValues(ollamaConfig{})
	if base != defaultOllamaBaseURL {
		t.Fatalf("expected default base url, got %q", base)
	}
	if model != defaultOllamaModel {
		t.Fatalf("expected default model, got %q", model)
	}
}

func TestNormalizedOllamaValues_Custom(t *testing.T) {
	base, model := normalizedOllamaValues(ollamaConfig{
		BaseURL: "http://myhost:1234/",
		Model:   "llama3",
	})
	if base != "http://myhost:1234" {
		t.Fatalf("expected trailing slash trimmed, got %q", base)
	}
	if model != "llama3" {
		t.Fatalf("expected model='llama3', got %q", model)
	}
}

func TestNormalizedOpenAIValues_Defaults(t *testing.T) {
	orig := os.Getenv("OPENAI_API_KEY")
	os.Unsetenv("OPENAI_API_KEY")
	defer func() {
		if orig != "" {
			os.Setenv("OPENAI_API_KEY", orig)
		}
	}()

	base, model, key := normalizedOpenAIValues(openAIConfig{})
	if base != defaultOpenAIBaseURL {
		t.Fatalf("expected default base url, got %q", base)
	}
	if model != defaultOpenAIModel {
		t.Fatalf("expected default model, got %q", model)
	}
	if key != "" {
		t.Fatalf("expected empty key, got %q", key)
	}
}

func TestNormalizedOpenAIValues_ConfigKey(t *testing.T) {
	orig := os.Getenv("OPENAI_API_KEY")
	os.Unsetenv("OPENAI_API_KEY")
	defer func() {
		if orig != "" {
			os.Setenv("OPENAI_API_KEY", orig)
		}
	}()

	_, _, key := normalizedOpenAIValues(openAIConfig{APIKey: "sk-test123"})
	if key != "sk-test123" {
		t.Fatalf("expected key from config, got %q", key)
	}
}

func TestNormalizedOpenAIValues_EnvKey(t *testing.T) {
	orig := os.Getenv("OPENAI_API_KEY")
	os.Setenv("OPENAI_API_KEY", "sk-env-key")
	defer func() {
		if orig != "" {
			os.Setenv("OPENAI_API_KEY", orig)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
	}()

	_, _, key := normalizedOpenAIValues(openAIConfig{})
	if key != "sk-env-key" {
		t.Fatalf("expected key from env, got %q", key)
	}
}

func TestLoadUserConfig_FileNotExist(t *testing.T) {
	orig := os.Getenv("DM_AGENT_CONFIG")
	tmp := filepath.Join(t.TempDir(), "nonexistent.json")
	os.Setenv("DM_AGENT_CONFIG", tmp)
	defer func() {
		if orig != "" {
			os.Setenv("DM_AGENT_CONFIG", orig)
		} else {
			os.Unsetenv("DM_AGENT_CONFIG")
		}
	}()

	cfg, err := loadUserConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Ollama.Model != "" || cfg.OpenAI.APIKey != "" {
		t.Fatalf("expected zero config when file missing, got %+v", cfg)
	}
}

func TestLoadUserConfig_ValidFile(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "agent.json")
	data := `{"ollama":{"model":"test-model"},"openai":{"api_key":"sk-x","model":"gpt-4"}}`
	if err := os.WriteFile(tmp, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	orig := os.Getenv("DM_AGENT_CONFIG")
	os.Setenv("DM_AGENT_CONFIG", tmp)
	defer func() {
		if orig != "" {
			os.Setenv("DM_AGENT_CONFIG", orig)
		} else {
			os.Unsetenv("DM_AGENT_CONFIG")
		}
	}()

	cfg, err := loadUserConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Ollama.Model != "test-model" {
		t.Fatalf("expected ollama model='test-model', got %q", cfg.Ollama.Model)
	}
	if cfg.OpenAI.APIKey != "sk-x" {
		t.Fatalf("expected openai key='sk-x', got %q", cfg.OpenAI.APIKey)
	}
	if cfg.OpenAI.Model != "gpt-4" {
		t.Fatalf("expected openai model='gpt-4', got %q", cfg.OpenAI.Model)
	}
}

func TestLoadUserConfig_InvalidJSON(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "agent.json")
	if err := os.WriteFile(tmp, []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	orig := os.Getenv("DM_AGENT_CONFIG")
	os.Setenv("DM_AGENT_CONFIG", tmp)
	defer func() {
		if orig != "" {
			os.Setenv("DM_AGENT_CONFIG", orig)
		} else {
			os.Unsetenv("DM_AGENT_CONFIG")
		}
	}()

	_, err := loadUserConfig()
	if err == nil {
		t.Fatal("expected error for invalid json config")
	}
}

func TestApplyOllamaOverrides(t *testing.T) {
	cfg := userConfig{}
	applyOllamaOverrides(&cfg, AskOptions{Model: "custom-model", BaseURL: "http://host:999"})
	if cfg.Ollama.Model != "custom-model" {
		t.Fatalf("expected model override, got %q", cfg.Ollama.Model)
	}
	if cfg.Ollama.BaseURL != "http://host:999" {
		t.Fatalf("expected base url override, got %q", cfg.Ollama.BaseURL)
	}
}

func TestApplyOpenAIOverrides(t *testing.T) {
	cfg := userConfig{}
	applyOpenAIOverrides(&cfg, AskOptions{Model: "gpt-5", BaseURL: "https://custom.api"})
	if cfg.OpenAI.Model != "gpt-5" {
		t.Fatalf("expected model override, got %q", cfg.OpenAI.Model)
	}
	if cfg.OpenAI.BaseURL != "https://custom.api" {
		t.Fatalf("expected base url override, got %q", cfg.OpenAI.BaseURL)
	}
}

func TestApplyOverrides_EmptyDoesNotOverwrite(t *testing.T) {
	cfg := userConfig{
		Ollama: ollamaConfig{Model: "existing"},
		OpenAI: openAIConfig{Model: "existing"},
	}
	applyOllamaOverrides(&cfg, AskOptions{})
	applyOpenAIOverrides(&cfg, AskOptions{})
	if cfg.Ollama.Model != "existing" {
		t.Fatalf("expected ollama model preserved, got %q", cfg.Ollama.Model)
	}
	if cfg.OpenAI.Model != "existing" {
		t.Fatalf("expected openai model preserved, got %q", cfg.OpenAI.Model)
	}
}

func TestDoWithRetry_SuccessOnFirst(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	res, err := doWithRetry(func() (*http.Request, error) {
		return http.NewRequest(http.MethodGet, srv.URL, nil)
	})
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestDoWithRetry_RetriesOn500(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	// Use a very short retry delay for testing
	origDelay := retryDelay
	defer func() { retryDelay = origDelay }()

	res, err := doWithRetry(func() (*http.Request, error) {
		return http.NewRequest(http.MethodGet, srv.URL, nil)
	})
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("expected 2 calls (1 retry), got %d", calls)
	}
}

func TestDoWithRetry_NoRetryOn4xx(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(400)
	}))
	defer srv.Close()

	res, err := doWithRetry(func() (*http.Request, error) {
		return http.NewRequest(http.MethodGet, srv.URL, nil)
	})
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("expected 1 call (no retry on 4xx), got %d", calls)
	}
}

func TestDoWithRetry_RetriesOn429(t *testing.T) {
	origDelay := retryDelay
	retryDelay = 1 // near-instant for test speed
	defer func() { retryDelay = origDelay }()

	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n <= 2 {
			w.WriteHeader(429)
			return
		}
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	res, err := doWithRetry(func() (*http.Request, error) {
		return http.NewRequest(http.MethodGet, srv.URL, nil)
	})
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if atomic.LoadInt32(&calls) != 3 {
		t.Fatalf("expected 3 calls (2 retries after 429), got %d", calls)
	}
}

func TestDoWithRetry_ExhaustsRetries(t *testing.T) {
	origDelay := retryDelay
	retryDelay = 1
	defer func() { retryDelay = origDelay }()

	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(500)
	}))
	defer srv.Close()

	_, err := doWithRetry(func() (*http.Request, error) {
		return http.NewRequest(http.MethodGet, srv.URL, nil)
	})
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if atomic.LoadInt32(&calls) != 3 {
		t.Fatalf("expected 3 calls (1 initial + 2 retries), got %d", calls)
	}
}
