package app

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"cli/internal/agent"
	"cli/internal/plugins"
	"cli/internal/ui"
	"cli/tools"
)

const askMaxSteps = 4
const askDecisionCacheTTL = 3 * time.Minute

const (
	riskPolicyStrict = "strict"
	riskPolicyNormal = "normal"
	riskPolicyOff    = "off"
)

type decisionCacheEntry struct {
	at    time.Time
	value agent.DecisionResult
}

type decisionCacheStore struct {
	mu    sync.RWMutex
	ttl   time.Duration
	items map[string]decisionCacheEntry
}

func newDecisionCacheStore(ttl time.Duration) *decisionCacheStore {
	return &decisionCacheStore{
		ttl:   ttl,
		items: map[string]decisionCacheEntry{},
	}
}

func (c *decisionCacheStore) Get(key string, now time.Time) (agent.DecisionResult, bool) {
	c.mu.RLock()
	entry, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return agent.DecisionResult{}, false
	}
	if now.Sub(entry.at) > c.ttl {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return agent.DecisionResult{}, false
	}
	return entry.value, true
}

func (c *decisionCacheStore) Set(key string, value agent.DecisionResult, now time.Time) {
	c.mu.Lock()
	c.items[key] = decisionCacheEntry{
		at:    now,
		value: value,
	}
	c.mu.Unlock()
}

var askDecisionCache = newDecisionCacheStore(askDecisionCacheTTL)

type askActionRecord struct {
	Step   int
	Action string
	Target string
	Args   string
	Result string
}



type askJSONStep struct {
	Step       int    `json:"step"`
	Action     string `json:"action"`
	Target     string `json:"target,omitempty"`
	Args       string `json:"args,omitempty"`
	Reason     string `json:"reason,omitempty"`
	Risk       string `json:"risk,omitempty"`
	RiskReason string `json:"risk_reason,omitempty"`
	Status     string `json:"status"`
}

type askJSONOutput struct {
	Provider string        `json:"provider,omitempty"`
	Model    string        `json:"model,omitempty"`
	Action   string        `json:"action"`
	Answer   string        `json:"answer,omitempty"`
	Steps    []askJSONStep `json:"steps,omitempty"`
	Error    string        `json:"error,omitempty"`
}

