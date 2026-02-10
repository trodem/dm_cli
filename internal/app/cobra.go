package app

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type exitCodeError struct {
	code int
}

func (e exitCodeError) Error() string {
	return fmt.Sprintf("exit code %d", e.code)
}

func Run(args []string) int {
	var opts flags

	root := &cobra.Command{
		Use:   "dm",
		Short: "Personal CLI for jumps, project actions, and knowledge search",
		Long:  "dm is a personal CLI to jump to folders, run aliases/actions, and search knowledge notes.",
		Example: "dm help\n" +
			"dm help plugins\n" +
			"dm help <function_name>",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, positional []string) error {
			code := runLegacy(legacyArgsWithFlags(opts, positional))
			if code != 0 {
				return exitCodeError{code: code}
			}
			return nil
		},
	}

	root.PersistentFlags().BoolVar(&opts.NoCache, "no-cache", false, "disable config cache")
	root.PersistentFlags().StringVar(&opts.Profile, "profile", "", "use profile")
	root.PersistentFlags().BoolP("tools", "t", false, "shortcut for 'tools' command")
	root.PersistentFlags().BoolP("plugins", "p", false, "shortcut for 'plugins' command")
	root.CompletionOptions.DisableDefaultCmd = true

	addCobraSubcommands(root, &opts)
	addPluginAwareHelpCommand(root, &opts)
	addCompletionCommands(root)
	applySubcommandHelpTemplate(root)

	root.SetArgs(rewriteGroupShortcuts(args))

	if err := root.Execute(); err != nil {
		var codeErr exitCodeError
		if errors.As(err, &codeErr) {
			return codeErr.code
		}
		msg := strings.TrimSpace(err.Error())
		if strings.HasPrefix(msg, "unknown help topic") {
			_, rest := parseFlags(rewriteGroupShortcuts(args))
			if len(rest) >= 2 && rest[0] == "help" {
				rt, loadErr := loadRuntime(opts)
				if loadErr != nil {
					fmt.Println("Error:", loadErr)
					return 1
				}
				return runPlugin(rt.BaseDir, []string{"info", rest[1]})
			}
		}
		if strings.HasPrefix(msg, "unknown command") {
			rt, loadErr := loadRuntime(opts)
			if loadErr != nil {
				fmt.Println("Error:", loadErr)
				return 1
			}
			_, rest := parseFlags(rewriteGroupShortcuts(args))
			return runTargetOrSearch(rt.BaseDir, rt.Config, rest)
		}
		if msg != "" {
			fmt.Println("Error:", msg)
		}
		return 1
	}
	return 0
}

func legacyArgsWithFlags(opts flags, positional []string) []string {
	legacyArgs := make([]string, 0, len(positional)+6)
	if opts.NoCache {
		legacyArgs = append(legacyArgs, "--no-cache")
	}
	if opts.Profile != "" {
		legacyArgs = append(legacyArgs, "--profile", opts.Profile)
	}
	legacyArgs = append(legacyArgs, positional...)
	return legacyArgs
}

func addPluginAwareHelpCommand(root *cobra.Command, opts *flags) {
	helpCmd := &cobra.Command{
		Use:   "help [command]",
		Short: "Help about any command or plugin/function",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return root.Help()
			}
			target, _, err := root.Find(args)
			if err == nil && target != nil && target != root {
				return target.Help()
			}
			rt, loadErr := loadRuntime(*opts)
			if loadErr != nil {
				return loadErr
			}
			code := runPlugin(rt.BaseDir, []string{"info", args[0]})
			if code != 0 {
				return exitCodeError{code: code}
			}
			return nil
		},
	}
	root.SetHelpCommand(helpCmd)
}
