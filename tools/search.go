package tools

import (
	"bufio"
	"fmt"

	"cli/internal/filesearch"
)

func RunSearch(baseDir string, r *bufio.Reader) int {
	base := prompt(r, "Base path", baseDir)
	name := prompt(r, "Name contains", "")
	ext := prompt(r, "Extension (optional)", "")
	sortBy := prompt(r, "Sort (name|date|size)", "name")

	results, err := filesearch.Find(filesearch.Options{
		BasePath: base,
		NamePart: name,
		Ext:      ext,
		SortBy:   sortBy,
	})
	if err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	filesearch.RenderList(results)
	return 0
}
