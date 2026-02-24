package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func runAskPowerShellBuiltin(command string) int {
	cmdText := strings.TrimSpace(command)
	if cmdText == "" {
		fmt.Fprintln(os.Stderr, "Error: missing PowerShell command")
		return 1
	}
	if isCD, target := parsePowerShellCDCommand(cmdText); isCD {
		if strings.TrimSpace(target) == "" {
			fmt.Println(currentDirOrDot())
			return 0
		}
		cleanTarget := strings.Trim(strings.TrimSpace(target), "\"'")
		if cleanTarget == "" {
			fmt.Fprintln(os.Stderr, "Error: missing directory path")
			return 1
		}
		if !filepath.IsAbs(cleanTarget) {
			cleanTarget = filepath.Join(currentDirOrDot(), cleanTarget)
		}
		cleanTarget = filepath.Clean(cleanTarget)
		info, statErr := os.Stat(cleanTarget)
		if statErr != nil {
			fmt.Fprintln(os.Stderr, "Error: directory not found:", cleanTarget)
			return 1
		}
		if !info.IsDir() {
			fmt.Fprintln(os.Stderr, "Error: path is not a directory:", cleanTarget)
			return 1
		}
		if chErr := os.Chdir(cleanTarget); chErr != nil {
			fmt.Fprintln(os.Stderr, "Error: cannot change directory:", chErr)
			return 1
		}
		fmt.Println(currentDirOrDot())
		return 0
	}

	psExe := "powershell"
	if _, err := exec.LookPath("pwsh"); err == nil {
		psExe = "pwsh"
	} else if _, err := exec.LookPath("powershell"); err != nil {
		fmt.Fprintln(os.Stderr, "Error: cannot find PowerShell executable (pwsh or powershell)")
		return 1
	}

	cmd := exec.Command(psExe, "-NoProfile", "-Command", cmdText)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintln(os.Stderr, "Error:", err)
		return 1
	}
	return 0
}

func parsePowerShellCDCommand(raw string) (bool, string) {
	s := strings.TrimSpace(raw)
	lc := strings.ToLower(s)
	switch {
	case lc == "cd" || lc == "/cd":
		return true, ""
	case strings.HasPrefix(lc, "cd "):
		return true, strings.TrimSpace(s[3:])
	case strings.HasPrefix(lc, "/cd "):
		return true, strings.TrimSpace(s[4:])
	default:
		return false, ""
	}
}

func currentDirOrDot() string {
	wd, err := os.Getwd()
	if err != nil || strings.TrimSpace(wd) == "" {
		return "."
	}
	return wd
}
