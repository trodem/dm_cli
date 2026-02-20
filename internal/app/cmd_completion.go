package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func addCompletionCommands(root *cobra.Command) {
	completionCmd := &cobra.Command{
		Use:   "completion",
		Short: "Generate or install shell completion",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	completionCmd.AddCommand(&cobra.Command{
		Use:   "powershell",
		Short: "Generate PowerShell completion script",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenPowerShellCompletion(os.Stdout)
		},
	})
	completionCmd.AddCommand(&cobra.Command{
		Use:   "bash",
		Short: "Generate Bash completion script",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenBashCompletionV2(os.Stdout, true)
		},
	})
	completionCmd.AddCommand(&cobra.Command{
		Use:   "zsh",
		Short: "Generate Zsh completion script",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenZshCompletion(os.Stdout)
		},
	})
	completionCmd.AddCommand(&cobra.Command{
		Use:   "fish",
		Short: "Generate Fish completion script",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		},
	})

	var shellName string
	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Install completion script for a shell",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			scriptPath, profilePath, err := installCompletion(cmd.Root(), homeDir, shellName)
			if err != nil {
				return err
			}
			fmt.Println("OK: completion installed")
			fmt.Println("Shell:", shellName)
			fmt.Println("Script:", scriptPath)
			if strings.TrimSpace(profilePath) != "" {
				fmt.Println("Profile:", profilePath)
			}
			return nil
		},
	}
	installCmd.Flags().StringVar(&shellName, "shell", "powershell", "target shell: powershell|bash|zsh|fish")
	completionCmd.AddCommand(installCmd)

	root.AddCommand(completionCmd)
}

func installCompletion(root *cobra.Command, homeDir, shellName string) (string, string, error) {
	if strings.TrimSpace(homeDir) == "" {
		return "", "", fmt.Errorf("invalid home directory")
	}
	switch strings.ToLower(strings.TrimSpace(shellName)) {
	case "powershell":
		psDir := filepath.Join(homeDir, "Documents", "PowerShell")
		if err := os.MkdirAll(psDir, 0755); err != nil {
			return "", "", err
		}
		scriptPath := filepath.Join(psDir, "dm-completion.ps1")
		if err := writeCompletionScript(scriptPath, func(f *os.File) error {
			return root.GenPowerShellCompletion(f)
		}); err != nil {
			return "", "", err
		}
		profilePath := filepath.Join(psDir, "Microsoft.PowerShell_profile.ps1")
		sourceLine := fmt.Sprintf(". '%s'", scriptPath)
		if err := ensureProfileLine(profilePath, sourceLine); err != nil {
			return "", "", err
		}
		return scriptPath, profilePath, nil
	case "bash":
		scriptPath := filepath.Join(homeDir, ".dm-completion.bash")
		if err := writeCompletionScript(scriptPath, func(f *os.File) error {
			return root.GenBashCompletionV2(f, true)
		}); err != nil {
			return "", "", err
		}
		profilePath := filepath.Join(homeDir, ".bashrc")
		sourceLine := fmt.Sprintf("source '%s'", scriptPath)
		if err := ensureProfileLine(profilePath, sourceLine); err != nil {
			return "", "", err
		}
		return scriptPath, profilePath, nil
	case "zsh":
		scriptPath := filepath.Join(homeDir, ".dm-completion.zsh")
		if err := writeCompletionScript(scriptPath, func(f *os.File) error {
			return root.GenZshCompletion(f)
		}); err != nil {
			return "", "", err
		}
		profilePath := filepath.Join(homeDir, ".zshrc")
		sourceLine := fmt.Sprintf("source '%s'", scriptPath)
		if err := ensureProfileLine(profilePath, sourceLine); err != nil {
			return "", "", err
		}
		return scriptPath, profilePath, nil
	case "fish":
		fishDir := filepath.Join(homeDir, ".config", "fish", "completions")
		if err := os.MkdirAll(fishDir, 0755); err != nil {
			return "", "", err
		}
		scriptPath := filepath.Join(fishDir, "dm.fish")
		if err := writeCompletionScript(scriptPath, func(f *os.File) error {
			return root.GenFishCompletion(f, true)
		}); err != nil {
			return "", "", err
		}
		return scriptPath, "", nil
	default:
		return "", "", fmt.Errorf("unsupported shell: %s", shellName)
	}
}

func writeCompletionScript(path string, gen func(*os.File) error) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := gen(f); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func ensureProfileLine(profilePath, line string) error {
	existing, err := os.ReadFile(profilePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if strings.Contains(string(existing), line) {
		return nil
	}

	f, err := os.OpenFile(profilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if len(existing) > 0 && !strings.HasSuffix(string(existing), "\n") {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}
	_, err = f.WriteString(line + "\n")
	return err
}
