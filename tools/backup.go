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

	"cli/internal/store"
)

func RunPackBackup(baseDir string, r *bufio.Reader) int {
	active, _ := store.GetActivePack(baseDir)
	pack := prompt(r, "Pack name", active)
	if strings.TrimSpace(pack) == "" {
		fmt.Println("Error: pack name is required.")
		return 1
	}
	if !store.PackExists(baseDir, pack) {
		fmt.Println("Error: pack not found.")
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
	outPath := filepath.Join(outDir, fmt.Sprintf("pack-%s-%s.zip", pack, ts))
	packDir := filepath.Join(baseDir, "packs", pack)

	if err := zipDir(packDir, outPath); err != nil {
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
