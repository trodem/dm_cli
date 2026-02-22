package ui

import (
	_ "embed"
	"fmt"
	"strings"
)

type SplashData struct {
	BaseDir    string
	Version    string
	ExeBuiltAt string
}

func PrintSplash(d SplashData) {
	fmt.Println(Accent(strings.TrimRight(logoText, "\n")))
	fmt.Println()

	ver := d.Version
	if ver == "" {
		ver = "dev"
	}
	meta := ver
	if strings.TrimSpace(d.ExeBuiltAt) != "" {
		meta += Muted("  built " + d.ExeBuiltAt)
	}
	fmt.Println("  " + meta)
	fmt.Println("  " + Muted(d.BaseDir))
	fmt.Println()

	fmt.Println("  " + Accent("Commands"))
	printCmd("dm ask", "AI agent (chat, plugins, tools)")
	printCmd("dm ask -f <file>", "AI agent with file context")
	printCmd("dm tools", "Interactive tools menu")
	printCmd("dm plugins", "Plugin browser & runner")
	printCmd("dm doctor", "Runtime diagnostics")
	fmt.Println()

	fmt.Println("  " + Accent("Shortcuts"))
	printCmd("dm -t", "tools")
	printCmd("dm -p", "plugins")
	printCmd("dm -o ps_profile", "open $PROFILE")
	fmt.Println()
	fmt.Println("  " + Muted("Run dm <command> --help for details"))
}

func printCmd(cmd, desc string) {
	pad := 24 - len(cmd)
	if pad < 2 {
		pad = 2
	}
	fmt.Printf("  %s%s%s\n", Prompt(cmd), strings.Repeat(" ", pad), Muted(desc))
}

//go:embed logo.txt
var logoText string
