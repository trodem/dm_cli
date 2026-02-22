package app

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cli/internal/agent"
	"cli/internal/plugins"
	"cli/internal/ui"
	"cli/tools"
)

const askMaxSteps = 4
const askDecisionCacheTTL = 3 * time.Minute
const askHistoryMaxLen = 2000
const askPreviousPromptsMax = 6
const askDescMaxLen = 80

const (
	riskPolicyStrict = "strict"
	riskPolicyNormal = "normal"
	riskPolicyOff    = "off"
)

type askActionRecord struct {
	Step   int
	Action string
	Target string
	Args   string
	Result string
}

type askSessionParams struct {
	baseDir         string
	prompt          string
	opts            agent.AskOptions
	confirmTools    bool
	riskPolicy      string
	previousPrompts []string
	jsonOut         bool
	catalog         string
	toolsCatalog    string
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

type askStepContext struct {
	baseDir      string
	prompt       string
	opts         agent.AskOptions
	confirmTools bool
	riskPolicy   string
	jsonOut      bool
	step         int
	out          askOutputWriter
	history      *[]askActionRecord
	catalog      *string
}

func runAskOnceWithSession(p askSessionParams) int {
	catalog := p.catalog
	toolsCatalog := p.toolsCatalog
	if catalog == "" {
		catalog = buildPluginCatalog(p.baseDir)
	}
	if toolsCatalog == "" {
		toolsCatalog = buildToolsCatalog()
	}
	envContext := buildEnvContext()
	history := []askActionRecord{}

	var out askOutputWriter
	if p.jsonOut {
		out = newAskJSONWriter()
	} else {
		out = &askTTYWriter{}
	}

	lastSignature := ""
	for step := 1; step <= askMaxSteps; step++ {
		decisionPrompt := buildAskPlannerPrompt(p.prompt, history, p.previousPrompts)
		decision, _, err := decideWithCache(decisionPrompt, catalog, toolsCatalog, p.opts, envContext)
		if err != nil {
			out.Error(err.Error())
			return 1
		}
		out.ProviderInfo(decision.Provider, decision.Model)

		if decision.Action == "answer" || strings.TrimSpace(decision.Action) == "" {
			out.Answer(decision.Answer)
			return 0
		}

		sig := decisionSignature(decision)
		if sig != "" && sig == lastSignature {
			out.LoopDetected(decision.Answer)
			return 0
		}
		lastSignature = sig

		ctx := askStepContext{
			baseDir:      p.baseDir,
			prompt:       p.prompt,
			opts:         p.opts,
			confirmTools: p.confirmTools,
			riskPolicy:   p.riskPolicy,
			jsonOut:      p.jsonOut,
			step:         step,
			out:          out,
			history:      &history,
			catalog:      &catalog,
		}

		var shouldContinue bool
		var exitCode int

		switch decision.Action {
		case "run_plugin":
			shouldContinue, exitCode = handleRunPlugin(ctx, decision)
		case "run_tool":
			shouldContinue, exitCode = handleRunTool(ctx, decision)
		case "create_function":
			shouldContinue, exitCode = handleCreateFunction(ctx, decision)
		default:
			out.Answer(decision.Answer)
			return 0
		}

		if !shouldContinue {
			return exitCode
		}
	}
	return 0
}

func handleRunPlugin(ctx askStepContext, decision agent.DecisionResult) (bool, int) {
	if strings.TrimSpace(decision.Plugin) == "" {
		ctx.out.Error("agent selected run_plugin without plugin name")
		return false, 1
	}
	if _, err := plugins.GetInfo(ctx.baseDir, decision.Plugin); err != nil {
		ctx.out.ErrorWithAnswer("agent selected unknown plugin: "+decision.Plugin, decision.Answer)
		return false, 1
	}

	var runArgs []string
	var argsDisplay string
	if len(decision.PluginArgs) > 0 {
		runArgs = pluginArgsToPS(decision.PluginArgs)
		argsDisplay = formatPluginArgs(decision.PluginArgs)
	} else {
		runArgs = decision.Args
		argsDisplay = strings.Join(decision.Args, " ")
	}

	risk, riskReason := assessDecisionRisk(decision)
	ctx.out.StepInfo(ctx.step, askMaxSteps, plannedActionSummary(decision), decision.Reason, risk, riskReason)

	stepRecord := askJSONStep{
		Step: ctx.step, Action: "run_plugin", Target: decision.Plugin,
		Args: argsDisplay, Reason: strings.TrimSpace(decision.Reason),
		Risk: risk, RiskReason: riskReason, Status: "pending",
	}

	if shouldConfirmAction(ctx.confirmTools, ctx.riskPolicy, risk) {
		reader := bufio.NewReader(os.Stdin)
		if !confirmAgentAction(reader, risk) {
			stepRecord.Status = "canceled"
			ctx.out.AddStep(stepRecord)
			ctx.out.Canceled(decision.Answer)
			return false, 0
		}
	}

	runResult := plugins.RunWithOutput(ctx.baseDir, decision.Plugin, runArgs)
	if runResult.Err != nil {
		stepRecord.Status = "error"
		ctx.out.AddStep(stepRecord)
		if ctx.jsonOut {
			ctx.out.ErrorWithAnswer(runResult.Err.Error(), decision.Answer)
			return false, 1
		}
		printAgentActionError(runResult.Err)
		errOutput := truncateForHistory(runResult.Output, askHistoryMaxLen)
		errMsg := runResult.Err.Error()
		if errOutput != "" {
			errMsg += "\n" + errOutput
		}
		*ctx.history = append(*ctx.history, askActionRecord{
			Step: ctx.step, Action: "run_plugin", Target: decision.Plugin,
			Args: argsDisplay, Result: "error: " + truncateForHistory(errMsg, askHistoryMaxLen),
		})
		return true, 0
	}

	stepRecord.Status = "ok"
	ctx.out.AddStep(stepRecord)
	capturedOutput := truncateForHistory(runResult.Output, askHistoryMaxLen)
	historyResult := "ok"
	if capturedOutput != "" {
		historyResult = "ok; output:\n" + capturedOutput
	}
	*ctx.history = append(*ctx.history, askActionRecord{
		Step: ctx.step, Action: "run_plugin", Target: decision.Plugin,
		Args: argsDisplay, Result: historyResult,
	})
	ctx.out.PartialAnswer(decision.Answer)
	if ctx.step == askMaxSteps {
		ctx.out.MaxStepsReached(decision.Answer)
		return false, 0
	}
	return true, 0
}

func handleRunTool(ctx askStepContext, decision agent.DecisionResult) (bool, int) {
	toolName := strings.TrimSpace(decision.Tool)
	if toolName == "" {
		ctx.out.Error("agent selected run_tool without tool name")
		return false, 1
	}
	if !isKnownTool(toolName) {
		ctx.out.ErrorWithAnswer("agent selected unknown tool: "+toolName, decision.Answer)
		return false, 1
	}

	risk, riskReason := assessDecisionRisk(decision)
	ctx.out.StepInfo(ctx.step, askMaxSteps, plannedActionSummary(decision), decision.Reason, risk, riskReason)

	stepRecord := askJSONStep{
		Step: ctx.step, Action: "run_tool", Target: toolName,
		Args: formatToolArgs(decision.ToolArgs), Reason: strings.TrimSpace(decision.Reason),
		Risk: risk, RiskReason: riskReason, Status: "pending",
	}

	if shouldConfirmAction(ctx.confirmTools, ctx.riskPolicy, risk) {
		reader := bufio.NewReader(os.Stdin)
		if !confirmAgentAction(reader, risk) {
			stepRecord.Status = "canceled"
			ctx.out.AddStep(stepRecord)
			ctx.out.Canceled(decision.Answer)
			return false, 0
		}
	}

	run := tools.RunByNameWithParamsDetailed(ctx.baseDir, toolName, decision.ToolArgs)
	if run.Code != 0 {
		stepRecord.Status = "error"
		ctx.out.AddStep(stepRecord)
		if ctx.jsonOut {
			ctx.out.ErrorWithAnswer(fmt.Sprintf("tool execution failed: %s", toolName), decision.Answer)
			return false, run.Code
		}
		*ctx.history = append(*ctx.history, askActionRecord{
			Step: ctx.step, Action: "run_tool", Target: toolName,
			Args: formatToolArgs(decision.ToolArgs),
			Result: fmt.Sprintf("error: tool execution failed (exit code %d)", run.Code),
		})
		return true, 0
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
		run = tools.RunByNameWithParamsDetailed(ctx.baseDir, toolName, run.ContinueParams)
		if run.Code != 0 {
			stepRecord.Status = "error"
			ctx.out.AddStep(stepRecord)
			if ctx.jsonOut {
				ctx.out.ErrorWithAnswer(fmt.Sprintf("tool continuation failed: %s", toolName), decision.Answer)
				return false, run.Code
			}
			*ctx.history = append(*ctx.history, askActionRecord{
				Step: ctx.step, Action: "run_tool", Target: toolName,
				Args:   formatToolArgs(decision.ToolArgs),
				Result: fmt.Sprintf("error: tool continuation failed (exit code %d)", run.Code),
			})
			break
		}
	}

	stepRecord.Status = "ok"
	ctx.out.AddStep(stepRecord)
	*ctx.history = append(*ctx.history, askActionRecord{
		Step: ctx.step, Action: "run_tool", Target: toolName,
		Args: formatToolArgs(decision.ToolArgs), Result: "ok",
	})
	ctx.out.PartialAnswer(decision.Answer)
	if ctx.step == askMaxSteps {
		ctx.out.MaxStepsReached(decision.Answer)
		return false, 0
	}
	return true, 0
}

func handleCreateFunction(ctx askStepContext, decision agent.DecisionResult) (bool, int) {
	if ctx.jsonOut {
		ctx.out.Answer("create_function is not supported in JSON mode")
		return false, 0
	}
	desc := strings.TrimSpace(decision.FunctionDescription)
	if desc == "" {
		ctx.out.Error("agent proposed create_function but provided no description")
		return false, 1
	}
	ctx.out.StepInfo(ctx.step, askMaxSteps, plannedActionSummary(decision), decision.Reason, "HIGH", "generates and writes new code")

	reader := bufio.NewReader(os.Stdin)
	fmt.Print(ui.Prompt("Create a new function? [y/N]: "))
	confirm1 := strings.ToLower(strings.TrimSpace(readLine(reader)))
	if confirm1 != "y" && confirm1 != "yes" {
		ctx.out.Canceled(decision.Answer)
		return false, 0
	}

	fmt.Println(ui.Muted("Generating function..."))
	summaries := listToolkitSummaries(ctx.baseDir)
	builderReq := agent.BuilderRequest{
		FunctionDescription: desc,
		ExistingToolkits:    summaries,
		UserRequest:         ctx.prompt,
	}
	built, buildErr := agent.BuildFunction(builderReq, ctx.opts)
	if buildErr != nil {
		ctx.out.Error("generating function: " + buildErr.Error())
		return false, 1
	}
	if valErr := validatePowerShellSyntax(built.FunctionCode); valErr != nil {
		fmt.Println(ui.Warn("Generated code has syntax errors:"))
		fmt.Println(valErr)
		fmt.Println(ui.Muted("Aborting — code will NOT be written to disk."))
		*ctx.history = append(*ctx.history, askActionRecord{
			Step: ctx.step, Action: "create_function", Target: built.FunctionName,
			Result: "syntax validation failed: " + valErr.Error(),
		})
		return true, 0
	}

	fmt.Println()
	fmt.Println(ui.Accent("=== Generated function: " + built.FunctionName + " ==="))
	fmt.Println(built.FunctionCode)
	fmt.Println(ui.Accent("=== End of generated code ==="))
	fmt.Println()
	if strings.TrimSpace(built.Explanation) != "" {
		fmt.Println(ui.Muted("Explanation: " + built.Explanation))
	}
	if built.IsNewToolkit {
		fmt.Println(ui.Muted("Target: new toolkit " + built.TargetFile + " (prefix: " + built.NewPrefix + "_*)"))
	} else {
		fmt.Println(ui.Muted("Target: " + built.TargetFile))
	}
	fmt.Println()
	fmt.Print(ui.Prompt("Approve and write this code? [y/N]: "))
	confirm2 := strings.ToLower(strings.TrimSpace(readLine(reader)))
	if confirm2 != "y" && confirm2 != "yes" {
		fmt.Println(ui.Warn("Code not written. Canceled."))
		return false, 0
	}

	pluginsDir := filepath.Join(ctx.baseDir, "plugins")
	needsNewToolkit := built.IsNewToolkit
	targetPath := built.TargetFile
	if !needsNewToolkit {
		if !filepath.IsAbs(targetPath) {
			targetPath = filepath.Join(pluginsDir, targetPath)
		}
		if _, statErr := os.Stat(targetPath); os.IsNotExist(statErr) {
			needsNewToolkit = true
			fmt.Println(ui.Muted("Target file not found, creating new toolkit instead."))
		}
	}
	if needsNewToolkit {
		baseName := filepath.Base(built.TargetFile)
		toolkitName := strings.TrimSuffix(baseName, "_Toolkit.ps1")
		toolkitName = strings.TrimSuffix(toolkitName, ".ps1")
		prefix := built.NewPrefix
		if prefix == "" {
			prefix = derivePrefix([]string{built.FunctionName})
		}
		writtenPath, writeErr := createNewToolkit(pluginsDir, toolkitName, prefix, built.FunctionCode)
		if writeErr != nil {
			ctx.out.Error("writing toolkit: " + writeErr.Error())
			return false, 1
		}
		fmt.Println(ui.Accent("Created: " + writtenPath))
	} else {
		if err := appendFunctionToToolkit(targetPath, built.FunctionCode); err != nil {
			ctx.out.Error("writing function: " + err.Error())
			return false, 1
		}
		_ = updateToolkitFunctionsIndex(targetPath, built.FunctionName)
		fmt.Println(ui.Accent("Added " + built.FunctionName + " to " + targetPath))
	}

	*ctx.catalog = buildPluginCatalog(ctx.baseDir)
	*ctx.history = append(*ctx.history, askActionRecord{
		Step: ctx.step, Action: "create_function", Target: built.FunctionName,
		Result: "ok; function created",
	})
	fmt.Println(ui.Muted("Plugin catalog updated. Running the new function..."))
	return true, 0
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

func buildEnvContext() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return "- Working directory: " + cwd
}

func decisionSignature(decision agent.DecisionResult) string {
	switch decision.Action {
	case "run_plugin":
		argsPart := formatPluginArgs(decision.PluginArgs)
		if argsPart == "" {
			argsPart = strings.Join(decision.Args, " ")
		}
		return "run_plugin|" + strings.TrimSpace(decision.Plugin) + "|" + argsPart
	case "run_tool":
		return "run_tool|" + strings.TrimSpace(decision.Tool) + "|" + formatToolArgs(decision.ToolArgs)
	case "create_function":
		return "create_function|" + strings.TrimSpace(decision.FunctionDescription)
	default:
		return ""
	}
}

func runAskInteractiveWithRisk(baseDir string, opts agent.AskOptions, confirmTools bool, riskPolicy string, initialPrompt string) int {
	session, err := agent.ResolveSessionProvider(opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return 1
	}
	sessionOpts := session.Options
	promptLabel := fmt.Sprintf("ask(%s,%s)> ", session.Provider, session.Model)

	catalog := buildPluginCatalog(baseDir)
	toolsCatalog := buildToolsCatalog()

	fmt.Println("Ask mode. Type your question.")
	fmt.Println("Exit commands: /exit, exit, quit")
	reader := bufio.NewReader(os.Stdin)
	previousPrompts := []string{}

	if strings.TrimSpace(initialPrompt) != "" {
		fmt.Printf("%s%s\n", ui.Warn(promptLabel), initialPrompt)
		_ = runAskOnceWithSession(askSessionParams{
			baseDir: baseDir, prompt: initialPrompt, opts: sessionOpts,
			confirmTools: confirmTools, riskPolicy: riskPolicy,
			previousPrompts: previousPrompts, catalog: catalog, toolsCatalog: toolsCatalog,
		})
		previousPrompts = append(previousPrompts, initialPrompt)
	}

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
		_ = runAskOnceWithSession(askSessionParams{
			baseDir: baseDir, prompt: prompt, opts: sessionOpts,
			confirmTools: confirmTools, riskPolicy: riskPolicy,
			previousPrompts: previousPrompts, catalog: catalog, toolsCatalog: toolsCatalog,
		})
		previousPrompts = append(previousPrompts, prompt)
		if len(previousPrompts) > askPreviousPromptsMax {
			previousPrompts = previousPrompts[len(previousPrompts)-askPreviousPromptsMax:]
		}
	}
}
