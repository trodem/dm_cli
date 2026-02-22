package app

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"cli/internal/plugins"
	"cli/internal/ui"
)

func runPluginMenu(baseDir string) int {
	reader := bufio.NewReader(os.Stdin)
	for {
		files, err := plugins.ListFunctionFiles(baseDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			return 1
		}
		if len(files) == 0 {
			fmt.Println("No plugin function files found.")
			return 0
		}

		fmt.Println()
		fmt.Println(ui.Accent("Plugin Files"))
		fmt.Println(ui.Muted("------------"))
		for i, f := range files {
			label := pluginMenuLabel(i)
			rel := strings.TrimPrefix(strings.ReplaceAll(f.Path, "\\", "/"), strings.ReplaceAll(filepath.Join(baseDir, "plugins"), "\\", "/")+"/")
			fmt.Printf("%2d) [%s] %s %s\n", i+1, ui.Warn(label), ui.Accent(rel), ui.Muted(fmt.Sprintf("(%d)", len(f.Functions))))
		}
		fmt.Println(" 0) " + ui.Error("[x] Exit"))
		fmt.Print(ui.Prompt("Select file > "))
		choice := strings.TrimSpace(readLine(reader))
		if choice == "" || strings.EqualFold(choice, "x") || choice == "0" {
			return 0
		}
		fileIndex, ok := parsePluginMenuChoice(choice, len(files))
		if !ok {
			fmt.Println(ui.Error("Invalid selection."))
			continue
		}
		code := runPluginFunctionsMenu(baseDir, files[fileIndex], reader)
		if code != 0 {
			return code
		}
	}
}

func runPluginFunctionsMenu(baseDir string, file plugins.FunctionFile, reader *bufio.Reader) int {
	infoByName := map[string]plugins.Info{}
	for _, name := range file.Functions {
		if info, err := plugins.GetInfo(baseDir, name); err == nil {
			infoByName[name] = info
		}
	}

	for {
		fmt.Println()
		fmt.Printf("%s %s\n", ui.Accent("Functions:"), ui.Accent(strings.ReplaceAll(file.Path, "\\", "/")))
		fmt.Println(ui.Muted("----------------"))
		for i, name := range file.Functions {
			info, ok := infoByName[name]
			line := fmt.Sprintf("%2d) [%s] %s", i+1, ui.Warn(pluginMenuLabel(i)), ui.Accent(name))
			if ok && len(info.Parameters) > 0 {
				line += " " + ui.Warn("[args]")
			}
			if ok && strings.TrimSpace(info.Synopsis) != "" {
				line += " " + ui.Muted("- "+truncateText(info.Synopsis, 72))
			}
			fmt.Println(line)
		}
		fmt.Println(" 0) " + ui.Error("[x] Exit"))
		fmt.Println(ui.Muted(" h <n|letter>) Help"))
		fmt.Print(ui.Prompt("Select function > "))

		choice := strings.TrimSpace(readLine(reader))
		lc := strings.ToLower(choice)
		switch lc {
		case "", "0", "x", "exit":
			return 0
		}

		if strings.HasPrefix(lc, "h ") {
			target := strings.TrimSpace(choice[2:])
			idx, ok := parsePluginMenuChoice(target, len(file.Functions))
			if !ok {
				fmt.Println(ui.Error("Invalid help selection."))
				continue
			}
			_ = runPlugin(baseDir, []string{"info", file.Functions[idx]})
			waitForEnter(reader)
			continue
		}

		funcIndex, ok := parsePluginMenuChoice(choice, len(file.Functions))
		if !ok {
			fmt.Println(ui.Error("Invalid selection."))
			continue
		}
		fn := file.Functions[funcIndex]
		var (
			paramCount int
			argsHint   string
		)
		if info, ok := infoByName[fn]; ok {
			paramCount = len(info.Parameters)
			if len(info.Parameters) > 0 {
				fmt.Println(ui.Accent("Parameters:"))
				for _, p := range info.Parameters {
					fmt.Println("-", p)
				}
			}
			if len(info.Examples) > 0 {
				fmt.Println(ui.Accent("Example:"))
				fmt.Println("-", info.Examples[0])
				argsHint = argsHintFromExample(fn, info.Examples[0])
			}
		}
		runArgs := []string{"run", fn}
		if paramCount == 0 {
			_ = runPlugin(baseDir, runArgs)
			waitForEnter(reader)
			continue
		}
		if strings.TrimSpace(argsHint) != "" {
			fmt.Println(ui.Accent("Args hint:"), argsHint)
		}
		fmt.Print(ui.Prompt("Args (optional) > "))
		rawArgs := strings.TrimSpace(readLine(reader))
		parsedArgs, err := splitMenuArgs(rawArgs)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			continue
		}
		runArgs = append(runArgs, parsedArgs...)
		_ = runPlugin(baseDir, runArgs)
		waitForEnter(reader)
	}
}

func parsePluginMenuChoice(choice string, count int) (int, bool) {
	trimmed := strings.TrimSpace(choice)
	if trimmed == "" {
		return -1, false
	}
	if n, err := strconv.Atoi(trimmed); err == nil {
		if n >= 1 && n <= count {
			return n - 1, true
		}
		return -1, false
	}
	lc := strings.ToLower(trimmed)
	if len(lc) == 1 {
		ch := lc[0]
		if ch >= 'a' && ch <= 'z' {
			idx := int(ch - 'a')
			if idx >= 0 && idx < count {
				return idx, true
			}
		}
	}
	return -1, false
}

func pluginMenuLabel(i int) string {
	if i < 26 {
		return string(rune('a' + i))
	}
	return "?"
}

func splitMenuArgs(s string) ([]string, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	var (
		args  []string
		cur   strings.Builder
		quote rune
	)
	flush := func() {
		if cur.Len() > 0 {
			args = append(args, cur.String())
			cur.Reset()
		}
	}
	for _, r := range s {
		if quote != 0 {
			if r == quote {
				quote = 0
				continue
			}
			cur.WriteRune(r)
			continue
		}
		if r == '"' || r == '\'' {
			quote = r
			continue
		}
		if r == ' ' || r == '\t' {
			flush()
			continue
		}
		cur.WriteRune(r)
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quoted argument")
	}
	flush()
	return args, nil
}

func readLine(r *bufio.Reader) string {
	s, _ := r.ReadString('\n')
	return strings.TrimSpace(s)
}

func waitForEnter(r *bufio.Reader) {
	fmt.Print(ui.Prompt("Press Enter to continue..."))
	_, _ = r.ReadString('\n')
}

func truncateText(s string, max int) string {
	txt := strings.TrimSpace(s)
	if max <= 0 || len(txt) <= max {
		return txt
	}
	if max <= 3 {
		return txt[:max]
	}
	return txt[:max-3] + "..."
}

func argsHintFromExample(functionName, example string) string {
	ex := strings.TrimSpace(example)
	if ex == "" {
		return ""
	}
	prefix := "dm " + functionName
	lowerEx := strings.ToLower(ex)
	lowerPrefix := strings.ToLower(prefix)
	if strings.HasPrefix(lowerEx, lowerPrefix) {
		hint := strings.TrimSpace(ex[len(prefix):])
		return hint
	}
	return ""
}
