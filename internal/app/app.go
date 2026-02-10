package app

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"cli/internal/config"
	"cli/internal/plugins"
	"cli/internal/runner"
	"cli/internal/ui"
	"cli/tools"
)

func runLegacy(args []string) int {
	opts, rest := parseFlags(args)
	rt, err := loadRuntime(opts)
	if err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	baseDir := rt.BaseDir
	cfg := rt.Config

	if len(rest) == 0 {
		ui.PrintSplash(ui.SplashData{
			BaseDir: baseDir,
			Version: Version,
		})
		return 0
	}

	args = rest
	if len(args) > 0 && args[0] == "$profile" {
		return showPowerShellSymbols(resolveUserPowerShellProfilePath(), "$PROFILE")
	}

	// global commands
	switch args[0] {
	case "aliases", "config":
		ui.PrintAliases(cfg)
		return 0
	case "profile", "ps_profile":
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
	case "list":
		return runList(cfg, args[1:])
	case "add":
		return runAdd(baseDir, args[1:])
	case "validate":
		return runValidate(baseDir, cfg)
	case "plugins":
		return runPlugin(baseDir, args[1:])
	case "tools":
		if len(args) == 1 {
			return tools.RunMenu(baseDir)
		}
		return tools.RunByName(baseDir, args[1])
	case "run":
		if len(args) < 2 {
			fmt.Println("Usage: dm run <alias>")
			return 0
		}
		name := args[1]
		runner.RunAlias(cfg, name, "")
		return 0
	}

	// PROJECT MODE: dm <project> <action>
	return runTargetOrSearch(baseDir, cfg, args)
}

func runTargetOrSearch(baseDir string, cfg config.Config, args []string) int {
	if len(args) == 0 {
		return 0
	}

	// PROJECT MODE: dm <project> <action>
	if _, ok := cfg.Projects[args[0]]; ok && len(args) >= 2 {
		action := args[1]
		runner.RunProjectCommand(cfg, args[0], action, baseDir)
		return 0
	}

	// INTERACTIVE TARGET: dm <name>
	name := args[0]

	// target can be jump or project
	targetPath, isJump := cfg.Jump[name]
	_, isProject := cfg.Projects[name]

	if !isJump && !isProject {
		err := plugins.Run(baseDir, args[0], args[1:])
		if err != nil {
			if plugins.IsNotFound(err) {
				fmt.Println("Error:", err)
				return 1
			}
			fmt.Println("Error:", err)
			return 1
		}
		return 0
	}

	if isProject {
		targetPath = cfg.Projects[name].Path
	}

	targetPath = config.ResolvePath(baseDir, targetPath)
	ui.ShowMenu(cfg, name, targetPath, baseDir)
	return 0
}

func exeDir() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(p), nil
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

type runtimeContext struct {
	BaseDir string
	Config  config.Config
}

func loadRuntime(opts flags) (runtimeContext, error) {
	baseDir, err := exeDir()
	if err != nil {
		return runtimeContext{}, fmt.Errorf("cannot determine executable directory: %w", err)
	}
	cfgPath := filepath.Join(baseDir, "dm.json")
	cfg, err := config.Load(cfgPath, config.Options{
		Profile:  opts.Profile,
		UseCache: !opts.NoCache,
	})
	if err != nil {
		return runtimeContext{}, fmt.Errorf("loading config: %w", err)
	}
	return runtimeContext{
		BaseDir: baseDir,
		Config:  cfg,
	}, nil
}

type flags struct {
	Profile string
	NoCache bool
}

func parseFlags(args []string) (flags, []string) {
	var out []string
	var f flags
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
		if arg == "--no-cache" {
			f.NoCache = true
			continue
		}
		if arg == "--profile" && i+1 < len(args) {
			f.Profile = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(arg, "--profile=") {
			f.Profile = strings.TrimPrefix(arg, "--profile=")
			continue
		}
		out = append(out, arg)
	}
	return f, out
}