func runAskOnceWithSession(baseDir, prompt string, opts agent.AskOptions, confirmTools bool, riskPolicy string, previousPrompts []string, jsonOut bool) int {
	catalog := buildPluginCatalog(baseDir)
	toolsCatalog := buildToolsCatalog()
	history := []askActionRecord{}
	jsonResult := askJSONOutput{Action: "answer", Steps: []askJSONStep{}}
	lastSignature := ""
	for step := 1; step <= askMaxSteps; step++ {
		decisionPrompt := buildAskPlannerPrompt(prompt, history, previousPrompts)
		decision, _, err := decideWithCache(decisionPrompt, catalog, toolsCatalog, opts)
		if err != nil {
			if jsonOut {
				jsonResult.Action = "error"
				jsonResult.Error = err.Error()
				emitAskJSON(jsonResult)
			} else {
				fmt.Println("Error:", err)
			}
			return 1
		}
		jsonResult.Provider = decision.Provider
		jsonResult.Model = decision.Model
		if !jsonOut {
			fmt.Printf("[%s | %s]\n", decision.Provider, decision.Model)
		}

		if decision.Action == "answer" || strings.TrimSpace(decision.Action) == "" {
			jsonResult.Action = "answer"
			jsonResult.Answer = decision.Answer
			if jsonOut {
				emitAskJSON(jsonResult)
			} else {
				fmt.Println(decision.Answer)
			}
			return 0
		}

		sig := decisionSignature(decision)
		if sig != "" && sig == lastSignature {
			if jsonOut {
				jsonResult.Action = "answer"
				jsonResult.Answer = strings.TrimSpace(decision.Answer)
				if jsonResult.Answer == "" {
					jsonResult.Answer = "Agent repeated the same action; stopped to avoid loop."
				}
				emitAskJSON(jsonResult)
			} else {
				fmt.Println(ui.Warn("Agent repeated the same action; stopping to avoid loop."))
				if strings.TrimSpace(decision.Answer) != "" {
					fmt.Println(decision.Answer)
				}
			}
			return 0
		}
		lastSignature = sig

		if decision.Action == "run_plugin" {
			if strings.TrimSpace(decision.Plugin) == "" {
				if jsonOut {
					jsonResult.Action = "error"
					jsonResult.Error = "agent selected run_plugin without plugin name"
					emitAskJSON(jsonResult)
				} else {
					fmt.Println("Error: agent selected run_plugin without plugin name")
				}
				return 1
			}
			if _, err := plugins.GetInfo(baseDir, decision.Plugin); err != nil {
				if jsonOut {
					jsonResult.Action = "error"
					jsonResult.Error = "agent selected unknown plugin: " + decision.Plugin
					jsonResult.Answer = strings.TrimSpace(decision.Answer)
					emitAskJSON(jsonResult)
				} else {
					fmt.Println("Error: agent selected unknown plugin:", decision.Plugin)
					if strings.TrimSpace(decision.Answer) != "" {
						fmt.Println(decision.Answer)
					}
				}
				return 1
			}
			risk, riskReason := assessDecisionRisk(decision)
			if !jsonOut {
				if strings.TrimSpace(decision.Reason) != "" {
					fmt.Println("Reason:", decision.Reason)
				}
				fmt.Printf("%s %d/%d: %s\n", ui.Accent("Plan step"), step, askMaxSteps, plannedActionSummary(decision))
				fmt.Printf("%s %s (%s)\n", ui.Warn("Risk:"), strings.ToUpper(risk), riskReason)
			}
			stepRecord := askJSONStep{
				Step:       step,
				Action:     "run_plugin",
				Target:     decision.Plugin,
				Args:       strings.Join(decision.Args, " "),
				Reason:     strings.TrimSpace(decision.Reason),
				Risk:       risk,
				RiskReason: riskReason,
				Status:     "pending",
			}
			if shouldConfirmAction(confirmTools, riskPolicy, risk) {
				reader := bufio.NewReader(os.Stdin)
				if !confirmAgentAction(reader, risk) {
					stepRecord.Status = "canceled"
					jsonResult.Steps = append(jsonResult.Steps, stepRecord)
					if jsonOut {
						jsonResult.Action = "answer"
						jsonResult.Answer = strings.TrimSpace(decision.Answer)
						emitAskJSON(jsonResult)
					} else {
						fmt.Println(ui.Warn("Canceled."))
						if strings.TrimSpace(decision.Answer) != "" {
							fmt.Println(decision.Answer)
						}
					}
					return 0
				}
			}
			if err := plugins.Run(baseDir, decision.Plugin, decision.Args); err != nil {
				stepRecord.Status = "error"
				jsonResult.Steps = append(jsonResult.Steps, stepRecord)
				if jsonOut {
					jsonResult.Action = "error"
					jsonResult.Error = err.Error()
					jsonResult.Answer = strings.TrimSpace(decision.Answer)
					emitAskJSON(jsonResult)
				} else {
					printAgentActionError(err)
				}
				return 1
			}
			stepRecord.Status = "ok"
			jsonResult.Steps = append(jsonResult.Steps, stepRecord)
			history = append(history, askActionRecord{
				Step:   step,
				Action: "run_plugin",
				Target: decision.Plugin,
				Args:   strings.Join(decision.Args, " "),
				Result: "ok",
			})
			if strings.TrimSpace(decision.Answer) != "" {
				if jsonOut {
					jsonResult.Answer = decision.Answer
				} else {
					fmt.Println(decision.Answer)
				}
			}
			if step == askMaxSteps {
				if jsonOut {
					jsonResult.Action = "answer"
					if strings.TrimSpace(jsonResult.Answer) == "" {
						jsonResult.Answer = "Reached max agent steps; stopping."
					}
					emitAskJSON(jsonResult)
				} else {
					fmt.Println(ui.Warn("Reached max agent steps; stopping."))
				}
				return 0
			}
			continue
		}

		if decision.Action == "run_tool" {
			toolName := strings.TrimSpace(decision.Tool)
			if toolName == "" {
				if jsonOut {
					jsonResult.Action = "error"
					jsonResult.Error = "agent selected run_tool without tool name"
					emitAskJSON(jsonResult)
				} else {
					fmt.Println("Error: agent selected run_tool without tool name")
				}
				return 1
			}
			if !isKnownTool(toolName) {
				if jsonOut {
					jsonResult.Action = "error"
					jsonResult.Error = "agent selected unknown tool: " + toolName
					jsonResult.Answer = strings.TrimSpace(decision.Answer)
					emitAskJSON(jsonResult)
				} else {
					fmt.Println("Error: agent selected unknown tool:", toolName)
					if strings.TrimSpace(decision.Answer) != "" {
						fmt.Println(decision.Answer)
					}
				}
				return 1
			}
			risk, riskReason := assessDecisionRisk(decision)
			if !jsonOut {
				if strings.TrimSpace(decision.Reason) != "" {
					fmt.Println("Reason:", decision.Reason)
				}
				fmt.Printf("%s %d/%d: %s\n", ui.Accent("Plan step"), step, askMaxSteps, plannedActionSummary(decision))
				fmt.Printf("%s %s (%s)\n", ui.Warn("Risk:"), strings.ToUpper(risk), riskReason)
			}
			stepRecord := askJSONStep{
				Step:       step,
				Action:     "run_tool",
				Target:     toolName,
				Args:       formatToolArgs(decision.ToolArgs),
				Reason:     strings.TrimSpace(decision.Reason),
				Risk:       risk,
				RiskReason: riskReason,
				Status:     "pending",
			}
			if shouldConfirmAction(confirmTools, riskPolicy, risk) {
				reader := bufio.NewReader(os.Stdin)
				if !confirmAgentAction(reader, risk) {
					stepRecord.Status = "canceled"
					jsonResult.Steps = append(jsonResult.Steps, stepRecord)
					if jsonOut {
						jsonResult.Action = "answer"
						jsonResult.Answer = strings.TrimSpace(decision.Answer)
						emitAskJSON(jsonResult)
					} else {
						fmt.Println(ui.Warn("Canceled."))
						if strings.TrimSpace(decision.Answer) != "" {
							fmt.Println(decision.Answer)
						}
					}
					return 0
				}
			}
			run := tools.RunByNameWithParamsDetailed(baseDir, toolName, decision.ToolArgs)
			if run.Code != 0 {
				stepRecord.Status = "error"
				jsonResult.Steps = append(jsonResult.Steps, stepRecord)
				if jsonOut {
					jsonResult.Action = "error"
					jsonResult.Error = fmt.Sprintf("tool execution failed: %s", toolName)
					jsonResult.Answer = strings.TrimSpace(decision.Answer)
					emitAskJSON(jsonResult)
				}
				return run.Code
			}
			reader := bufio.NewReader(os.Stdin)
			for run.CanContinue {
				promptText := run.ContinuePrompt
				if strings.TrimSpace(promptText) == "" {
					promptText = "Show more results? [Y/n]: "
				}
				fmt.Print(ui.Prompt(promptText))
				nextChoice := strings.ToLower(strings.TrimSpace(readLine(reader)))
				if nextChoice == "n" || nextChoice == "no" {
					break
				}
				run = tools.RunByNameWithParamsDetailed(baseDir, toolName, run.ContinueParams)
				if run.Code != 0 {
					stepRecord.Status = "error"
					jsonResult.Steps = append(jsonResult.Steps, stepRecord)
					if jsonOut {
						jsonResult.Action = "error"
						jsonResult.Error = fmt.Sprintf("tool continuation failed: %s", toolName)
						jsonResult.Answer = strings.TrimSpace(decision.Answer)
						emitAskJSON(jsonResult)
					}
					return run.Code
				}
			}
			stepRecord.Status = "ok"
			jsonResult.Steps = append(jsonResult.Steps, stepRecord)
			history = append(history, askActionRecord{
				Step:   step,
				Action: "run_tool",
				Target: toolName,
				Args:   formatToolArgs(decision.ToolArgs),
				Result: "ok",
			})
			if strings.TrimSpace(decision.Answer) != "" {
				if jsonOut {
					jsonResult.Answer = decision.Answer
				} else {
					fmt.Println(decision.Answer)
				}
			}
			if step == askMaxSteps {
				if jsonOut {
					jsonResult.Action = "answer"
					if strings.TrimSpace(jsonResult.Answer) == "" {
						jsonResult.Answer = "Reached max agent steps; stopping."
					}
					emitAskJSON(jsonResult)
				} else {
					fmt.Println(ui.Warn("Reached max agent steps; stopping."))
				}
				return 0
			}
			continue
		}

		jsonResult.Action = "answer"
		jsonResult.Answer = decision.Answer
		if jsonOut {
			emitAskJSON(jsonResult)
		} else {
			fmt.Println(decision.Answer)
		}
		return 0
	}
	return 0
}

