package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadWithInclude(t *testing.T) {
	dir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(dir, "cfg"), 0755); err != nil {
		t.Fatal(err)
	}
	mainCfg := `{
  "include": ["cfg/base.json"]
}`
	if err := os.WriteFile(filepath.Join(dir, "dm.json"), []byte(mainCfg), 0644); err != nil {
		t.Fatal(err)
	}
	includeCfg := `{
  "jump": {"dev": "E:/dev"},
  "run": {"gs": "git status"},
  "search": { "knowledge": "knowledge" }
}`
	if err := os.WriteFile(filepath.Join(dir, "cfg", "base.json"), []byte(includeCfg), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(filepath.Join(dir, "dm.json"), Options{UseCache: false})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Jump["dev"] != "E:/dev" {
		t.Fatalf("expected jump dev, got %v", cfg.Jump["dev"])
	}
	if cfg.Run["gs"] != "git status" {
		t.Fatalf("expected run gs, got %v", cfg.Run["gs"])
	}
	if cfg.Search.Knowledge != "knowledge" {
		t.Fatalf("expected knowledge, got %v", cfg.Search.Knowledge)
	}
}

func TestLoadWithProfile(t *testing.T) {
	dir := t.TempDir()

	mainCfg := `{
  "include": ["cfg/base.json"],
  "profiles": {
    "work": { "include": ["cfg/work.json"] }
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "dm.json"), []byte(mainCfg), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "cfg"), 0755); err != nil {
		t.Fatal(err)
	}
	defaultCfg := `{"jump": {"home": "C:/home"}}`
	if err := os.WriteFile(filepath.Join(dir, "cfg", "base.json"), []byte(defaultCfg), 0644); err != nil {
		t.Fatal(err)
	}
	workCfg := `{"jump": {"office": "C:/office"}}`
	if err := os.WriteFile(filepath.Join(dir, "cfg", "work.json"), []byte(workCfg), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(filepath.Join(dir, "dm.json"), Options{Profile: "work", UseCache: false})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.Jump["office"]; !ok {
		t.Fatalf("expected office jump")
	}
	if _, ok := cfg.Jump["home"]; ok {
		t.Fatalf("did not expect default jump when profile overrides include")
	}
}

func TestCache(t *testing.T) {
	dir := t.TempDir()

	mainCfg := `{"include": ["cfg/base.json"]}`
	if err := os.WriteFile(filepath.Join(dir, "dm.json"), []byte(mainCfg), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "cfg"), 0755); err != nil {
		t.Fatal(err)
	}
	jumpCfg := `{"jump": {"dev": "E:/dev"}}`
	if err := os.WriteFile(filepath.Join(dir, "cfg", "base.json"), []byte(jumpCfg), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(filepath.Join(dir, "dm.json"), Options{UseCache: true})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Jump["dev"] != "E:/dev" {
		t.Fatalf("expected jump dev, got %v", cfg.Jump["dev"])
	}

	cfg2, err := Load(filepath.Join(dir, "dm.json"), Options{UseCache: true})
	if err != nil {
		t.Fatal(err)
	}
	if cfg2.Jump["dev"] != "E:/dev" {
		t.Fatalf("expected jump dev from cache, got %v", cfg2.Jump["dev"])
	}
}

func TestNoImplicitPackInclude(t *testing.T) {
	dir := t.TempDir()

	cfg, err := Load(filepath.Join(dir, "dm.json"), Options{UseCache: false})
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Jump) != 0 || len(cfg.Run) != 0 || len(cfg.Projects) != 0 {
		t.Fatalf("expected empty config when dm.json is missing, got %#v", cfg)
	}
}
