package ui

import (
	"os"
	"strings"
	"testing"
)

func withEnv(key, value string, fn func()) {
	orig, had := os.LookupEnv(key)
	if value == "" {
		os.Unsetenv(key)
	} else {
		os.Setenv(key, value)
	}
	defer func() {
		if had {
			os.Setenv(key, orig)
		} else {
			os.Unsetenv(key)
		}
	}()
	fn()
}

func TestColorFunctions_WithColor(t *testing.T) {
	withEnv("NO_COLOR", "", func() {
		withEnv("TERM", "", func() {
			cases := []struct {
				name string
				fn   func(string) string
				code string
			}{
				{"Accent", Accent, "36"},
				{"OK", OK, "32"},
				{"Warn", Warn, "33"},
				{"Error", Error, "31"},
				{"Muted", Muted, "90"},
				{"Prompt", Prompt, "96"},
			}
			for _, tc := range cases {
				result := tc.fn("hello")
				expected := "\x1b[" + tc.code + "mhello\x1b[0m"
				if result != expected {
					t.Errorf("%s(\"hello\") = %q, want %q", tc.name, result, expected)
				}
			}
		})
	})
}

func TestColorFunctions_NoColor(t *testing.T) {
	withEnv("NO_COLOR", "1", func() {
		cases := []struct {
			name string
			fn   func(string) string
		}{
			{"Accent", Accent},
			{"OK", OK},
			{"Warn", Warn},
			{"Error", Error},
			{"Muted", Muted},
			{"Prompt", Prompt},
		}
		for _, tc := range cases {
			result := tc.fn("hello")
			if result != "hello" {
				t.Errorf("%s should return plain text with NO_COLOR set, got %q", tc.name, result)
			}
		}
	})
}

func TestColorFunctions_DumbTerm(t *testing.T) {
	withEnv("NO_COLOR", "", func() {
		withEnv("TERM", "dumb", func() {
			result := Accent("test")
			if result != "test" {
				t.Errorf("Accent should return plain text with TERM=dumb, got %q", result)
			}
		})
	})
}

func TestSupportsColor_Default(t *testing.T) {
	withEnv("NO_COLOR", "", func() {
		withEnv("TERM", "", func() {
			if !supportsColor() {
				t.Error("expected supportsColor()=true with no NO_COLOR and no TERM=dumb")
			}
		})
	})
}

func TestSupportsColor_NoColor(t *testing.T) {
	withEnv("NO_COLOR", "1", func() {
		if supportsColor() {
			t.Error("expected supportsColor()=false with NO_COLOR=1")
		}
	})
}

func TestSupportsColor_NoColorEmpty(t *testing.T) {
	withEnv("NO_COLOR", "", func() {
		os.Setenv("NO_COLOR", "")
		if supportsColor() {
			t.Error("expected supportsColor()=false when NO_COLOR is set (even empty)")
		}
	})
}

func TestSupportsColor_DumbTerm(t *testing.T) {
	withEnv("NO_COLOR", "", func() {
		withEnv("TERM", "dumb", func() {
			if supportsColor() {
				t.Error("expected supportsColor()=false with TERM=dumb")
			}
		})
	})
}

func TestToolkitLabel_SectionOutput(t *testing.T) {
	withEnv("NO_COLOR", "1", func() {
		result := Accent("== Tools ==")
		if result != "== Tools ==" {
			t.Errorf("expected plain text, got %q", result)
		}
	})
}

func TestEmptyText(t *testing.T) {
	withEnv("NO_COLOR", "", func() {
		withEnv("TERM", "", func() {
			result := Accent("")
			if !strings.Contains(result, "\x1b[36m") {
				t.Error("expected ANSI code even for empty text")
			}
		})
	})
}