func buildAskPlannerPrompt(original string, history []askActionRecord, previousPrompts []string) string {
	base := strings.TrimSpace(original)
	if len(history) == 0 && len(previousPrompts) == 0 {
		return base
	}
	lines := []string{
		"Original user request:",
		base,
	}
	if len(previousPrompts) > 0 {
		lines = append(lines, "", "Previous prompts in this interactive session:")
		for i, p := range previousPrompts {
			if strings.TrimSpace(p) == "" {
				continue
			}
			lines = append(lines, fmt.Sprintf("- prev %d: %s", i+1, strings.TrimSpace(p)))
		}
	}
	if len(history) > 0 {
		lines = append(lines, "", "Actions already executed in this session:")
		for _, h := range history {
			line := fmt.Sprintf("- step %d: %s target=%s", h.Step, h.Action, h.Target)
			if strings.TrimSpace(h.Args) != "" {
				line += " args=" + h.Args
			}
			if strings.TrimSpace(h.Result) != "" {
				line += " result=" + h.Result
			}
			lines = append(lines, line)
		}
	}
	lines = append(lines,
		"",
		"Decide the next best step. If the task is complete, return action=answer with the final response.",
	)
	return strings.Join(lines, "\n")
}

func decideWithCache(prompt, pluginCatalog, toolCatalog string, opts agent.AskOptions) (agent.DecisionResult, bool, error) {
	key := decisionCacheKey(prompt, pluginCatalog, toolCatalog, opts)
	now := time.Now()
	if cached, ok := askDecisionCache.Get(key, now); ok {
		return cached, true, nil
	}
	decision, err := agent.DecideWithPlugins(prompt, pluginCatalog, toolCatalog, opts)
	if err != nil {
		return agent.DecisionResult{}, false, err
	}
	askDecisionCache.Set(key, decision, now)
	return decision, false, nil
}

