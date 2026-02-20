package tools

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func RunBackupAuto(baseDir string, params map[string]string) int {
	source := strings.TrimSpace(params["source"])
	if source == "" {
		source = currentWorkingDir(baseDir)
	}
	source = normalizeInputPath(source, currentWorkingDir(baseDir))
	if err := validateExistingDir(source, "source dir"); err != nil {
		fmt.Println("Error:", err)
		return 1
	}

	outDir := strings.TrimSpace(params["output"])
	if outDir == "" {
		outDir = filepath.Join(baseDir, "backups")
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Println("Error:", err)
		return 1
	}

	ts := time.Now().Format("20060102-1504")
	baseName := filepath.Base(source)
	if baseName == "" || baseName == "." || baseName == string(filepath.Separator) {
		baseName = "backup"
	}
	outPath := filepath.Join(outDir, fmt.Sprintf("%s-%s.zip", baseName, ts))

	if err := zipDir(source, outPath); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	fmt.Println("Saved:", outPath)
	return 0
}

func RunPackBackup(baseDir string, r *bufio.Reader) int {
	defaultSource := currentWorkingDir(baseDir)
	sourceDir := prompt(r, "Source dir", defaultSource)
	sourceDir = normalizeInputPath(sourceDir, defaultSource)
	if err := validateExistingDir(sourceDir, "source dir"); err != nil {
		fmt.Println("Error:", err)
		return 1
	}

	outDir := prompt(r, "Output dir", filepath.Join(baseDir, "backups"))
	if strings.TrimSpace(outDir) == "" {
		fmt.Println("Error: output dir is required.")
		return 1
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Println("Error:", err)
		return 1
	}

	ts := time.Now().Format("20060102-1504")
	baseName := filepath.Base(sourceDir)
	if baseName == "" || baseName == "." || baseName == string(filepath.Separator) {
		baseName = "backup"
	}
	outPath := filepath.Join(outDir, fmt.Sprintf("%s-%s.zip", baseName, ts))

	if err := zipDir(sourceDir, outPath); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	fmt.Println("Saved:", outPath)
	return 0
}

func zipDir(srcDir, zipPath string) error {
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		w, err := zw.Create(rel)
		if err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		_, err = io.Copy(w, in)
		return err
	})
}
