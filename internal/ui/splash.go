package ui

import (
	_ "embed"
	"fmt"
	"strings"
)

type SplashData struct {
	BaseDir string
}

func PrintSplash(d SplashData) {
	fmt.Println(Accent(strings.TrimRight(logoText, "\n")))
	fmt.Println(Accent("dm - Demtrodev CLI"))
	fmt.Println()
	fmt.Println(Accent("Workspace"))
	fmt.Println(Muted("---------"))
	fmt.Printf("%s %s\n", Muted("Base dir   :"), d.BaseDir)
	fmt.Println()
	fmt.Println(Accent("Quick Start"))
	fmt.Println(Muted("-----------"))
	fmt.Println(Prompt("dm help"))
	fmt.Println(Prompt("dm tools or dm -t"))
	fmt.Println(Prompt("dm plugins or dm -p"))
	fmt.Println(Prompt("dm completion install"))
}

//go:embed logo.txt
var logoText string
