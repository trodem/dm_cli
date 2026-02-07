package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cli/internal/config"
	"cli/internal/plugins"
	"cli/internal/runner"
	"cli/internal/search"
	"cli/internal/store"
	"cli/internal/ui"
)

func Run(args []string) int {
	baseDir, err := exeDir()
	if err != nil {
		fmt.Println("Errore: non riesco a determinare la cartella dell'eseguibile:", err)
		return 1
	}

	opts, rest := parseFlags(args)

	cfgPath := filepath.Join(baseDir, "dm.json")
	if opts.Pack == "" {
		if active, err := store.GetActivePack(baseDir); err == nil && active != "" {
			opts.Pack = active
		}
	}
	cfg, err := config.Load(cfgPath, config.Options{
		Profile:  opts.Profile,
		UseCache: !opts.NoCache,
		Pack:     opts.Pack,
	})
	if err != nil {
		fmt.Println("Errore caricando dm.json:", err)
		return 1
	}

	if len(rest) == 0 {
		ui.PrintHelp(cfg)
		return 0
	}

	args = rest

	// comandi globali
	switch args[0] {
	case "help":
		ui.PrintHelp(cfg)
		return 0
	case "aliases", "config":
		ui.PrintAliases(cfg)
		return 0
	case "list":
		return runList(cfg, args[1:])
	case "add":
		return runAdd(baseDir, opts, args[1:])
	case "pack":
		return runPack(baseDir, args[1:])
	case "validate":
		return runValidate(cfg)
	case "plugin":
		return runPlugin(baseDir, args[1:])
	case "find", "search":
		if len(args) < 2 {
			fmt.Println("Uso: dm find <query>")
			return 0
		}
		knowledgeDir := config.ResolvePath(baseDir, cfg.Search.Knowledge)
		query := strings.Join(args[1:], " ")
		search.InKnowledge(knowledgeDir, query)
		return 0
	case "run":
		if len(args) < 2 {
			fmt.Println("Uso: dm run <alias>")
			return 0
		}
		name := args[1]
		runner.RunAlias(cfg, name, "")
		return 0
	}

	// PROJECT MODE: dm <project> <action>
	if _, ok := cfg.Projects[args[0]]; ok && len(args) >= 2 {
		action := args[1]
		runner.RunProjectCommand(cfg, args[0], action, baseDir)
		return 0
	}

	// INTERACTIVE TARGET: dm <name>
	name := args[0]

	// target puo' essere jump o project
	targetPath, isJump := cfg.Jump[name]
	_, isProject := cfg.Projects[name]

	if !isJump && !isProject {
		// fallback: come query di ricerca
		knowledgeDir := config.ResolvePath(baseDir, cfg.Search.Knowledge)
		search.InKnowledge(knowledgeDir, strings.Join(args, " "))
		return 0
	}

	if isProject {
		targetPath = cfg.Projects[name].Path
	}

	targetPath = config.ResolvePath(baseDir, targetPath)
	ui.ShowMenu(cfg, name, targetPath, baseDir)
	return 0
}

func exeDir() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(p), nil
}

type flags struct {
	Profile string
	NoCache bool
	Pack    string
}

func parseFlags(args []string) (flags, []string) {
	var out []string
	var f flags
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--no-cache" {
			f.NoCache = true
			continue
		}
		if arg == "--profile" && i+1 < len(args) {
			f.Profile = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(arg, "--profile=") {
			f.Profile = strings.TrimPrefix(arg, "--profile=")
			continue
		}
		if arg == "--pack" && i+1 < len(args) {
			f.Pack = args[i+1]
			i++
			continue
		}
		if arg == "-p" && i+1 < len(args) {
			f.Pack = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(arg, "--pack=") {
			f.Pack = strings.TrimPrefix(arg, "--pack=")
			continue
		}
		out = append(out, arg)
	}
	return f, out
}

