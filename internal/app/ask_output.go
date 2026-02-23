package app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"cli/internal/ui"
)

type askOutputWriter interface {
	ProviderInfo(provider, model string)
	StepInfo(step, maxSteps int, summary, reason, risk, riskReason string)
	Answer(answer string)
	PartialAnswer(answer string)
	Error(msg string)
	ErrorWithAnswer(msg, answer string)
	Canceled(answer string)
	MaxStepsReached(answer string)
	LoopDetected(answer string)
	AddStep(step askJSONStep)
}

type askTTYWriter struct {
	providerShown bool
}

func (w *askTTYWriter) ProviderInfo(provider, model string) {
	if !w.providerShown {
		slog.Debug("provider", "name", provider, "model", model)
	}
}

func (w *askTTYWriter) StepInfo(step, maxSteps int, summary, reason, risk, riskReason string) {
	slog.Debug("agent step",
		"step", fmt.Sprintf("%d/%d", step, maxSteps),
		"reason", reason,
		"risk", risk,
		"risk_reason", riskReason,
	)
	fmt.Println()
	fmt.Printf("  %s %s\n", ui.Accent(">"), humanizeSummary(summary))
	if strings.ToLower(risk) != "low" {
		riskLabel := ui.Warn(strings.ToUpper(risk))
		if strings.ToLower(risk) == "high" {
			riskLabel = ui.Error(strings.ToUpper(risk))
		}
		fmt.Printf("  %s %s\n", ui.Muted("Risk:"), riskLabel)
	}
}

func (w *askTTYWriter) Answer(answer string) {
	fmt.Println()
	fmt.Println(ui.RenderMarkdown(answer))
}

func (w *askTTYWriter) PartialAnswer(answer string) {
	if strings.TrimSpace(answer) != "" {
		fmt.Println()
		fmt.Println(ui.Muted(ui.RenderMarkdown(answer)))
	}
}

func (w *askTTYWriter) Error(msg string) {
	fmt.Println()
	fmt.Println(ui.Error("Error: " + msg))
}

func (w *askTTYWriter) ErrorWithAnswer(msg, answer string) {
	fmt.Println()
	fmt.Println(ui.Error("Error: " + msg))
	if strings.TrimSpace(answer) != "" {
		fmt.Println(ui.RenderMarkdown(answer))
	}
}

func (w *askTTYWriter) Canceled(answer string) {
	fmt.Println()
	fmt.Println(ui.Warn("Canceled."))
	if strings.TrimSpace(answer) != "" {
		fmt.Println(ui.RenderMarkdown(answer))
	}
}

func (w *askTTYWriter) MaxStepsReached(_ string) {
	fmt.Println()
	fmt.Println(ui.Warn("Reached max steps."))
}

func (w *askTTYWriter) LoopDetected(answer string) {
	fmt.Println()
	fmt.Println(ui.Warn("Stopped to avoid repeated action."))
	if strings.TrimSpace(answer) != "" {
		fmt.Println(ui.RenderMarkdown(answer))
	}
}

func (w *askTTYWriter) AddStep(_ askJSONStep) {}

func humanizeSummary(summary string) string {
	if strings.HasPrefix(summary, "plugin ") {
		rest := strings.TrimPrefix(summary, "plugin ")
		parts := strings.SplitN(rest, " ", 2)
		name := parts[0]
		label := "Running " + ui.Accent(name)
		if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
			label += " " + ui.Muted(parts[1])
		}
		return label
	}
	if strings.HasPrefix(summary, "tool ") {
		rest := strings.TrimPrefix(summary, "tool ")
		parts := strings.SplitN(rest, " ", 2)
		name := parts[0]
		label := "Running tool " + ui.Accent(name)
		if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
			label += " " + ui.Muted(parts[1])
		}
		return label
	}
	if strings.HasPrefix(summary, "create function:") {
		desc := strings.TrimPrefix(summary, "create function: ")
		return "Creating new function: " + ui.Accent(desc)
	}
	return summary
}

type askJSONWriter struct {
	result askJSONOutput
}

func newAskJSONWriter() *askJSONWriter {
	return &askJSONWriter{
		result: askJSONOutput{Action: "answer", Steps: []askJSONStep{}},
	}
}

func (w *askJSONWriter) ProviderInfo(provider, model string) {
	w.result.Provider = provider
	w.result.Model = model
}

func (w *askJSONWriter) StepInfo(_, _ int, _, _, _, _ string) {}

func (w *askJSONWriter) Answer(answer string) {
	w.result.Action = "answer"
	w.result.Answer = answer
	w.emit()
}

func (w *askJSONWriter) PartialAnswer(answer string) {
	if strings.TrimSpace(answer) != "" {
		w.result.Answer = answer
	}
}

func (w *askJSONWriter) Error(msg string) {
	w.result.Action = "error"
	w.result.Error = msg
	w.emit()
}

func (w *askJSONWriter) ErrorWithAnswer(msg, answer string) {
	w.result.Action = "error"
	w.result.Error = msg
	w.result.Answer = strings.TrimSpace(answer)
	w.emit()
}

func (w *askJSONWriter) Canceled(answer string) {
	w.result.Action = "answer"
	w.result.Answer = strings.TrimSpace(answer)
	w.emit()
}

func (w *askJSONWriter) MaxStepsReached(answer string) {
	w.result.Action = "answer"
	if strings.TrimSpace(answer) != "" {
		w.result.Answer = answer
	}
	if strings.TrimSpace(w.result.Answer) == "" {
		w.result.Answer = "Reached max agent steps; stopping."
	}
	w.emit()
}

func (w *askJSONWriter) LoopDetected(answer string) {
	w.result.Action = "answer"
	w.result.Answer = strings.TrimSpace(answer)
	if w.result.Answer == "" {
		w.result.Answer = "Agent repeated the same action; stopped to avoid loop."
	}
	w.emit()
}

func (w *askJSONWriter) AddStep(step askJSONStep) {
	w.result.Steps = append(w.result.Steps, step)
}

func (w *askJSONWriter) emit() {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(w.result)
}
