package app

import (
	"fmt"
	"strings"

	"cli/internal/agent"
	"cli/internal/doctor"
	"cli/internal/plugins"
	"cli/tools"

	"github.com/spf13/cobra"
)

func addCobraSubcommands(root *cobra.Command) {
	root.AddCommand(&cobra.Command{
		Use:   "ps_profile",
		Short: "Show functions and aliases from PowerShell $PROFILE",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			code := showPowerShellSymbols(resolveUserPowerShellProfilePath(), "$PROFILE")
			if code != 0 {
				return exitCodeError{code: code}
			}
			return nil
		},
	})
	openCmd := &cobra.Command{
		Use:   "open",
		Short: "Open profile files in Notepad",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	openCmd.AddCommand(&cobra.Command{
		Use:   "ps_profile",
		Short: "Open PowerShell $PROFILE in Notepad",
		Args:  cobra.NoArgs,
		ValidArgs: []string{
			"ps_profile",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return openUserPowerShellProfileInNotepad()
		},
	})
	root.AddCommand(openCmd)
	root.AddCommand(newPluginCommand())
	root.AddCommand(newToolsCommand())
	root.AddCommand(newAliasCommand())
	var doctorJSON bool
	doctorCmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run diagnostics for agent, providers, plugins, and tool paths",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime()
			if err != nil {
				return err
			}
			report := doctor.Run(rt.BaseDir)
			if doctorJSON {
				if err := doctor.RenderJSON(report); err != nil {
					return err
				}
			} else {
				doctor.RenderText(report)
			}
			if report.ErrorCount > 0 {
				return exitCodeError{code: 1}
			}
			return nil
		},
	}
	doctorCmd.Flags().BoolVar(&doctorJSON, "json", false, "render diagnostics as JSON")
	root.AddCommand(doctorCmd)
	var askProvider string
	var askModel string
	var askBaseURL string
	var askConfirmTools bool
	var askNoConfirmTools bool
	var askRiskPolicy string
	var askResponseMode string
	var askJSON bool
	var askFiles []string
	var askScope string
	var askAsPowerShell bool
	askCmd := &cobra.Command{
		Use:   "ask <prompt...>",
		Short: "Ask AI (openai|ollama|auto)",
		Long: "Uses provider selected by --provider (default: openai). " +
			"With --provider auto, dm tries Ollama first and falls back to OpenAI.",
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if askAsPowerShell {
				if len(args) == 0 {
					return fmt.Errorf("--as-powershell (-a) requires a command")
				}
				code := runAskPowerShellBuiltin(strings.Join(args, " "))
				if code != 0 {
					return exitCodeError{code: code}
				}
				return nil
			}

			askOpts := agent.AskOptions{
				Provider: askProvider,
				Model:    askModel,
				BaseURL:  askBaseURL,
			}
			confirmTools := askConfirmTools
			if askNoConfirmTools {
				confirmTools = false
			}
			riskPolicy, riskErr := normalizeRiskPolicy(askRiskPolicy)
			if riskErr != nil {
				return riskErr
			}
			responseMode, modeErr := normalizeResponseMode(askResponseMode)
			if modeErr != nil {
				return modeErr
			}
			rt, err := loadRuntime()
			if err != nil {
				return err
			}
			var fileCtx string
			if len(askFiles) > 0 {
				fc, fcErr := buildFileContext(askFiles)
				if fcErr != nil {
					return fcErr
				}
				fileCtx = fc
			}
			if askJSON {
				if len(args) == 0 {
					return fmt.Errorf("--json requires a prompt (non-interactive mode)")
				}
				code, _ := runAskOnceWithSession(askSessionParams{
					baseDir: rt.BaseDir, prompt: strings.Join(args, " "), opts: askOpts,
					confirmTools: confirmTools, riskPolicy: riskPolicy, responseMode: responseMode, jsonOut: true,
					fileContext: fileCtx, scope: askScope,
				})
				if code != 0 {
					return exitCodeError{code: code}
				}
				return nil
			}
			var initialPrompt string
			if len(args) > 0 {
				initialPrompt = strings.Join(args, " ")
			}
			code := runAskInteractiveWithRisk(rt.BaseDir, askOpts, confirmTools, riskPolicy, responseMode, initialPrompt, fileCtx, askScope)
			if code != 0 {
				return exitCodeError{code: code}
			}
			return nil
		},
	}
	askCmd.Flags().StringVar(&askProvider, "provider", "openai", "provider: openai|auto|ollama")
	askCmd.Flags().StringVar(&askModel, "model", "", "override model for selected provider")
	askCmd.Flags().StringVar(&askBaseURL, "base-url", "", "override base URL for selected provider")
	askCmd.Flags().BoolVar(&askConfirmTools, "confirm-tools", true, "ask confirmation before agent runs a plugin/function/tool")
	askCmd.Flags().BoolVar(&askNoConfirmTools, "no-confirm-tools", false, "disable confirmation before agent actions")
	askCmd.MarkFlagsMutuallyExclusive("confirm-tools", "no-confirm-tools")
	askCmd.Flags().StringVar(&askRiskPolicy, "risk-policy", riskPolicyNormal, "risk policy: strict|normal|off")
	askCmd.Flags().StringVar(&askResponseMode, "response-mode", responseModeRawFirst, "response mode: raw-first|llm-first")
	askCmd.Flags().BoolVar(&askJSON, "json", false, "print structured JSON output (non-interactive only)")
	askCmd.Flags().StringArrayVarP(&askFiles, "file", "f", nil, "attach file as context (repeatable)")
	askCmd.Flags().StringVarP(&askScope, "scope", "s", "", "limit plugin catalog to a toolkit prefix or domain (e.g. stibs, m365, docker)")
	askCmd.Flags().BoolVarP(&askAsPowerShell, "as-powershell", "a", false, "run prompt as a direct PowerShell command (bypass AI)")
	askCmd.MarkFlagsMutuallyExclusive("as-powershell", "json")
	root.AddCommand(askCmd)
}