func decisionCacheKey(prompt, pluginCatalog, toolCatalog string, opts agent.AskOptions) string {
	normalized := strings.Join([]string{
		strings.TrimSpace(prompt),
		strings.TrimSpace(pluginCatalog),
		strings.TrimSpace(toolCatalog),
		strings.ToLower(strings.TrimSpace(opts.Provider)),
		strings.TrimSpace(opts.Model),
		strings.TrimSpace(opts.BaseURL),
	}, "\n---\n")
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

func decisionSignature(decision agent.DecisionResult) string {
	switch decision.Action {
	case "run_plugin":
		return "run_plugin|" + strings.TrimSpace(decision.Plugin) + "|" + strings.Join(decision.Args, " ")
	case "run_tool":
		return "run_tool|" + strings.TrimSpace(decision.Tool) + "|" + formatToolArgs(decision.ToolArgs)
	default:
		return ""
	}
}

func runAskInteractiveWithRisk(baseDir string, opts agent.AskOptions, confirmTools bool, riskPolicy string) int {
	session, err := agent.ResolveSessionProvider(opts)
	if err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	sessionOpts := session.Options
	promptLabel := fmt.Sprintf("ask(%s,%s)> ", session.Provider, session.Model)

	fmt.Println("Ask mode. Type your question.")
	fmt.Println("Exit commands: /exit, exit, quit")
	reader := bufio.NewReader(os.Stdin)
	previousPrompts := []string{}
	for {
		fmt.Print(ui.Warn(promptLabel))
		line, readErr := reader.ReadString('\n')
		if readErr != nil && strings.TrimSpace(line) == "" {
			fmt.Println()
			return 0
		}
		prompt := strings.TrimSpace(line)
		switch strings.ToLower(prompt) {
		case "":
			continue
		case "/exit", "exit", "quit":
			return 0
		}
		_ = runAskOnceWithSession(baseDir, prompt, sessionOpts, confirmTools, riskPolicy, previousPrompts, false)
		previousPrompts = append(previousPrompts, prompt)
		if len(previousPrompts) > 6 {
			previousPrompts = previousPrompts[len(previousPrompts)-6:]
		}
	}
}

func normalizeRiskPolicy(raw string) (string, error) {
	p := strings.ToLower(strings.TrimSpace(raw))
	switch p {
	case "", riskPolicyNormal:
		return riskPolicyNormal, nil
	case riskPolicyStrict, riskPolicyOff:
		return p, nil
	default:
		return "", fmt.Errorf("invalid --risk-policy %q (use strict|normal|off)", raw)
	}
}

func shouldConfirmAction(confirmTools bool, riskPolicy, risk string) bool {
	switch riskPolicy {
	case riskPolicyOff:
		return confirmTools
	case riskPolicyStrict:
		return true
	default:
		return confirmTools || risk == "high"
	}
}

func confirmAgentAction(reader *bufio.Reader, risk string) bool {
	if risk == "high" {
		fmt.Print(ui.Prompt("Confirm HIGH risk action? [y/N]: "))
		confirm := strings.ToLower(strings.TrimSpace(readLine(reader)))
		return confirm == "y" || confirm == "yes"
	}
	fmt.Print(ui.Prompt("Confirm agent action? [Y/n]: "))
	confirm := strings.ToLower(strings.TrimSpace(readLine(reader)))
	return !(confirm == "n" || confirm == "no")
}

func assessDecisionRisk(decision agent.DecisionResult) (string, string) {
	if decision.Action == "run_tool" {
		tool := strings.ToLower(strings.TrimSpace(decision.Tool))
		switch tool {
		case "clean", "c":
			apply := strings.ToLower(strings.TrimSpace(decision.ToolArgs["apply"]))
			if apply == "1" || apply == "true" || apply == "yes" || apply == "y" {
				return "high", "delete empty directories"
			}
			return "low", "preview only"
		case "rename", "r":
			return "medium", "batch rename files"
		case "backup", "b":
			return "medium", "writes backup archive"
		default:
			return "low", "read/inspect operation"
		}
	}
	if decision.Action == "run_plugin" {
		name := strings.ToLower(strings.TrimSpace(decision.Plugin))
		if strings.Contains(name, "reset") || strings.Contains(name, "delete") || strings.Contains(name, "drop") || strings.Contains(name, "rm") {
			return "high", "plugin may perform destructive operations"
		}
		return "medium", "external plugin execution"
	}
	return "low", "response only"
}

func buildPluginCatalog(baseDir string) string {
	items, err := plugins.ListEntries(baseDir, true)
	if err != nil || len(items) == 0 {
		return "(none)"
	}
	lines := make([]string, 0, len(items))
	for _, item := range items {
		info, _ := plugins.GetInfo(baseDir, item.Name)
		line := fmt.Sprintf("- %s (%s)", item.Name, item.Kind)
		if strings.TrimSpace(info.Synopsis) != "" {
			line += ": " + info.Synopsis
		}
		if len(info.Parameters) > 0 {
			line += " | params: " + strings.Join(info.Parameters, "; ")
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func buildToolsCatalog() string {
	return strings.Join([]string{
		"- search: Search files by name/extension | tool_args: base, ext, name, sort, limit, offset",
		"- rename: Batch rename files with preview | tool_args: base, from, to, name, case_sensitive",
		"- recent: Show recent files | tool_args: base, limit, offset",
		"- backup: Create a folder zip backup | tool_args: source, output",
		"- clean: Delete empty folders | tool_args: base, apply (true for delete, otherwise preview)",
		"- system: Show system/network snapshot (no args needed)",
	}, "\n")
}

func isKnownTool(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "search", "s", "rename", "r", "recent", "rec", "backup", "b", "clean", "c", "system", "sys", "htop":
		return true
	default:
		return false
	}
}

func formatToolArgs(args map[string]string) string {
	if len(args) == 0 {
		return ""
	}
	keys := make([]string, 0, len(args))
	for k := range args {
		v := strings.TrimSpace(args[k])
		lc := strings.ToLower(v)
		if v == "" || lc == "<nil>" || lc == "null" {
			continue
		}
		keys = append(keys, k)
	}
	if len(keys) == 0 {
		return ""
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, args[k]))
	}
	return strings.Join(parts, ", ")
}

func plannedActionSummary(decision agent.DecisionResult) string {
	switch strings.ToLower(strings.TrimSpace(decision.Action)) {
	case "run_plugin":
		s := "plugin " + strings.TrimSpace(decision.Plugin)
		if len(decision.Args) > 0 {
			s += " " + strings.Join(decision.Args, " ")
		}
		return s
	case "run_tool":
		s := "tool " + strings.TrimSpace(decision.Tool)
		if args := formatToolArgs(decision.ToolArgs); strings.TrimSpace(args) != "" {
			s += " (" + args + ")"
		}
		return s
	default:
		if strings.TrimSpace(decision.Answer) != "" {
			return "answer"
		}
		return "noop"
	}
}

func emitAskJSON(v askJSONOutput) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

var missingPathErr = regexp.MustCompile(`(?i)required path '([^']+)' does not exist`)

func printAgentActionError(err error) {
	fmt.Println("Error:", err)
	combined := strings.TrimSpace(err.Error() + "\n" + plugins.ErrorOutput(err))
	m := missingPathErr.FindStringSubmatch(combined)
	if len(m) == 2 {
		fmt.Println(ui.Warn("Missing required path: " + m[1]))
		fmt.Println(ui.Muted("Fix the path in plugin variables/config, then retry."))
	}
}
