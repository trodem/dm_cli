package app

import (
	"fmt"
	"strings"

	"cli/internal/config"
	"cli/internal/runner"
	"cli/internal/search"
	"cli/internal/store"
	"cli/internal/ui"
	"cli/tools"

	"github.com/spf13/cobra"
)

func addCobraSubcommands(root *cobra.Command, opts *flags) {
	root.AddCommand(&cobra.Command{
		Use:     "aliases",
		Aliases: []string{"a"},
		Short:   "Show aliases and projects",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(*opts)
			if err != nil {
				return err
			}
			ui.PrintAliases(rt.Config)
			return nil
		},
	})
	root.AddCommand(&cobra.Command{
		Use:     "config",
		Aliases: []string{"cfg"},
		Short:   "Show aliases and projects",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(*opts)
			if err != nil {
				return err
			}
			ui.PrintAliases(rt.Config)
			return nil
		},
	})
	root.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List config entries",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(*opts)
			if err != nil {
				return err
			}
			code := runList(rt.Config, args)
			if code != 0 {
				return exitCodeError{code: code}
			}
			return nil
		},
	})
	root.AddCommand(&cobra.Command{
		Use:   "add",
		Short: "Add config entries",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(*opts)
			if err != nil {
				return err
			}
			code := runAdd(rt.BaseDir, *opts, args)
			if code != 0 {
				return exitCodeError{code: code}
			}
			return nil
		},
	})
	root.AddCommand(newPackCommand(opts))
	root.AddCommand(newPluginCommand(opts))
	root.AddCommand(newToolsCommand(opts))
	root.AddCommand(&cobra.Command{
		Use:   "find <query...>",
		Short: "Search knowledge markdown",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFindCommand(*opts, args)
		},
	})
	root.AddCommand(&cobra.Command{
		Use:     "search <query...>",
		Aliases: []string{"f"},
		Short:   "Search knowledge markdown",
		Hidden:  true,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFindCommand(*opts, args)
		},
	})
	root.AddCommand(&cobra.Command{
		Use:   "run <alias>",
		Short: "Run alias from config",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(*opts)
			if err != nil {
				return err
			}
			runner.RunAlias(rt.Config, args[0], "")
			return nil
		},
	})
	root.AddCommand(&cobra.Command{
		Use:   "validate",
		Short: "Validate configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(*opts)
			if err != nil {
				return err
			}
			code := runValidate(rt.Config)
			if code != 0 {
				return exitCodeError{code: code}
			}
			return nil
		},
	})
}

func newPackCommand(opts *flags) *cobra.Command {
	packCmd := &cobra.Command{
		Use:   "pack",
		Short: "Manage packs",
		Long: "Create, inspect, activate and manage packs.\n\n" +
			"A pack groups:\n" +
			"- shortcuts (jump)\n" +
			"- aliases (run)\n" +
			"- projects/actions\n" +
			"- knowledge path used by search",
		Example: "dm pack list\n" +
			"dm pack use git\n" +
			"dm pack vim --help",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	runPackArgs := func(args ...string) error {
		rt, err := loadRuntime(*opts)
		if err != nil {
			return err
		}
		code := runPack(rt.BaseDir, args)
		if code != 0 {
			return exitCodeError{code: code}
		}
		return nil
	}

	packCmd.AddCommand(&cobra.Command{
		Use:   "new <name>",
		Short: "Create a new pack",
		Long: "Create packs/<name>/ with:\n" +
			"- pack.json\n" +
			"- knowledge/",
		Example: "dm pack new work",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPackArgs("new", args[0])
		},
	})
	packCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List available packs",
		Long:  "Show all packs that contain packs/<name>/pack.json.",
		Example: "dm pack list\n" +
			"dm -k list",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPackArgs("list")
		},
	})
	packCmd.AddCommand(&cobra.Command{
		Use:   "info <name>",
		Short: "Show pack info",
		Long: "Show pack metadata and counts:\n" +
			"- path\n" +
			"- knowledge path\n" +
			"- jumps/runs/projects/actions",
		Example: "dm pack info git",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPackArgs("info", args[0])
		},
	})
	packCmd.AddCommand(&cobra.Command{
		Use:   "use <name>",
		Short: "Set active pack",
		Long: "Set the default pack used when --pack/-p is not passed.\n" +
			"Stored in .dm.active-pack next to the executable.",
		Example: "dm pack use git\n" +
			"dm -k use vim",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPackArgs("use", args[0])
		},
	})
	packCmd.AddCommand(&cobra.Command{
		Use:   "current",
		Short: "Show active pack",
		Long:  "Print the currently active pack, if set.",
		Example: "dm pack current",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPackArgs("current")
		},
	})
	packCmd.AddCommand(&cobra.Command{
		Use:   "unset",
		Short: "Unset active pack",
		Long:  "Remove the currently active pack selection.",
		Example: "dm pack unset",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPackArgs("unset")
		},
	})
	addDynamicPackProfileCommands(packCmd, runPackArgs)

	return packCmd
}