func newAliasCommand() *cobra.Command {
	aliasCmd := &cobra.Command{
		Use:   "alias",
		Short: "Manage local ask aliases",
		Long:  "Store and run local aliases backed by dm.aliases.json.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	aliasCmd.AddCommand(&cobra.Command{
		Use:   "ls",
		Short: "List aliases",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime()
			if err != nil {
				return err
			}
			aliases, err := loadAskAliases(rt.BaseDir)
			if err != nil {
				return err
			}
			if len(aliases) == 0 {
				fmt.Println("No aliases configured.")
				return nil
			}
			for _, k := range sortedAliasNames(aliases) {
				fmt.Printf("%s -> %s\n", k, aliases[k])
			}
			return nil
		},
	})

	aliasCmd.AddCommand(&cobra.Command{
		Use:   "add <name> <command...>",
		Short: "Create or update an alias",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime()
			if err != nil {
				return err
			}
			name, err := normalizeAskAliasName(args[0])
			if err != nil {
				return err
			}
			command := strings.TrimSpace(strings.Join(args[1:], " "))
			if command == "" {
				return fmt.Errorf("alias command is required")
			}
			aliases, err := loadAskAliases(rt.BaseDir)
			if err != nil {
				return err
			}
			aliases[name] = command
			if err := saveAskAliases(rt.BaseDir, aliases); err != nil {
				return err
			}
			fmt.Printf("Saved alias: %s -> %s\n", name, command)
			return nil
		},
	})

	aliasCmd.AddCommand(&cobra.Command{
		Use:   "rm <name>",
		Short: "Remove an alias",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime()
			if err != nil {
				return err
			}
			name, err := normalizeAskAliasName(args[0])
			if err != nil {
				return err
			}
			aliases, err := loadAskAliases(rt.BaseDir)
			if err != nil {
				return err
			}
			if _, ok := aliases[name]; !ok {
				return fmt.Errorf("alias not found: %s", name)
			}
			delete(aliases, name)
			if err := saveAskAliases(rt.BaseDir, aliases); err != nil {
				return err
			}
			fmt.Printf("Removed alias: %s\n", name)
			return nil
		},
	})

	aliasCmd.AddCommand(&cobra.Command{
		Use:   "run <name> [extra args...]",
		Short: "Run alias as PowerShell command",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime()
			if err != nil {
				return err
			}
			name, err := normalizeAskAliasName(args[0])
			if err != nil {
				return err
			}
			aliases, err := loadAskAliases(rt.BaseDir)
			if err != nil {
				return err
			}
			baseCommand, ok := aliases[name]
			if !ok {
				return fmt.Errorf("alias not found: %s", name)
			}
			fullCommand := baseCommand
			if len(args) > 1 {
				fullCommand += " " + strings.Join(args[1:], " ")
			}
			code := runAskPowerShellBuiltin(fullCommand)
			if code != 0 {
				return exitCodeError{code: code}
			}
			return nil
		},
	})

	aliasCmd.AddCommand(&cobra.Command{
		Use:   "sync",
		Short: "Sync aliases to $PROFILE",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime()
			if err != nil {
				return err
			}
			aliases, err := loadAskAliases(rt.BaseDir)
			if err != nil {
				return err
			}
			if err := syncAskAliasesToProfile(aliases); err != nil {
				return err
			}
			profilePath := strings.TrimSpace(askAliasProfilePathResolver())
			if profilePath == "" {
				fmt.Println("Aliases synced. $PROFILE path is not available on this system.")
				return nil
			}
			fmt.Printf("Aliases synced to $PROFILE: %s\n", profilePath)
			return nil
		},
	})

	return aliasCmd
}

