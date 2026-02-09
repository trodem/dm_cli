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
	PrintSection("Target")
	PrintKV("Name", name)
	PrintKV("Path", targetPath)

	// base menu
	PrintSection("Actions")
	PrintMenuLine("1", "[p] Print path (for: cd $(dm ...))", true)
	PrintMenuLine("2", "[o] Open Explorer/Finder", false)
	PrintMenuLine("3", "[v] Open VS Code (code .)", false)
	PrintMenuLine("4", "[t] Open new terminal here", false)

	// if project: list actions
	actionKeys := []string{}
	if isProject && len(proj.Commands) > 0 {
		actionKeys = sortedKeys(proj.Commands)
		PrintMenuLine("5", "[a] Project actions...", false)
	}
	PrintMenuLine("0", "[x] Exit", false)

	fmt.Print("\n" + Prompt("Select option > "))
	choice := readLine()

	switch choice {
	case "1", "p", "P", "":
		// default: path
		fmt.Println(filepath.ToSlash(targetPath))
		return
	case "2", "o", "O":
		platform.OpenFileBrowser(targetPath)
		return
	case "3", "v", "V":
		platform.OpenVSCode(targetPath)
		return
	case "4", "t", "T":
		platform.OpenTerminal(targetPath)
		return
	case "5", "a", "A":
		if len(actionKeys) == 0 {
			fmt.Println(Warn("No actions defined for this project."))
			return
		}
		PrintSection("Project Actions")
		for i, a := range actionKeys {
			PrintMenuLine(fmt.Sprintf("%d", i+1), a, false)
		}
		fmt.Print("\n" + Prompt("Select action > "))
		sel := readLine()
		idx := parseIndex(sel)
		if idx < 1 || idx > len(actionKeys) {
			fmt.Println(Error("Invalid selection."))
			return
		}
		runner.RunProjectCommand(cfg, name, actionKeys[idx-1], baseDir)
		return
	case "0", "x", "X":
		return
	default:
		fmt.Println(Error("Invalid selection."))
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
