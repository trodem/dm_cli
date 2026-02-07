package ui

import "fmt"

type SplashData struct {
	BaseDir    string
	PackCount  int
	ActivePack string
	ConfigPath string
	ConfigUsed bool
	Ascii      string
}

func PrintSplash(d SplashData) {
	if d.Ascii != "" {
		fmt.Println(d.Ascii)
	} else {
		fmt.Println("dm")
	}
	fmt.Println("dm - Demtrodev CLI")
	fmt.Println()
	fmt.Println("Project Info")
	fmt.Println("------------")
	fmt.Printf("Base dir   : %s\n", d.BaseDir)
	fmt.Printf("Packs      : %d\n", d.PackCount)
	if d.ActivePack != "" {
		fmt.Printf("Active pack: %s\n", d.ActivePack)
	} else {
		fmt.Printf("Active pack: none\n")
	}
	if d.ConfigUsed {
		fmt.Printf("Config     : %s\n", d.ConfigPath)
	} else {
		fmt.Printf("Config     : default (packs/*/pack.json)\n")
	}
	fmt.Println()
	fmt.Println("Quick Start")
	fmt.Println("-----------")
	fmt.Println("dm help")
	fmt.Println("dm pack list")
	fmt.Println("dm pack use <name>")
	fmt.Println("dm tools")
}
