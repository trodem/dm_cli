package ui

import (
	_ "embed"
	"fmt"
	"strings"
)

type SplashData struct {
	BaseDir string
	Version string
}

func PrintSplash(d SplashData) {
	fmt.Println(Accent(strings.TrimRight(logoText, "\n")))
	fmt.Println(Accent("dm - Demtrodev CLI"))
	fmt.Println()
	fmt.Println(Accent("Workspace"))
	fmt.Println(Muted("---------"))
	fmt.Printf("%s %s\n", Muted("Base dir   :"), d.BaseDir)
	fmt.Printf("%s %s\n", Muted("Version    :"), d.Version)
	fmt.Println()
	fmt.Println(Accent("Quick Start"))
	fmt.Println(Muted("-----------"))
	fmt.Println(Prompt("dm -t or dm tools") + Muted("    Tools menu"))
	fmt.Println(Prompt("dm -p or dm plugins") + Muted("  Pluging menu"))
	fmt.Println(Muted("-----------"))
	fmt.Println(Prompt("dm -o ps_profile") + Muted("  open $PROFILE in Notepad"))
	fmt.Println(Prompt("dm -o profile") + Muted("     open plugins/functions/0_powershell_profile.ps1 in Notepad"))
	fmt.Println(Prompt("dm -p profile") + Muted("     list functions/aliases from plugin profile file"))
	fmt.Println(Prompt("dm ps_profile") + Muted("     List functions/aliases from PowerShell $PROFILE"))
	fmt.Println(Prompt("dm cp profile") + Muted("     Overwrite $PROFILE from plugin profile file"))
	fmt.Println(Prompt("dm completion install"))
}

//go:embed logo.txt
var logoText string
