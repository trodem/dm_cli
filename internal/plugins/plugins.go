package plugins

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

type Plugin struct {
	Name string
	Path string
}

type Entry struct {
	Name string
	Kind string // script|function
	Path string
}

type FunctionFile struct {
	Path      string
	Functions []string
}

type ParamDetail struct {
	Name        string
	Type        string
	Mandatory   bool
	Switch      bool
	ValidateSet []string
	Default     string
}

type Info struct {
	Name         string
	Kind         string
	Path         string
	Sources      []string
	Runner       string
	Synopsis     string
	Description  string
	Parameters   []string
	ParamDetails []ParamDetail
	Examples     []string
}

type RunError struct {
	Err    error
	Output string
}

func (e *RunError) Error() string {
	return e.Err.Error()
}

func (e *RunError) Unwrap() error {
	return e.Err
}

var ErrNotFound = errors.New("plugin not found")
var psFunctionLine = regexp.MustCompile(`(?i)^\s*function\s+([a-z0-9_-]+)\b`)
var psNamedTag = regexp.MustCompile(`(?i)^\.(synopsis|description|example|parameter)\b(?:\s+([a-z0-9_-]+))?\s*$`)

func List(baseDir string) ([]Plugin, error) {
	items, err := ListEntries(baseDir, true)
	if err != nil {
		return nil, err
	}
	out := make([]Plugin, 0, len(items))
	for _, it := range items {
		out = append(out, Plugin{Name: it.Name, Path: it.Path})
	}
	return out, nil
}

func ListEntries(baseDir string, includeFunctions bool) ([]Entry, error) {
	dir := filepath.Join(baseDir, "plugins")
	cacheKey := listEntriesCacheKey(dir, includeFunctions)
	if cached, ok := getCachedEntryList(cacheKey); ok {
		return cached, nil
	}
	dirStamp := statStamp(dir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			out := []Entry{}
			setCachedEntryList(cacheKey, dir, out, dirStamp, map[string]int64{})
			return out, nil
		}
		return nil, err
	}

	bestByName := map[string]Entry{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !isSupportedPlugin(name) {
			continue
		}
		baseName := pluginName(name)
		candidate := Entry{Name: baseName, Kind: "script", Path: filepath.Join(dir, name)}
		current, ok := bestByName[baseName]
		if !ok || pluginScore(candidate.Path) < pluginScore(current.Path) {
			bestByName[baseName] = candidate
		}
	}

	out := make([]Entry, 0, len(bestByName))
	for _, p := range bestByName {
		out = append(out, p)
	}

	if includeFunctions {
		fnMap, _, err := collectPowerShellFunctions(dir)
		if err != nil {
			return nil, err
		}
		for name, path := range fnMap {
			if _, ok := bestByName[name]; ok {
				continue
			}
			out = append(out, Entry{Name: name, Kind: "function", Path: path})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Name == out[j].Name {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Name < out[j].Name
	})
	stamps := buildEntryListFileStamps(out)
	setCachedEntryList(cacheKey, dir, out, dirStamp, stamps)
	return out, nil
}

func ListFunctionFiles(baseDir string) ([]FunctionFile, error) {
	dir := filepath.Join(baseDir, "plugins")
	files, err := listPowerShellFunctionFiles(dir)
	if err != nil {
		return nil, err
	}
	out := make([]FunctionFile, 0, len(files))
	for _, p := range files {
		names, err := readPowerShellFunctionNames(p)
		if err != nil {
			return nil, err
		}
		if len(names) == 0 {
			continue
		}
		sort.Strings(names)
		out = append(out, FunctionFile{
			Path:      p,
			Functions: names,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Path) < strings.ToLower(out[j].Path)
	})
	return out, nil
}

