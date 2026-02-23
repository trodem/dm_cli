package tools

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"cli/internal/ui"
)

const (
	diffMaxDiffLines = 200
	diffDefaultLines = 80
)

func RunDiff(r *bufio.Reader) int {
	mode := prompt(r, "Mode (git|files)", "git")

	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "git", "":
		return printGitDiff(diffDefaultLines)
	case "files":
		a := prompt(r, "File A", "")
		b := prompt(r, "File B", "")
		if a == "" || b == "" {
			fmt.Println(ui.Error("Error:"), "both file paths are required.")
			return 1
		}
		return printFileDiff(a, b)
	default:
		fmt.Println(ui.Error("Error:"), "unknown mode. Use 'git' or 'files'.")
		return 1
	}
}

func RunDiffAutoDetailed(baseDir string, params map[string]string) AutoRunResult {
	mode := strings.ToLower(strings.TrimSpace(params["mode"]))
	if mode == "" {
		mode = "git"
	}

	limit := diffDefaultLines
	if v := strings.TrimSpace(params["limit"]); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			limit = n
		}
	}
	if limit > diffMaxDiffLines {
		limit = diffMaxDiffLines
	}

	switch mode {
	case "git":
		return AutoRunResult{Code: printGitDiff(limit)}
	case "files":
		a := strings.TrimSpace(params["file_a"])
		b := strings.TrimSpace(params["file_b"])
		if a == "" || b == "" {
			fmt.Println("Error: file_a and file_b are required for files mode.")
			return AutoRunResult{Code: 1}
		}
		return AutoRunResult{Code: printFileDiff(a, b)}
	default:
		fmt.Println("Error: unknown mode. Use 'git' or 'files'.")
		return AutoRunResult{Code: 1}
	}
}

func printGitDiff(limit int) int {
	if _, err := exec.LookPath("git"); err != nil {
		fmt.Println("Error: git is not installed or not in PATH. Install git to use this tool.")
		return 1
	}

	if !isGitRepo() {
		fmt.Println("Error: current directory is not a git repository. Navigate to a project with git init or git clone first.")
		return 1
	}

	branch := gitOneLiner("branch", "--show-current")
	if branch != "" {
		fmt.Printf("Branch: %s\n", branch)
	}

	lastCommit := gitOneLiner("log", "-1", "--format=%h %s (%cr)")
	if lastCommit != "" {
		fmt.Printf("Last commit: %s\n", lastCommit)
	}

	status := gitOutput("status", "--short")
	if strings.TrimSpace(status) == "" {
		fmt.Println("\nWorking tree clean — nothing to commit.")
		return 0
	}

	fmt.Printf("\nChanged files:\n%s\n", status)

	diffStat := gitOutput("diff", "--stat", "HEAD")
	if strings.TrimSpace(diffStat) == "" {
		diffStat = gitOutput("diff", "--stat", "--cached")
	}
	if strings.TrimSpace(diffStat) != "" {
		fmt.Printf("\nStats:\n%s\n", diffStat)
	}

	diff := gitOutput("diff", "HEAD")
	if strings.TrimSpace(diff) == "" {
		diff = gitOutput("diff", "--cached")
	}
	if strings.TrimSpace(diff) == "" {
		diff = gitOutput("diff")
	}

	if strings.TrimSpace(diff) != "" {
		lines := strings.Split(diff, "\n")
		if len(lines) > limit {
			fmt.Printf("\nDiff (first %d of %d lines):\n", limit, len(lines))
			fmt.Println(strings.Join(lines[:limit], "\n"))
			fmt.Printf("... %d more lines (use limit=%d to see more)\n", len(lines)-limit, len(lines))
		} else {
			fmt.Printf("\nDiff:\n%s\n", diff)
		}
	}

	return 0
}

func printFileDiff(fileA, fileB string) int {
	if _, err := exec.LookPath("git"); err != nil {
		fmt.Println("Error: git is not installed (needed for diff).")
		return 1
	}

	out, err := exec.Command("git", "diff", "--no-index", "--", fileA, fileB).CombinedOutput()
	result := strings.TrimSpace(string(out))
	if err != nil && result == "" {
		fmt.Printf("Error: could not diff files: %s\n", err)
		return 1
	}

	if result == "" {
		fmt.Println("Files are identical.")
		return 0
	}

	fmt.Println(result)
	return 0
}

func isGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	out, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

func gitOneLiner(args ...string) string {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func gitOutput(args ...string) string {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimRight(string(out), "\r\n")
}