func runValidate(baseDir string, cfg config.Config) int {
	issues := config.Validate(cfg)
	if len(issues) == 0 {
		fmt.Println("OK: valid configuration")
		return 0
	}
	for _, issue := range issues {
		fmt.Printf("%s: %s\n", issue.Level, issue.Message)
	}
	return 1
}

func runList(cfg config.Config, args []string) int {
	if len(args) == 0 {
		fmt.Println("Usage: dm list <jumps|runs|projects|actions>")
		return 0
	}
	switch args[0] {
	case "jumps":
		ui.PrintMap(cfg.Jump)
	case "runs":
		ui.PrintMap(cfg.Run)
	case "projects":
		ui.PrintProjects(cfg.Projects)
	case "actions":
		if len(args) < 2 {
			fmt.Println("Usage: dm list actions <project>")
			return 0
		}
		p, ok := cfg.Projects[args[1]]
		if !ok {
			fmt.Println("Project not found:", args[1])
			return 0
		}
		ui.PrintMap(p.Commands)
	default:
		fmt.Println("Usage: dm list <jumps|runs|projects|actions>")
	}
	return 0
}

func runAdd(baseDir string, args []string) int {
	if len(args) < 1 {
		fmt.Println("Usage: dm add <jump|run|project|action> ...")
		return 0
	}
	switch args[0] {
	case "jump":
		if len(args) < 3 {
			fmt.Println("Usage: dm add jump <name> <path>")
			return 0
		}
		err := updateRootConfig(baseDir, func(cfg *config.Config) {
			cfg.Jump[args[1]] = args[2]
		})
		if err != nil {
			fmt.Println("Error:", err)
			return 1
		}
		fmt.Println("OK: jump added")
		return 0
	case "run":
		if len(args) < 3 {
			fmt.Println("Usage: dm add run <name> <command>")
			return 0
		}
		err := updateRootConfig(baseDir, func(cfg *config.Config) {
			cfg.Run[args[1]] = strings.Join(args[2:], " ")
		})
		if err != nil {
			fmt.Println("Error:", err)
			return 1
		}
		fmt.Println("OK: run added")
		return 0
	case "project":
		if len(args) < 3 {
			fmt.Println("Usage: dm add project <name> <path>")
			return 0
		}
		err := updateRootConfig(baseDir, func(cfg *config.Config) {
			cfg.Projects[args[1]] = config.Project{
				Path:     args[2],
				Commands: map[string]string{},
			}
		})
		if err != nil {
			fmt.Println("Error:", err)
			return 1
		}
		fmt.Println("OK: project added")
		return 0
	case "action":
		if len(args) < 4 {
			fmt.Println("Usage: dm add action <project> <name> <command>")
			return 0
		}
		project := args[1]
		action := args[2]
		cmd := strings.Join(args[3:], " ")
		cfg, err := config.Load(filepath.Join(baseDir, "dm.json"), config.Options{UseCache: false})
		if err != nil {
			fmt.Println("Error:", err)
			return 1
		}
		p, ok := cfg.Projects[project]
		if !ok {
			fmt.Println("Project not found. Add the project first.")
			return 1
		}
		if p.Commands == nil {
			p.Commands = map[string]string{}
		}
		p.Commands[action] = cmd
		err = updateRootConfig(baseDir, func(cfg *config.Config) {
			cfg.Projects[project] = p
		})
		if err != nil {
			fmt.Println("Error:", err)
			return 1
		}
		fmt.Println("OK: action added")
		return 0
	default:
		fmt.Println("Usage: dm add <jump|run|project|action> ...")
		return 0
	}
}

func updateRootConfig(baseDir string, mutate func(cfg *config.Config)) error {
	path := filepath.Join(baseDir, "dm.json")
	cfg := config.Config{}
	if fileExists(path) {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if len(strings.TrimSpace(string(data))) > 0 {
			if err := json.Unmarshal(data, &cfg); err != nil {
				return err
			}
		}
	}
	if cfg.Jump == nil {
		cfg.Jump = map[string]string{}
	}
	if cfg.Run == nil {
		cfg.Run = map[string]string{}
	}
	if cfg.Projects == nil {
		cfg.Projects = map[string]config.Project{}
	}
	mutate(&cfg)
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(path, out, 0644)
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
		fmt.Println("Usage: dm plugins <list|info|run|menu> ...")
		return 0
	}
}

