package tools

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func RunQuickNote(baseDir string, r *bufio.Reader) int {
	defaultPath := filepath.Join(currentWorkingDir(baseDir), "inbox.md")
	notePath := prompt(r, "Note file", defaultPath)
	notePath = normalizeInputPath(notePath, defaultPath)
	text := prompt(r, "Note", "")
	if strings.TrimSpace(text) == "" {
		fmt.Println("Error: note is empty.")
		return 1
	}

	if err := os.MkdirAll(filepath.Dir(notePath), 0755); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	line := fmt.Sprintf("- %s %s\n", time.Now().Format("2006-01-02 15:04"), text)
	f, err := os.OpenFile(notePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	defer f.Close()
	if _, err := f.WriteString(line); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	fmt.Println("Saved:", notePath)
	return 0
}
