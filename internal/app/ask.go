package app

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"cli/internal/agent"
	"cli/internal/plugins"
	"cli/internal/ui"
	"cli/tools"
)

const askMaxSteps = 4

type askActionRecord struct {
	Step   int
	Action string
	Target string
	Args   string
	Result string
}

func parseLegacyAskArgs(args []string) (agent.AskOptions, bool, string, error) {
	var opts agent.AskOptions
	confirmTools := true
	var promptParts []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--provider":
			if i+1 >= len(args) {
				return opts, confirmTools, "", fmt.Errorf("missing value for --provider")
			}
			opts.Provider = args[i+1]
			i++
		case "--model":
			if i+1 >= len(args) {
				return opts, confirmTools, "", fmt.Errorf("missing value for --model")
			}
			opts.Model = args[i+1]
			i++
		case "--base-url":
			if i+1 >= len(args) {
				return opts, confirmTools, "", fmt.Errorf("missing value for --base-url")
			}
			opts.BaseURL = args[i+1]
			i++
		case "--confirm-tools":
			confirmTools = true
		case "--no-confirm-tools":
			confirmTools = false
		default:
			promptParts = append(promptParts, args[i])
		}
	}
	return opts, confirmTools, strings.Join(promptParts, " "), nil
}

func runAskOnce(baseDir, prompt string, opts agent.AskOptions, confirmTools bool) int {
	return runAskOnceWithSession(baseDir, prompt, opts, confirmTools, nil)
}

func runAskOnceWithSession(baseDir, prompt string, opts agent.AskOptions, confirmTools bool, previousPrompts []string) int {
	catalog := buildPluginCatalog(baseDir)
	toolsCatalog := buildToolsCatalog()
	history := []askActionRecord{}
	lastSignature := ""
	for step := 1; step <= askMaxSteps; step++ {
		decisionPrompt := buildAskPlannerPrompt(prompt, history, previousPrompts)
		decision, err := agent.DecideWithPlugins(decisionPrompt, catalog, toolsCatalog, opts)
		if err != nil {
			fmt.Println("Error:", err)
			return 1
		}
		fmt.Printf("[%s | %s]\n", decision.Provider, decision.Model)

		if decision.Action == "answer" || strings.TrimSpace(decision.Action) == "" {
			fmt.Println(decision.Answer)
			return 0
		}

		sig := decisionSignature(decision)
		if sig != "" && sig == lastSignature {
			fmt.Println(ui.Warn("Agent repeated the same action; stopping to avoid loop."))
			if strings.TrimSpace(decision.Answer) != "" {
				fmt.Println(decision.Answer)
			}
			return 0
		}
		lastSignature = sig

		if decision.Action == "run_plugin" {
			if strings.TrimSpace(decision.Plugin) == "" {
				fmt.Println("Error: agent selected run_plugin without plugin name")
				return 1
			}
			if _, err := plugins.GetInfo(baseDir, decision.Plugin); err != nil {
				fmt.Println("Error: agent selected unknown plugin:", decision.Plugin)
				if strings.TrimSpace(decision.Answer) != "" {
					fmt.Println(decision.Answer)
				}
				return 1
			}
			if strings.TrimSpace(decision.Reason) != "" {
				fmt.Println("Reason:", decision.Reason)
			}
			fmt.Printf("Running plugin: %s", decision.Plugin)
			if len(decision.Args) > 0 {
				fmt.Printf(" %s", strings.Join(decision.Args, " "))
			}
			fmt.Println()
			if confirmTools {
				reader := bufio.NewReader(os.Stdin)
				fmt.Print(ui.Prompt("Confirm agent action? [Y/n]: "))
				confirm := strings.ToLower(strings.TrimSpace(readLine(reader)))
				if confirm == "n" || confirm == "no" {
					fmt.Println(ui.Warn("Canceled."))
					if strings.TrimSpace(decision.Answer) != "" {
						fmt.Println(decision.Answer)
					}
					return 0
				}
			}
			if err := plugins.Run(baseDir, decision.Plugin, decision.Args); err != nil {
				printAgentActionError(err)
				return 1
			}
			history = append(history, askActionRecord{
				Step:   step,
				Action: "run_plugin",
				Target: decision.Plugin,
				Args:   strings.Join(decision.Args, " "),
				Result: "ok",
			})
			if strings.TrimSpace(decision.Answer) != "" {
				fmt.Println(decision.Answer)
			}
			if step == askMaxSteps {
				fmt.Println(ui.Warn("Reached max agent steps; stopping."))
				return 0
			}
			continue
		}

		if decision.Action == "run_tool" {
			toolName := strings.TrimSpace(decision.Tool)
			if toolName == "" {
				fmt.Println("Error: agent selected run_tool without tool name")
				return 1
			}
			if !isKnownTool(toolName) {
				fmt.Println("Error: agent selected unknown tool:", toolName)
				if strings.TrimSpace(decision.Answer) != "" {
					fmt.Println(decision.Answer)
				}
				return 1
			}
			if strings.TrimSpace(decision.Reason) != "" {
				fmt.Println("Reason:", decision.Reason)
			}
			fmt.Println("Running tool:", toolName)
			if len(decision.ToolArgs) > 0 {
				fmt.Println("Tool args:", formatToolArgs(decision.ToolArgs))
			}
			if confirmTools {
				reader := bufio.NewReader(os.Stdin)
				fmt.Print(ui.Prompt("Confirm agent action? [Y/n]: "))
				confirm := strings.ToLower(strings.TrimSpace(readLine(reader)))
				if confirm == "n" || confirm == "no" {
					fmt.Println(ui.Warn("Canceled."))
					if strings.TrimSpace(decision.Answer) != "" {
						fmt.Println(decision.Answer)
					}
					return 0
				}
			}
			run := tools.RunByNameWithParamsDetailed(baseDir, toolName, decision.ToolArgs)
			if run.Code != 0 {
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
					return run.Code
				}
			}
			history = append(history, askActionRecord{
				Step:   step,
				Action: "run_tool",
				Target: toolName,
				Args:   formatToolArgs(decision.ToolArgs),
				Result: "ok",
			})
			if strings.TrimSpace(decision.Answer) != "" {
				fmt.Println(decision.Answer)
			}
			if step == askMaxSteps {
				fmt.Println(ui.Warn("Reached max agent steps; stopping."))
				return 0
			}
			continue
		}

		fmt.Println(decision.Answer)
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

func runAskInteractive(baseDir string, opts agent.AskOptions, confirmTools bool) int {
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
		_ = runAskOnceWithSession(baseDir, prompt, sessionOpts, confirmTools, previousPrompts)
		previousPrompts = append(previousPrompts, prompt)
		if len(previousPrompts) > 6 {
			previousPrompts = previousPrompts[len(previousPrompts)-6:]
		}
	}
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
		"- note: Append a quick note to a file",
		"- recent: Show recent files | tool_args: base, limit, offset",
		"- backup: Create a folder zip backup",
		"- clean: Delete empty folders | tool_args: base, apply (true for delete, otherwise preview)",
		"- system: Show system/network snapshot",
	}, "\n")
}

func isKnownTool(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "search", "s", "rename", "r", "note", "n", "recent", "rec", "e", "backup", "b", "clean", "c", "system", "sys", "htop", "y":
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
