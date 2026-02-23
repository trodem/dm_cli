package ui

import (
	"fmt"
	"os"
	"strings"
)

func PrintSection(title string) {
	fmt.Println()
	fmt.Println(Accent("== " + title + " =="))
}

func PrintKV(label, value string) {
	label = strings.TrimSpace(label)
	if label == "" {
		label = "value"
	}
	fmt.Printf("%-12s %s\n", label+":", value)
}

func Accent(text string) string {
	return colorize("36", text) // cyan
}

func OK(text string) string {
	return colorize("32", text) // green
}

func Warn(text string) string {
	return colorize("33", text) // yellow
}

func Error(text string) string {
	return colorize("31", text) // red
}

func Muted(text string) string {
	return colorize("90", text) // gray
}

func Prompt(text string) string {
	return colorize("96", text) // bright cyan
}

func colorize(code, text string) string {
	if !supportsColor() {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

func supportsColor() bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	term := strings.ToLower(strings.TrimSpace(os.Getenv("TERM")))
	return term != "dumb"
}
