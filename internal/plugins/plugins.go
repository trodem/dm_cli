package plugins

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

type Plugin struct {
	Name string
	Path string
}

func List(baseDir string) ([]Plugin, error) {
	dir := filepath.Join(baseDir, "plugins")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Plugin{}, nil
		}
		return nil, err
	}

	var plugins []Plugin
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !isSupportedPlugin(name) {
			continue
		}
		plugins = append(plugins, Plugin{
			Name: pluginName(name),
			Path: filepath.Join(dir, name),
		})
	}

	sort.Slice(plugins, func(i, j int) bool { return plugins[i].Name < plugins[j].Name })
	return plugins, nil
}

func Run(baseDir, name string, args []string) error {
	dir := filepath.Join(baseDir, "plugins")
	candidate, err := findPlugin(dir, name)
	if err != nil {
		return err
	}
	if candidate == "" {
		return fmt.Errorf("plugin not found: %s", name)
	}
	return execPlugin(candidate, args)
}

func findPlugin(dir, name string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !isSupportedPlugin(e.Name()) {
			continue
		}
		if pluginName(e.Name()) == name {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	return "", nil
}

func execPlugin(path string, args []string) error {
	ext := strings.ToLower(filepath.Ext(path))
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		switch ext {
		case ".ps1":
			cmd = exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-File", path)
		case ".cmd", ".bat":
			cmd = exec.Command("cmd", "/C", path)
		case ".exe":
			cmd = exec.Command(path)
		default:
			return errors.New("unsupported plugin type on windows")
		}
	default:
		switch ext {
		case ".sh":
			cmd = exec.Command("sh", path)
		default:
			cmd = exec.Command(path)
		}
	}

	if len(args) > 0 {
		cmd.Args = append(cmd.Args, args...)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func isSupportedPlugin(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch runtime.GOOS {
	case "windows":
		return ext == ".ps1" || ext == ".cmd" || ext == ".bat" || ext == ".exe"
	default:
		return ext == ".sh" || ext == "" || ext == ".out"
	}
}

func pluginName(name string) string {
	ext := filepath.Ext(name)
	if ext == "" {
		return name
	}
	return strings.TrimSuffix(name, ext)
}