var psFunctionName = regexp.MustCompile(`(?i)^\s*function\s+([a-z0-9_-]+)\b`)
var psSetAlias = regexp.MustCompile(`(?i)^\s*(?:set-alias|new-alias)\s+([^\s]+)\s+([^\s#]+)`)

func resolveUserPowerShellProfilePath() string {
	home, _ := os.UserHomeDir()
	if strings.TrimSpace(home) == "" {
		return ""
	}
	ps7 := filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")
	if fileExists(ps7) {
		return ps7
	}
	winPS := filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1")
	if fileExists(winPS) {
		return winPS
	}
	return ps7
}

func showPowerShellSymbols(path, label string) int {
	if strings.TrimSpace(path) == "" {
		fmt.Println("Error: PowerShell profile path is not available.")
		return 1
	}
	funcs, aliases, err := parsePowerShellSymbols(path)
	if err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	fmt.Println("Profile:", label)
	fmt.Println("Path   :", path)
	fmt.Println()
	fmt.Println("Functions")
	fmt.Println("---------")
	if len(funcs) == 0 {
		fmt.Println("(none)")
	} else {
		for _, n := range funcs {
			fmt.Println(n)
		}
	}
	fmt.Println()
	fmt.Println("Aliases")
	fmt.Println("-------")
	if len(aliases) == 0 {
		fmt.Println("(none)")
	} else {
		for _, n := range aliases {
			fmt.Println(n)
		}
	}
	return 0
}

func parsePowerShellSymbols(path string) ([]string, []string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	lines := strings.Split(string(data), "\n")
	funcSeen := map[string]struct{}{}
	aliasSeen := map[string]struct{}{}
	funcs := []string{}
	aliases := []string{}

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if m := psFunctionName.FindStringSubmatch(line); len(m) == 2 {
			name := strings.TrimSpace(m[1])
			if name != "" {
				if _, ok := funcSeen[name]; !ok {
					funcSeen[name] = struct{}{}
					funcs = append(funcs, name)
				}
			}
			continue
		}
		if m := psSetAlias.FindStringSubmatch(line); len(m) == 3 {
			aliasName := strings.TrimSpace(m[1])
			target := strings.TrimSpace(m[2])
			if aliasName != "" {
				if _, ok := aliasSeen[aliasName]; !ok {
					aliasSeen[aliasName] = struct{}{}
					aliases = append(aliases, aliasName+" -> "+target)
				}
			}
		}
	}

	sort.Strings(funcs)
	sort.Strings(aliases)
	return funcs, aliases, nil
}

func copyPowerShellProfileFromPlugin(baseDir string) error {
	src := filepath.Join(baseDir, "plugins", "functions", "0_powershell_profile.ps1")
	if !fileExists(src) {
		return fmt.Errorf("source not found: %s", src)
	}
	dst := resolveUserPowerShellProfilePath()
	if strings.TrimSpace(dst) == "" {
		return fmt.Errorf("PowerShell profile path is not available")
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func openInNotepad(path string) error {
	cmd := exec.Command("notepad.exe", path)
	return cmd.Start()
}

func openPluginPowerShellProfileInNotepad(baseDir string) error {
	src := filepath.Join(baseDir, "plugins", "functions", "0_powershell_profile.ps1")
	return openInNotepad(src)
}

func openUserPowerShellProfileInNotepad() error {
	dst := resolveUserPowerShellProfilePath()
	if strings.TrimSpace(dst) == "" {
		return fmt.Errorf("PowerShell profile path is not available")
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	if !fileExists(dst) {
		if err := os.WriteFile(dst, []byte{}, 0644); err != nil {
			return err
		}
	}
	return openInNotepad(dst)
}

func runPluginMenu(baseDir string) int {
	reader := bufio.NewReader(os.Stdin)
	for {
		files, err := plugins.ListFunctionFiles(baseDir)
		if err != nil {
			fmt.Println("Error:", err)
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
			fmt.Println("Error:", err)
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
