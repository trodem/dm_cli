package plugins

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	psParamMandatory  = regexp.MustCompile(`(?i)\[Parameter\s*\([^)]*Mandatory\b`)
	psParamVarLine    = regexp.MustCompile(`(?i)^\s*(?:\[([^\]]+)\])?\s*\$(\w+)`)
	psValidateSetLine = regexp.MustCompile(`(?i)\[ValidateSet\s*\(([^)]+)\)\]`)
	psDefaultValue    = regexp.MustCompile(`\$\w+\s*=\s*(.+)`)
)

type functionHelp struct {
	Synopsis    string
	Description string
	Parameters  []string
	Examples    []string
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

func findPowerShellFunction(pluginsDir, name string) (string, []string, bool, error) {
	catalog, files, err := collectPowerShellFunctions(pluginsDir)
	if err != nil {
		return "", nil, false, err
	}
	src, ok := catalog[name]
	if !ok {
		return "", nil, false, nil
	}
	return src, files, true, nil
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
