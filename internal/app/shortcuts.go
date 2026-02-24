package app

import "strings"

func mapGroupShortcut(arg string) ([]string, bool) {
	switch arg {
	case "-a", "--add-alias":
		return []string{"alias", "add"}, true
	case "-t", "--tools":
		return []string{"tools"}, true
	case "-p", "--plugins":
		return []string{"plugins"}, true
	case "-o", "--open":
		return []string{"open"}, true
	case "-r", "--run-alias":
		return []string{"alias", "run"}, true
	default:
		return nil, false
	}
}

func rewriteGroupShortcuts(args []string) []string {
	out := make([]string, 0, len(args))
	commandResolved := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if commandResolved {
			out = append(out, arg)
			continue
		}
		groupTokens, ok := mapGroupShortcut(arg)
		if !ok {
			out = append(out, arg)
			if !strings.HasPrefix(arg, "-") {
				commandResolved = true
			}
			continue
		}
		out = append(out, groupTokens...)
		commandResolved = true
		if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			out = append(out, args[i+1])
			i++
		}
	}
	return out
}
