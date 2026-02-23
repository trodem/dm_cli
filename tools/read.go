package tools

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"cli/internal/ui"
)

const (
	readDefaultLimit = 100
	readMaxLimit     = 500
	readMaxFileBytes = 256 * 1024 // 256 KB
)

func RunRead(r *bufio.Reader) int {
	path := prompt(r, "File path", "")
	if strings.TrimSpace(path) == "" {
		fmt.Println(ui.Error("Error:"), "file path is required.")
		return 1
	}
	path = normalizeInputPath(path, currentWorkingDir("."))

	offsetStr := prompt(r, "Start line (default 1)", "1")
	offset, _ := strconv.Atoi(offsetStr)
	if offset < 1 {
		offset = 1
	}

	limitStr := prompt(r, fmt.Sprintf("Max lines (default %d)", readDefaultLimit), strconv.Itoa(readDefaultLimit))
	limit, _ := strconv.Atoi(limitStr)
	if limit < 1 {
		limit = readDefaultLimit
	}

	return printFileContents(path, offset, limit)
}

func RunReadAutoDetailed(baseDir string, params map[string]string) AutoRunResult {
	raw := strings.TrimSpace(params["path"])
	if raw == "" {
		fmt.Println("Error: path is required.")
		return AutoRunResult{Code: 1}
	}
	path := resolveReadPath(raw, baseDir)

	offset := 1
	if v := strings.TrimSpace(params["offset"]); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			offset = n
		}
	}

	limit := readDefaultLimit
	if v := strings.TrimSpace(params["limit"]); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			limit = n
		}
	}
	if limit > readMaxLimit {
		limit = readMaxLimit
	}

	code := printFileContents(path, offset, limit)
	return AutoRunResult{Code: code}
}

func printFileContents(path string, startLine, limit int) int {
	info, err := os.Stat(path)
	if err != nil {
		fmt.Printf("Error: file not found: %s\n", path)
		return 1
	}
	if info.IsDir() {
		entries, readErr := os.ReadDir(path)
		if readErr != nil {
			fmt.Printf("Error: cannot read directory: %s\n", path)
			return 1
		}
		fmt.Printf("Directory: %s (%d entries)\n", path, len(entries))
		shown := 0
		for _, e := range entries {
			if shown >= limit {
				fmt.Printf("... and %d more entries\n", len(entries)-shown)
				break
			}
			kind := "file"
			if e.IsDir() {
				kind = "dir "
			}
			fi, _ := e.Info()
			size := ""
			if fi != nil && !e.IsDir() {
				size = formatReadSize(fi.Size())
			}
			fmt.Printf("  %s  %-40s %s\n", kind, e.Name(), size)
			shown++
		}
		return 0
	}

	if info.Size() > readMaxFileBytes {
		fmt.Printf("Error: file too large (%s, max %s): %s\n",
			formatReadSize(info.Size()), formatReadSize(readMaxFileBytes), path)
		return 1
	}

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("Error: cannot read file: %s\n", err)
		return 1
	}

	if !utf8.Valid(data) {
		fmt.Printf("Error: file appears to be binary: %s\n", path)
		return 1
	}

	lines := strings.Split(string(data), "\n")
	totalLines := len(lines)

	if startLine > totalLines {
		fmt.Printf("File has %d lines, start line %d is beyond end.\n", totalLines, startLine)
		return 0
	}

	from := startLine - 1
	to := from + limit
	if to > totalLines {
		to = totalLines
	}
	window := lines[from:to]

	fmt.Printf("File: %s (%d lines total, showing %d-%d)\n", filepath.Base(path), totalLines, startLine, from+len(window))
	for i, line := range window {
		lineNum := from + i + 1
		fmt.Printf("%4d | %s\n", lineNum, line)
	}

	remaining := totalLines - to
	if remaining > 0 {
		fmt.Printf("... %d more lines (use offset=%d to continue)\n", remaining, to+1)
	}
	return 0
}

func resolveReadPath(raw, baseDir string) string {
	p := strings.TrimSpace(raw)
	p = strings.Trim(p, `"'`)

	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}

	home, _ := os.UserHomeDir()
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
		if home != "" {
			return filepath.Clean(filepath.Join(home, p[2:]))
		}
	}

	cwd := currentWorkingDir(baseDir)
	return filepath.Clean(filepath.Join(cwd, p))
}

func formatReadSize(n int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
	)
	switch {
	case n >= mb:
		return fmt.Sprintf("%.1fMB", float64(n)/float64(mb))
	case n >= kb:
		return fmt.Sprintf("%.1fKB", float64(n)/float64(kb))
	default:
		return fmt.Sprintf("%dB", n)
	}
}
