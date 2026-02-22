package app

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"cli/internal/plugins"
	"cli/tools"
)

func buildPluginCatalog(baseDir string) string {
	items, err := plugins.ListEntries(baseDir, true)
	if err != nil || len(items) == 0 {
		return "(none)"
	}

	type catalogEntry struct {
		item plugins.Entry
		line string
	}

	groups := map[string][]catalogEntry{}
	groupOrder := []string{}

	for _, item := range items {
		info, _ := plugins.GetInfo(baseDir, item.Name)
		line := fmt.Sprintf("- %s", item.Name)
		if strings.TrimSpace(info.Synopsis) != "" {
			line += ": " + info.Synopsis
		}
		if len(info.ParamDetails) > 0 {
			line += " | params: " + formatParamDetailsForCatalog(info.ParamDetails)
		} else if len(info.Parameters) > 0 {
			line += " | params: " + strings.Join(info.Parameters, "; ")
		}

		key := toolkitGroupKey(item.Path)
		if _, exists := groups[key]; !exists {
			groupOrder = append(groupOrder, key)
		}
		groups[key] = append(groups[key], catalogEntry{item: item, line: line})
	}

	sort.Strings(groupOrder)

	var out []string
	for _, key := range groupOrder {
		label := toolkitLabel(key)
		out = append(out, fmt.Sprintf("\n[%s]", label))
		for _, entry := range groups[key] {
			out = append(out, entry.line)
		}
	}
	return strings.Join(out, "\n")
}

func toolkitGroupKey(filePath string) string {
	normalized := strings.ReplaceAll(filePath, "\\", "/")
	base := filepath.Base(normalized)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

func toolkitLabel(groupKey string) string {
	name := groupKey
	if len(name) >= 2 && name[0] >= '0' && name[0] <= '9' && name[1] == '_' {
		name = name[2:]
	}
	name = strings.TrimSuffix(name, "_Toolkit")
	return strings.ReplaceAll(name, "_", " ")
}

func formatParamDetailsForCatalog(details []plugins.ParamDetail) string {
	parts := make([]string, 0, len(details))
	for _, d := range details {
		s := d.Name
		if d.Switch {
			s += " [switch]"
		} else if d.Type != "" {
			s += " [" + d.Type + "]"
		}
		if d.Mandatory {
			s += " (required)"
		}
		if len(d.ValidateSet) > 0 {
			s += " values=" + strings.Join(d.ValidateSet, "|")
		}
		if d.Default != "" {
			s += " default=" + d.Default
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, "; ")
}

func buildToolsCatalog() string {
	return tools.BuildAgentCatalog()
}

func isKnownTool(name string) bool {
	return tools.IsKnownTool(name)
}