func runValidate(cfg config.Config) int {
	issues := config.Validate(cfg)
	if len(issues) == 0 {
		fmt.Println("OK: configurazione valida")
		return 0
	}
	for _, issue := range issues {
		fmt.Printf("%s: %s\n", issue.Level, issue.Message)
	}
	return 1
}

func runList(cfg config.Config, args []string) int {
	if len(args) == 0 {
		fmt.Println("Uso: dm list <jumps|runs|projects|actions>")
		return 0
	}
	switch args[0] {
	case "jumps":
		ui.PrintMap(cfg.Jump)
	case "runs":
		ui.PrintMap(cfg.Run)
	case "projects":
		ui.PrintProjects(cfg.Projects)
	case "actions":
		if len(args) < 2 {
			fmt.Println("Uso: dm list actions <project>")
			return 0
		}
		p, ok := cfg.Projects[args[1]]
		if !ok {
			fmt.Println("Project non trovato:", args[1])
			return 0
		}
		ui.PrintMap(p.Commands)
	default:
		fmt.Println("Uso: dm list <jumps|runs|projects|actions>")
	}
	return 0
}

func runAdd(baseDir string, opts flags, args []string) int {
	if len(args) < 1 {
		fmt.Println("Uso: dm add <jump|run|project|action> ...")
		return 0
	}
	pack := opts.Pack
	if pack == "" {
		fmt.Println("Errore: nessun pack attivo. Usa -p <pack> o dm pack use <name>.")
		return 1
	}
	switch args[0] {
	case "jump":
		if len(args) < 3 {
			fmt.Println("Uso: dm add jump <name> <path>")
			return 0
		}
		path := filepath.Join(baseDir, "packs", pack, "pack.json")
		pf, err := store.LoadPackFile(path)
		if err != nil {
			fmt.Println("Errore:", err)
			return 1
		}
		pf.Jump[args[1]] = args[2]
		if err := store.SavePackFile(path, pf); err != nil {
			fmt.Println("Errore:", err)
			return 1
		}
		fmt.Println("OK: jump aggiunto")
		return 0
	case "run":
		if len(args) < 3 {
			fmt.Println("Uso: dm add run <name> <command>")
			return 0
		}
		path := filepath.Join(baseDir, "packs", pack, "pack.json")
		pf, err := store.LoadPackFile(path)
		if err != nil {
			fmt.Println("Errore:", err)
			return 1
		}
		pf.Run[args[1]] = strings.Join(args[2:], " ")
		if err := store.SavePackFile(path, pf); err != nil {
			fmt.Println("Errore:", err)
			return 1
		}
		fmt.Println("OK: run aggiunto")
		return 0
	case "project":
		if len(args) < 3 {
			fmt.Println("Uso: dm add project <name> <path>")
			return 0
		}
		path := filepath.Join(baseDir, "packs", pack, "pack.json")
		pf, err := store.LoadPackFile(path)
		if err != nil {
			fmt.Println("Errore:", err)
			return 1
		}
		if pf.Projects == nil {
			pf.Projects = map[string]store.Project{}
		}
		pf.Projects[args[1]] = store.Project{
			Path:     args[2],
			Commands: map[string]string{},
		}
		if err := store.SavePackFile(path, pf); err != nil {
			fmt.Println("Errore:", err)
			return 1
		}
		fmt.Println("OK: project aggiunto")
		return 0
	case "action":
		if len(args) < 4 {
			fmt.Println("Uso: dm add action <project> <name> <command>")
			return 0
		}
		project := args[1]
		action := args[2]
		cmd := strings.Join(args[3:], " ")

		path := filepath.Join(baseDir, "packs", pack, "pack.json")
		pf, err := store.LoadPackFile(path)
		if err != nil {
			fmt.Println("Errore:", err)
			return 1
		}
		p, ok := pf.Projects[project]
		if !ok {
			fmt.Println("Project non trovato. Aggiungi prima il project.")
			return 1
		}
		if p.Commands == nil {
			p.Commands = map[string]string{}
		}
		p.Commands[action] = cmd
		pf.Projects[project] = p
		if err := store.SavePackFile(path, pf); err != nil {
			fmt.Println("Errore:", err)
			return 1
		}
		fmt.Println("OK: action aggiunta")
		return 0
	default:
		fmt.Println("Uso: dm add <jump|run|project|action> ...")
		return 0
	}
}

