package filesearch

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Result struct {
	Path    string
	Size    int64
	ModTime time.Time
}

type Options struct {
	BasePath string
	NamePart string
	Ext      string
	SortBy   string
}

func Find(opts Options) ([]Result, error) {
	base := opts.BasePath
	if base == "" {
		base = "."
	}
	namePart := strings.ToLower(strings.TrimSpace(opts.NamePart))
	ext := strings.ToLower(strings.TrimSpace(opts.Ext))
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	var results []Result
	err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		name := strings.ToLower(d.Name())
		if namePart != "" && !strings.Contains(name, namePart) {
			return nil
		}
		if ext != "" && strings.ToLower(filepath.Ext(name)) != ext {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		results = append(results, Result{
			Path:    path,
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sortResults(results, opts.SortBy)
	return results, nil
}

func RenderList(results []Result) {
	if len(results) == 0 {
		fmt.Println("Nessun file trovato.")
		return
	}
	for _, r := range results {
		fmt.Printf("%s | %s | %s\n", r.ModTime.Format("2006-01-02 15:04"), formatSize(r.Size), r.Path)
	}
}

func sortResults(results []Result, sortBy string) {
	switch strings.ToLower(strings.TrimSpace(sortBy)) {
	case "date":
		sort.Slice(results, func(i, j int) bool {
			return results[i].ModTime.After(results[j].ModTime)
		})
	case "size":
		sort.Slice(results, func(i, j int) bool {
			return results[i].Size > results[j].Size
		})
	default:
		sort.Slice(results, func(i, j int) bool {
			return strings.ToLower(results[i].Path) < strings.ToLower(results[j].Path)
		})
	}
}

func formatSize(n int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case n >= gb:
		return fmt.Sprintf("%.2fGB", float64(n)/float64(gb))
	case n >= mb:
		return fmt.Sprintf("%.2fMB", float64(n)/float64(mb))
	case n >= kb:
		return fmt.Sprintf("%.2fKB", float64(n)/float64(kb))
	default:
		return fmt.Sprintf("%dB", n)
	}
}

func FormatSize(n int64) string {
	return formatSize(n)
}
