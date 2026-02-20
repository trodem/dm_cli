package app

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cli/internal/toolkitgen"
	"cli/internal/ui"

	"github.com/spf13/cobra"
)

func newToolkitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "toolkit",
		Short: "Create and maintain plugin toolkits with a guided UX",
		Long:  "Built-in toolkit generator for creating and evolving plugin toolkits.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime()
			if err != nil {
				return err
			}
			return runToolkitWizard(rt.BaseDir)
		},
	}

	var (
		newName        string
		newPrefix      string
		newCategory    string
		newDescription string
		newForce       bool
	)
	newCmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new toolkit scaffold",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime()
			if err != nil {
				return err
			}
			return runToolkitNew(rt.BaseDir, newName, newPrefix, newCategory, newDescription, newForce)
		},
	}
	newCmd.Flags().StringVar(&newName, "name", "", "toolkit name (e.g. MSWord)")
	newCmd.Flags().StringVar(&newPrefix, "prefix", "", "function prefix (e.g. word)")
	newCmd.Flags().StringVar(&newCategory, "category", "", "subfolder under plugins/functions")
	newCmd.Flags().StringVar(&newDescription, "description", "", "toolkit description")
	newCmd.Flags().BoolVar(&newForce, "force", false, "overwrite target file if it already exists")
	cmd.AddCommand(newCmd)

	var (
		addFile         string
		addPrefix       string
		addFunc         string
		addSynopsis     string
		addDescription  string
		addConfirm      bool
		addParams       []string
		addSwitches     []string
		addRequireVars  []string
		addRequireHelps []string
	)
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a function to an existing toolkit",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime()
			if err != nil {
				return err
			}
			return runToolkitAdd(rt.BaseDir, toolkitAddInput{
				File:           addFile,
				Prefix:         addPrefix,
				FuncName:       addFunc,
				Synopsis:       addSynopsis,
				Description:    addDescription,
				Confirm:        addConfirm,
				Params:         addParams,
				Switches:       addSwitches,
				RequireVars:    addRequireVars,
				RequireHelpers: addRequireHelps,
			})
		},
	}
	addCmd.Flags().StringVar(&addFile, "file", "", "target toolkit .ps1 file")
	addCmd.Flags().StringVar(&addPrefix, "prefix", "", "function prefix")
	addCmd.Flags().StringVar(&addFunc, "func", "", "function suffix")
	addCmd.Flags().StringVar(&addSynopsis, "synopsis", "", "help synopsis")
	addCmd.Flags().StringVar(&addDescription, "description", "", "help description")
	addCmd.Flags().BoolVar(&addConfirm, "confirm", false, "add confirmation guard with -Force")
	addCmd.Flags().StringArrayVar(&addParams, "param", nil, "mandatory string parameter (repeatable)")
	addCmd.Flags().StringArrayVar(&addSwitches, "switch", nil, "switch parameter (repeatable)")
	addCmd.Flags().StringArrayVar(&addRequireVars, "require-var", nil, "ensure variable in plugins/variables.ps1, format NAME=default")
	addCmd.Flags().StringArrayVar(&addRequireHelps, "require-helper", nil, "ensure helper in plugins/utils.ps1")
	cmd.AddCommand(addCmd)

	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate toolkit files and functions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime()
			if err != nil {
				return err
			}
			return runToolkitValidate(rt.BaseDir)
		},
	}
	cmd.AddCommand(validateCmd)

	return cmd
}

type toolkitAddInput struct {
	File           string
	Prefix         string
	FuncName       string
	Synopsis       string
	Description    string
	Confirm        bool
	Params         []string
	Switches       []string
	RequireVars    []string
	RequireHelpers []string
}