func GetInfo(baseDir, name string) (Info, error) {
	dir := filepath.Join(baseDir, "plugins")
	cacheKey := infoCacheKey(dir, name)
	if cached, ok := getCachedInfo(cacheKey); ok {
		return cached, nil
	}
	dirStamp := statStamp(dir)

	candidate, err := findPlugin(dir, name)
	if err != nil {
		return Info{}, err
	}
	if candidate != "" {
		out := Info{
			Name:    name,
			Kind:    "script",
			Path:    candidate,
			Sources: []string{candidate},
			Runner:  runnerForPath(candidate),
		}
		setCachedInfo(cacheKey, dir, out, dirStamp, buildInfoFileStamps(out))
		return out, nil
	}

	fnPath, loadFiles, found, err := findPowerShellFunction(dir, name)
	if err != nil {
		return Info{}, err
	}
	if !found {
		return Info{}, fmt.Errorf("%w: %s", ErrNotFound, name)
	}

	help, _ := parsePowerShellFunctionHelp(fnPath, name)
	paramDetails := parsePowerShellParamBlock(fnPath, name)
	sources := sourcesForFunction(loadFiles, name)
	if len(sources) == 0 {
		sources = []string{fnPath}
	}

	out := Info{
		Name:         name,
		Kind:         "function",
		Path:         fnPath,
		Sources:      sources,
		Runner:       "powershell function bridge",
		Synopsis:     help.Synopsis,
		Description:  help.Description,
		Parameters:   help.Parameters,
		ParamDetails: paramDetails,
		Examples:     help.Examples,
	}
	setCachedInfo(cacheKey, dir, out, dirStamp, buildInfoFileStamps(out))
	return out, nil
}

func buildEntryListFileStamps(items []Entry) map[string]int64 {
	stamps := map[string]int64{}
	for _, it := range items {
		p := strings.TrimSpace(it.Path)
		if p == "" {
			continue
		}
		if _, seen := stamps[p]; seen {
			continue
		}
		stamps[p] = statStamp(p)
	}
	return stamps
}

func buildInfoFileStamps(info Info) map[string]int64 {
	stamps := map[string]int64{}
	add := func(path string) {
		p := strings.TrimSpace(path)
		if p == "" {
			return
		}
		if _, seen := stamps[p]; seen {
			return
		}
		stamps[p] = statStamp(p)
	}
	add(info.Path)
	for _, src := range info.Sources {
		add(src)
	}
	return stamps
}

type RunResult struct {
	Output string
	Err    error
}

func Run(baseDir, name string, args []string) error {
	r := RunWithOutput(baseDir, name, args)
	return r.Err
}

func RunWithOutput(baseDir, name string, args []string) RunResult {
	dir := filepath.Join(baseDir, "plugins")
	candidate, err := findPlugin(dir, name)
	if err != nil {
		return RunResult{Err: err}
	}
	if candidate == "" {
		_, loadFiles, found, fErr := findPowerShellFunction(dir, name)
		if fErr != nil {
			return RunResult{Err: fErr}
		}
		if !found {
			return RunResult{Err: fmt.Errorf("%w: %s", ErrNotFound, name)}
		}
		out, runErr := runPowerShellFunctionCapture(loadFiles, name, args)
		return RunResult{Output: out, Err: runErr}
	}
	out, runErr := execPluginCapture(candidate, args)
	return RunResult{Output: out, Err: runErr}
}

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

func ErrorOutput(err error) string {
	var re *RunError
	if errors.As(err, &re) {
		return strings.TrimSpace(re.Output)
	}
	return ""
}

func findPlugin(dir, name string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	var matches []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !isSupportedPlugin(e.Name()) {
			continue
		}
		if pluginName(e.Name()) == name {
			matches = append(matches, filepath.Join(dir, e.Name()))
		}
	}
	if len(matches) == 0 {
		return "", nil
	}

	sort.Slice(matches, func(i, j int) bool {
		si := pluginScore(matches[i])
		sj := pluginScore(matches[j])
		if si == sj {
			return strings.ToLower(matches[i]) < strings.ToLower(matches[j])
		}
		return si < sj
	})
	return matches[0], nil
}

