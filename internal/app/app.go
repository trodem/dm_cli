package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cli/internal/plugins"
)

func runPluginOrSuggest(baseDir string, args []string) int {
	if len(args) == 0 {
		return 0
	}
	if err := plugins.Run(baseDir, args[0], args[1:]); err != nil {
		if plugins.IsNotFound(err) {
			fmt.Fprintln(os.Stderr, "Error:", err)
			if suggestion := suggestTopLevelName(baseDir, args[0]); suggestion != "" {
				fmt.Fprintf(os.Stderr, "Did you mean: dm %s\n", suggestion)
			}
			return 1
		}
		fmt.Fprintln(os.Stderr, "Error:", err)
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

func loadRuntime() (runtimeContext, error) {
	baseDir, err := exeDir()
	if err != nil {
		return runtimeContext{}, fmt.Errorf("cannot determine executable directory: %w", err)
	}
	return runtimeContext{BaseDir: baseDir}, nil
}

func parseFlags(args []string) []string {
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
	return out
}

func runPlugin(baseDir string, args []string) int {
	if len(args) == 0 {
		return runPluginMenu(baseDir)
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
			fmt.Fprintln(os.Stderr, "Error:", err)
			return 1
		}
		if len(items) == 0 {
			fmt.Println("No plugins found.")
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
			fmt.Fprintln(os.Stderr, "Error:", err)
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
			fmt.Fprintln(os.Stderr, "Error:", err)
			return 1
		}
		return 0
	default:
		if suggestion := suggestClosest(args[0], []string{"list", "info", "run", "menu"}, 3); suggestion != "" {
			fmt.Printf("Did you mean: dm plugins %s\n", suggestion)
		}
		fmt.Println("Usage: dm plugins <list|info|run|menu> ...")
		return 0
	}
}

func suggestTopLevelName(baseDir string, input string) string {
	candidates := []string{
		"ps_profile", "cp", "open", "doctor", "plugins", "tools", "ask", "completion", "help",
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
	ra := []rune(a)
	rb := []rune(b)
	if len(ra) == 0 {
		return len(rb)
	}
	if len(rb) == 0 {
		return len(ra)
	}
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := 0; j <= len(rb); j++ {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 0
			if ra[i-1] != rb[j-1] {
				cost = 1
			}
			del := prev[j] + 1
			ins := cur[j-1] + 1
			sub := prev[j-1] + cost
			cur[j] = min(del, ins, sub)
		}
		prev, cur = cur, prev
	}
	return prev[len(rb)]
}
