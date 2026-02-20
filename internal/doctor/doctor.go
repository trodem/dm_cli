package doctor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cli/internal/plugins"
)

type Level string

const (
	LevelOK    Level = "OK"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"
)

type Check struct {
	Level   Level  `json:"level"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

type Report struct {
	GeneratedAt time.Time `json:"generated_at"`
	Checks      []Check   `json:"checks"`
	OKCount     int       `json:"ok_count"`
	WarnCount   int       `json:"warn_count"`
	ErrorCount  int       `json:"error_count"`
}

func Run(baseDir string) Report {
	r := Report{GeneratedAt: time.Now()}
	r.add(checkAgentConfig())
	r.add(checkOllama())
	r.add(checkOpenAI())
	r.add(checkPlugins(baseDir))
	r.add(checkCommonToolPaths())
	return r
}

func (r *Report) add(c Check) {
	r.Checks = append(r.Checks, c)
	switch c.Level {
	case LevelOK:
		r.OKCount++
	case LevelWarn:
		r.WarnCount++
	case LevelError:
		r.ErrorCount++
	}
}

func RenderText(r Report) {
	fmt.Printf("Doctor report (%s)\n", r.GeneratedAt.Format(time.RFC3339))
	for _, c := range r.Checks {
		fmt.Printf("[%s] %-18s %s\n", c.Level, c.Name, c.Message)
	}
	fmt.Printf("Summary: OK=%d WARN=%d ERROR=%d\n", r.OKCount, r.WarnCount, r.ErrorCount)
}

func RenderJSON(r Report) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

func checkAgentConfig() Check {
	path := agentConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Check{
				Level:   LevelWarn,
				Name:    "agent-config",
				Message: fmt.Sprintf("config not found: %s", path),
			}
		}
		return Check{
			Level:   LevelError,
			Name:    "agent-config",
			Message: fmt.Sprintf("cannot read config: %v", err),
		}
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return Check{
			Level:   LevelError,
			Name:    "agent-config",
			Message: fmt.Sprintf("invalid JSON in %s: %v", path, err),
		}
	}
	return Check{
		Level:   LevelOK,
		Name:    "agent-config",
		Message: fmt.Sprintf("loaded %s", path),
	}
}

func checkOllama() Check {
	baseURL, model := ollamaConfig()
	client := &http.Client{Timeout: 3 * time.Second}
	res, err := client.Get(strings.TrimRight(baseURL, "/") + "/api/tags")
	if err != nil {
		return Check{
			Level:   LevelWarn,
			Name:    "ollama",
			Message: fmt.Sprintf("unreachable at %s (%v)", baseURL, err),
		}
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return Check{
			Level:   LevelWarn,
			Name:    "ollama",
			Message: fmt.Sprintf("endpoint returned %s", res.Status),
		}
	}
	return Check{
		Level:   LevelOK,
		Name:    "ollama",
		Message: fmt.Sprintf("reachable at %s (model=%s)", baseURL, model),
	}
}

func checkOpenAI() Check {
	baseURL, model, key := openAIConfig()
	if strings.TrimSpace(key) == "" {
		return Check{
			Level:   LevelWarn,
			Name:    "openai",
			Message: "missing API key (set in dm.agent.json or OPENAI_API_KEY)",
		}
	}
	return Check{
		Level:   LevelOK,
		Name:    "openai",
		Message: fmt.Sprintf("configured (base_url=%s model=%s)", baseURL, model),
	}
}

func checkPlugins(baseDir string) Check {
	items, err := plugins.ListEntries(baseDir, true)
	if err != nil {
		return Check{
			Level:   LevelError,
			Name:    "plugins",
			Message: fmt.Sprintf("scan failed: %v", err),
		}
	}
	if len(items) == 0 {
		return Check{
			Level:   LevelWarn,
			Name:    "plugins",
			Message: "no plugins found",
		}
	}
	return Check{
		Level:   LevelOK,
		Name:    "plugins",
		Message: fmt.Sprintf("found %d plugin/function entries", len(items)),
	}
}

func checkCommonToolPaths() Check {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return Check{
			Level:   LevelWarn,
			Name:    "tool-paths",
			Message: "cannot resolve user home directory",
		}
	}
	paths := []string{
		filepath.Join(home, "Downloads"),
		filepath.Join(home, "Desktop"),
		filepath.Join(home, "Documents"),
	}
	missing := make([]string, 0, len(paths))
	for _, p := range paths {
		if info, statErr := os.Stat(p); statErr != nil || !info.IsDir() {
			missing = append(missing, p)
		}
	}
	if len(missing) > 0 {
		return Check{
			Level:   LevelWarn,
			Name:    "tool-paths",
			Message: "missing: " + strings.Join(missing, ", "),
		}
	}
	return Check{
		Level:   LevelOK,
		Name:    "tool-paths",
		Message: "common user paths are available",
	}
}

func agentConfigPath() string {
	if p := strings.TrimSpace(os.Getenv("DM_AGENT_CONFIG")); p != "" {
		return p
	}
	exe, err := os.Executable()
	if err == nil && strings.TrimSpace(exe) != "" {
		return filepath.Join(filepath.Dir(exe), "dm.agent.json")
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "dm.agent.json"
	}
	return filepath.Join(home, ".config", "dm", "agent.json")
}

func readAgentConfigMap() map[string]any {
	path := agentConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func ollamaConfig() (baseURL, model string) {
	baseURL = "http://127.0.0.1:11434"
	model = "deepseek-coder-v2:latest"
	raw := readAgentConfigMap()
	if ollama, ok := raw["ollama"].(map[string]any); ok {
		if v, ok := ollama["base_url"].(string); ok && strings.TrimSpace(v) != "" {
			baseURL = strings.TrimSpace(v)
		}
		if v, ok := ollama["model"].(string); ok && strings.TrimSpace(v) != "" {
			model = strings.TrimSpace(v)
		}
	}
	return baseURL, model
}

func openAIConfig() (baseURL, model, key string) {
	baseURL = "https://api.openai.com/v1"
	model = "gpt-4o-mini"
	key = strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	raw := readAgentConfigMap()
	if openai, ok := raw["openai"].(map[string]any); ok {
		if v, ok := openai["base_url"].(string); ok && strings.TrimSpace(v) != "" {
			baseURL = strings.TrimSpace(v)
		}
		if v, ok := openai["model"].(string); ok && strings.TrimSpace(v) != "" {
			model = strings.TrimSpace(v)
		}
		if k, ok := openai["api_key"].(string); ok && strings.TrimSpace(k) != "" {
			key = strings.TrimSpace(k)
		}
	}
	return baseURL, model, key
}
