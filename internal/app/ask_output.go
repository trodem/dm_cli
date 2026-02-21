package app

import (
	"encoding/json"
	"fmt"
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
	Finalize()
}

type askTTYWriter struct{}

func (w *askTTYWriter) ProviderInfo(provider, model string) {
	fmt.Printf("[%s | %s]\n", provider, model)
}

func (w *askTTYWriter) StepInfo(step, maxSteps int, summary, reason, risk, riskReason string) {
	if strings.TrimSpace(reason) != "" {
		fmt.Println("Reason:", reason)
	}
	fmt.Printf("%s %d/%d: %s\n", ui.Accent("Plan step"), step, maxSteps, summary)
	fmt.Printf("%s %s (%s)\n", ui.Warn("Risk:"), strings.ToUpper(risk), riskReason)
}

func (w *askTTYWriter) Answer(answer string) {
	fmt.Println(answer)
}

func (w *askTTYWriter) PartialAnswer(answer string) {
	if strings.TrimSpace(answer) != "" {
		fmt.Println(answer)
	}
}

func (w *askTTYWriter) Error(msg string) {
	fmt.Println("Error:", msg)
}

func (w *askTTYWriter) ErrorWithAnswer(msg, answer string) {
	fmt.Println("Error:", msg)
	if strings.TrimSpace(answer) != "" {
		fmt.Println(answer)
	}
}

func (w *askTTYWriter) Canceled(answer string) {
	fmt.Println(ui.Warn("Canceled."))
	if strings.TrimSpace(answer) != "" {
		fmt.Println(answer)
	}
}

func (w *askTTYWriter) MaxStepsReached(_ string) {
	fmt.Println(ui.Warn("Reached max agent steps; stopping."))
}

func (w *askTTYWriter) LoopDetected(answer string) {
	fmt.Println(ui.Warn("Agent repeated the same action; stopping to avoid loop."))
	if strings.TrimSpace(answer) != "" {
		fmt.Println(answer)
	}
}

func (w *askTTYWriter) AddStep(_ askJSONStep) {}

func (w *askTTYWriter) Finalize() {}

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

func (w *askJSONWriter) Finalize() {
	w.emit()
}

func (w *askJSONWriter) emit() {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(w.result)
}
