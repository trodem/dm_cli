package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadWithInclude(t *testing.T) {
	dir := t.TempDir()

	mainCfg := `{
  "include": ["packs/*/pack.json"]
}`
	if err := os.WriteFile(filepath.Join(dir, "tellme.json"), []byte(mainCfg), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "packs", "core"), 0755); err != nil {
		t.Fatal(err)
	}
	packCfg := `{
  "jump": {"dev": "E:/dev"},
  "run": {"gs": "git status"},
  "search": { "knowledge": "packs/core/knowledge" }
}`
	if err := os.WriteFile(filepath.Join(dir, "packs", "core", "pack.json"), []byte(packCfg), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(filepath.Join(dir, "tellme.json"), Options{UseCache: false})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Jump["dev"] != "E:/dev" {
		t.Fatalf("expected jump dev, got %v", cfg.Jump["dev"])
	}
	if cfg.Run["gs"] != "git status" {
		t.Fatalf("expected run gs, got %v", cfg.Run["gs"])
	}
	if cfg.Search.Knowledge != "packs/core/knowledge" {
		t.Fatalf("expected knowledge, got %v", cfg.Search.Knowledge)
	}
}

func TestLoadWithProfile(t *testing.T) {
	dir := t.TempDir()

	mainCfg := `{
  "include": ["packs/*/pack.json"],
  "profiles": {
    "work": { "include": ["packs/work/pack.json"] }
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "tellme.json"), []byte(mainCfg), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "packs", "core"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "packs", "work"), 0755); err != nil {
		t.Fatal(err)
	}
	defaultCfg := `{"jump": {"home": "C:/home"}}`
	if err := os.WriteFile(filepath.Join(dir, "packs", "core", "pack.json"), []byte(defaultCfg), 0644); err != nil {
		t.Fatal(err)
	}
	workCfg := `{"jump": {"office": "C:/office"}}`
	if err := os.WriteFile(filepath.Join(dir, "packs", "work", "pack.json"), []byte(workCfg), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(filepath.Join(dir, "tellme.json"), Options{Profile: "work", UseCache: false})
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

	mainCfg := `{"include": ["packs/*/pack.json"]}`
	if err := os.WriteFile(filepath.Join(dir, "tellme.json"), []byte(mainCfg), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "packs", "core"), 0755); err != nil {
		t.Fatal(err)
	}
	jumpCfg := `{"jump": {"dev": "E:/dev"}}`
	if err := os.WriteFile(filepath.Join(dir, "packs", "core", "pack.json"), []byte(jumpCfg), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(filepath.Join(dir, "tellme.json"), Options{UseCache: true})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Jump["dev"] != "E:/dev" {
		t.Fatalf("expected jump dev, got %v", cfg.Jump["dev"])
	}

	cfg2, err := Load(filepath.Join(dir, "tellme.json"), Options{UseCache: true})
	if err != nil {
		t.Fatal(err)
	}
	if cfg2.Jump["dev"] != "E:/dev" {
		t.Fatalf("expected jump dev from cache, got %v", cfg2.Jump["dev"])
	}
}

func TestPackDefaultKnowledge(t *testing.T) {
	dir := t.TempDir()

	mainCfg := `{"include": ["packs/*/pack.json"]}`
	if err := os.WriteFile(filepath.Join(dir, "tellme.json"), []byte(mainCfg), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "packs", "core"), 0755); err != nil {
		t.Fatal(err)
	}
	packCfg := `{"jump": {"dev": "E:/dev"}}`
	if err := os.WriteFile(filepath.Join(dir, "packs", "core", "pack.json"), []byte(packCfg), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(filepath.Join(dir, "tellme.json"), Options{Pack: "core", UseCache: false})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Search.Knowledge != filepath.Join("packs", "core", "knowledge") {
		t.Fatalf("expected default knowledge, got %v", cfg.Search.Knowledge)
	}
}
