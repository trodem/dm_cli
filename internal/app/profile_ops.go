package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var psFunctionName = regexp.MustCompile(`(?i)^\s*function\s+([a-z0-9_-]+)\b`)
var psSetAlias = regexp.MustCompile(`(?i)^\s*(?:set-alias|new-alias)\s+([^\s]+)\s+([^\s#]+)`)

func resolveUserPowerShellProfilePath() string {
	home, _ := os.UserHomeDir()
	if strings.TrimSpace(home) == "" {
		return ""
	}
	ps7 := filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")
	if fileExists(ps7) {
		return ps7
	}
	winPS := filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1")
	if fileExists(winPS) {
		return winPS
	}
	return ps7
}

func showPowerShellSymbols(path, label string) int {
	if strings.TrimSpace(path) == "" {
		fmt.Println("Error: PowerShell profile path is not available.")
		return 1
	}
	funcs, aliases, err := parsePowerShellSymbols(path)
	if err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	fmt.Println("Profile:", label)
	fmt.Println("Path   :", path)
	fmt.Println()
	fmt.Println("Functions")
	fmt.Println("---------")
	if len(funcs) == 0 {
		fmt.Println("(none)")
	} else {
		for _, n := range funcs {
			fmt.Println(n)
		}
	}
	fmt.Println()
	fmt.Println("Aliases")
	fmt.Println("-------")
	if len(aliases) == 0 {
		fmt.Println("(none)")
	} else {
		for _, n := range aliases {
			fmt.Println(n)
		}
	}
	return 0
}

func parsePowerShellSymbols(path string) ([]string, []string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	lines := strings.Split(string(data), "\n")
	funcSeen := map[string]struct{}{}
	aliasSeen := map[string]struct{}{}
	funcs := []string{}
	aliases := []string{}

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if m := psFunctionName.FindStringSubmatch(line); len(m) == 2 {
			name := strings.TrimSpace(m[1])
			if name != "" {
				if _, ok := funcSeen[name]; !ok {
					funcSeen[name] = struct{}{}
					funcs = append(funcs, name)
				}
			}
			continue
		}
		if m := psSetAlias.FindStringSubmatch(line); len(m) == 3 {
			aliasName := strings.TrimSpace(m[1])
			target := strings.TrimSpace(m[2])
			if aliasName != "" {
				if _, ok := aliasSeen[aliasName]; !ok {
					aliasSeen[aliasName] = struct{}{}
					aliases = append(aliases, aliasName+" -> "+target)
				}
			}
		}
	}

	sort.Strings(funcs)
	sort.Strings(aliases)
	return funcs, aliases, nil
}

func openInNotepad(path string) error {
	cmd := exec.Command("notepad.exe", path)
	return cmd.Start()
}

func openUserPowerShellProfileInNotepad() error {
	dst := resolveUserPowerShellProfilePath()
	if strings.TrimSpace(dst) == "" {
		return fmt.Errorf("PowerShell profile path is not available")
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	if !fileExists(dst) {
		if err := os.WriteFile(dst, []byte{}, 0644); err != nil {
			return err
		}
	}
	return openInNotepad(dst)
}