func pluginScore(path string) int {
	ext := strings.ToLower(filepath.Ext(path))
	order := preferredPluginExtOrder()
	for i, v := range order {
		if ext == v {
			return i
		}
	}
	return len(order) + 1
}

func findPowerShellFunction(pluginsDir, name string) (string, []string, bool, error) {
	catalog, files, err := collectPowerShellFunctions(pluginsDir)
	if err != nil {
		return "", nil, false, err
	}
	path, ok := catalog[name]
	if !ok {
		return "", nil, false, nil
	}
	return path, files, true, nil
}

func collectPowerShellFunctions(pluginsDir string) (map[string]string, []string, error) {
	files, err := listPowerShellFunctionFiles(pluginsDir)
	if err != nil {
		return nil, nil, err
	}
	catalog := map[string]string{}
	for _, p := range files {
		names, err := readPowerShellFunctionNames(p)
		if err != nil {
			return nil, nil, err
		}
		for _, n := range names {
			if _, exists := catalog[n]; exists {
				continue
			}
			catalog[n] = p
		}
	}
	return catalog, files, nil
}

func listPowerShellFunctionFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !isPowerShellFunctionSource(d.Name()) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool {
		si := functionSourceScore(files[i])
		sj := functionSourceScore(files[j])
		if si == sj {
			return strings.ToLower(files[i]) < strings.ToLower(files[j])
		}
		return si < sj
	})
	return files, nil
}

func isPowerShellFunctionSource(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".ps1" || ext == ".psm1" || ext == ".txt"
}

func functionSourceScore(path string) int {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".ps1":
		return 0
	case ".psm1":
		return 1
	case ".txt":
		return 2
	default:
		return 3
	}
}

