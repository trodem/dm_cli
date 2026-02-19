package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cli/internal/doctor"
	"cli/internal/plugins"
	"cli/internal/ui"
	"cli/tools"
)

func runLegacy(args []string) int {
	_, rest := parseFlags(args)
	rt, err := loadRuntime(flags{})
	if err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	baseDir := rt.BaseDir

	if len(rest) == 0 {
		exeBuiltAt, _ := executableBuildTime()
		ui.PrintSplash(ui.SplashData{
			BaseDir:    baseDir,
			Version:    Version,
			ExeBuiltAt: exeBuiltAt,
		})
		return 0
	}

	args = rest
	if len(args) > 0 && args[0] == "$profile" {
		return showPowerShellSymbols(resolveUserPowerShellProfilePath(), "$PROFILE")
	}

	switch args[0] {
	case "ps_profile":
		return showPowerShellSymbols(resolveUserPowerShellProfilePath(), "$PROFILE")
	case "cp":
		if len(args) < 2 {
			fmt.Println("Usage: dm cp profile")
			return 0
		}
		if strings.EqualFold(args[1], "profile") {
			if err := copyPowerShellProfileFromPlugin(baseDir); err != nil {
				fmt.Println("Error:", err)
				return 1
			}
			fmt.Println("OK: profile overwritten from plugins/functions/0_powershell_profile.ps1")
			return 0
		}
		fmt.Println("Usage: dm cp profile")
		return 0
	case "open":
		if len(args) < 2 {
			fmt.Println("Usage: dm open <ps_profile|profile>")
			return 0
		}
		switch strings.ToLower(strings.TrimSpace(args[1])) {
		case "ps_profile":
			if err := openUserPowerShellProfileInNotepad(); err != nil {
				fmt.Println("Error:", err)
				return 1
			}
			return 0
		case "profile", "profile-source", "profile-src":
			if err := openPluginPowerShellProfileInNotepad(baseDir); err != nil {
				fmt.Println("Error:", err)
				return 1
			}
			return 0
		default:
			fmt.Println("Usage: dm open <ps_profile|profile>")
			return 0
		}
	case "doctor":
		useJSON := false
		for _, a := range args[1:] {
			if strings.TrimSpace(a) == "--json" {
				useJSON = true
			}
		}
		report := doctor.Run(baseDir)
		if useJSON {
			if err := doctor.RenderJSON(report); err != nil {
				fmt.Println("Error:", err)
				return 1
			}
		} else {
			doctor.RenderText(report)
		}
		if report.ErrorCount > 0 {
			return 1
		}
		return 0
	case "plugins":
		return runPlugin(baseDir, args[1:])
	case "tools":
		if len(args) == 1 {
			return tools.RunMenu(baseDir)
		}
		return tools.RunByName(baseDir, args[1])
	case "ask":
		askOpts, confirmTools, riskPolicy, prompt, err := parseLegacyAskArgs(args[1:])
		if err != nil {
			fmt.Println("Error:", err)
			return 1
		}
		if strings.TrimSpace(prompt) == "" {
			return runAskInteractiveWithRisk(baseDir, askOpts, confirmTools, riskPolicy)
		}
		return runAskOnceWithSession(baseDir, prompt, askOpts, confirmTools, riskPolicy, nil, false)
	case "toolkit":
		if len(args) == 1 {
			if err := runToolkitWizard(baseDir); err != nil {
				fmt.Println("Error:", err)
				return 1
			}
			return 0
		}
		fmt.Println("Usage: dm toolkit [new|add|validate]")
		return 0
	}

	return runPluginOrSuggest(baseDir, args)
}

func runPluginOrSuggest(baseDir string, args []string) int {
	if len(args) == 0 {
		return 0
	}
	if err := plugins.Run(baseDir, args[0], args[1:]); err != nil {
		if plugins.IsNotFound(err) {
			fmt.Println("Error:", err)
			if suggestion := suggestTopLevelName(baseDir, args[0]); suggestion != "" {
				fmt.Printf("Did you mean: dm %s\n", suggestion)
			}
			return 1
		}
		fmt.Println("Error:", err)
		return 1
	}
	return 0
}

func exeDir() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(p), nil
}

func executableBuildTime() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	info, err := os.Stat(exePath)
	if err != nil {
		return "", err
	}
	return info.ModTime().Local().Format("2006-01-02 15:04:05"), nil
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

type runtimeContext struct {
	BaseDir string
}

func loadRuntime(opts flags) (runtimeContext, error) {
	_ = opts
	baseDir, err := exeDir()
	if err != nil {
		return runtimeContext{}, fmt.Errorf("cannot determine executable directory: %w", err)
	}
	return runtimeContext{BaseDir: baseDir}, nil
}

