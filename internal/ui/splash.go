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
	fmt.Println(Accent("dm - Demtrodev CLI"))
	fmt.Println()
	fmt.Println(Accent("Workspace"))
	fmt.Println(Muted("---------"))
	fmt.Printf("%s %s\n", Muted("Base dir   :"), d.BaseDir)
	fmt.Printf("%s %s\n", Muted("Version    :"), d.Version)
	if strings.TrimSpace(d.ExeBuiltAt) != "" {
		fmt.Printf("%s %s\n", Muted("Exe built  :"), d.ExeBuiltAt)
	}
	fmt.Println()
	fmt.Println(Accent("Quick Start"))
	fmt.Println(Muted("-----------"))
	fmt.Println(Prompt("dm -t or dm tools") + Muted("       Tools menu"))
	fmt.Println(Prompt("dm -p or dm plugins") + Muted("     Plugin menu"))
	fmt.Println(Prompt("dm ask") + Muted("                  Agent ask mode"))
	fmt.Println(Prompt("dm doctor") + Muted("               Runtime diagnostics"))
	fmt.Println(Muted("-----------"))
	fmt.Println(Prompt("dm -o ps_profile") + Muted("        open $PROFILE in Notepad"))
	fmt.Println(Prompt("dm ps_profile") + Muted("           List functions/aliases from $PROFILE"))
	fmt.Println(Prompt("dm completion install"))
}

//go:embed logo.txt
var logoText string
