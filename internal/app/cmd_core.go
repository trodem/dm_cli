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

func addCobraSubcommands(root *cobra.Command, opts *flags) {
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
	cpCmd := &cobra.Command{
		Use:   "cp",
		Short: "Copy helper targets",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cpCmd.AddCommand(&cobra.Command{
		Use:   "profile",
		Short: "Overwrite PowerShell $PROFILE from plugins/functions/0_powershell_profile.ps1",
		Args:  cobra.NoArgs,
		ValidArgs: []string{
			"profile",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(*opts)
			if err != nil {
				return err
			}
			if err := copyPowerShellProfileFromPlugin(rt.BaseDir); err != nil {
				return err
			}
			fmt.Println("OK: profile overwritten from plugins/functions/0_powershell_profile.ps1")
			return nil
		},
	})
	root.AddCommand(cpCmd)
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
	openCmd.AddCommand(&cobra.Command{
		Use:     "profile",
		Aliases: []string{"profile-source", "profile-src"},
		Short:   "Open plugins/functions/0_powershell_profile.ps1 in Notepad",
		Args:    cobra.NoArgs,
		ValidArgs: []string{
			"profile",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(*opts)
			if err != nil {
				return err
			}
			return openPluginPowerShellProfileInNotepad(rt.BaseDir)
		},
	})
	root.AddCommand(openCmd)
	root.AddCommand(newPluginCommand(opts))
	root.AddCommand(newToolsCommand(opts))
	root.AddCommand(newToolkitCommand(opts))
	var doctorJSON bool
	doctorCmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run diagnostics for agent, providers, plugins, and tool paths",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(*opts)
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
	askCmd := &cobra.Command{
		Use:   "ask <prompt...>",
		Short: "Ask AI (openai|ollama|auto)",
		Long: "Uses provider selected by --provider (default: openai). " +
			"With --provider auto, dm tries Ollama first and falls back to OpenAI.",
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
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
			rt, err := loadRuntime(*opts)
			if err != nil {
				return err
			}
			if len(args) == 0 {
				code := runAskInteractiveWithRisk(rt.BaseDir, askOpts, confirmTools, riskPolicy)
				if code != 0 {
					return exitCodeError{code: code}
				}
				return nil
			}
			code := runAskOnceWithSession(rt.BaseDir, strings.Join(args, " "), askOpts, confirmTools, riskPolicy, nil)
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
	askCmd.Flags().StringVar(&askRiskPolicy, "risk-policy", riskPolicyNormal, "risk policy: strict|normal|off")
	root.AddCommand(askCmd)
}

func newPluginCommand(opts *flags) *cobra.Command {
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
		Use:   "$profile",
		Short: "Show functions and aliases from plugins/functions/0_powershell_profile.ps1",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPluginArgs("$profile")
		},
	})
	pluginCmd.AddCommand(&cobra.Command{
		Use:               "info <name>",
		Short:             "Show plugin/function details",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completePluginEntryNames(opts),
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
		ValidArgsFunction: completePluginEntryNames(opts),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := append([]string{"run"}, args...)
			return runPluginArgs(out...)
		},
	})

	return pluginCmd
}

func completePluginEntryNames(opts *flags) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		rt, err := loadRuntime(*opts)
		if err != nil {
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

func newToolsCommand(opts *flags) *cobra.Command {
	toolsCmd := &cobra.Command{
		Use:     "tools [tool]",
		Aliases: []string{"tool"},
		Short:   "Run tools menu or a specific tool",
		Long:    "Interactive tools for search, rename, quick notes, recent files, backup, cleanup, and system snapshot.",
		Example: "dm tools\ndm tools search\ndm tools system\ndm -t s",
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
		"Append a quick note",
		"Asks for note file path and text, then appends a timestamped line.",
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
		"Create a folder zip backup",
		"Asks for source directory and output directory, then creates a timestamped zip.",
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