func newPluginCommand() *cobra.Command {
	runPluginArgs := func(args ...string) error {
		rt, err := loadRuntime()
		if err != nil {
			return err
		}
		code := runPlugin(rt.BaseDir, args)
		if code != 0 {
			return exitCodeError{code: code}
		}
		return nil
	}

	pluginCmd := &cobra.Command{
		Use:   "plugins",
		Short: "Manage plugins",
		Long:  "List and execute scripts/functions from the plugins directory.",
		Example: "dm plugins list\n" +
			"dm plugins list --functions\n" +
			"dm plugins info restart_backend\n" +
			"dm plugins menu\n" +
			"dm plugins run paint",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPluginArgs()
		},
	}

	var listFunctions bool
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available plugins",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if listFunctions {
				return runPluginArgs("list", "--functions")
			}
			return runPluginArgs("list")
		},
	}
	listCmd.Flags().BoolVarP(&listFunctions, "functions", "f", false, "include discovered PowerShell functions")
	pluginCmd.AddCommand(listCmd)
	pluginCmd.AddCommand(&cobra.Command{
		Use:               "info <name>",
		Short:             "Show plugin/function details",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completePluginEntryNames(),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPluginArgs("info", args[0])
		},
	})
	pluginCmd.AddCommand(&cobra.Command{
		Use:   "menu",
		Short: "Open interactive plugin menu",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPluginArgs("menu")
		},
	})
	pluginCmd.AddCommand(&cobra.Command{
		Use:               "run <name> [args...]",
		Short:             "Run a plugin",
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: completePluginEntryNames(),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := append([]string{"run"}, args...)
			return runPluginArgs(out...)
		},
	})

	return pluginCmd
}

func completePluginEntryNames() func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		rt, rtErr := loadRuntime()
		if rtErr != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		items, err := plugins.ListEntries(rt.BaseDir, true)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		prefix := strings.ToLower(strings.TrimSpace(toComplete))
		out := make([]string, 0, len(items))
		for _, it := range items {
			name := strings.TrimSpace(it.Name)
			if name == "" {
				continue
			}
			if prefix == "" || strings.HasPrefix(strings.ToLower(name), prefix) {
				out = append(out, name)
			}
		}
		return out, cobra.ShellCompDirectiveNoFileComp
	}
}

func newToolsCommand() *cobra.Command {
	toolsCmd := &cobra.Command{
		Use:     "tools [tool]",
		Aliases: []string{"tool"},
		Short:   "Run tools menu or a specific tool",
		Long:    "Interactive tools for search, rename, recent files, cleanup, and system snapshot.",
		Example: "dm tools\ndm tools search\ndm tools system\ndm -t s",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime()
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
				rt, err := loadRuntime()
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
		"recent",
		"Show recent files",
		"Asks for base path and limit, then lists most recently modified files.",
		"dm tools recent",
		"recent",
		"rec",
	)
	addToolSubcommand(
		"clean",
		"Delete empty folders",
		"Asks for base path, previews empty folders, and asks for confirmation before deletion.",
		"dm tools clean",
		"clean",
		"c",
	)
	addToolSubcommand(
		"system",
		"Show system/network snapshot",
		"Shows host, CPU, memory, disks, interfaces, Wi-Fi networks, and ARP LAN neighbors.",
		"dm tools system\ndm tools sys\ndm tools htop",
		"system",
		"sys",
		"htop",
	)

	return toolsCmd
}