func addDynamicPackProfileCommands(packCmd *cobra.Command, runPackArgs func(args ...string) error) {
	baseDir, err := exeDir()
	if err != nil {
		return
	}
	packs, err := store.ListPacks(baseDir)
	if err != nil {
		return
	}
	reserved := map[string]struct{}{
		"new": {}, "list": {}, "info": {}, "use": {}, "current": {}, "unset": {}, "help": {},
	}
	for _, name := range packs {
		if _, blocked := reserved[name]; blocked {
			continue
		}
		n := name
		info, err := store.GetPackInfo(baseDir, n)
		if err != nil {
			continue
		}
		longText := fmt.Sprintf(
			"Pack: %s\nPath: %s\nKnowledge: %s\nJumps: %d\nRuns: %d\nProjects: %d\nActions: %d",
			info.Name,
			info.Path,
			info.Knowledge,
			info.Jumps,
			info.Runs,
			info.Projects,
			info.Actions,
		)
		packCmd.AddCommand(&cobra.Command{
			Use:   n,
			Short: fmt.Sprintf("Pack profile %s", n),
			Long:  longText,
			Example: fmt.Sprintf(
				"dm pack %s --help\ndm pack use %s\ndm pack info %s\ndm -p %s find <query>",
				n, n, n, n,
			),
			Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				return cmd.Help()
			},
		})
	}
}

func newPluginCommand(opts *flags) *cobra.Command {
	pluginCmd := &cobra.Command{
		Use:   "plugin",
		Short: "Manage plugins",
		Long:  "List and execute scripts from the plugins directory.",
		Example: "dm plugin list\n" +
			"dm plugin run paint",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	runPluginArgs := func(args ...string) error {
		rt, err := loadRuntime(*opts)
		if err != nil {
			return err
		}
		code := runPlugin(rt.BaseDir, args)
		if code != 0 {
			return exitCodeError{code: code}
		}
		return nil
	}

	pluginCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List available plugins",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPluginArgs("list")
		},
	})
	pluginCmd.AddCommand(&cobra.Command{
		Use:   "run <name> [args...]",
		Short: "Run a plugin",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := append([]string{"run"}, args...)
			return runPluginArgs(out...)
		},
	})

	return pluginCmd
}

func newToolsCommand(opts *flags) *cobra.Command {
	toolsCmd := &cobra.Command{
		Use:     "tools [tool]",
		Aliases: []string{"tool"},
		Short:   "Run tools menu or a specific tool",
		Long:    "Interactive tools for search, rename, quick notes, recent files, backup, and cleanup.",
		Example: "dm tools\ndm tools search\ndm -t s",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(*opts)
			if err != nil {
				return err
			}
			var code int
			if len(args) == 0 {
				code = tools.RunMenu(rt.BaseDir)
			} else {
				code = tools.RunByName(rt.BaseDir, args[0])
			}
			if code != 0 {
				return exitCodeError{code: code}
			}
			return nil
		},
	}

	addToolSubcommand := func(use, short, long, example, canonical string, aliases ...string) {
		toolsCmd.AddCommand(&cobra.Command{
			Use:     use,
			Aliases: aliases,
			Short:   short,
			Long:    long,
			Example: example,
			Args:    cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				rt, err := loadRuntime(*opts)
				if err != nil {
					return err
				}
				code := tools.RunByName(rt.BaseDir, canonical)
				if code != 0 {
					return exitCodeError{code: code}
				}
				return nil
			},
		})
	}

	addToolSubcommand(
		"search",
		"Search files by name/extension",
		"Asks for base path, optional name fragment, extension and sort mode (name/date/size).",
		"dm tools search\ndm -t s",
		"search",
		"s",
	)
	addToolSubcommand(
		"rename",
		"Batch rename files with preview",
		"Asks for base path, filter and replace rules, then shows a preview before applying changes.",
		"dm tools rename",
		"rename",
		"r",
	)
	addToolSubcommand(
		"note",
		"Append a quick note to pack inbox",
		"Asks for pack name and note text, then appends to packs/<pack>/knowledge/inbox.md.",
		"dm tools note",
		"note",
		"n",
	)
	addToolSubcommand(
		"recent",
		"Show recent files",
		"Asks for base path and limit, then lists most recently modified files.",
		"dm tools recent",
		"recent",
		"rec",
	)
	addToolSubcommand(
		"backup",
		"Create a pack zip backup",
		"Asks for pack and output directory, then creates a timestamped zip in backups.",
		"dm tools backup",
		"backup",
		"b",
	)
	addToolSubcommand(
		"clean",
		"Delete empty folders",
		"Asks for base path, previews empty folders, and asks for confirmation before deletion.",
		"dm tools clean",
		"clean",
		"c",
	)

	return toolsCmd
}

func runFindCommand(opts flags, args []string) error {
	if len(args) < 1 {
		fmt.Println("Uso: dm find <query>")
		return nil
	}
	rt, err := loadRuntime(opts)
	if err != nil {
		return err
	}
	knowledgeDir := config.ResolvePath(rt.BaseDir, rt.Config.Search.Knowledge)
	search.InKnowledge(knowledgeDir, strings.Join(args, " "))
	return nil
}
