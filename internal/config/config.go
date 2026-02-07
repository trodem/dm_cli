package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Config struct {
	Jump     map[string]string  `json:"jump"`
	Run      map[string]string  `json:"run"`
	Projects map[string]Project `json:"projects"`
	Search   SearchConfig       `json:"search"`
	Include  []string           `json:"include"`
	Profiles map[string]Profile `json:"profiles"`
}

type Project struct {
	Path     string            `json:"path"`
	Commands map[string]string `json:"commands"`
}

type SearchConfig struct {
	Knowledge string `json:"knowledge"`
}

type Profile struct {
	Include []string    `json:"include"`
	Search  SearchConfig `json:"search"`
}

type Options struct {
	Profile  string
	UseCache bool
	Pack     string
}

type CacheFile struct {
	Sources map[string]int64 `json:"sources"`
	Config  Config           `json:"config"`
}

func Load(path string, opts Options) (Config, error) {
	baseDir := filepath.Dir(path)

	baseCfg, err := loadFile(path)
	if err != nil {
		return baseCfg, err
	}

	if opts.Profile != "" {
		applyProfile(&baseCfg, opts.Profile)
	}

	if opts.Pack != "" {
		baseCfg.Include = []string{filepath.Join("packs", opts.Pack, "pack.json")}
	}

	sources, err := collectSources(path, baseCfg.Include, baseDir)
	if err != nil {
		return baseCfg, err
	}

	if opts.UseCache {
		cachePath := cacheFilePath(baseDir, opts.Profile)
		if cached, ok := loadValidCache(cachePath, sources); ok {
			normalize(&cached)
			return cached, nil
		}
	}

	cfg := baseCfg
	if len(cfg.Include) > 0 {
		if err := applyIncludes(&cfg, baseDir); err != nil {
			return cfg, err
		}
	}

	normalize(&cfg)
	applyPackDefaults(&cfg, opts.Pack)

	if opts.UseCache {
		cachePath := cacheFilePath(baseDir, opts.Profile)
		_ = writeCache(cachePath, sources, cfg)
	}

	return cfg, nil
}

func ResolvePath(baseDir, p string) string {
	if p == "" {
		return p
	}
	// supporta path con slash stile E:/...
	p = filepath.FromSlash(p)
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(baseDir, p)
}

func loadFile(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func applyIncludes(target *Config, baseDir string) error {
	for _, pattern := range target.Include {
		p := ResolvePath(baseDir, pattern)
		matches, err := filepath.Glob(p)
		if err != nil {
			return err
		}
		sort.Strings(matches)
		for _, path := range matches {
			inc, err := loadFile(path)
			if err != nil {
				return err
			}
			mergeConfig(target, inc)
		}
	}
	return nil
}

func applyProfile(cfg *Config, name string) {
	if cfg.Profiles == nil {
		return
	}
	p, ok := cfg.Profiles[name]
	if !ok {
		return
	}
	if len(p.Include) > 0 {
		cfg.Include = append([]string{}, p.Include...)
	}
	if p.Search.Knowledge != "" {
		cfg.Search.Knowledge = p.Search.Knowledge
	}
}

func applyPackDefaults(cfg *Config, pack string) {
	if pack == "" {
		return
	}
	if strings.TrimSpace(cfg.Search.Knowledge) == "" {
		cfg.Search.Knowledge = filepath.Join("packs", pack, "knowledge")
	}
}

func collectSources(configPath string, include []string, baseDir string) (map[string]int64, error) {
	sources := map[string]int64{}
	if err := addSource(sources, configPath); err != nil {
		return nil, err
	}
	for _, pattern := range include {
		p := ResolvePath(baseDir, pattern)
		matches, err := filepath.Glob(p)
		if err != nil {
			return nil, err
		}
		sort.Strings(matches)
		for _, m := range matches {
			if err := addSource(sources, m); err != nil {
				return nil, err
			}
		}
	}
	return sources, nil
}

func addSource(sources map[string]int64, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	sources[path] = info.ModTime().UnixNano()
	return nil
}

func cacheFilePath(baseDir, profile string) string {
	if profile == "" {
		return filepath.Join(baseDir, ".tellme.cache.json")
	}
	return filepath.Join(baseDir, ".tellme.cache."+profile+".json")
}

func loadValidCache(path string, sources map[string]int64) (Config, bool) {
	var cf CacheFile
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, false
	}
	if err := json.Unmarshal(data, &cf); err != nil {
		return Config{}, false
	}
	if !sourcesMatch(cf.Sources, sources) {
		return Config{}, false
	}
	return cf.Config, true
}

func sourcesMatch(a, b map[string]int64) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func writeCache(path string, sources map[string]int64, cfg Config) error {
	cf := CacheFile{
		Sources: sources,
		Config:  cfg,
	}
	data, err := json.MarshalIndent(cf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func mergeConfig(dst *Config, src Config) {
	if dst.Jump == nil {
		dst.Jump = map[string]string{}
	}
	if dst.Run == nil {
		dst.Run = map[string]string{}
	}
	if dst.Projects == nil {
		dst.Projects = map[string]Project{}
	}

	for k, v := range src.Jump {
		dst.Jump[k] = v
	}
	for k, v := range src.Run {
		dst.Run[k] = v
	}
	for name, p := range src.Projects {
		existing, ok := dst.Projects[name]
		if !ok {
			dst.Projects[name] = p
			continue
		}
		if p.Path != "" {
			existing.Path = p.Path
		}
		if existing.Commands == nil {
			existing.Commands = map[string]string{}
		}
		for ck, cv := range p.Commands {
			existing.Commands[ck] = cv
		}
		dst.Projects[name] = existing
	}
	if src.Search.Knowledge != "" {
		dst.Search.Knowledge = src.Search.Knowledge
	}
}

func normalize(cfg *Config) {
	if cfg.Jump == nil {
		cfg.Jump = map[string]string{}
	}
	if cfg.Run == nil {
		cfg.Run = map[string]string{}
	}
	if cfg.Projects == nil {
		cfg.Projects = map[string]Project{}
	}
}
