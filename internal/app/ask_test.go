package app

import (
	"strings"
	"testing"
	"time"

	"cli/internal/agent"
)


func TestBuildAskPlannerPromptWithHistory(t *testing.T) {
	history := []askActionRecord{
		{Step: 1, Action: "run_tool", Target: "search", Args: "name=report, ext=pdf", Result: "ok"},
	}
	got := buildAskPlannerPrompt("trova i pdf recenti", history, []string{"prima richiesta"})
	if !strings.Contains(got, "Original user request:") {
		t.Fatalf("expected original request section, got: %s", got)
	}
	if !strings.Contains(got, "Previous prompts in this interactive session:") {
		t.Fatalf("expected previous prompts section, got: %s", got)
	}
	if !strings.Contains(got, "step 1: run_tool target=search") {
		t.Fatalf("expected history section, got: %s", got)
	}
}

func TestDecisionSignature(t *testing.T) {
	pluginSig := decisionSignature(agent.DecisionResult{
		Action: "run_plugin",
		Plugin: "g_status",
		Args:   []string{"--short"},
	})
	if pluginSig == "" || !strings.Contains(pluginSig, "run_plugin|g_status") {
		t.Fatalf("unexpected plugin signature: %q", pluginSig)
	}
	toolSig := decisionSignature(agent.DecisionResult{
		Action:   "run_tool",
		Tool:     "search",
		ToolArgs: map[string]string{"name": "report", "ext": "pdf"},
	})
	if toolSig == "" || !strings.Contains(toolSig, "run_tool|search|") {
		t.Fatalf("unexpected tool signature: %q", toolSig)
	}
}

func TestNormalizeRiskPolicy(t *testing.T) {
	v, err := normalizeRiskPolicy("off")
	if err != nil {
		t.Fatal(err)
	}
	if v != riskPolicyOff {
		t.Fatalf("unexpected policy: %q", v)
	}
	if _, err := normalizeRiskPolicy("invalid"); err == nil {
		t.Fatal("expected error for invalid risk policy")
	}
}

func TestAssessDecisionRisk(t *testing.T) {
	risk, _ := assessDecisionRisk(agent.DecisionResult{
		Action:   "run_tool",
		Tool:     "clean",
		ToolArgs: map[string]string{"apply": "true"},
	})
	if risk != "high" {
		t.Fatalf("expected high risk, got %q", risk)
	}
}

func TestPlannedActionSummaryPlugin(t *testing.T) {
	got := plannedActionSummary(agent.DecisionResult{
		Action: "run_plugin",
		Plugin: "g_status",
		Args:   []string{"--short"},
	})
	if got != "plugin g_status --short" {
		t.Fatalf("unexpected summary: %q", got)
	}
}

func TestPlannedActionSummaryTool(t *testing.T) {
	got := plannedActionSummary(agent.DecisionResult{
		Action:   "run_tool",
		Tool:     "search",
		ToolArgs: map[string]string{"name": "report", "ext": "pdf"},
	})
	if !strings.HasPrefix(got, "tool search") {
		t.Fatalf("unexpected summary: %q", got)
	}
	if !strings.Contains(got, "name=report") || !strings.Contains(got, "ext=pdf") {
		t.Fatalf("expected tool args in summary, got %q", got)
	}
}

func TestDecisionCacheKeyStableWithTrim(t *testing.T) {
	opts := agent.AskOptions{
		Provider: " OpenAI ",
		Model:    "gpt-4o-mini",
		BaseURL:  " https://api.openai.com/v1 ",
	}
	k1 := decisionCacheKey("  test  ", "  pcat  ", "  tcat  ", opts)
	k2 := decisionCacheKey("test", "pcat", "tcat", agent.AskOptions{
		Provider: "openai",
		Model:    "gpt-4o-mini",
		BaseURL:  "https://api.openai.com/v1",
	})
	if k1 != k2 {
		t.Fatalf("expected equal cache keys, got %q != %q", k1, k2)
	}
}

func TestDecisionCacheStoreTTL(t *testing.T) {
	cache := newDecisionCacheStore(50 * time.Millisecond)
	key := "k"
	value := agent.DecisionResult{Action: "answer", Answer: "ok"}
	start := time.Now()
	cache.Set(key, value, start)

	got, ok := cache.Get(key, start.Add(10*time.Millisecond))
	if !ok || got.Answer != "ok" {
		t.Fatalf("expected cached value before ttl, got ok=%v value=%+v", ok, got)
	}

	if _, ok := cache.Get(key, start.Add(100*time.Millisecond)); ok {
		t.Fatal("expected cache miss after ttl expiration")
	}
}

func TestIsKnownTool(t *testing.T) {
	known := []string{"search", "s", "rename", "r", "recent", "rec", "backup", "b", "clean", "c", "system", "sys", "htop"}
	for _, name := range known {
		if !isKnownTool(name) {
			t.Fatalf("expected %q to be a known tool", name)
		}
	}
	unknown := []string{"e", "y", "foo", "1", ""}
	for _, name := range unknown {
		if isKnownTool(name) {
			t.Fatalf("expected %q to NOT be a known tool", name)
		}
	}
}
