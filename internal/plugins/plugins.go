package plugins

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

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

type RunResult struct {
	Output string
	Err    error
}

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

func isSupportedPlugin(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".ps1" || ext == ".cmd" || ext == ".bat" || ext == ".exe" || ext == ".sh" || ext == "" || ext == ".out"
}

func pluginName(name string) string {
	ext := filepath.Ext(name)
	if ext == "" {
		return name
	}
	return strings.TrimSuffix(name, ext)
}
