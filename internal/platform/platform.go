package platform

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

func OpenFileBrowser(path string) {
	if runtime.GOOS == "windows" {
		_ = exec.Command("explorer", path).Start()
		return
	}
	if runtime.GOOS == "darwin" {
		_ = exec.Command("open", path).Start()
		return
	}
	_ = exec.Command("xdg-open", path).Start()
}

func OpenVSCode(path string) {
	_ = exec.Command("code", path).Start()
}

func OpenTerminal(path string) {
	// apre un nuovo terminale nella dir, senza toccare profili/alias
	if runtime.GOOS == "windows" {
		// nuova finestra pwsh
		cmd := exec.Command("cmd", "/C", "start", "pwsh", "-NoExit", "-Command", fmt.Sprintf(`Set-Location -LiteralPath "%s"`, EscapeQuotes(path)))
		_ = cmd.Start()
		return
	}

	// linux/mac: prova $TERM emul. (best effort)
	_ = exec.Command("x-terminal-emulator", "--working-directory", path).Start()
}

func EscapeQuotes(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}
