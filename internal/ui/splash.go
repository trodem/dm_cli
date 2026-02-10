package ui

import (
	_ "embed"
	"fmt"
	"strings"
)

type SplashData struct {
	BaseDir    string
	PackCount  int
	ActivePack string
	ConfigPath string
	ConfigUsed bool
}

func PrintSplash(d SplashData) {
	fmt.Println(Accent(strings.TrimRight(logoText, "\n")))
	fmt.Println(Accent("dm - Demtrodev CLI"))
	fmt.Println()
	fmt.Println(Accent("Workspace"))
	fmt.Println(Muted("---------"))
	fmt.Printf("%s %s\n", Muted("Base dir   :"), d.BaseDir)
	fmt.Printf("%s %d\n", Muted("Packs      :"), d.PackCount)
	if d.ActivePack != "" {
		fmt.Printf("%s %s\n", Muted("Active pack:"), d.ActivePack)
	} else {
		fmt.Printf("%s %s\n", Muted("Active pack:"), Warn("none"))
	}
	if d.ConfigUsed {
		fmt.Printf("%s %s\n", Muted("Config     :"), d.ConfigPath)
	} else {
		fmt.Printf("%s %s\n", Muted("Config     :"), "default (packs/*/pack.json)")
	}
	fmt.Println()
	fmt.Println(Accent("Quick Start"))
	fmt.Println(Muted("-----------"))
	fmt.Println(Prompt("dm help"))
	fmt.Println(Prompt("dm pack list"))
	fmt.Println(Prompt("dm pack current"))
	fmt.Println(Prompt("dm pack use <name>"))
	fmt.Println(Prompt("dm tools"))
	fmt.Println(Prompt("dm -t s"))
	fmt.Println(Prompt("dm -k list"))
	fmt.Println(Prompt("dm -g"))
	fmt.Println(Prompt("dm plugin list"))
	fmt.Println(Prompt("dm completion install"))
}

//go:embed logo.txt
var logoText string
