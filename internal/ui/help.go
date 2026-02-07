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
dm - personal CLI

Comandi:
  dm <name>                 menu interattivo (jump/project)
  dm <project> <action>     esegue comando di progetto
  dm run <alias>            esegue alias run
  dm find <query>           cerca nei markdown knowledge
  dm list <type>            elenca jumps/runs/projects/actions
  dm add <type>             aggiunge jump/run/project/action
  dm pack new <name>        crea un nuovo pack
  dm pack list              elenca i pack
  dm pack info <name>       mostra info pack
  dm pack use <name>        imposta pack attivo
  dm pack current           mostra pack attivo
  dm pack unset             rimuove pack attivo
  dm validate               valida configurazione
  dm plugin <cmd>           gestisce plugins
  dm aliases                mostra config in modo leggibile
  dm help                   aiuto

Flags:
  --profile <name>              usa un profilo specifico
  --pack <name> / -p <name>     usa un pack specifico
  --no-cache                    disabilita cache config
`)

	fmt.Println("Suggerimento CD (senza modifiche shell):")
	fmt.Println(`  cd $(dm <name>  => scegli "Print path")`)
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
