package app

import (
	"reflect"
	"testing"
)

func TestParseFlagsToolsShortcut(t *testing.T) {
	out := parseFlags([]string{"-t"})
	want := []string{"tools"}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("expected %v, got %v", want, out)
	}
}

func TestParseFlagsToolsShortcutWithTarget(t *testing.T) {
	out := parseFlags([]string{"-t", "search"})
	want := []string{"tools", "search"}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("expected %v, got %v", want, out)
	}
}

func TestParseFlagsToolsShortcutWithUnrelatedFlags(t *testing.T) {
	out := parseFlags([]string{"--verbose", "-t", "s"})
	want := []string{"--verbose", "tools", "s"}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("expected %v, got %v", want, out)
	}
}

func TestParseFlagsPluginsShortcut(t *testing.T) {
	out := parseFlags([]string{"-p", "list"})
	want := []string{"plugins", "list"}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("expected %v, got %v", want, out)
	}
}

func TestParseFlagsOpenShortcut(t *testing.T) {
	out := parseFlags([]string{"-o", "profile"})
	want := []string{"open", "profile"}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("expected %v, got %v", want, out)
	}
}

func TestRunPluginOrSuggestUnknownReturnsError(t *testing.T) {
	baseDir := t.TempDir()
	code := runPluginOrSuggest(baseDir, []string{"not-existing-command"})
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

func TestSuggestClosest(t *testing.T) {
	candidates := []string{"plugins", "tools", "open"}
	got := suggestClosest("plguins", candidates, 3)
	if got != "plugins" {
		t.Fatalf("expected plugins, got %q", got)
	}
}

func TestSuggestClosestNoMatch(t *testing.T) {
	candidates := []string{"plugins", "tools", "open"}
	got := suggestClosest("xyz", candidates, 2)
	if got != "" {
		t.Fatalf("expected empty suggestion, got %q", got)
	}
}
