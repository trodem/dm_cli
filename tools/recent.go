package tools

import (
	"bufio"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"cli/internal/filesearch"
	"cli/internal/ui"
)

type recentItem struct {
	Path    string
	ModTime time.Time
	Size    int64
}

func RunRecent(r *bufio.Reader) int {
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
	limitStr := prompt(r, "Limit", "20")
	limit, err := strconv.Atoi(strings.TrimSpace(limitStr))
	if err != nil || limit <= 0 {
		fmt.Println("Error: invalid limit.")
		return 1
	}

	_, _, code := runRecentQuery(base, 0, limit)
	return code
}

func RunRecentAuto(baseDir string, params map[string]string) int {
	return RunRecentAutoDetailed(baseDir, params).Code
}

func RunRecentAutoDetailed(baseDir string, params map[string]string) AutoRunResult {
	base := strings.TrimSpace(params["base"])
	if base == "" {
		base = currentWorkingDir(baseDir)
	}
	base = normalizeAgentPath(base, baseDir)
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
	shown, total, code := runRecentQuery(base, offset, limit)
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
			ContinuePrompt: fmt.Sprintf("Show next %d recent files? [Y/n]: ", limit),
			ContinueParams: next,
		}
	}
	return AutoRunResult{Code: 0}
}

func runRecentQuery(base string, offset, limit int) (int, int, int) {
	items, err := collectRecent(base)
	if err != nil {
		fmt.Println("Error:", err)
		return 0, 0, 1
	}
	if len(items) == 0 {
		fmt.Println("No files found.")
		return 0, 0, 0
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ModTime.After(items[j].ModTime)
	})
	if offset < 0 {
		offset = 0
	}
	if offset >= len(items) {
		fmt.Println("No more files.")
		return 0, len(items), 0
	}
	show := items[offset:]
	if len(show) > limit {
		show = show[:limit]
	}
	start := offset + 1
	end := offset + len(show)
	fmt.Printf("Showing %d-%d of %d files\n", start, end, len(items))

	for _, it := range show {
		fmt.Printf("%s | %s | %s\n", it.ModTime.Format("2006-01-02 15:04"), filesearch.FormatSize(it.Size), it.Path)
	}
	if len(items) > end {
		fmt.Println(ui.Muted(fmt.Sprintf("... and %d more", len(items)-end)))
	}
	return len(show), len(items), 0
}

func collectRecent(base string) ([]recentItem, error) {
	var items []recentItem
	err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		items = append(items, recentItem{
			Path:    path,
			ModTime: info.ModTime(),
			Size:    info.Size(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return items, nil
}