func readPowerShellFunctionNames(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var out []string
	seen := map[string]struct{}{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		m := psFunctionLine.FindStringSubmatch(line)
		if len(m) != 2 {
			continue
		}
		name := strings.TrimSpace(m[1])
		if name == "" {
			continue
		}
		if !isPublicFunctionName(name) {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func isPublicFunctionName(name string) bool {
	return !strings.HasPrefix(name, "_")
}

type functionHelp struct {
	Synopsis    string
	Description string
	Parameters  []string
	Examples    []string
}

func parsePowerShellFunctionHelp(path, functionName string) (functionHelp, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return functionHelp{}, err
	}
	lines := strings.Split(string(data), "\n")
	fnIdx := -1
	for i, line := range lines {
		m := psFunctionLine.FindStringSubmatch(line)
		if len(m) == 2 && strings.EqualFold(strings.TrimSpace(m[1]), functionName) {
			fnIdx = i
			break
		}
	}
	if fnIdx == -1 {
		return functionHelp{}, nil
	}

	end := fnIdx - 1
	for end >= 0 && strings.TrimSpace(lines[end]) == "" {
		end--
	}
	if end < 0 || strings.TrimSpace(lines[end]) != "#>" {
		return functionHelp{}, nil
	}
	start := end - 1
	for start >= 0 && strings.TrimSpace(lines[start]) != "<#" {
		start--
	}
	if start < 0 {
		return functionHelp{}, nil
	}

	block := lines[start+1 : end]
	return parseCommentBlockHelp(block), nil
}

func parseCommentBlockHelp(lines []string) functionHelp {
	helper := functionHelp{}
	var mode string
	var paramName string
	paramText := map[string][]string{}
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if m := psNamedTag.FindStringSubmatch(line); len(m) >= 2 {
			mode = strings.ToLower(m[1])
			if mode == "parameter" {
				paramName = strings.TrimSpace(m[2])
				if paramName != "" {
					if _, ok := paramText[paramName]; !ok {
						paramText[paramName] = []string{}
					}
				}
			} else {
				paramName = ""
			}
			continue
		}
		switch mode {
		case "synopsis":
			helper.Synopsis = strings.TrimSpace(strings.TrimSpace(helper.Synopsis + " " + line))
		case "description":
			helper.Description = strings.TrimSpace(strings.TrimSpace(helper.Description + " " + line))
		case "example":
			helper.Examples = append(helper.Examples, line)
		case "parameter":
			if paramName != "" {
				paramText[paramName] = append(paramText[paramName], line)
			}
		}
	}
	if len(paramText) > 0 {
		names := make([]string, 0, len(paramText))
		for name := range paramText {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			text := strings.TrimSpace(strings.Join(paramText[name], " "))
			if text == "" {
				helper.Parameters = append(helper.Parameters, name)
			} else {
				helper.Parameters = append(helper.Parameters, fmt.Sprintf("%s: %s", name, text))
			}
		}
	}
	return helper
}

var (
	psParamMandatory  = regexp.MustCompile(`(?i)\[Parameter\s*\([^)]*Mandatory\b`)
	psParamVarLine    = regexp.MustCompile(`(?i)^\s*(?:\[([^\]]+)\])?\s*\$(\w+)`)
	psValidateSetLine = regexp.MustCompile(`(?i)\[ValidateSet\s*\(([^)]+)\)\]`)
	psDefaultValue    = regexp.MustCompile(`\$\w+\s*=\s*(.+)`)
)

func parsePowerShellParamBlock(path, functionName string) []ParamDetail {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(data), "\n")
	fnIdx := -1
	for i, line := range lines {
		m := psFunctionLine.FindStringSubmatch(line)
		if len(m) == 2 && strings.EqualFold(strings.TrimSpace(m[1]), functionName) {
			fnIdx = i
			break
		}
	}
	if fnIdx == -1 {
		return nil
	}

	paramStart := -1
	for i := fnIdx + 1; i < len(lines) && i < fnIdx+10; i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(strings.ToLower(trimmed), "param") && strings.Contains(trimmed, "(") {
			paramStart = i
			break
		}
	}
	if paramStart == -1 {
		return nil
	}

	depth := 0
	var blockLines []string
	for i := paramStart; i < len(lines); i++ {
		line := lines[i]
		depth += strings.Count(line, "(") - strings.Count(line, ")")
		blockLines = append(blockLines, line)
		if depth <= 0 {
			break
		}
	}

	var params []ParamDetail
	var pendingMandatory bool
	var pendingValidateSet []string
	for _, raw := range blockLines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if psParamMandatory.MatchString(line) {
			pendingMandatory = true
		}
		if m := psValidateSetLine.FindStringSubmatch(line); len(m) == 2 {
			vals := strings.Split(m[1], ",")
			for _, v := range vals {
				v = strings.TrimSpace(v)
				v = strings.Trim(v, `"'`)
				if v != "" {
					pendingValidateSet = append(pendingValidateSet, v)
				}
			}
		}
		if m := psParamVarLine.FindStringSubmatch(line); len(m) == 3 {
			typeName := strings.TrimSpace(m[1])
			paramName := strings.TrimSpace(m[2])
			if paramName == "" {
				continue
			}
			pd := ParamDetail{
				Name:        paramName,
				Type:        typeName,
				Mandatory:   pendingMandatory,
				Switch:      strings.EqualFold(typeName, "switch"),
				ValidateSet: pendingValidateSet,
			}
			if dm := psDefaultValue.FindStringSubmatch(line); len(dm) == 2 {
				pd.Default = strings.TrimSpace(strings.TrimRight(dm[1], ","))
				pd.Default = strings.Trim(pd.Default, `"'`)
			}
			params = append(params, pd)
			pendingMandatory = false
			pendingValidateSet = nil
		}
	}
	return params
}

