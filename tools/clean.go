package tools

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cli/internal/ui"
)

func RunCleanEmpty(r *bufio.Reader) int {
	base := prompt(r, "Base path", currentWorkingDir("."))
	if strings.TrimSpace(base) == "" {
		fmt.Println("Error: base path is required.")
		return 1
	}

	dirs, err := findEmptyDirs(base)
	if err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	if len(dirs) == 0 {
		fmt.Println("No empty folders found.")
		return 0
	}

	fmt.Println("\nEmpty folders:")
	for _, d := range dirs {
		fmt.Println(d)
	}

	confirm := prompt(r, "Delete these folders? [y/N]", "N")
	if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
		fmt.Println(ui.Warn("Canceled."))
		return 0
	}

	for _, d := range dirs {
		_ = os.Remove(d)
	}
	fmt.Println("Done.")
	return 0
}

func findEmptyDirs(base string) ([]string, error) {
	var dirs []string
	err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		if path == base {
			return nil
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil
		}
		if len(entries) == 0 {
			dirs = append(dirs, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// remove deepest first
	sort.Slice(dirs, func(i, j int) bool {
		return len(dirs[i]) > len(dirs[j])
	})
	return dirs, nil
}
