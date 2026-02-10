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

type Info struct {
	Name        string
	Kind        string
	Path        string
	Sources     []string
	Runner      string
	Synopsis    string
	Description string
	Parameters  []string
	Examples    []string
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
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
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

	candidate, err := findPlugin(dir, name)
	if err != nil {
		return Info{}, err
	}
	if candidate != "" {
		return Info{
			Name:    name,
			Kind:    "script",
			Path:    candidate,
			Sources: []string{candidate},
			Runner:  runnerForPath(candidate),
		}, nil
	}

	fnPath, loadFiles, found, err := findPowerShellFunction(dir, name)
	if err != nil {
		return Info{}, err
	}
	if !found {
		return Info{}, fmt.Errorf("%w: %s", ErrNotFound, name)
	}

	help, _ := parsePowerShellFunctionHelp(fnPath, name)
	sources := sourcesForFunction(loadFiles, name)
	if len(sources) == 0 {
		sources = []string{fnPath}
	}

	return Info{
		Name:        name,
		Kind:        "function",
		Path:        fnPath,
		Sources:     sources,
		Runner:      "powershell function bridge",
		Synopsis:    help.Synopsis,
		Description: help.Description,
		Parameters:  help.Parameters,
		Examples:    help.Examples,
	}, nil
}

func Run(baseDir, name string, args []string) error {
	dir := filepath.Join(baseDir, "plugins")
	candidate, err := findPlugin(dir, name)
	if err != nil {
		return err
	}
	if candidate == "" {
		_, loadFiles, found, fErr := findPowerShellFunction(dir, name)
		if fErr != nil {
			return fErr
		}
		if !found {
			return fmt.Errorf("%w: %s", ErrNotFound, name)
		}
		return runPowerShellFunction(loadFiles, name, args)
	}
	return execPlugin(candidate, args)
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

func runPowerShellFunction(profilePaths []string, functionName string, args []string) error {
	ps := firstAvailableBinary("pwsh", "powershell")
	if ps == "" {
		return errors.New("pwsh/powershell executable not found")
	}
	quotedPaths := make([]string, 0, len(profilePaths))
	for _, p := range profilePaths {
		quotedPaths = append(quotedPaths, quotePowerShellArg(p))
	}
	script := "$dmProfilePaths=@(" + strings.Join(quotedPaths, ",") + "); $oldEap=$ErrorActionPreference; $ErrorActionPreference='SilentlyContinue'; foreach($dmProfilePath in $dmProfilePaths){ if(Test-Path $dmProfilePath){ Invoke-Expression (Get-Content -Raw $dmProfilePath) } }; $ErrorActionPreference=$oldEap; " + functionName
	if len(args) > 0 {
		quoted := make([]string, 0, len(args))
		for _, a := range args {
			quoted = append(quoted, quotePowerShellArg(a))
		}
		script += " " + strings.Join(quoted, " ")
	}
	cmd := exec.Command(ps, "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)
	var output bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &output)
	cmd.Stderr = io.MultiWriter(os.Stderr, &output)
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return &RunError{Err: err, Output: output.String()}
	}
	return nil
}

func execPlugin(path string, args []string) error {
	ext := strings.ToLower(filepath.Ext(path))
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		switch ext {
		case ".ps1":
			ps := firstAvailableBinary("powershell", "pwsh")
			if ps == "" {
				return errors.New("powershell executable not found")
			}
			cmd = exec.Command(ps, "-ExecutionPolicy", "Bypass", "-File", path)
		case ".sh":
			sh := firstAvailableBinary("sh", "bash")
			if sh == "" {
				return errors.New("sh/bash executable not found")
			}
			cmd = exec.Command(sh, path)
		case ".cmd", ".bat":
			cmd = exec.Command("cmd", "/C", path)
		case ".exe", "", ".out":
			cmd = exec.Command(path)
		default:
			return errors.New("unsupported plugin type on windows")
		}
	default:
		switch ext {
		case ".ps1":
			ps := firstAvailableBinary("pwsh", "powershell")
			if ps == "" {
				return errors.New("pwsh/powershell executable not found")
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
		return &RunError{Err: err, Output: output.String()}
	}
	return nil
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
