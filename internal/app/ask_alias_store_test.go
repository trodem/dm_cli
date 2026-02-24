package app

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeAskAliasName(t *testing.T) {
	got, err := normalizeAskAliasName(" Downloads_1 ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "downloads_1" {
		t.Fatalf("expected downloads_1, got %q", got)
	}
	if _, err := normalizeAskAliasName("bad name"); err == nil {
		t.Fatal("expected invalid alias name error")
	}
	if _, err := normalizeAskAliasName("help"); err == nil {
		t.Fatal("expected reserved alias name error")
	}
}

func TestLoadSaveAskAliases(t *testing.T) {
	baseDir := t.TempDir()
	profilePath := filepath.Join(baseDir, "Microsoft.PowerShell_profile.ps1")
	prevResolver := askAliasProfilePathResolver
	askAliasProfilePathResolver = func() string { return profilePath }
	defer func() { askAliasProfilePathResolver = prevResolver }()

	aliases := map[string]string{
		"ll": "Get-ChildItem -Force",
		"d":  "cd C:\\Users\\Demtro\\Downloads",
	}
	if err := saveAskAliases(baseDir, aliases); err != nil {
		t.Fatal(err)
	}
	got, err := loadAskAliases(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, aliases) {
		t.Fatalf("unexpected aliases: got=%v want=%v", got, aliases)
	}
	profileRaw, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatal(err)
	}
	profileText := string(profileRaw)
	if !strings.Contains(profileText, dmAliasProfileBegin) || !strings.Contains(profileText, dmAliasProfileEnd) {
		t.Fatalf("expected dm alias markers in profile, got: %q", profileText)
	}
	if !strings.Contains(profileText, "'ll' = 'Get-ChildItem -Force'") {
		t.Fatalf("expected alias payload in profile, got: %q", profileText)
	}
}

func TestLoadAskAliasesMissingFile(t *testing.T) {
	baseDir := t.TempDir()
	got, err := loadAskAliases(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty aliases, got %v", got)
	}
}

func TestSortedAliasNames(t *testing.T) {
	keys := sortedAliasNames(map[string]string{
		"z": "a",
		"a": "b",
	})
	want := []string{"a", "z"}
	if !reflect.DeepEqual(keys, want) {
		t.Fatalf("unexpected sorted keys: got=%v want=%v", keys, want)
	}
}

func TestAskAliasFilePath(t *testing.T) {
	baseDir := t.TempDir()
	got := askAliasFilePath(baseDir)
	want := filepath.Join(baseDir, "dm.aliases.json")
	if got != want {
		t.Fatalf("unexpected path: got=%q want=%q", got, want)
	}
}

func TestLoadAskAliasesSkipsInvalidEntries(t *testing.T) {
	baseDir := t.TempDir()
	path := askAliasFilePath(baseDir)
	content := `{
  "ok_alias": "Get-Location",
  "bad alias": "Get-Date",
  "empty": ""
}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := loadAskAliases(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"ok_alias": "Get-Location"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected filtered aliases: got=%v want=%v", got, want)
	}
}

func TestUpsertAskAliasesProfileBlock(t *testing.T) {
	old := "Write-Host 'hello'\n\n" + dmAliasProfileBegin + "\nold\n" + dmAliasProfileEnd + "\n"
	block := renderAskAliasesProfileBlock(map[string]string{"cli": "Get-Location"})
	out := upsertAskAliasesProfileBlock(old, block)
	if strings.Count(out, dmAliasProfileBegin) != 1 {
		t.Fatalf("expected one begin marker, got %q", out)
	}
	if !strings.Contains(out, "'cli' = 'Get-Location'") {
		t.Fatalf("expected updated alias in profile block, got %q", out)
	}
}
