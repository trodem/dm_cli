package search

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func InKnowledge(knowledgeDir, query string) {
	if query == "" {
		fmt.Println("Uso: tellme find <query>")
		return
	}
	if knowledgeDir == "" {
		fmt.Println("Knowledge path non configurato.")
		return
	}

	fmt.Printf("\nfind: %s\n", query)
	fmt.Println("------------------------")

	if tryRipgrep(knowledgeDir, query) {
		fmt.Println("------------------------")
		fmt.Println()
		return
	}

	q := strings.ToLower(query)

	_ = filepath.Walk(knowledgeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			return nil
		}

		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		lines := strings.Split(string(b), "\n")
		for i, line := range lines {
			if strings.Contains(strings.ToLower(line), q) {
				fmt.Printf("%s:%d: %s\n", info.Name(), i+1, line)
			}
		}
		return nil
	})

	fmt.Println("------------------------")
	fmt.Println()
}

func tryRipgrep(knowledgeDir, query string) bool {
	if _, err := exec.LookPath("rg"); err != nil {
		return false
	}

	cmd := exec.Command("rg", "--no-heading", "--line-number", "--smart-case", query, knowledgeDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}