func runPlugin(baseDir string, args []string) int {
	if len(args) == 0 {
		fmt.Println("Uso: dm plugin <list|run> ...")
		return 0
	}
	switch args[0] {
	case "list":
		items, err := plugins.List(baseDir)
		if err != nil {
			fmt.Println("Errore:", err)
			return 1
		}
		if len(items) == 0 {
			fmt.Println("Nessun plugin trovato.")
			return 0
		}
		for _, p := range items {
			fmt.Println(p.Name)
		}
		return 0
	case "run":
		if len(args) < 2 {
			fmt.Println("Uso: dm plugin run <name> [args...]")
			return 0
		}
		if err := plugins.Run(baseDir, args[1], args[2:]); err != nil {
			fmt.Println("Errore:", err)
			return 1
		}
		return 0
	default:
		fmt.Println("Uso: dm plugin <list|run> ...")
		return 0
	}
}

func runPack(baseDir string, args []string) int {
	if len(args) < 1 {
		fmt.Println("Uso: dm pack <new|list|info|use|current|unset> [name]")
		return 0
	}
	switch args[0] {
	case "new":
		if len(args) < 2 {
			fmt.Println("Uso: dm pack new <name>")
			return 0
		}
		name := args[1]
		if err := store.CreatePack(baseDir, name); err != nil {
			fmt.Println("Errore:", err)
			return 1
		}
		fmt.Println("OK: pack creato")
		return 0
	case "list":
		items, err := store.ListPacks(baseDir)
		if err != nil {
			fmt.Println("Errore:", err)
			return 1
		}
		if len(items) == 0 {
			fmt.Println("Nessun pack trovato.")
			return 0
		}
		for _, name := range items {
			fmt.Println(name)
		}
		return 0
	case "info":
		if len(args) < 2 {
			fmt.Println("Uso: dm pack info <name>")
			return 0
		}
		info, err := store.GetPackInfo(baseDir, args[1])
		if err != nil {
			fmt.Println("Errore:", err)
			return 1
		}
		fmt.Printf("pack: %s\n", info.Name)
		fmt.Printf("path: %s\n", info.Path)
		if info.Knowledge != "" {
			fmt.Printf("knowledge: %s\n", info.Knowledge)
		}
		fmt.Printf("jumps: %d\n", info.Jumps)
		fmt.Printf("runs: %d\n", info.Runs)
		fmt.Printf("projects: %d\n", info.Projects)
		fmt.Printf("actions: %d\n", info.Actions)
		return 0
	case "use":
		if len(args) < 2 {
			fmt.Println("Uso: dm pack use <name>")
			return 0
		}
		name := args[1]
		if !store.PackExists(baseDir, name) {
			fmt.Println("Pack non trovato:", name)
			return 1
		}
		if err := store.SetActivePack(baseDir, name); err != nil {
			fmt.Println("Errore:", err)
			return 1
		}
		fmt.Println("OK: pack attivo ->", name)
		return 0
	case "current":
		name, err := store.GetActivePack(baseDir)
		if err != nil {
			fmt.Println("Errore:", err)
			return 1
		}
		if name == "" {
			fmt.Println("Nessun pack attivo.")
			return 0
		}
		fmt.Println(name)
		return 0
	case "unset":
		if err := store.ClearActivePack(baseDir); err != nil {
			fmt.Println("Errore:", err)
			return 1
		}
		fmt.Println("OK: pack attivo rimosso")
		return 0
	default:
		fmt.Println("Uso: dm pack <new|list|info|use|current|unset> [name]")
		return 0
	}
}
