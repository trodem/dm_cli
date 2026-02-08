package ui

import (
	"fmt"
	"sort"
	"strings"

	"cli/internal/config"
)

func PrintAliases(cfg config.Config) {
	fmt.Println("\nJUMP")
	fmt.Println("------------------------")
	PrintMap(cfg.Jump)

	fmt.Println("\nRUN")
	fmt.Println("------------------------")
	PrintMap(cfg.Run)

	fmt.Println("\nPROJECTS")
	fmt.Println("------------------------")
	PrintProjects(cfg.Projects)
	fmt.Println()
}

func PrintMap(m map[string]string) {
	keys := sortedKeys(m)
	for _, k := range keys {
		fmt.Printf("%-12s -> %s\n", k, m[k])
	}
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedKeysProjects(m map[string]config.Project) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func PrintProjects(m map[string]config.Project) {
	names := sortedKeysProjects(m)
	for _, n := range names {
		p := m[n]
		fmt.Printf("%-12s -> %s\n", n, p.Path)
		if len(p.Commands) > 0 {
			cmdNames := sortedKeys(p.Commands)
			fmt.Printf("  actions: %s\n", strings.Join(cmdNames, ", "))
		}
	}
}
