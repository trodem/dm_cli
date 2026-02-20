package tools

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cli/internal/ui"
)

type toolMenuItem struct {
	Key      string
	Name     string
	Synopsis string
}

type AutoRunResult struct {
	Code           int
	CanContinue    bool
	ContinuePrompt string
	ContinueParams map[string]string
}

var toolMenuItems = []toolMenuItem{
	{Key: "s", Name: "search", Synopsis: "Search files by name/extension"},
	{Key: "r", Name: "rename", Synopsis: "Batch rename files with preview"},
	{Key: "e", Name: "recent", Synopsis: "Show recent files"},
	{Key: "b", Name: "backup", Synopsis: "Create a folder zip backup"},
	{Key: "c", Name: "clean", Synopsis: "Delete empty folders"},
	{Key: "y", Name: "system", Synopsis: "Show system/network snapshot"},
}

func RunMenu(baseDir string) int {
	reader := bufio.NewReader(os.Stdin)

	for {
		ui.PrintSection("Tools")
		for i, item := range toolMenuItems {
			fmt.Printf("%2d) [%s] %s %s\n", i+1, ui.Warn(item.Key), ui.Accent(item.Name), ui.Muted("- "+item.Synopsis))
		}
		fmt.Println(" 0) " + ui.Error("[x] Exit"))
		fmt.Println(ui.Muted(" h <n|letter>) Help"))
		fmt.Print(ui.Prompt("Select tool > "))

		choice := strings.TrimSpace(readLine(reader))
		lc := strings.ToLower(choice)
		switch choice {
		case "0", "x", "X", "exit", "Exit", "":
			return 0
		default:
			if strings.HasPrefix(lc, "h ") {
				target := strings.TrimSpace(choice[2:])
				idx, ok := parseToolMenuChoice(target, len(toolMenuItems))
				if !ok {
					fmt.Println(ui.Error("Invalid help selection."))
					continue
				}
				item := toolMenuItems[idx]
				fmt.Println(ui.Accent("Tool:"), item.Name)
				fmt.Println(ui.Accent("Summary:"), item.Synopsis)
				waitForEnter(reader)
				continue
			}
			idx, ok := parseToolMenuChoice(choice, len(toolMenuItems))
			if !ok {
				fmt.Println(ui.Error("Invalid selection."))
				continue
			}
			_ = RunByNameWithReader(baseDir, toolMenuItems[idx].Name, reader)
			waitForEnter(reader)
		}
	}
}

func RunByName(baseDir, name string) int {
	return RunByNameWithReader(baseDir, name, bufio.NewReader(os.Stdin))
}

func RunByNameWithParams(baseDir, name string, params map[string]string) int {
	return RunByNameWithParamsDetailed(baseDir, name, params).Code
}

func RunByNameWithParamsDetailed(baseDir, name string, params map[string]string) AutoRunResult {
	switch normalizeToolName(name) {
	case "search":
		return RunSearchAutoDetailed(baseDir, params)
	case "rename":
		return RunRenameAutoDetailed(baseDir, params)
	case "recent":
		return RunRecentAutoDetailed(baseDir, params)
	case "clean":
		return AutoRunResult{Code: RunCleanEmptyAuto(baseDir, params)}
	case "backup":
		return AutoRunResult{Code: RunBackupAuto(baseDir, params)}
	case "system":
		return AutoRunResult{Code: RunSystemAuto()}
	default:
		return AutoRunResult{Code: RunByName(baseDir, name)}
	}
}

func RunByNameWithReader(baseDir, name string, reader *bufio.Reader) int {
	switch normalizeToolName(name) {
	case "search":
		return RunSearch(reader)
	case "rename":
		return RunRename(baseDir, reader)
	case "recent":
		return RunRecent(reader)
	case "backup":
		return RunPackBackup(baseDir, reader)
	case "clean":
		return RunCleanEmpty(reader)
	case "system":
		return RunSystem(reader)
	default:
		fmt.Println(ui.Error("Invalid tool:"), name)
		fmt.Println(ui.Muted("Use: search|rename|recent|backup|clean|system"))
		return 1
	}
}

func normalizeToolName(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "1", "search", "s":
		return "search"
	case "2", "rename", "r":
		return "rename"
	case "3", "recent", "rec":
		return "recent"
	case "e":
		return "recent"
	case "4", "backup", "b":
		return "backup"
	case "5", "clean", "c":
		return "clean"
	case "6", "system", "sys", "htop":
		return "system"
	case "y":
		return "system"
	default:
		return ""
	}
}

func parseToolMenuChoice(choice string, count int) (int, bool) {
	v := strings.ToLower(strings.TrimSpace(choice))
	if v == "" {
		return -1, false
	}
	// number
	n := 0
	allDigits := true
	for _, ch := range v {
		if ch < '0' || ch > '9' {
			allDigits = false
			break
		}
		n = n*10 + int(ch-'0')
	}
	if allDigits {
		if n >= 1 && n <= count {
			return n - 1, true
		}
		return -1, false
	}
	// letter
	if len(v) == 1 {
		for i, item := range toolMenuItems {
			if item.Key == v {
				return i, true
			}
		}
	}
	// direct name
	for i, item := range toolMenuItems {
		if item.Name == v {
			return i, true
		}
	}
	return -1, false
}

func prompt(r *bufio.Reader, label, def string) string {
	if def != "" {
		fmt.Printf("%s ", ui.Prompt(fmt.Sprintf("%s [%s]:", label, def)))
	} else {
		fmt.Printf("%s ", ui.Prompt(label+":"))
	}
	text, _ := r.ReadString('\n')
	text = strings.TrimSpace(text)
	if text == "" {
		return def
	}
	return text
}

func readLine(r *bufio.Reader) string {
	s, _ := r.ReadString('\n')
	return strings.TrimSpace(s)
}

func waitForEnter(r *bufio.Reader) {
	fmt.Print(ui.Prompt("Press Enter to continue..."))
	_, _ = r.ReadString('\n')
}

func currentWorkingDir(fallback string) string {
	wd, err := os.Getwd()
	if err != nil || strings.TrimSpace(wd) == "" {
		return fallback
	}
	return wd
}

func normalizeInputPath(raw, fallback string) string {
	p := strings.TrimSpace(raw)
	p = strings.Trim(p, `"'`)
	if p == "" {
		p = fallback
	}
	if strings.TrimSpace(p) == "" {
		p = "."
	}
	return filepath.Clean(p)
}

func validateExistingDir(path, label string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%s not found: %s", label, path)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory: %s", label, path)
	}
	return nil
}