type flags struct{}

func parseFlags(args []string) (flags, []string) {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if group, ok := mapGroupShortcut(arg); ok {
			out = append(out, group)
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				out = append(out, args[i+1])
				i++
			}
			continue
		}
		out = append(out, arg)
	}
	return flags{}, out
}

func runPlugin(baseDir string, args []string) int {
	if len(args) == 0 {
		return runPluginMenu(baseDir)
	}
	if args[0] == "$profile" || strings.EqualFold(args[0], "profile") {
		path := filepath.Join(baseDir, "plugins", "functions", "0_powershell_profile.ps1")
		return showPowerShellSymbols(path, "plugins/functions/0_powershell_profile.ps1")
	}
	switch args[0] {
	case "menu":
		return runPluginMenu(baseDir)
	case "list":
		includeFunctions := false
		for _, arg := range args[1:] {
			if arg == "--functions" || arg == "-f" {
				includeFunctions = true
			}
		}
		items, err := plugins.ListEntries(baseDir, includeFunctions)
		if err != nil {
			fmt.Println("Error:", err)
			return 1
		}
		if len(items) == 0 {
			fmt.Println("No plugins/functions found.")
			return 0
		}
		for _, item := range items {
			if includeFunctions {
				if item.Kind == "function" {
					fmt.Println(item.Name)
				}
				continue
			}
			if item.Kind == "script" {
				fmt.Println(item.Name)
			}
		}
		return 0
	case "info":
		if len(args) < 2 {
			fmt.Println("Usage: dm plugins info <name>")
			return 0
		}
		info, err := plugins.GetInfo(baseDir, args[1])
		if err != nil {
			fmt.Println("Error:", err)
			return 1
		}
		fmt.Println("Name      :", info.Name)
		fmt.Println("Kind      :", info.Kind)
		fmt.Println("Path      :", info.Path)
		fmt.Println("Runner    :", info.Runner)
		if len(info.Sources) > 1 {
			fmt.Println("Sources   :", strings.Join(info.Sources, ", "))
		}
		if strings.TrimSpace(info.Synopsis) != "" {
			fmt.Println("Synopsis  :", info.Synopsis)
		}
		if strings.TrimSpace(info.Description) != "" {
			fmt.Println("Description:", info.Description)
		}
		if len(info.Parameters) > 0 {
			fmt.Println("Parameters:")
			for _, p := range info.Parameters {
				fmt.Println("-", p)
			}
		}
		if len(info.Examples) > 0 {
			fmt.Println("Examples:")
			for _, ex := range info.Examples {
				fmt.Println("-", ex)
			}
		}
		return 0
	case "run":
		if len(args) < 2 {
			fmt.Println("Usage: dm plugins run <name> [args...]")
			return 0
		}
		if err := plugins.Run(baseDir, args[1], args[2:]); err != nil {
			fmt.Println("Error:", err)
			return 1
		}
		return 0
	default:
		if suggestion := suggestClosest(args[0], []string{"list", "info", "run", "menu", "profile"}, 3); suggestion != "" {
			fmt.Printf("Did you mean: dm plugins %s\n", suggestion)
		}
		fmt.Println("Usage: dm plugins <list|info|run|menu> ...")
		return 0
	}
}

func suggestTopLevelName(baseDir string, input string) string {
	candidates := []string{
		"ps_profile", "cp", "open", "doctor", "plugins", "tools", "toolkit", "ask", "completion", "help",
	}
	if items, err := plugins.ListEntries(baseDir, true); err == nil {
		for _, it := range items {
			if strings.TrimSpace(it.Name) != "" {
				candidates = append(candidates, it.Name)
			}
		}
	}
	return suggestClosest(input, candidates, 3)
}

func suggestClosest(input string, candidates []string, maxDistance int) string {
	in := strings.ToLower(strings.TrimSpace(input))
	if in == "" {
		return ""
	}
	seen := map[string]struct{}{}
	best := ""
	bestDist := maxDistance + 1
	for _, raw := range candidates {
		c := strings.TrimSpace(raw)
		if c == "" {
			continue
		}
		key := strings.ToLower(c)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if key == in {
			return c
		}
		d := editDistance(in, key)
		if d < bestDist {
			bestDist = d
			best = c
		}
	}
	if bestDist <= maxDistance {
		return best
	}
	return ""
}

func editDistance(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := 0; j <= len(b); j++ {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			del := prev[j] + 1
			ins := cur[j-1] + 1
			sub := prev[j-1] + cost
			cur[j] = min3(del, ins, sub)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

func min3(a, b, c int) int {
	if a <= b && a <= c {
		return a
	}
	if b <= a && b <= c {
		return b
	}
	return c
}
