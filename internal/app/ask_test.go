package app

import (
	"os"
	"strings"
	"testing"
	"time"

	"cli/internal/agent"
)


func TestBuildAskPlannerPromptWithHistory(t *testing.T) {
	history := []askActionRecord{
		{Step: 1, Action: "run_tool", Target: "search", Args: "name=report, ext=pdf", Result: "ok"},
	}
	got := buildAskPlannerPrompt("trova i pdf recenti", history, []string{"prima richiesta"}, nil)
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

func TestNormalizeResponseMode(t *testing.T) {
	v, err := normalizeResponseMode("")
	if err != nil {
		t.Fatal(err)
	}
	if v != responseModeRawFirst {
		t.Fatalf("unexpected mode: %q", v)
	}
	v, err = normalizeResponseMode("llm-first")
	if err != nil {
		t.Fatal(err)
	}
	if v != responseModeLLMFirst {
		t.Fatalf("unexpected mode: %q", v)
	}
	if _, err := normalizeResponseMode("invalid"); err == nil {
		t.Fatal("expected error for invalid response mode")
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
	k1 := decisionCacheKey("  test  ", "  pcat  ", "  tcat  ", opts, "  ctx  ")
	k2 := decisionCacheKey("test", "pcat", "tcat", agent.AskOptions{
		Provider: "openai",
		Model:    "gpt-4o-mini",
		BaseURL:  "https://api.openai.com/v1",
	}, "ctx")
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

func TestPluginArgsToPS(t *testing.T) {
	args := pluginArgsToPS(map[string]string{
		"Host":  "server1",
		"Force": "true",
		"Port":  "8080",
		"Skip":  "false",
	})
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-Force") {
		t.Fatalf("expected -Force switch, got %q", joined)
	}
	if !strings.Contains(joined, "-Host server1") {
		t.Fatalf("expected -Host server1, got %q", joined)
	}
	if !strings.Contains(joined, "-Port 8080") {
		t.Fatalf("expected -Port 8080, got %q", joined)
	}
	if strings.Contains(joined, "-Skip") {
		t.Fatalf("expected -Skip false to be omitted, got %q", joined)
	}
}

func TestPluginArgsToPS_Empty(t *testing.T) {
	args := pluginArgsToPS(nil)
	if len(args) != 0 {
		t.Fatalf("expected empty args, got %v", args)
	}
}

func TestPluginArgsToPS_DashPrefix(t *testing.T) {
	args := pluginArgsToPS(map[string]string{
		"-Name": "value",
	})
	if len(args) != 2 || args[0] != "-Name" || args[1] != "value" {
		t.Fatalf("expected [-Name value], got %v", args)
	}
}

func TestFormatPluginArgs(t *testing.T) {
	got := formatPluginArgs(map[string]string{
		"Host": "server1",
		"Port": "8080",
	})
	if !strings.Contains(got, "-Host server1") {
		t.Fatalf("expected -Host server1, got %q", got)
	}
	if !strings.Contains(got, "-Port 8080") {
		t.Fatalf("expected -Port 8080, got %q", got)
	}
}

func TestFormatPluginArgs_Empty(t *testing.T) {
	got := formatPluginArgs(nil)
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestDecisionSignatureWithPluginArgs(t *testing.T) {
	sig := decisionSignature(agent.DecisionResult{
		Action:     "run_plugin",
		Plugin:     "sys_uptime",
		PluginArgs: map[string]string{"Host": "server1"},
	})
	if !strings.Contains(sig, "run_plugin|sys_uptime|") {
		t.Fatalf("unexpected signature: %q", sig)
	}
	if !strings.Contains(sig, "-Host server1") {
		t.Fatalf("expected -Host server1 in signature, got %q", sig)
	}
}

func TestPlannedActionSummaryPluginArgs(t *testing.T) {
	got := plannedActionSummary(agent.DecisionResult{
		Action:     "run_plugin",
		Plugin:     "sys_uptime",
		PluginArgs: map[string]string{"Host": "server1"},
	})
	if !strings.Contains(got, "plugin sys_uptime") {
		t.Fatalf("expected plugin name, got %q", got)
	}
	if !strings.Contains(got, "-Host server1") {
		t.Fatalf("expected plugin args, got %q", got)
	}
}

func TestTrimToTokenBudget_UnderBudget(t *testing.T) {
	session, prev := trimToTokenBudget("short prompt", "session data", "prev data", 20000)
	if session != "session data" || prev != "prev data" {
		t.Fatalf("expected no trimming, got session=%q prev=%q", session, prev)
	}
}

func TestTrimToTokenBudget_OverBudget(t *testing.T) {
	bigSession := strings.Repeat("x", 80001)
	session, prev := trimToTokenBudget("prompt", bigSession, "prev", 100)
	total := estimateTokens("prompt") + estimateTokens(session) + estimateTokens(prev)
	if total > 200 {
		t.Fatalf("expected trimmed total under budget, got %d tokens", total)
	}
}

func TestEstimateTokens(t *testing.T) {
	if got := estimateTokens("hello world"); got != 2 {
		t.Fatalf("expected ~2 tokens for 11 chars, got %d", got)
	}
}

func TestExtractFriendlyError_MissingParam(t *testing.T) {
	raw := `xls_sheets: C:\Temp\dm-plugin-123.ps1:11:1
Line |
  11 |  & 'xls_sheets' @dmNamedArgs @dmPositionalArgs
     |  ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
     | A parameter cannot be found that matches parameter name 'Name'.
exit status 1`
	got := extractFriendlyError(raw)
	if !strings.Contains(got, "'Name'") {
		t.Fatalf("expected friendly error about 'Name', got %q", got)
	}
}

func TestExtractFriendlyError_Mandatory(t *testing.T) {
	raw := `Cannot process command because of one or more missing mandatory parameters: Table Value.`
	got := extractFriendlyError(raw)
	if !strings.Contains(got, "Table Value") {
		t.Fatalf("expected mandatory params, got %q", got)
	}
}

func TestBuildFileContext_SingleFile(t *testing.T) {
	tmp := t.TempDir()
	path := tmp + "/test.txt"
	if err := os.WriteFile(path, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}
	ctx, err := buildFileContext([]string{path})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ctx, "hello world") {
		t.Fatalf("expected file content in context, got %q", ctx)
	}
	if !strings.Contains(ctx, "test.txt") {
		t.Fatalf("expected filename in context, got %q", ctx)
	}
}

func TestBuildFileContext_MultipleFiles(t *testing.T) {
	tmp := t.TempDir()
	p1 := tmp + "/a.txt"
	p2 := tmp + "/b.txt"
	_ = os.WriteFile(p1, []byte("alpha"), 0644)
	_ = os.WriteFile(p2, []byte("beta"), 0644)
	ctx, err := buildFileContext([]string{p1, p2})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ctx, "alpha") || !strings.Contains(ctx, "beta") {
		t.Fatalf("expected both files in context, got %q", ctx)
	}
}

func TestBuildFileContext_MissingFile(t *testing.T) {
	_, err := buildFileContext([]string{"/nonexistent/file.txt"})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestBuildFileContext_Directory(t *testing.T) {
	tmp := t.TempDir()
	_, err := buildFileContext([]string{tmp})
	if err == nil {
		t.Fatal("expected error for directory")
	}
}

func TestBuildFileContext_TooLarge(t *testing.T) {
	tmp := t.TempDir()
	path := tmp + "/big.bin"
	data := make([]byte, fileContextMaxBytes+1)
	_ = os.WriteFile(path, data, 0644)
	_, err := buildFileContext([]string{path})
	if err == nil {
		t.Fatal("expected error for oversized file")
	}
}

func TestIsKnownTool(t *testing.T) {
	known := []string{"search", "s", "rename", "r", "recent", "rec", "clean", "c", "system", "sys", "htop", "e", "y"}
	for _, name := range known {
		if !isKnownTool(name) {
			t.Fatalf("expected %q to be a known tool", name)
		}
	}
	unknown := []string{"foo", ""}
	for _, name := range unknown {
		if isKnownTool(name) {
			t.Fatalf("expected %q to NOT be a known tool", name)
		}
	}
}
