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

Comandi principali:
  dm <name>                 apre menu per jump/project
  dm <project> <action>     esegue un comando di progetto
  dm run <alias>            esegue un alias definito in run
  dm find <query>           cerca nei markdown knowledge
  dm tools                  strumenti (search/rename)

Gestione pack:
  dm pack new <name>        crea un nuovo pack
  dm pack list              elenca i pack
  dm pack info <name>       mostra info pack (counts)
  dm pack help <name>       mostra help del pack
  dm pack use <name>        imposta pack attivo
  dm pack current           mostra pack attivo
  dm pack unset             rimuove pack attivo

Gestione config:
  dm list <type>            elenca jumps/runs/projects/actions
  dm add <type>             aggiunge jump/run/project/action
  dm validate               valida configurazione
  dm aliases                mostra config in modo leggibile

Plugins:
  dm plugin list
  dm plugin run <name> [args...]

Help:
  dm help
  dm help tools
  dm help plugin
  dm help pack <name>

Flags:
  --profile <name>              usa un profilo specifico
  --pack <name> / -p <name>     usa un pack specifico
  --no-cache                    disabilita cache config

Esempi rapidi:
  dm pack list
  dm pack use git
  dm -p git find branch
  dm run gs
  dm git-tools gcommit
  dm tools
`)

	fmt.Println("Suggerimento CD (senza modifiche shell):")
	fmt.Println(`  cd $(dm <name>  => scegli "Print path")`)
	fmt.Println()
}

func PrintToolsHelp() {
	fmt.Println(`
dm tools

Menu:
  1) Search files
  2) Rename files

Search files:
- Recursive search by name contains or extension.
- Sort by name/date/size.

Rename files:
- Interactive batch rename with preview and confirmation.
- Simple text (contains + replace).
- Name filter is case-insensitive.
- Replace is case-insensitive by default; prompt allows case-sensitive.

Examples:
  dm tools
`)
}

func PrintPluginHelp() {
	fmt.Println(`
dm plugin list
dm plugin run <name> [args...]

Descrizione:
- I plugin sono script in plugins/.
- Windows: .ps1, .cmd, .bat, .exe
- Linux/mac: .sh o eseguibili

Esempi:
  dm plugin list
  dm plugin run myscript --foo bar
`)
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
