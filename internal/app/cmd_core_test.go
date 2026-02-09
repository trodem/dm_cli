package app

import (
	"reflect"
	"testing"

	"cli/internal/store"
)

func TestPackDoctorIssues_OK(t *testing.T) {
	pf := store.PackFile{
		SchemaVersion: 1,
		Description:   "desc",
		Summary:       "sum",
		Examples:      []string{"dm -p x find y"},
		Jump:          map[string]string{"x": "C:/x"},
		Run:           map[string]string{},
		Projects:      map[string]store.Project{},
		Search:        store.SearchConfig{Knowledge: "packs/x/knowledge"},
	}
	issues := packDoctorIssues(pf, "packs/x/knowledge", "C:/packs/x/knowledge", true)
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %v", issues)
	}
}

func TestPackDoctorIssues_MissingFields(t *testing.T) {
	pf := store.PackFile{
		Jump:     map[string]string{},
		Run:      map[string]string{},
		Projects: map[string]store.Project{},
		Search:   store.SearchConfig{},
	}
	issues := packDoctorIssues(pf, "", "", false)
	if len(issues) < 4 {
		t.Fatalf("expected multiple issues, got %v", issues)
	}
}

func TestPackDoctorIssues_UnsupportedSchema(t *testing.T) {
	pf := store.PackFile{
		SchemaVersion: 2,
		Description:   "desc",
		Summary:       "sum",
		Examples:      []string{"dm -p x find y"},
		Jump:          map[string]string{"x": "C:/x"},
		Run:           map[string]string{},
		Projects:      map[string]store.Project{},
		Search:        store.SearchConfig{Knowledge: "packs/x/knowledge"},
	}
	issues := packDoctorIssues(pf, "packs/x/knowledge", "C:/packs/x/knowledge", true)
	found := false
	for _, issue := range issues {
		if issue == "schema_version 2 is not supported (expected 1)" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected unsupported schema issue, got %v", issues)
	}
}

func TestPackProfileLegacyArgsWithoutCommand(t *testing.T) {
	got := packProfileLegacyArgs("vim", nil)
	want := []string{"--pack", "vim"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestPackProfileLegacyArgsWithCommand(t *testing.T) {
	got := packProfileLegacyArgs("vim", []string{"run", "vim"})
	want := []string{"--pack", "vim", "run", "vim"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}
