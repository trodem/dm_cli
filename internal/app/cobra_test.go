package app

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRewriteGroupShortcutsToolsBare(t *testing.T) {
	got := rewriteGroupShortcuts([]string{"-t"})
	want := []string{"tools"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestRewriteGroupShortcutsToolsWithTarget(t *testing.T) {
	got := rewriteGroupShortcuts([]string{"-t", "s"})
	want := []string{"tools", "s"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestRewriteGroupShortcutsMixedArgs(t *testing.T) {
	got := rewriteGroupShortcuts([]string{"--verbose", "-t", "search"})
	want := []string{"--verbose", "tools", "search"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestRewriteGroupShortcutsPlugin(t *testing.T) {
	got := rewriteGroupShortcuts([]string{"-p", "run"})
	want := []string{"plugins", "run"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestRewriteGroupShortcutsOpen(t *testing.T) {
	got := rewriteGroupShortcuts([]string{"-o", "profile"})
	want := []string{"open", "profile"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestRewriteGroupShortcutsRunAlias(t *testing.T) {
	got := rewriteGroupShortcuts([]string{"-r", "cli"})
	want := []string{"alias", "run", "cli"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestRewriteGroupShortcutsAddAlias(t *testing.T) {
	got := rewriteGroupShortcuts([]string{"-a", "cli", "Get-Location"})
	want := []string{"alias", "add", "cli", "Get-Location"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestRewriteGroupShortcutsKeepsAskAsPowerShell(t *testing.T) {
	got := rewriteGroupShortcuts([]string{"ask", "-a", "Get-Location"})
	want := []string{"ask", "-a", "Get-Location"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestInstallCompletionPowerShell(t *testing.T) {
	home := t.TempDir()
	root := &cobra.Command{Use: "dm"}

	scriptPath, profilePath, err := installCompletion(root, home, "powershell")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(scriptPath); err != nil {
		t.Fatalf("expected script file, got error: %v", err)
	}
	if _, err := os.Stat(profilePath); err != nil {
		t.Fatalf("expected profile file, got error: %v", err)
	}

	profileData, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatal(err)
	}
	wantLine := ". '" + filepath.Join(home, "Documents", "PowerShell", "dm-completion.ps1") + "'"
	if !strings.Contains(string(profileData), wantLine) {
		t.Fatalf("expected profile to contain %q", wantLine)
	}

	_, _, err = installCompletion(root, home, "powershell")
	if err != nil {
		t.Fatal(err)
	}
	profileData2, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(profileData2), wantLine) != 1 {
		t.Fatalf("expected profile line once, got %d", strings.Count(string(profileData2), wantLine))
	}
}

func TestInstallCompletionBash(t *testing.T) {
	home := t.TempDir()
	root := &cobra.Command{Use: "dm"}

	scriptPath, profilePath, err := installCompletion(root, home, "bash")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(scriptPath); err != nil {
		t.Fatalf("expected script file, got error: %v", err)
	}
	if profilePath != filepath.Join(home, ".bashrc") {
		t.Fatalf("unexpected profile path: %s", profilePath)
	}
	profileData, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatal(err)
	}
	wantLine := "source '" + filepath.Join(home, ".dm-completion.bash") + "'"
	if !strings.Contains(string(profileData), wantLine) {
		t.Fatalf("expected profile to contain %q", wantLine)
	}
}

func TestInstallCompletionFish(t *testing.T) {
	home := t.TempDir()
	root := &cobra.Command{Use: "dm"}

	scriptPath, profilePath, err := installCompletion(root, home, "fish")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(scriptPath); err != nil {
		t.Fatalf("expected script file, got error: %v", err)
	}
	if profilePath != "" {
		t.Fatalf("expected empty profile path for fish, got %q", profilePath)
	}
}

func TestInstallCompletionUnsupportedShell(t *testing.T) {
	home := t.TempDir()
	root := &cobra.Command{Use: "dm"}

	_, _, err := installCompletion(root, home, "tcsh")
	if err == nil {
		t.Fatal("expected unsupported shell error")
	}
}

func TestAddCobraSubcommandsIncludesDoctor(t *testing.T) {
	root := &cobra.Command{Use: "dm"}
	addCobraSubcommands(root)

	cmd, _, err := root.Find([]string{"doctor"})
	if err != nil {
		t.Fatalf("expected doctor command, got error: %v", err)
	}
	if cmd == nil || cmd.Name() != "doctor" {
		t.Fatalf("expected doctor command, got %#v", cmd)
	}
}

func TestAddCobraSubcommandsIncludesAlias(t *testing.T) {
	root := &cobra.Command{Use: "dm"}
	addCobraSubcommands(root)

	cmd, _, err := root.Find([]string{"alias"})
	if err != nil {
		t.Fatalf("expected alias command, got error: %v", err)
	}
	if cmd == nil || cmd.Name() != "alias" {
		t.Fatalf("expected alias command, got %#v", cmd)
	}
}

func TestAliasCommandIncludesSync(t *testing.T) {
	root := &cobra.Command{Use: "dm"}
	addCobraSubcommands(root)

	cmd, _, err := root.Find([]string{"alias", "sync"})
	if err != nil {
		t.Fatalf("expected alias sync command, got error: %v", err)
	}
	if cmd == nil || cmd.Name() != "sync" {
		t.Fatalf("expected sync command, got %#v", cmd)
	}
}

func TestAskCommandIncludesAsPowerShellFlag(t *testing.T) {
	root := &cobra.Command{Use: "dm"}
	addCobraSubcommands(root)

	cmd, _, err := root.Find([]string{"ask"})
	if err != nil {
		t.Fatalf("expected ask command, got error: %v", err)
	}
	if cmd == nil || cmd.Name() != "ask" {
		t.Fatalf("expected ask command, got %#v", cmd)
	}

	f := cmd.Flags().Lookup("as-powershell")
	if f == nil {
		t.Fatal("expected --as-powershell flag on ask")
	}
	if f.Shorthand != "a" {
		t.Fatalf("expected shorthand -a, got %q", f.Shorthand)
	}
}
