package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type PackFile struct {
	Description string             `json:"description"`
	Summary     string             `json:"summary"`
	Owner       string             `json:"owner"`
	Tags        []string           `json:"tags"`
	Examples    []string           `json:"examples"`
	Jump        map[string]string  `json:"jump"`
	Run         map[string]string  `json:"run"`
	Projects    map[string]Project `json:"projects"`
	Search      SearchConfig       `json:"search"`
}

type Project struct {
	Path     string            `json:"path"`
	Commands map[string]string `json:"commands"`
}

type SearchConfig struct {
	Knowledge string `json:"knowledge"`
}

func CreatePack(baseDir, name string) error {
	packDir := filepath.Join(baseDir, "packs", name)
	knowledgeDir := filepath.Join(packDir, "knowledge")
	if err := os.MkdirAll(knowledgeDir, 0755); err != nil {
		return err
	}
	packPath := filepath.Join(packDir, "pack.json")
	pf := PackFile{
		Description: "Pack " + name,
		Summary:     "Commands and knowledge for " + name,
		Examples: []string{
			"dm -p " + name + " find <query>",
			"dm -p " + name + " run <alias>",
		},
		Jump:     map[string]string{},
		Run:      map[string]string{},
		Projects: map[string]Project{},
		Search: SearchConfig{
			Knowledge: filepath.Join("packs", name, "knowledge"),
		},
	}
	if err := writeJSON(packPath, pf); err != nil {
		return err
	}
	return nil
}

func ListPacks(baseDir string) ([]string, error) {
	packsDir := filepath.Join(baseDir, "packs")
	entries, err := os.ReadDir(packsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		packPath := filepath.Join(packsDir, e.Name(), "pack.json")
		if _, err := os.Stat(packPath); err == nil {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

type PackInfo struct {
	Name        string
	Path        string
	Description string
	Summary     string
	Owner       string
	Tags        []string
	Examples    []string
	Knowledge   string
	Jumps       int
	Runs        int
	Projects    int
	Actions     int
}

func GetPackInfo(baseDir, name string) (PackInfo, error) {
	packPath := filepath.Join(baseDir, "packs", name, "pack.json")
	pf, err := LoadPackFile(packPath)
	if err != nil {
		return PackInfo{}, err
	}
	info := PackInfo{
		Name:        name,
		Path:        packPath,
		Description: pf.Description,
		Summary:     pf.Summary,
		Owner:       pf.Owner,
		Tags:        append([]string{}, pf.Tags...),
		Examples:    append([]string{}, pf.Examples...),
		Knowledge:   pf.Search.Knowledge,
		Jumps:       len(pf.Jump),
		Runs:        len(pf.Run),
		Projects:    len(pf.Projects),
		Actions:     countActions(pf.Projects),
	}
	return info, nil
}

func countActions(projects map[string]Project) int {
	total := 0
	for _, p := range projects {
		total += len(p.Commands)
	}
	return total
}

func PackExists(baseDir, name string) bool {
	packPath := filepath.Join(baseDir, "packs", name, "pack.json")
	_, err := os.Stat(packPath)
	return err == nil
}

func ActivePackPath(baseDir string) string {
	return filepath.Join(baseDir, ".dm.active-pack")
}

func GetActivePack(baseDir string) (string, error) {
	path := ActivePackPath(baseDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func SetActivePack(baseDir, name string) error {
	path := ActivePackPath(baseDir)
	return os.WriteFile(path, []byte(name+"\n"), 0644)
}

func ClearActivePack(baseDir string) error {
	path := ActivePackPath(baseDir)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func LoadPackFile(path string) (PackFile, error) {
	var pf PackFile
	if err := readJSON(path, &pf); err != nil {
		if os.IsNotExist(err) {
			return PackFile{
				Jump:     map[string]string{},
				Run:      map[string]string{},
				Projects: map[string]Project{},
			}, nil
		}
		return pf, err
	}
	if pf.Jump == nil {
		pf.Jump = map[string]string{}
	}
	if pf.Run == nil {
		pf.Run = map[string]string{}
	}
	if pf.Projects == nil {
		pf.Projects = map[string]Project{}
	}
	if pf.Tags == nil {
		pf.Tags = []string{}
	}
	if pf.Examples == nil {
		pf.Examples = []string{}
	}
	return pf, nil
}

func SavePackFile(path string, pf PackFile) error {
	if pf.Jump == nil {
		pf.Jump = map[string]string{}
	}
	if pf.Run == nil {
		pf.Run = map[string]string{}
	}
	if pf.Projects == nil {
		pf.Projects = map[string]Project{}
	}
	if pf.Tags == nil {
		pf.Tags = []string{}
	}
	if pf.Examples == nil {
		pf.Examples = []string{}
	}
	return writeJSON(path, pf)
}

func readJSON(path string, dst any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}

func writeJSON(path string, v any) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}
