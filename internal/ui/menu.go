package ui

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cli/internal/config"
	"cli/internal/platform"
	"cli/internal/runner"
)

func ShowMenu(cfg config.Config, name, targetPath, baseDir string) {
	isProject := false
	var proj config.Project
	if p, ok := cfg.Projects[name]; ok {
		isProject = true
		proj = p
	}

	fmt.Println()
	fmt.Printf("Target: %s\n", name)
	fmt.Printf("Path:   %s\n\n", targetPath)

	// menu base
	fmt.Println("Scegli azione:")
	fmt.Println("  1) Print path (per cd $(dm ...))")
	fmt.Println("  2) Open Explorer/Finder")
	fmt.Println("  3) Open VS Code (code .)")
	fmt.Println("  4) Open new terminal here")

	// se progetto: lista azioni
	actionKeys := []string{}
	if isProject && len(proj.Commands) > 0 {
		actionKeys = sortedKeys(proj.Commands)
		fmt.Println("  5) Project actions...")
	}

	fmt.Print("\n> ")
	choice := readLine()

	switch choice {
	case "1", "":
		// default: path
		fmt.Println(filepath.ToSlash(targetPath))
		return
	case "2":
		platform.OpenFileBrowser(targetPath)
		return
	case "3":
		platform.OpenVSCode(targetPath)
		return
	case "4":
		platform.OpenTerminal(targetPath)
		return
	case "5":
		if len(actionKeys) == 0 {
			fmt.Println("Nessuna action definita per questo progetto.")
			return
		}
		fmt.Println("\nAzioni disponibili:")
		for i, a := range actionKeys {
			fmt.Printf("  %d) %s\n", i+1, a)
		}
		fmt.Print("\n> ")
		sel := readLine()
		idx := parseIndex(sel)
		if idx < 1 || idx > len(actionKeys) {
			fmt.Println("Scelta non valida.")
			return
		}
		runner.RunProjectCommand(cfg, name, actionKeys[idx-1], baseDir)
		return
	default:
		fmt.Println("Scelta non valida.")
		return
	}
}

func readLine() string {
	r := bufio.NewReader(os.Stdin)
	s, _ := r.ReadString('\n')
	return strings.TrimSpace(s)
}

func parseIndex(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0
		}
		n = n*10 + int(ch-'0')
	}
	return n
}
