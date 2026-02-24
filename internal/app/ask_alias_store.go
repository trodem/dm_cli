package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	dmAliasProfileBegin = "# >>> dm aliases begin >>>"
	dmAliasProfileEnd   = "# <<< dm aliases end <<<"
)

var askAliasProfilePathResolver = resolveUserPowerShellProfilePath

func askAliasFilePath(baseDir string) string {
	return filepath.Join(baseDir, "dm.aliases.json")
}

func normalizeAskAliasName(name string) (string, error) {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "" {
		return "", fmt.Errorf("alias name is required")
	}
	for _, ch := range n {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' || ch == '.' {
			continue
		}
		return "", fmt.Errorf("invalid alias name %q (allowed: a-z, 0-9, -, _, .)", name)
	}
	reserved := map[string]bool{
		"help": true, "status": true, "reset": true, "clear": true, "cls": true,
		"exit": true, "quit": true, "pwd": true, "cd": true, "alias": true,
	}
	if reserved[n] {
		return "", fmt.Errorf("alias name %q is reserved", name)
	}
	return n, nil
}

func loadAskAliases(baseDir string) (map[string]string, error) {
	path := askAliasFilePath(baseDir)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	var aliases map[string]string
	if err := json.Unmarshal(raw, &aliases); err != nil {
		return nil, err
	}
	if aliases == nil {
		return map[string]string{}, nil
	}
	out := map[string]string{}
	for k, v := range aliases {
		normName, nameErr := normalizeAskAliasName(k)
		if nameErr != nil {
			continue
		}
		cmd := strings.TrimSpace(v)
		if cmd == "" {
			continue
		}
		out[normName] = cmd
	}
	return out, nil
}

func saveAskAliases(baseDir string, aliases map[string]string) error {
	path := askAliasFilePath(baseDir)
	if aliases == nil {
		aliases = map[string]string{}
	}
	data, err := json.MarshalIndent(aliases, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}
	if err := syncAskAliasesToProfile(aliases); err != nil {
		return fmt.Errorf("saved dm.aliases.json but failed to sync $PROFILE: %w", err)
	}
	return nil
}

func sortedAliasNames(aliases map[string]string) []string {
	keys := make([]string, 0, len(aliases))
	for k := range aliases {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func syncAskAliasesToProfile(aliases map[string]string) error {
	profilePath := strings.TrimSpace(askAliasProfilePathResolver())
	if profilePath == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(profilePath), 0755); err != nil {
		return err
	}
	existing := ""
	if raw, err := os.ReadFile(profilePath); err == nil {
		existing = string(raw)
	} else if !os.IsNotExist(err) {
		return err
	}
	block := renderAskAliasesProfileBlock(aliases)
	updated := upsertAskAliasesProfileBlock(existing, block)
	return os.WriteFile(profilePath, []byte(updated), 0644)
}

func renderAskAliasesProfileBlock(aliases map[string]string) string {
	var b strings.Builder
	b.WriteString(dmAliasProfileBegin + "\n")
	b.WriteString("$script:dmAliases = @{\n")
	for _, k := range sortedAliasNames(aliases) {
		v := strings.TrimSpace(aliases[k])
		if v == "" {
			continue
		}
		b.WriteString("    '")
		b.WriteString(escapePowerShellSingleQuoted(k))
		b.WriteString("' = '")
		b.WriteString(escapePowerShellSingleQuoted(v))
		b.WriteString("'\n")
	}
	b.WriteString("}\n")
	b.WriteString("foreach ($entry in $script:dmAliases.GetEnumerator()) {\n")
	b.WriteString("    $name = $entry.Key\n")
	b.WriteString("    $cmd = $entry.Value\n")
	b.WriteString("    $fn = 'dm_alias_' + ($name -replace '[^a-zA-Z0-9_]', '_')\n")
	b.WriteString("    Set-Item -Path ('Function:' + $fn) -Value ([ScriptBlock]::Create($cmd))\n")
	b.WriteString("    Set-Alias -Name $name -Value $fn -Scope Global\n")
	b.WriteString("}\n")
	b.WriteString(dmAliasProfileEnd + "\n")
	return b.String()
}

func upsertAskAliasesProfileBlock(existing, block string) string {
	start := strings.Index(existing, dmAliasProfileBegin)
	end := strings.Index(existing, dmAliasProfileEnd)
	if start >= 0 && end >= start {
		end += len(dmAliasProfileEnd)
		if end < len(existing) && existing[end] == '\n' {
			end++
		}
		updated := existing[:start] + block + existing[end:]
		return ensureTrailingNewline(updated)
	}
	existing = strings.TrimRight(existing, "\r\n")
	if existing == "" {
		return ensureTrailingNewline(block)
	}
	return ensureTrailingNewline(existing + "\n\n" + block)
}

func escapePowerShellSingleQuoted(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func ensureTrailingNewline(s string) string {
	if strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}
