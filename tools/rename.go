package tools

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"cli/internal/renamer"
	"cli/internal/ui"
)

func RunRename(baseDir string, r *bufio.Reader) int {
	cleanBase := normalizeInputPath(prompt(r, "Base path", currentWorkingDir(baseDir)), currentWorkingDir(baseDir))
	if err := validateExistingDir(cleanBase, "base path"); err != nil {
		fmt.Println(ui.Error("Error:"), err)
		fmt.Println(ui.Muted("Hint: use '.' for current dir or '..' for parent dir."))
		return 1
	}
	opts := renamer.Options{
		BasePath:      cleanBase,
		NamePart:      prompt(r, "Name contains (optional)", ""),
		From:          prompt(r, "Replace from", ""),
		To:            prompt(r, "Replace to (empty = delete)", ""),
		Recursive:     true,
		UseRegex:      false,
		CaseSensitive: strings.ToLower(strings.TrimSpace(prompt(r, "Case sensitive for replace? (y/N)", "N"))) == "y",
	}

	if strings.TrimSpace(opts.From) == "" {
		fmt.Println("Error: replace-from is required.")
		return 1
	}

	plan, err := renamer.BuildPlan(opts)
	if err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	if len(plan) == 0 {
		fmt.Println("No files to rename.")
		return 0
	}

	fmt.Println("\nPreview:")
	for _, item := range plan {
		fmt.Printf("%s -> %s\n", item.OldPath, item.NewPath)
	}

	confirm := prompt(r, "Proceed? [y/N]", "N")
	if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
		fmt.Println(ui.Warn("Canceled."))
		return 0
	}

	if err := renamer.ApplyPlan(plan); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	fmt.Println("Done.")
	return 0
}

func RunRenameAutoDetailed(baseDir string, params map[string]string) AutoRunResult {
	reader := bufio.NewReader(os.Stdin)
	cwd := currentWorkingDir(baseDir)

	base, ok := params["base"]
	if !ok || strings.TrimSpace(base) == "" {
		base = prompt(reader, "Base path", cwd)
	}
	base = normalizeInputPath(base, cwd)
	if err := validateExistingDir(base, "base path"); err != nil {
		fmt.Println(ui.Error("Error:"), err)
		fmt.Println(ui.Muted("Hint: use '.' for current dir or '..' for parent dir."))
		return AutoRunResult{Code: 1}
	}

	from, ok := params["from"]
	if !ok || strings.TrimSpace(from) == "" {
		from = prompt(reader, "Replace from", "")
	}
	from = strings.TrimSpace(from)
	if from == "" {
		fmt.Println("Error: replace-from is required.")
		return AutoRunResult{Code: 1}
	}

	namePart := strings.TrimSpace(params["name"])
	if _, has := params["name"]; !has {
		namePart = prompt(reader, "Name contains (optional)", "")
	}

	to, hasTo := params["to"]
	if !hasTo {
		to = prompt(reader, "Replace to (empty = delete)", "")
	}

	caseSensitive := false
	if rawCase, has := params["case_sensitive"]; has {
		v := strings.ToLower(strings.TrimSpace(rawCase))
		caseSensitive = v == "1" || v == "true" || v == "yes" || v == "y"
	} else {
		caseSensitive = strings.ToLower(strings.TrimSpace(prompt(reader, "Case sensitive for replace? (y/N)", "N"))) == "y"
	}

	opts := renamer.Options{
		BasePath:      base,
		NamePart:      namePart,
		From:          from,
		To:            to,
		Recursive:     true,
		UseRegex:      false,
		CaseSensitive: caseSensitive,
	}

	plan, err := renamer.BuildPlan(opts)
	if err != nil {
		fmt.Println("Error:", err)
		return AutoRunResult{Code: 1}
	}
	if len(plan) == 0 {
		fmt.Println("No files to rename.")
		return AutoRunResult{Code: 0}
	}

	fmt.Println("\nPreview:")
	for _, item := range plan {
		fmt.Printf("%s -> %s\n", item.OldPath, item.NewPath)
	}

	confirm := prompt(reader, "Apply these renames? [y/N]", "N")
	if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
		fmt.Println(ui.Warn("Canceled."))
		return AutoRunResult{Code: 0}
	}

	if err := renamer.ApplyPlan(plan); err != nil {
		fmt.Println("Error:", err)
		return AutoRunResult{Code: 1}
	}
	fmt.Println("Done.")
	return AutoRunResult{Code: 0}
}
