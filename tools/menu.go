package tools

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func RunMenu(baseDir string) int {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("\nTools:")
		fmt.Println("  1) Search files")
		fmt.Println("  2) Rename files")
		fmt.Println("  0) Exit")
		fmt.Print("\n> ")

		choice := readLine(reader)
		switch choice {
		case "1":
			_ = RunSearch(baseDir, reader)
		case "2":
			_ = RunRename(baseDir, reader)
		case "0", "exit", "Exit", "":
			return 0
		default:
			fmt.Println("Invalid choice.")
		}
	}
}

func prompt(r *bufio.Reader, label, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	text, _ := r.ReadString('\n')
	text = strings.TrimSpace(text)
	if text == "" {
		return def
	}
	return text
}

func readLine(r *bufio.Reader) string {
	s, _ := r.ReadString('\n')
	return strings.TrimSpace(s)
}
