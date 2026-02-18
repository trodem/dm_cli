package app

import (
	"strings"
	"testing"

	"cli/internal/agent"
)

func TestParseLegacyAskArgs(t *testing.T) {
	opts, confirm, riskPolicy, prompt, err := parseLegacyAskArgs([]string{
		"--provider", "ollama",
		"--model", "deepseek-coder-v2:latest",
		"--base-url", "http://127.0.0.1:11434",
		"--risk-policy", "strict",
		"--no-confirm-tools",
		"spiegami", "questo", "errore",
	})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Provider != "ollama" {
		t.Fatalf("expected provider ollama, got %q", opts.Provider)
	}
	if opts.Model != "deepseek-coder-v2:latest" {
		t.Fatalf("unexpected model: %q", opts.Model)
	}
	if opts.BaseURL != "http://127.0.0.1:11434" {
		t.Fatalf("unexpected base-url: %q", opts.BaseURL)
	}
	if confirm {
		t.Fatalf("expected confirmTools=false")
	}
	if riskPolicy != riskPolicyStrict {
		t.Fatalf("expected strict risk policy, got %q", riskPolicy)
	}
	if prompt != "spiegami questo errore" {
		t.Fatalf("unexpected prompt: %q", prompt)
	}
}

func TestParseLegacyAskArgsMissingProviderValue(t *testing.T) {
	_, _, _, _, err := parseLegacyAskArgs([]string{"--provider"})
	if err == nil {
		t.Fatal("expected error for missing --provider value")
	}
}

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
