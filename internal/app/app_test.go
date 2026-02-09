package app

import (
	"reflect"
	"testing"

	"cli/internal/config"
)

func TestParseFlagsToolsShortcut(t *testing.T) {
	_, out := parseFlags([]string{"-t"})
	want := []string{"tools"}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("expected %v, got %v", want, out)
	}
}

func TestParseFlagsToolsShortcutWithTarget(t *testing.T) {
	_, out := parseFlags([]string{"-t", "search"})
	want := []string{"tools", "search"}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("expected %v, got %v", want, out)
	}
}

func TestParseFlagsToolsShortcutWithAliasAndPack(t *testing.T) {
	f, out := parseFlags([]string{"-p", "git", "-t", "s"})
	want := []string{"tools", "s"}
	if f.Pack != "git" {
		t.Fatalf("expected pack git, got %q", f.Pack)
	}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("expected %v, got %v", want, out)
	}
}

func TestParseFlagsPacksShortcut(t *testing.T) {
	_, out := parseFlags([]string{"-k", "list"})
	want := []string{"pack", "list"}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("expected %v, got %v", want, out)
	}
}

func TestParseFlagsPluginsShortcut(t *testing.T) {
	_, out := parseFlags([]string{"-g", "list"})
	want := []string{"plugin", "list"}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("expected %v, got %v", want, out)
	}
}

func TestRunTargetOrSearchUnknownReturnsError(t *testing.T) {
	baseDir := t.TempDir()
	cfg := config.Config{
		Jump:     map[string]string{},
		Run:      map[string]string{},
		Projects: map[string]config.Project{},
	}

	code := runTargetOrSearch(baseDir, cfg, []string{"not-existing-command"})
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
}

func TestParsePluginMenuChoice(t *testing.T) {
	tests := []struct {
		in    string
		count int
		want  int
		ok    bool
	}{
		{"1", 5, 0, true},
		{"3", 5, 2, true},
		{"a", 5, 0, true},
		{"c", 5, 2, true},
		{"z", 5, -1, false},
		{"0", 5, -1, false},
	}
	for _, tt := range tests {
		got, ok := parsePluginMenuChoice(tt.in, tt.count)
		if got != tt.want || ok != tt.ok {
			t.Fatalf("parsePluginMenuChoice(%q,%d) => (%d,%v), want (%d,%v)", tt.in, tt.count, got, ok, tt.want, tt.ok)
		}
	}
}

func TestSplitMenuArgs(t *testing.T) {
	got, err := splitMenuArgs(`-Message "hello world" -Confirm`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"-Message", "hello world", "-Confirm"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}
