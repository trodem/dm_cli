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
		Use:           "dm",
		Short:         "Personal CLI for jumps, project actions, and knowledge search",
		Long:          "dm is a personal CLI to jump to folders, run aliases/actions, and search knowledge notes.",
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
	root.PersistentFlags().StringVarP(&opts.Pack, "pack", "p", "", "use pack")
	root.PersistentFlags().BoolP("tools", "t", false, "shortcut for 'tools' command")
	root.PersistentFlags().BoolP("packs", "k", false, "shortcut for 'pack' command")
	root.PersistentFlags().BoolP("plugins", "g", false, "shortcut for 'plugin' command")
	root.CompletionOptions.DisableDefaultCmd = true

	addCobraSubcommands(root, &opts)
	addCompletionCommands(root)
	applySubcommandHelpTemplate(root)

	root.SetArgs(rewriteGroupShortcuts(args))

	if err := root.Execute(); err != nil {
		var codeErr exitCodeError
		if errors.As(err, &codeErr) {
			return codeErr.code
		}
		msg := strings.TrimSpace(err.Error())
		if strings.HasPrefix(msg, "unknown command") {
			rt, loadErr := loadRuntime(opts)
			if loadErr != nil {
				fmt.Println("Errore:", loadErr)
				return 1
			}
			_, rest := parseFlags(rewriteGroupShortcuts(args))
			return runTargetOrSearch(rt.BaseDir, rt.Config, rest)
		}
		if msg != "" {
			fmt.Println("Errore:", msg)
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
	if opts.Pack != "" {
		legacyArgs = append(legacyArgs, "--pack", opts.Pack)
	}
	legacyArgs = append(legacyArgs, positional...)
	return legacyArgs
}
