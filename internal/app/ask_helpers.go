package app

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"cli/internal/agent"
	"cli/internal/plugins"
	"cli/internal/ui"
)

func pluginArgsToPS(pluginArgs map[string]string) []string {
	if len(pluginArgs) == 0 {
		return nil
	}
	keys := make([]string, 0, len(pluginArgs))
	for k := range pluginArgs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var args []string
	for _, k := range keys {
		v := strings.TrimSpace(pluginArgs[k])
		paramName := k
		if !strings.HasPrefix(paramName, "-") {
			paramName = "-" + paramName
		}
		lv := strings.ToLower(v)
		if lv == "true" || lv == "" {
			args = append(args, paramName)
			continue
		}
		if lv == "false" {
			continue
		}
		args = append(args, paramName, v)
	}
	return args
}

func formatPluginArgs(pluginArgs map[string]string) string {
	if len(pluginArgs) == 0 {
		return ""
	}
	keys := make([]string, 0, len(pluginArgs))
	for k := range pluginArgs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("-%s %s", k, pluginArgs[k]))
	}
	return strings.Join(parts, " ")
}

func formatToolArgs(args map[string]string) string {
	if len(args) == 0 {
		return ""
	}
	keys := make([]string, 0, len(args))
	for k := range args {
		v := strings.TrimSpace(args[k])
		lc := strings.ToLower(v)
		if v == "" || lc == "<nil>" || lc == "null" {
			continue
		}
		keys = append(keys, k)
	}
	if len(keys) == 0 {
		return ""
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, args[k]))
	}
	return strings.Join(parts, ", ")
}

func plannedActionSummary(decision agent.DecisionResult) string {
	switch strings.ToLower(strings.TrimSpace(decision.Action)) {
	case "run_plugin":
		s := "plugin " + strings.TrimSpace(decision.Plugin)
		if a := formatPluginArgs(decision.PluginArgs); a != "" {
			s += " " + a
		} else if len(decision.Args) > 0 {
			s += " " + strings.Join(decision.Args, " ")
		}
		return s
	case "run_tool":
		s := "tool " + strings.TrimSpace(decision.Tool)
		if args := formatToolArgs(decision.ToolArgs); strings.TrimSpace(args) != "" {
			s += " (" + args + ")"
		}
		return s
	case "create_function":
		desc := strings.TrimSpace(decision.FunctionDescription)
		if len(desc) > askDescMaxLen {
			desc = desc[:askDescMaxLen] + "..."
		}
		return "create function: " + desc
	default:
		if strings.TrimSpace(decision.Answer) != "" {
			return "answer"
		}
		return "noop"
	}
}

func missingMandatoryParams(info plugins.Info, pluginArgs map[string]string) []string {
	var missing []string
	for _, p := range info.ParamDetails {
		if !p.Mandatory {
			continue
		}
		found := false
		for k, v := range pluginArgs {
			if strings.EqualFold(k, p.Name) && strings.TrimSpace(v) != "" {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, p.Name)
		}
	}
	return missing
}

var missingPathErr = regexp.MustCompile(`(?i)required path '([^']+)' does not exist`)

func truncateForHistory(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n... (truncated)"
}

func printAgentActionError(err error) {
	fmt.Fprintln(os.Stderr, "Error:", err)
	combined := strings.TrimSpace(err.Error() + "\n" + plugins.ErrorOutput(err))
	m := missingPathErr.FindStringSubmatch(combined)
	if len(m) == 2 {
		fmt.Println(ui.Warn("Missing required path: " + m[1]))
		fmt.Println(ui.Muted("Fix the path in plugin variables/config, then retry."))
	}
}