func sourcesForFunction(loadFiles []string, functionName string) []string {
	out := make([]string, 0)
	for _, p := range loadFiles {
		names, err := readPowerShellFunctionNames(p)
		if err != nil {
			continue
		}
		for _, n := range names {
			if n == functionName {
				out = append(out, p)
				break
			}
		}
	}
	return out
}

func preferredPluginExtOrder() []string {
	if shellLooksLikeBash() {
		if runtime.GOOS == "windows" {
			return []string{".sh", ".ps1", ".cmd", ".bat", ".exe", "", ".out"}
		}
		return []string{".sh", "", ".out", ".ps1"}
	}
	if runtime.GOOS == "windows" {
		return []string{".ps1", ".cmd", ".bat", ".exe", ".sh", "", ".out"}
	}
	return []string{".sh", "", ".out", ".ps1"}
}

func shellLooksLikeBash() bool {
	shell := strings.ToLower(strings.TrimSpace(os.Getenv("SHELL")))
	return strings.Contains(shell, "bash") || strings.Contains(shell, "zsh") || strings.Contains(shell, "fish")
}

func firstAvailableBinary(names ...string) string {
	for _, n := range names {
		if _, err := exec.LookPath(n); err == nil {
			return n
		}
	}
	return ""
}

func quotePowerShellArg(v string) string {
	return "'" + strings.ReplaceAll(v, "'", "''") + "'"
}

type psNamedArg struct {
	Name     string
	Value    string
	IsSwitch bool
}

func looksLikePowerShellNamedToken(v string) bool {
	token := strings.TrimSpace(v)
	if !strings.HasPrefix(token, "-") || token == "-" {
		return false
	}
	// Treat negative numbers (for example -1, -0.5) as values, not parameter names.
	if len(token) > 1 {
		ch := token[1]
		if (ch >= '0' && ch <= '9') || ch == '.' {
			return false
		}
	}
	return true
}

func splitPowerShellSplatArgs(args []string) ([]psNamedArg, []string) {
	named := make([]psNamedArg, 0, len(args))
	positional := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		current := strings.TrimSpace(args[i])
		if !looksLikePowerShellNamedToken(current) {
			positional = append(positional, args[i])
			continue
		}
		name := strings.TrimLeft(current, "-")
		if name == "" {
			positional = append(positional, args[i])
			continue
		}
		if i+1 < len(args) && !looksLikePowerShellNamedToken(args[i+1]) {
			named = append(named, psNamedArg{Name: name, Value: args[i+1]})
			i++
			continue
		}
		named = append(named, psNamedArg{Name: name, IsSwitch: true})
	}
	return named, positional
}

func buildPowerShellFunctionScript(profilePaths []string, functionName string, args []string) string {
	quotedPaths := make([]string, 0, len(profilePaths))
	for _, p := range profilePaths {
		quotedPaths = append(quotedPaths, quotePowerShellArg(p))
	}
	namedArgs, positionalArgs := splitPowerShellSplatArgs(args)

	lines := []string{
		"Set-StrictMode -Version Latest",
		"$ErrorActionPreference='Stop'",
		"$dmProfilePaths=@(" + strings.Join(quotedPaths, ",") + ")",
		"$dmNamedArgs=@{}",
		"$dmPositionalArgs=@()",
	}
	for _, a := range namedArgs {
		valueExpr := "$true"
		if !a.IsSwitch {
			valueExpr = quotePowerShellArg(a.Value)
		}
		lines = append(lines, "$dmNamedArgs["+quotePowerShellArg(a.Name)+"]="+valueExpr)
	}
	for _, a := range positionalArgs {
		lines = append(lines, "$dmPositionalArgs+="+quotePowerShellArg(a))
	}
	lines = append(lines,
		"foreach($dmProfilePath in $dmProfilePaths){ if(Test-Path -LiteralPath $dmProfilePath){ . $dmProfilePath } }",
		"if(-not(Get-Command -Name "+quotePowerShellArg(functionName)+" -CommandType Function -ErrorAction SilentlyContinue)){",
		"  throw \"Function '"+functionName+"' was not loaded from plugin sources.\"",
		"}",
		"& "+quotePowerShellArg(functionName)+" @dmNamedArgs @dmPositionalArgs",
	)
	return strings.Join(lines, "\n") + "\n"
}

