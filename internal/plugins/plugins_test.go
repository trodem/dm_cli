package plugins

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRunNotFound(t *testing.T) {
	baseDir := t.TempDir()
	err := Run(baseDir, "missing_plugin", nil)
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestListDeduplicatesByPluginName(t *testing.T) {
	baseDir := t.TempDir()
	pluginsDir := filepath.Join(baseDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	ps1 := filepath.Join(pluginsDir, "hello.ps1")
	sh := filepath.Join(pluginsDir, "hello.sh")
	if err := os.WriteFile(ps1, []byte("Write-Host hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sh, []byte("echo hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	items, err := List(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one plugin entry, got %d", len(items))
	}
	if items[0].Name != "hello" {
		t.Fatalf("expected plugin name hello, got %q", items[0].Name)
	}
}

func TestReadPowerShellFunctionNames(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profile.txt")
	content := "function stibs_restart_backend { }\nfunction _internal_helper { }\nfunction test_one { }\n# function ignored\nfunction stibs_restart_backend { }\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := readPowerShellFunctionNames(path)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"stibs_restart_backend", "test_one"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestCollectPowerShellFunctionsFromMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a_profile.ps1"), []byte("function one { }\nfunction shared { }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b_profile.txt"), []byte("function two { }\nfunction shared { }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "functions")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "c_profile.ps1"), []byte("function three { }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	catalog, files, err := collectPowerShellFunctions(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 {
		t.Fatalf("expected 3 source files, got %d", len(files))
	}
	if catalog["one"] == "" || catalog["two"] == "" || catalog["three"] == "" {
		t.Fatalf("expected one/two/three in catalog, got %v", catalog)
	}
	// first file wins on duplicate names
	if filepath.Base(catalog["shared"]) != "a_profile.ps1" {
		t.Fatalf("expected shared from a_profile.ps1, got %q", catalog["shared"])
	}
}

func TestListEntriesIncludesFunctions(t *testing.T) {
	baseDir := t.TempDir()
	pluginsDir := filepath.Join(baseDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginsDir, "tool.cmd"), []byte("@echo off"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginsDir, "profile.txt"), []byte("function restart_backend { }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	items, err := ListEntries(baseDir, true)
	if err != nil {
		t.Fatal(err)
	}
	foundScript := false
	foundFunction := false
	for _, it := range items {
		if it.Name == "tool" && it.Kind == "script" {
			foundScript = true
		}
		if it.Name == "restart_backend" && it.Kind == "function" {
			foundFunction = true
		}
	}
	if !foundScript || !foundFunction {
		t.Fatalf("expected script+function, got %+v", items)
	}
}

func TestGetInfoForFunctionWithCommentHelp(t *testing.T) {
	baseDir := t.TempDir()
	pluginsDir := filepath.Join(baseDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "<#\n.SYNOPSIS\nRestart backend service\n.DESCRIPTION\nRestarts docker backend in dev stack\n.PARAMETER force\nForce restart\n.EXAMPLE\ndm restart_backend\n#>\nfunction restart_backend {\n}\n"
	if err := os.WriteFile(filepath.Join(pluginsDir, "stibs.ps1"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	info, err := GetInfo(baseDir, "restart_backend")
	if err != nil {
		t.Fatal(err)
	}
	if info.Kind != "function" {
		t.Fatalf("expected function kind, got %q", info.Kind)
	}
	if info.Synopsis == "" || info.Description == "" {
		t.Fatalf("expected synopsis and description, got %+v", info)
	}
	if len(info.Parameters) == 0 || len(info.Examples) == 0 {
		t.Fatalf("expected parameters and examples, got %+v", info)
	}
}

func TestListFunctionFiles(t *testing.T) {
	baseDir := t.TempDir()
	pluginsDir := filepath.Join(baseDir, "plugins")
	subDir := filepath.Join(pluginsDir, "functions")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginsDir, "vars.ps1"), []byte("function _helper { }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "git.ps1"), []byte("function g_status { }\nfunction g_log { }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := ListFunctionFiles(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("expected exactly one function file, got %d", len(files))
	}
	if filepath.Base(files[0].Path) != "git.ps1" {
		t.Fatalf("expected git.ps1, got %s", files[0].Path)
	}
	if !reflect.DeepEqual(files[0].Functions, []string{"g_log", "g_status"}) {
		t.Fatalf("unexpected functions: %v", files[0].Functions)
	}
}
