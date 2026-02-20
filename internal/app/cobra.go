package app

import (
	"errors"
	"fmt"
	"strings"

	"cli/internal/ui"

	"github.com/spf13/cobra"
)

type exitCodeError struct {
	code int
}

func (e exitCodeError) Error() string {
	return fmt.Sprintf("exit code %d", e.code)
}

func Run(args []string) int {
	root := &cobra.Command{
		Use:   "dm",
		Short: "Personal CLI for tools, plugins, and AI helpers",
		Long:  "dm is a personal CLI for tools, plugins, and AI-driven workflows.",
		Example: "dm help\n" +
			"dm help plugins\n" +
			"dm help <function_name>",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, positional []string) error {
			rt, err := loadRuntime()
			if err != nil {
				return err
			}
			exeBuiltAt, _ := executableBuildTime()
			ui.PrintSplash(ui.SplashData{
				BaseDir:    rt.BaseDir,
				Version:    Version,
				ExeBuiltAt: exeBuiltAt,
			})
			return nil
		},
	}

	root.PersistentFlags().BoolP("tools", "t", false, "shortcut for 'tools' command")
	root.PersistentFlags().BoolP("plugins", "p", false, "shortcut for 'plugins' command")
	root.PersistentFlags().BoolP("open", "o", false, "shortcut for 'open' command")
	root.CompletionOptions.DisableDefaultCmd = true

	addCobraSubcommands(root)
	addPluginAwareHelpCommand(root)
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
			rest := parseFlags(rewriteGroupShortcuts(args))
			if len(rest) >= 2 && rest[0] == "help" {
				rt, loadErr := loadRuntime()
				if loadErr != nil {
					fmt.Println("Error:", loadErr)
					return 1
				}
				return runPlugin(rt.BaseDir, []string{"info", rest[1]})
			}
		}
		if strings.HasPrefix(msg, "unknown command") {
			rt, loadErr := loadRuntime()
			if loadErr != nil {
				fmt.Println("Error:", loadErr)
				return 1
			}
			rest := parseFlags(rewriteGroupShortcuts(args))
			if len(rest) > 0 && rest[0] == "$profile" {
				return showPowerShellSymbols(resolveUserPowerShellProfilePath(), "$PROFILE")
			}
			return runPluginOrSuggest(rt.BaseDir, rest)
		}
		if msg != "" {
			fmt.Println("Error:", msg)
		}
		return 1
	}
	return 0
}

func addPluginAwareHelpCommand(root *cobra.Command) {
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
			rt, loadErr := loadRuntime()
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