func runPowerShellFunction(profilePaths []string, functionName string, args []string) error {
	_, err := runPowerShellFunctionCapture(profilePaths, functionName, args)
	return err
}

func runPowerShellFunctionCapture(profilePaths []string, functionName string, args []string) (string, error) {
	ps := firstAvailableBinary("pwsh", "powershell")
	if ps == "" {
		return "", errors.New("pwsh/powershell executable not found")
	}

	scriptBody := buildPowerShellFunctionScript(profilePaths, functionName, args)

	tmp, tmpErr := os.CreateTemp("", "dm-plugin-*.ps1")
	if tmpErr != nil {
		return "", tmpErr
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer func() { _ = os.Remove(tmpPath) }()
	if writeErr := os.WriteFile(tmpPath, []byte(scriptBody), 0600); writeErr != nil {
		return "", writeErr
	}

	cmd := exec.Command(ps, "-NoProfile", "-NonInteractive", "-File", tmpPath)
	var output bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &output)
	cmd.Stderr = io.MultiWriter(os.Stderr, &output)
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return output.String(), &RunError{Err: err, Output: output.String()}
	}
	return output.String(), nil
}

func execPlugin(path string, args []string) error {
	_, err := execPluginCapture(path, args)
	return err
}

func execPluginCapture(path string, args []string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		switch ext {
		case ".ps1":
			ps := firstAvailableBinary("pwsh", "powershell")
			if ps == "" {
				return "", errors.New("powershell executable not found")
			}
			cmd = exec.Command(ps, "-NoProfile", "-NonInteractive", "-File", path)
		case ".sh":
			sh := firstAvailableBinary("sh", "bash")
			if sh == "" {
				return "", errors.New("sh/bash executable not found")
			}
			cmd = exec.Command(sh, path)
		case ".cmd", ".bat":
			cmd = exec.Command("cmd", "/C", path)
		case ".exe", "", ".out":
			cmd = exec.Command(path)
		default:
			return "", errors.New("unsupported plugin type on windows")
		}
	default:
		switch ext {
		case ".ps1":
			ps := firstAvailableBinary("pwsh", "powershell")
			if ps == "" {
				return "", errors.New("pwsh/powershell executable not found")
			}
			cmd = exec.Command(ps, "-File", path)
		case ".sh":
			cmd = exec.Command("sh", path)
		default:
			cmd = exec.Command(path)
		}
	}

	if len(args) > 0 {
		cmd.Args = append(cmd.Args, args...)
	}

	var output bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &output)
	cmd.Stderr = io.MultiWriter(os.Stderr, &output)
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return output.String(), &RunError{Err: err, Output: output.String()}
	}
	return output.String(), nil
}

func isSupportedPlugin(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".ps1" || ext == ".cmd" || ext == ".bat" || ext == ".exe" || ext == ".sh" || ext == "" || ext == ".out"
}

func runnerForPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch runtime.GOOS {
	case "windows":
		switch ext {
		case ".ps1":
			return "powershell -File"
		case ".sh":
			return "sh"
		case ".cmd", ".bat":
			return "cmd /C"
		case ".exe", "", ".out":
			return "direct"
		}
	default:
		switch ext {
		case ".ps1":
			return "pwsh -File"
		case ".sh":
			return "sh"
		default:
			return "direct"
		}
	}
	return "unknown"
}

func pluginName(name string) string {
	ext := filepath.Ext(name)
	if ext == "" {
		return name
	}
	return strings.TrimSuffix(name, ext)
}
