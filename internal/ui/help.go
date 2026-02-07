package ui

import (
	"fmt"
	"sort"
	"strings"

	"cli/internal/config"
)

func PrintHelp(cfg config.Config) {
	_ = cfg
	fmt.Println(`
tellme - personal CLI

Comandi:
  tellme <name>                 menu interattivo (jump/project)
  tellme <project> <action>     esegue comando di progetto
  tellme run <alias>            esegue alias run
  tellme find <query>           cerca nei markdown knowledge
  tellme list <type>            elenca jumps/runs/projects/actions
  tellme add <type>             aggiunge jump/run/project/action
  tellme pack new <name>        crea un nuovo pack
  tellme pack list              elenca i pack
  tellme pack info <name>       mostra info pack
  tellme pack use <name>        imposta pack attivo
  tellme pack current           mostra pack attivo
  tellme pack unset             rimuove pack attivo
  tellme validate               valida configurazione
  tellme plugin <cmd>           gestisce plugins
  tellme aliases                mostra config in modo leggibile
  tellme help                   aiuto

Flags:
  --profile <name>              usa un profilo specifico
  --pack <name> / -p <name>     usa un pack specifico
  --no-cache                    disabilita cache config
`)

	fmt.Println("Suggerimento CD (senza modifiche shell):")
	fmt.Println(`  cd $(tellme <name>  => scegli "Print path")`)
	fmt.Println()
}

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