func runToolkitWizard(baseDir string) error {
	reader := bufio.NewReader(os.Stdin)
	for {
		ui.PrintSection("Toolkit Generator")
		fmt.Println(" 1) " + ui.Accent("Create new toolkit"))
		fmt.Println(" 2) " + ui.Accent("Add function to toolkit"))
		fmt.Println(" 3) " + ui.Accent("Validate toolkits"))
		fmt.Println(" 0) " + ui.Error("Exit"))
		fmt.Print(ui.Prompt("Select option > "))
		choice := strings.TrimSpace(readLine(reader))

		switch strings.ToLower(choice) {
		case "0", "x", "exit", "":
			return nil
		case "1", "new":
			name := promptValue(reader, "Toolkit name", "")
			prefix := promptValue(reader, "Prefix", strings.ToLower(strings.TrimSpace(name)))
			category := promptValue(reader, "Category (optional)", "")
			description := promptValue(reader, "Description (optional)", "")
			force := promptYesNo(reader, "Overwrite existing file if needed", false)
			if err := runToolkitNew(baseDir, name, prefix, category, description, force); err != nil {
				fmt.Println(ui.Error("Error:"), err)
			}
			waitEnter(reader)
		case "2", "add":
			file := promptValue(reader, "Toolkit file (e.g. plugins/functions/office/MSWord_Toolkit.ps1)", "")
			prefix := promptValue(reader, "Prefix", "")
			fn := promptValue(reader, "Function suffix (e.g. export_pdf)", "")
			syn := promptValue(reader, "Synopsis (optional)", "")
			desc := promptValue(reader, "Description (optional)", "")
			params := splitCSV(promptValue(reader, "Params CSV (optional)", ""))
			switches := splitCSV(promptValue(reader, "Switches CSV (optional)", ""))
			confirm := promptYesNo(reader, "Add confirmation guard (-Force + prompt)", false)
			reqVars := splitCSV(promptValue(reader, "Require vars CSV NAME=default (optional)", ""))
			reqHelpers := splitCSV(promptValue(reader, "Require helpers CSV (optional)", ""))

			err := runToolkitAdd(baseDir, toolkitAddInput{
				File:           file,
				Prefix:         prefix,
				FuncName:       fn,
				Synopsis:       syn,
				Description:    desc,
				Confirm:        confirm,
				Params:         params,
				Switches:       switches,
				RequireVars:    reqVars,
				RequireHelpers: reqHelpers,
			})
			if err != nil {
				fmt.Println(ui.Error("Error:"), err)
			}
			waitEnter(reader)
		case "3", "validate":
			if err := runToolkitValidate(baseDir); err != nil {
				fmt.Println(ui.Error("Error:"), err)
			}
			waitEnter(reader)
		default:
			fmt.Println(ui.Error("Invalid selection."))
		}
	}
}

func runToolkitNew(baseDir, name, prefix, category, description string, force bool) error {
	path, err := toolkitgen.Init(toolkitgen.InitOptions{
		Repo:        baseDir,
		Name:        name,
		Prefix:      prefix,
		Category:    category,
		Description: description,
		Force:       force,
	})
	if err != nil {
		return err
	}
	rel, relErr := filepath.Rel(baseDir, path)
	if relErr != nil {
		rel = path
	}
	fmt.Println(ui.OK("Created toolkit:"), rel)
	return nil
}

func runToolkitAdd(baseDir string, in toolkitAddInput) error {
	path, err := toolkitgen.Add(toolkitgen.AddOptions{
		Repo:           baseDir,
		File:           in.File,
		Prefix:         in.Prefix,
		FuncName:       in.FuncName,
		Synopsis:       in.Synopsis,
		Description:    in.Description,
		Confirm:        in.Confirm,
		Params:         in.Params,
		Switches:       in.Switches,
		RequireVars:    in.RequireVars,
		RequireHelpers: in.RequireHelpers,
	})
	if err != nil {
		return err
	}
	rel, relErr := filepath.Rel(baseDir, path)
	if relErr != nil {
		rel = path
	}
	fmt.Println(ui.OK("Updated toolkit:"), rel)
	return nil
}

func runToolkitValidate(baseDir string) error {
	result, err := toolkitgen.Validate(baseDir)
	if err != nil {
		return err
	}
	if len(result.Issues) == 0 {
		fmt.Printf("%s %d file(s), %d function(s), no validation issues\n", ui.OK("OK:"), result.Files, result.Funcs)
		return nil
	}
	for _, it := range result.Issues {
		rel, relErr := filepath.Rel(baseDir, it.Path)
		if relErr != nil {
			rel = it.Path
		}
		fmt.Printf("%s:%d: %s\n", rel, it.Line, it.Msg)
	}
	return fmt.Errorf("validation failed with %d issue(s)", len(result.Issues))
}

func promptValue(r *bufio.Reader, label, def string) string {
	if strings.TrimSpace(def) != "" {
		fmt.Printf("%s ", ui.Prompt(fmt.Sprintf("%s [%s]:", label, def)))
	} else {
		fmt.Printf("%s ", ui.Prompt(label+":"))
	}
	v := strings.TrimSpace(readLine(r))
	if v == "" {
		return def
	}
	return v
}

func promptYesNo(r *bufio.Reader, label string, def bool) bool {
	defLabel := "N"
	if def {
		defLabel = "Y"
	}
	fmt.Printf("%s ", ui.Prompt(fmt.Sprintf("%s [y/N] (default %s):", label, defLabel)))
	raw := strings.ToLower(strings.TrimSpace(readLine(r)))
	if raw == "" {
		return def
	}
	return raw == "y" || raw == "yes"
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func waitEnter(r *bufio.Reader) {
	fmt.Print(ui.Prompt("Press Enter to continue..."))
	_, _ = r.ReadString('\n')
}
