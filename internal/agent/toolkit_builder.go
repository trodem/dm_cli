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

const toolkitConventions = `PowerShell Toolkit Conventions (MUST follow strictly):

NAMING:
- Public functions: <prefix>_<action>, all lowercase (e.g. excel_sheets, pdf_pages)
- The prefix must be short, unique, and domain-descriptive
- Parameter names MUST use PascalCase (e.g. $FilePath, $SheetName — NOT $file_path)

REQUIRED FUNCTION STRUCTURE — every function MUST follow this exact pattern:

<#
.SYNOPSIS
One-line summary of what the function does.
.DESCRIPTION
More detail when the synopsis alone is not enough.
.PARAMETER FilePath
Description of this parameter.
.EXAMPLE
prefix_action -FilePath "C:\data\file.xlsx"
#>
function prefix_action {
    param(
        [Parameter(Mandatory = $true)]
        [string]$FilePath
    )

    _assert_path_exists -Path $FilePath

    # ... function body ...

    return [pscustomobject]@{
        Result = $value
    }
}

CRITICAL RULES:
- Parameters MUST be inside a param() block
- Do NOT place [Parameter()] attributes outside param()
- Do NOT use trailing commas after the last parameter in param()
- Do NOT use Write-Host — return values directly
- Use throw for errors
- Use Test-Path -LiteralPath (not -Path)
- Place $null on the left of comparisons: $null -eq $x
- Return [pscustomobject]@{ ... } for structured output

GUARD HELPERS — the toolkit boilerplate already provides these:
- _assert_command_available -Name <tool>
- _assert_path_exists -Path <path>
You can CALL these helpers in your function. Do NOT redefine them.`

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
		"   Do NOT include toolkit boilerplate (headers, Set-StrictMode, $ErrorActionPreference, guard helpers).",
		"   The boilerplate is added automatically. You only write the function(s).",
		"5. Do NOT define _assert_command_available or _assert_path_exists — they already exist.",
		"   You may CALL them inside your function body.",
		"6. The function MUST return [pscustomobject] for structured output.",
		"7. Parameters MUST be inside a param() block. NEVER place [Parameter()] outside param().",
		"8. Parameter names MUST be PascalCase (e.g. $FilePath, not $file_path).",
		"9. Do NOT put a trailing comma after the last parameter in param().",
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
	m := jsonBlockRe.FindString(trimmed)
	if m == "" {
		return BuilderResult{}, fmt.Errorf("no json object found in builder response")
	}
	var result BuilderResult
	if err := json.Unmarshal([]byte(m), &result); err != nil {
		return BuilderResult{}, err
	}
	result.FunctionCode = strings.ReplaceAll(result.FunctionCode, "\\n", "\n")
	return result, nil
}
