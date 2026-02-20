package renamer

import (
	"os"
	"path/filepath"
	"testing"
)

func createFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestBuildPlan_SimpleReplace(t *testing.T) {
	dir := t.TempDir()
	createFile(t, filepath.Join(dir, "report_v1.txt"))
	createFile(t, filepath.Join(dir, "report_v2.txt"))
	createFile(t, filepath.Join(dir, "readme.md"))

	plan, err := BuildPlan(Options{
		BasePath: dir,
		From:     "report",
		To:       "summary",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan) != 2 {
		t.Fatalf("expected 2 renames, got %d", len(plan))
	}
	for _, item := range plan {
		base := filepath.Base(item.NewPath)
		if base != "summary_v1.txt" && base != "summary_v2.txt" {
			t.Fatalf("unexpected new name: %s", base)
		}
	}
}

func TestBuildPlan_CaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	createFile(t, filepath.Join(dir, "Report_V1.txt"))

	plan, err := BuildPlan(Options{
		BasePath:      dir,
		From:          "report",
		To:            "summary",
		CaseSensitive: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan) != 1 {
		t.Fatalf("expected 1 rename, got %d", len(plan))
	}
	if filepath.Base(plan[0].NewPath) != "summary_V1.txt" {
		t.Fatalf("unexpected new name: %s", filepath.Base(plan[0].NewPath))
	}
}

func TestBuildPlan_CaseSensitive(t *testing.T) {
	dir := t.TempDir()
	createFile(t, filepath.Join(dir, "Report_V1.txt"))

	plan, err := BuildPlan(Options{
		BasePath:      dir,
		From:          "report",
		To:            "summary",
		CaseSensitive: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan) != 0 {
		t.Fatalf("expected 0 renames (case mismatch), got %d", len(plan))
	}
}

func TestBuildPlan_NameFilter(t *testing.T) {
	dir := t.TempDir()
	createFile(t, filepath.Join(dir, "photo_old.jpg"))
	createFile(t, filepath.Join(dir, "doc_old.pdf"))

	plan, err := BuildPlan(Options{
		BasePath: dir,
		NamePart: "photo",
		From:     "old",
		To:       "new",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan) != 1 {
		t.Fatalf("expected 1 rename, got %d", len(plan))
	}
	if filepath.Base(plan[0].NewPath) != "photo_new.jpg" {
		t.Fatalf("unexpected new name: %s", filepath.Base(plan[0].NewPath))
	}
}

func TestBuildPlan_NoMatch(t *testing.T) {
	dir := t.TempDir()
	createFile(t, filepath.Join(dir, "file.txt"))

	plan, err := BuildPlan(Options{
		BasePath: dir,
		From:     "zzz",
		To:       "aaa",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan) != 0 {
		t.Fatalf("expected 0 renames, got %d", len(plan))
	}
}

func TestBuildPlan_Delete(t *testing.T) {
	dir := t.TempDir()
	createFile(t, filepath.Join(dir, "file_backup.txt"))

	plan, err := BuildPlan(Options{
		BasePath: dir,
		From:     "_backup",
		To:       "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan) != 1 {
		t.Fatalf("expected 1 rename, got %d", len(plan))
	}
	if filepath.Base(plan[0].NewPath) != "file.txt" {
		t.Fatalf("unexpected new name: %s", filepath.Base(plan[0].NewPath))
	}
}

func TestBuildPlan_Recursive(t *testing.T) {
	dir := t.TempDir()
	createFile(t, filepath.Join(dir, "a.old"))
	createFile(t, filepath.Join(dir, "sub", "b.old"))

	plan, err := BuildPlan(Options{
		BasePath:  dir,
		From:      ".old",
		To:        ".new",
		Recursive: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan) != 2 {
		t.Fatalf("expected 2 renames, got %d", len(plan))
	}
}

func TestBuildPlan_NonRecursive(t *testing.T) {
	dir := t.TempDir()
	createFile(t, filepath.Join(dir, "a.old"))
	createFile(t, filepath.Join(dir, "sub", "b.old"))

	plan, err := BuildPlan(Options{
		BasePath:  dir,
		From:      ".old",
		To:        ".new",
		Recursive: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan) != 1 {
		t.Fatalf("expected 1 rename (top level only), got %d", len(plan))
	}
}

func TestApplyPlan_Success(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "old.txt")
	newPath := filepath.Join(dir, "new.txt")
	createFile(t, oldPath)

	err := ApplyPlan([]PlanItem{{OldPath: oldPath, NewPath: newPath}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("expected new file to exist: %v", err)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatal("expected old file to be gone")
	}
}

func TestApplyPlan_DuplicateTarget(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.txt")
	target := filepath.Join(dir, "same.txt")
	createFile(t, a)
	createFile(t, b)

	err := ApplyPlan([]PlanItem{
		{OldPath: a, NewPath: target},
		{OldPath: b, NewPath: target},
	})
	if err == nil {
		t.Fatal("expected error for duplicate target")
	}
}

func TestApplyPlan_TargetExists(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, "old.txt")
	existing := filepath.Join(dir, "existing.txt")
	createFile(t, old)
	createFile(t, existing)

	err := ApplyPlan([]PlanItem{{OldPath: old, NewPath: existing}})
	if err == nil {
		t.Fatal("expected error when target already exists")
	}
}

func TestBuildPlan_Regex(t *testing.T) {
	dir := t.TempDir()
	createFile(t, filepath.Join(dir, "img001.png"))
	createFile(t, filepath.Join(dir, "img002.png"))
	createFile(t, filepath.Join(dir, "doc.txt"))

	plan, err := BuildPlan(Options{
		BasePath: dir,
		From:     `img(\d+)`,
		To:       "photo$1",
		UseRegex: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan) != 2 {
		t.Fatalf("expected 2 renames, got %d", len(plan))
	}
}

func TestReplaceInsensitive(t *testing.T) {
	got := replaceInsensitive("Hello_WORLD.txt", "world", "earth")
	if got != "Hello_earth.txt" {
		t.Fatalf("expected Hello_earth.txt, got %q", got)
	}
}

func TestReplaceInsensitive_EmptyFrom(t *testing.T) {
	got := replaceInsensitive("file.txt", "", "x")
	if got != "file.txt" {
		t.Fatalf("expected unchanged, got %q", got)
	}
}
