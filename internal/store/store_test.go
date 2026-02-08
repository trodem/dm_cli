package store

import (
	"path/filepath"
	"testing"
)

func TestCreatePackSetsDefaultMetadata(t *testing.T) {
	baseDir := t.TempDir()
	name := "work"

	if err := CreatePack(baseDir, name); err != nil {
		t.Fatal(err)
	}

	packPath := filepath.Join(baseDir, "packs", name, "pack.json")
	pf, err := LoadPackFile(packPath)
	if err != nil {
		t.Fatal(err)
	}
	if pf.Description != "Pack "+name {
		t.Fatalf("expected default description, got %q", pf.Description)
	}
	if pf.Summary == "" {
		t.Fatalf("expected default summary")
	}
	if len(pf.Examples) == 0 {
		t.Fatalf("expected default examples")
	}

	info, err := GetPackInfo(baseDir, name)
	if err != nil {
		t.Fatal(err)
	}
	if info.Description != "Pack "+name {
		t.Fatalf("expected info description, got %q", info.Description)
	}
	if info.Summary == "" {
		t.Fatalf("expected info summary")
	}
	if len(info.Examples) == 0 {
		t.Fatalf("expected info examples")
	}
}
