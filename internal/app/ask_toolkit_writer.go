package app

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"cli/internal/agent"
	"cli/internal/plugins"
)

var functionsIndexRe = regexp.MustCompile(`(?m)^#\s+FUNCTIONS\s*$`)

func listToolkitSummaries(baseDir string) []agent.ToolkitSummary {
	fnFiles, err := plugins.ListFunctionFiles(baseDir)
	if err != nil {
		return nil
	}
	summaries := make([]agent.ToolkitSummary, 0, len(fnFiles))
	for _, ff := range fnFiles {
		label := toolkitLabel(toolkitGroupKey(ff.Path))
		prefix := derivePrefix(ff.Functions)
		summaries = append(summaries, agent.ToolkitSummary{
			FilePath:  ff.Path,
			Label:     label,
			Prefix:    prefix,
			Functions: ff.Functions,
		})
	}
	return summaries
}

func derivePrefix(functions []string) string {
	if len(functions) == 0 {
		return ""
	}
	first := functions[0]
	idx := strings.Index(first, "_")
	if idx < 0 {
		return first
	}
	candidate := first[:idx]
	for _, fn := range functions[1:] {
		if !strings.HasPrefix(fn, candidate+"_") {
			return candidate
		}
	}
	longerIdx := strings.Index(functions[0][idx+1:], "_")
	if longerIdx >= 0 {
		longer := first[:idx+1+longerIdx]
		allMatch := true
		for _, fn := range functions {
			if !strings.HasPrefix(fn, longer+"_") {
				allMatch = false
				break
			}
		}
		if allMatch {
			return longer
		}
	}
	return candidate
}

func appendFunctionToToolkit(filePath, functionCode string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read toolkit file: %w", err)
	}
	text := string(content)
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	text += "\n" + strings.TrimSpace(functionCode) + "\n"
	return os.WriteFile(filePath, []byte(text), 0644)
}

func updateToolkitFunctionsIndex(filePath, functionName string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read toolkit file: %w", err)
	}
	text := string(content)
	loc := functionsIndexRe.FindStringIndex(text)
	if loc == nil {
		return nil
	}
	insertAfter := loc[1]
	lastFnLine := insertAfter
	for i := insertAfter; i < len(text); i++ {
		if text[i] == '\n' {
			line := strings.TrimSpace(text[lastFnLine:i])
			if strings.HasPrefix(line, "#") && strings.Contains(line, "=") && !strings.HasPrefix(line, "#   ") {
				break
			}
			if strings.HasPrefix(line, "#   ") || line == "#" || line == "" {
				lastFnLine = i + 1
				continue
			}
			break
		}
	}
	newEntry := fmt.Sprintf("#   %s\n", functionName)
	updated := text[:lastFnLine] + newEntry + text[lastFnLine:]
	return os.WriteFile(filePath, []byte(updated), 0644)
}

func createNewToolkit(pluginsDir, toolkitName, prefix, functionCode string) (string, error) {
	fileName := toolkitName + "_Toolkit.ps1"
	filePath := filepath.Join(pluginsDir, fileName)

	fnName := ""
	for _, line := range strings.Split(functionCode, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "function ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				fnName = strings.TrimSuffix(parts[1], "{")
				fnName = strings.TrimSpace(fnName)
			}
			break
		}
	}

	upperName := strings.ToUpper(strings.ReplaceAll(toolkitName, "_", " "))
	header := fmt.Sprintf(`# =============================================================================
# %s TOOLKIT – Auto-generated toolkit (standalone)
# Safety: Review generated functions before use.
# Entry point: %s_*
#
# FUNCTIONS
#   %s
# =============================================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# -----------------------------------------------------------------------------
# Internal helpers
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Ensure a command is available in PATH.
.PARAMETER Name
Command name to validate.
.EXAMPLE
_assert_command_available -Name docker
#>
function _assert_command_available {
    param([Parameter(Mandatory = $true)][string]$Name)
    if (-not (Get-Command -Name $Name -ErrorAction SilentlyContinue)) {
        throw "Required command '$Name' was not found in PATH."
    }
}

<#
.SYNOPSIS
Ensure a filesystem path exists.
.PARAMETER Path
Path to validate.
.EXAMPLE
_assert_path_exists -Path C:\Data
#>
function _assert_path_exists {
    param([Parameter(Mandatory = $true)][string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) {
        throw "Required path '$Path' does not exist."
    }
}

# -----------------------------------------------------------------------------
# Public functions
# -----------------------------------------------------------------------------

`, upperName, prefix, fnName)

	fullContent := header + strings.TrimSpace(functionCode) + "\n"
	return filePath, os.WriteFile(filePath, []byte(fullContent), 0644)
}

func validatePowerShellSyntax(code string) error {
	pwsh, err := exec.LookPath("pwsh")
	if err != nil {
		return nil
	}
	cmd := exec.Command(pwsh, "-NoProfile", "-Command", "[scriptblock]::Create($input)")
	cmd.Stdin = bytes.NewReader([]byte(code))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if runErr := cmd.Run(); runErr != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = runErr.Error()
		}
		return fmt.Errorf("PowerShell syntax error:\n%s", msg)
	}
	return nil
}
