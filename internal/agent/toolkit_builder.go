package agent

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ToolkitSummary struct {
	FilePath  string
	Label     string
	Prefix    string
	Functions []string
}

type BuilderRequest struct {
	FunctionDescription string
	ExistingToolkits    []ToolkitSummary
	UserRequest         string
}

type BuilderResult struct {
	FunctionName string `json:"function_name"`
	FunctionCode string `json:"function_code"`
	TargetFile   string `json:"target_file"`
	IsNewToolkit bool   `json:"is_new_toolkit"`
	NewPrefix    string `json:"new_prefix"`
	Explanation  string `json:"explanation"`
}

const toolkitConventions = `PowerShell Toolkit Conventions (MUST follow):

NAMING:
- Public functions: <prefix>_<action>, all lowercase (e.g. excel_sheets, pdf_pages)
- The prefix must be short, unique, and domain-descriptive
- Private helpers start with _ (e.g. _assert_command_available)

HELP BLOCKS (mandatory on every function):
<#
.SYNOPSIS
One-line summary.
.DESCRIPTION
More detail if synopsis is not self-explanatory.
.PARAMETER ParamName
Description.
.EXAMPLE
prefix_action -ParamName value
#>

PARAMETERS:
- Mark mandatory params: [Parameter(Mandatory = $true)]
- Use [ValidateSet(...)] for closed domains

RETURN VALUES:
- Return [pscustomobject]@{ ... } for multi-field outputs
- Never use Write-Host — return values directly

CODING STYLE:
- Use Test-Path -LiteralPath (not -Path)
- Place $null on the left: $null -eq $x
- Use throw for errors, not Write-Host
- Use Set-StrictMode -Version Latest and $ErrorActionPreference = "Stop"

GUARD HELPERS (define inside the toolkit, not imported):
- _assert_command_available -Name <tool>: check external tool exists in PATH
- _assert_path_exists -Path <path>: check filesystem path exists

STANDALONE:
- Every toolkit must be fully self-contained, zero cross-file dependencies
- Define its own guard helpers internally

TOOLKIT FILE STRUCTURE (for new toolkits):
1. Header banner (bordered with = lines) with name, safety, entry point, FUNCTIONS index
2. Set-StrictMode + $ErrorActionPreference
3. Internal helpers (prefixed with _)
4. Public functions`

func BuildFunction(req BuilderRequest, opts AskOptions) (BuilderResult, error) {
	var toolkitInfo strings.Builder
	for _, tk := range req.ExistingToolkits {
		toolkitInfo.WriteString(fmt.Sprintf("- File: %s | Prefix: %s_ | Functions: %s\n",
			tk.FilePath, tk.Prefix, strings.Join(tk.Functions, ", ")))
	}

	existingPrefixes := make([]string, 0, len(req.ExistingToolkits))
	for _, tk := range req.ExistingToolkits {
		existingPrefixes = append(existingPrefixes, tk.Prefix+"_*")
	}

	prompt := strings.Join([]string{
		"You are a PowerShell toolkit builder for a CLI assistant.",
		"Your job is to generate a PowerShell function that follows strict conventions.",
		"",
		toolkitConventions,
		"",
		"EXISTING TOOLKITS:",
		toolkitInfo.String(),
		"",
		"RESERVED PREFIXES (do NOT reuse):",
		strings.Join(existingPrefixes, ", "),
		"",
		"USER REQUEST:",
		req.UserRequest,
		"",
		"FUNCTION NEEDED:",
		req.FunctionDescription,
		"",
		"INSTRUCTIONS:",
		"1. Decide if this function fits in an existing toolkit or needs a new one.",
		"2. If it fits an existing toolkit, set target_file to that toolkit's file path and is_new_toolkit=false.",
		"3. If a new toolkit is needed, choose a short unique prefix and set is_new_toolkit=true.",
		"   For new toolkits, set target_file to the suggested file name (e.g. Excel_Toolkit.ps1).",
		"4. Generate ONLY the function code (with help block above it).",
		"   Do NOT include the full toolkit boilerplate (headers, strict mode, helpers) — only the function itself.",
		"   If the function needs guard helpers (_assert_command_available, _assert_path_exists), include them as separate functions BEFORE the main function.",
		"5. The function MUST return [pscustomobject] for structured output.",
		"",
		"Return ONLY valid JSON with this schema:",
		`{"function_name":"prefix_action","function_code":"<complete PowerShell code>","target_file":"path","is_new_toolkit":false,"new_prefix":"","explanation":"brief explanation"}`,
		"",
		"IMPORTANT: In function_code, use \\n for newlines. The code must be syntactically valid PowerShell.",
	}, "\n")

	raw, err := AskWithOptions(prompt, opts)
	if err != nil {
		return BuilderResult{}, fmt.Errorf("builder LLM call failed: %w", err)
	}

	result, err := parseBuilderJSON(raw.Text)
	if err != nil {
		return BuilderResult{}, fmt.Errorf("failed to parse builder response: %w", err)
	}

	if strings.TrimSpace(result.FunctionName) == "" {
		return BuilderResult{}, fmt.Errorf("builder returned empty function name")
	}
	if strings.TrimSpace(result.FunctionCode) == "" {
		return BuilderResult{}, fmt.Errorf("builder returned empty function code")
	}

	return result, nil
}

func parseBuilderJSON(text string) (BuilderResult, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return BuilderResult{}, fmt.Errorf("empty builder response")
	}
	payload := trimmed
	if !strings.HasPrefix(payload, "{") {
		m := jsonBlockRe.FindString(trimmed)
		if m == "" {
			return BuilderResult{}, fmt.Errorf("no json object found in builder response")
		}
		payload = m
	}
	var result BuilderResult
	if err := json.Unmarshal([]byte(payload), &result); err != nil {
		return BuilderResult{}, err
	}
	result.FunctionCode = strings.ReplaceAll(result.FunctionCode, "\\n", "\n")
	return result, nil
}
