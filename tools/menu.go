package tools

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cli/internal/ui"
)

type ToolDescriptor struct {
	Key       string
	Name      string
	Synopsis  string
	Aliases   []string
	AgentArgs string
	RiskLevel string
	RiskNote  string
}

type AutoRunResult struct {
	Code           int
	CanContinue    bool
	ContinuePrompt string
	ContinueParams map[string]string
}

var ToolRegistry = []ToolDescriptor{
	{Key: "s", Name: "search", Synopsis: "Search files by name/extension", Aliases: []string{"s"}, AgentArgs: "base, ext, name, sort, limit, offset", RiskLevel: "low", RiskNote: "read/inspect operation"},
	{Key: "r", Name: "rename", Synopsis: "Batch rename files with preview", Aliases: []string{"r"}, AgentArgs: "base, from, to, name, case_sensitive", RiskLevel: "medium", RiskNote: "batch rename files"},
	{Key: "e", Name: "recent", Synopsis: "Show recent files", Aliases: []string{"rec"}, AgentArgs: "base, limit, offset", RiskLevel: "low", RiskNote: "read/inspect operation"},
	{Key: "b", Name: "backup", Synopsis: "Create a folder zip backup", Aliases: []string{"b"}, AgentArgs: "source, output", RiskLevel: "medium", RiskNote: "writes backup archive"},
	{Key: "c", Name: "clean", Synopsis: "Delete empty folders", Aliases: []string{"c"}, AgentArgs: "base, apply (true for delete, otherwise preview)", RiskLevel: "low", RiskNote: "preview only"},
	{Key: "y", Name: "system", Synopsis: "Show system/network snapshot", Aliases: []string{"sys", "htop"}, AgentArgs: "", RiskLevel: "low", RiskNote: "read/inspect operation"},
}

func RunMenu(baseDir string) int {
	reader := bufio.NewReader(os.Stdin)

	for {
		ui.PrintSection("Tools")
		for i, item := range ToolRegistry {
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
				idx, ok := parseToolMenuChoice(target, len(ToolRegistry))
				if !ok {
					fmt.Println(ui.Error("Invalid help selection."))
					continue
				}
				item := ToolRegistry[idx]
				fmt.Println(ui.Accent("Tool:"), item.Name)
				fmt.Println(ui.Accent("Summary:"), item.Synopsis)
				waitForEnter(reader)
				continue
			}
			idx, ok := parseToolMenuChoice(choice, len(ToolRegistry))
			if !ok {
				fmt.Println(ui.Error("Invalid selection."))
				continue
			}
			_ = RunByNameWithReader(baseDir, ToolRegistry[idx].Name, reader)
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
	lc := strings.ToLower(strings.TrimSpace(name))
	for i, t := range ToolRegistry {
		if lc == t.Name || lc == t.Key || lc == fmt.Sprintf("%d", i+1) {
			return t.Name
		}
		for _, alias := range t.Aliases {
			if lc == strings.ToLower(alias) {
				return t.Name
			}
		}
	}
	return ""
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
		for i, item := range ToolRegistry {
			if item.Key == v {
				return i, true
			}
		}
	}
	// direct name
	for i, item := range ToolRegistry {
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

func IsKnownTool(name string) bool {
	return normalizeToolName(name) != ""
}

func BuildAgentCatalog() string {
	lines := make([]string, 0, len(ToolRegistry))
	for _, t := range ToolRegistry {
		line := "- " + t.Name + ": " + t.Synopsis
		if t.AgentArgs != "" {
			line += " | tool_args: " + t.AgentArgs
		} else {
			line += " (no args needed)"
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func ToolRisk(name string, args map[string]string) (string, string) {
	canonical := normalizeToolName(name)
	for _, t := range ToolRegistry {
		if t.Name != canonical {
			continue
		}
		if t.Name == "clean" {
			apply := strings.ToLower(strings.TrimSpace(args["apply"]))
			if apply == "1" || apply == "true" || apply == "yes" || apply == "y" {
				return "high", "delete empty directories"
			}
		}
		return t.RiskLevel, t.RiskNote
	}
	return "low", "read/inspect operation"
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
