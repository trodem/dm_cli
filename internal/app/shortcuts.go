package app

import "strings"

func mapGroupShortcut(arg string) (string, bool) {
	switch arg {
	case "-t", "--tools":
		return "tools", true
	case "-p", "--plugins":
		return "plugins", true
	default:
		return "", false
	}
}

func rewriteGroupShortcuts(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		group, ok := mapGroupShortcut(arg)
		if !ok {
			out = append(out, arg)
			continue
		}
		out = append(out, group)
		if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			out = append(out, args[i+1])
			i++
		}
	}
	return out
}
