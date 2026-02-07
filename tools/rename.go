package tools

import (
	"bufio"
	"fmt"
	"strings"

	"cli/internal/renamer"
)

func RunRename(baseDir string, r *bufio.Reader) int {
	opts := renamer.Options{
		BasePath:  prompt(r, "Base path", baseDir),
		NamePart:  prompt(r, "Name contains (optional)", ""),
		From:      prompt(r, "Replace from", ""),
		To:        prompt(r, "Replace to (empty = delete)", ""),
		Recursive: true,
		UseRegex:  false,
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
		fmt.Println("Canceled.")
		return 0
	}

	if err := renamer.ApplyPlan(plan); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	fmt.Println("Done.")
	return 0
}
