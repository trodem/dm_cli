package tools

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"cli/internal/filesearch"
	"cli/internal/platform"
	"cli/internal/ui"
)

func RunSearch(r *bufio.Reader) int {
	base := prompt(r, "Base path", currentWorkingDir("."))
	base = normalizeInputPath(base, currentWorkingDir("."))
	if strings.TrimSpace(base) == "" {
		fmt.Println("Error: base path is required.")
		return 1
	}
	if err := validateExistingDir(base, "base path"); err != nil {
		fmt.Println(ui.Error("Error:"), err)
		fmt.Println(ui.Muted("Hint: use '.' for current dir or '..' for parent dir."))
		return 1
	}
	name := prompt(r, "Name contains", "")
	ext := prompt(r, "Extension (optional)", "")
	sortBy := prompt(r, "Sort (name|date|size)", "name")

	results, err := filesearch.Find(filesearch.Options{
		BasePath: base,
		NamePart: name,
		Ext:      ext,
		SortBy:   sortBy,
	})
	if err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	if len(results) == 0 {
		fmt.Println("No files found.")
		return 0
	}
	for i, item := range results {
		idx := ui.Warn(fmt.Sprintf("%2d)", i+1))
		fmt.Printf("%s %s | %s | %s\n", idx, item.ModTime.Format("2006-01-02 15:04"), filesearch.FormatSize(item.Size), item.Path)
	}

	selection := prompt(r, "Select result to open (number, Enter to skip)", "")
	if strings.TrimSpace(selection) == "" {
		return 0
	}
	idx, ok := parseSelectionIndex(selection, len(results))
	if !ok {
		fmt.Println(ui.Error("Invalid selection."))
		return 1
	}
	platform.OpenFile(results[idx].Path)
	return 0
}

func RunSearchAuto(baseDir string, params map[string]string) int {
	return RunSearchAutoDetailed(baseDir, params).Code
}

func RunSearchAutoDetailed(baseDir string, params map[string]string) AutoRunResult {
	base := strings.TrimSpace(params["base"])
	if base == "" {
		base = currentWorkingDir(baseDir)
	}
	base = normalizeAgentPath(base, baseDir)
	name := strings.TrimSpace(params["name"])
	ext := strings.TrimSpace(params["ext"])
	sortBy := strings.TrimSpace(params["sort"])
	if sortBy == "" {
		sortBy = "name"
	}
	limit := 10
	if rawLimit := strings.TrimSpace(params["limit"]); rawLimit != "" {
		if n, err := strconv.Atoi(rawLimit); err == nil && n > 0 {
			limit = n
		}
	}
	offset := 0
	if rawOffset := strings.TrimSpace(params["offset"]); rawOffset != "" {
		if n, err := strconv.Atoi(rawOffset); err == nil && n >= 0 {
			offset = n
		}
	}
	shown, total, code := runSearchQuery(base, name, ext, sortBy, offset, limit)
	if code != 0 {
		return AutoRunResult{Code: code}
	}
	nextOffset := offset + shown
	if nextOffset < total {
		next := copyStringMap(params)
		next["offset"] = strconv.Itoa(nextOffset)
		next["limit"] = strconv.Itoa(limit)
		return AutoRunResult{
			Code:           0,
			CanContinue:    true,
			ContinuePrompt: fmt.Sprintf("Show next %d search results? [Y/n]: ", limit),
			ContinueParams: next,
		}
	}
	return AutoRunResult{Code: 0}
}

func runSearchQuery(base, name, ext, sortBy string, offset, limit int) (int, int, int) {
	results, err := filesearch.Find(filesearch.Options{
		BasePath: base,
		NamePart: name,
		Ext:      ext,
		SortBy:   sortBy,
	})
	if err != nil {
		fmt.Println("Error:", err)
		return 0, 0, 1
	}
	if len(results) == 0 {
		fmt.Println("No files found.")
		return 0, 0, 0
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= len(results) {
		fmt.Println("No more files.")
		return 0, len(results), 0
	}

	show := results[offset:]
	if limit > 0 && len(show) > limit {
		show = show[:limit]
	}
	start := offset + 1
	end := offset + len(show)
	fmt.Printf("Showing %d-%d of %d results\n", start, end, len(results))
	for i, item := range show {
		idx := ui.Warn(fmt.Sprintf("%2d)", offset+i+1))
		fmt.Printf("%s %s | %s | %s\n", idx, item.ModTime.Format("2006-01-02 15:04"), filesearch.FormatSize(item.Size), item.Path)
	}
	if limit > 0 && len(results) > limit {
		remaining := len(results) - end
		if remaining > 0 {
			fmt.Println(ui.Muted(fmt.Sprintf("... and %d more", remaining)))
		}
	}
	return len(show), len(results), 0
}

func normalizeAgentPath(raw, fallbackBaseDir string) string {
	p := strings.TrimSpace(raw)
	if p == "" {
		return normalizeInputPath(currentWorkingDir(fallbackBaseDir), currentWorkingDir(fallbackBaseDir))
	}
	lc := strings.ToLower(strings.ReplaceAll(p, "\\", "/"))
	home, _ := os.UserHomeDir()
	switch lc {
	case "downloads", "~/downloads":
		if strings.TrimSpace(home) != "" {
			return filepath.Join(home, "Downloads")
		}
	case "desktop", "~/desktop":
		if strings.TrimSpace(home) != "" {
			return filepath.Join(home, "Desktop")
		}
	case "documents", "~/documents":
		if strings.TrimSpace(home) != "" {
			return filepath.Join(home, "Documents")
		}
	}
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
		if strings.TrimSpace(home) != "" {
			p = filepath.Join(home, p[2:])
		}
	}
	return normalizeInputPath(p, currentWorkingDir(fallbackBaseDir))
}

func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func parseSelectionIndex(raw string, max int) (int, bool) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return -1, false
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return -1, false
	}
	if n < 1 || n > max {
		return -1, false
	}
	return n - 1, true
}
