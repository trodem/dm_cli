# dm

Personal CLI for tools, plugins, AI ask, and toolkit generation.

## Requirements
- Go 1.24+
- PowerShell available for `.ps1` plugins on Windows

## Build
```powershell
go build -o dm.exe .
```

Build with explicit version:
```powershell
go build -ldflags "-X cli/internal/app.Version=v0.2.0" -o dm.exe .
```

## Release
Create a versioned release folder in `dist/`:
```powershell
.\scripts\release.ps1 -Version v0.2.0
```

All targets:
```powershell
.\scripts\release.ps1 -Version v0.2.0 -AllTargets
```

For `windows/amd64`, `release.ps1` also updates root `./dm.exe`.

## Install
From a release folder:
```powershell
.\install.ps1
```

`install.ps1` also:
- copies `plugins/`
- creates `dm.agent.json` from `dm.agent.example.json` (if missing)
- tries `dm completion install`

## Core Commands
```bash
dm help
dm tools
dm plugins
dm ask
dm doctor
dm toolkit
dm completion
dm ps_profile
dm cp profile
dm -o ps_profile
dm -o profile
```

Group shortcuts:
- `-t`, `--tools` -> `tools`
- `-p`, `--plugins` -> `plugins`
- `-o`, `--open` -> `open`

## AI Agent (`dm ask`)
Providers:
- `openai` (default)
- `ollama`
- `auto` (tries Ollama first, then OpenAI)

Flags:
- `--provider openai|ollama|auto`
- `--model <name>`
- `--base-url <url>`
- `--confirm-tools` / `--no-confirm-tools`
- `--risk-policy strict|normal|off`
- `--json` (structured output, one-shot mode only)

Examples:
```bash
dm ask "spiegami questo errore"
dm ask --provider auto "cerca i file pdf in Downloads"
dm ask --json "trova file recenti in Downloads"
```

Config path priority:
1. `DM_AGENT_CONFIG`
2. `dm.agent.json` next to executable
3. `~/.config/dm/agent.json`

OpenAI key can also be set with `OPENAI_API_KEY`.

## Tools
Interactive menu:
```bash
dm tools
dm -t
```

Run directly:
```bash
dm tools search
dm tools rename
dm tools recent
dm tools backup
dm tools clean
dm tools system
```

Tool aliases:
- `search/s`
- `rename/r`
- `recent/rec`
- `backup/b`
- `clean/c`
- `system/sys/htop`

## Plugins
Recommended layout:
- `plugins/variables.ps1`
- `plugins/utils.ps1`
- `plugins/functions/*.ps1`

Commands:
```bash
dm -p
dm plugins list
dm plugins list --functions
dm plugins info <name>
dm plugins menu
dm plugins run <name> [args...]
dm <plugin_or_function> [args...]
```

Validate plugin help blocks:
```powershell
go run ./scripts/check_plugin_help.go
```

## Toolkit Generator
Built into `dm` (no external generator exe):
```powershell
dm toolkit
dm toolkit new --name MSWord --prefix word --category office
dm toolkit add --file plugins/functions/office/MSWord_Toolkit.ps1 --prefix word --func export_pdf --param InputPath --param OutputPath --confirm
dm toolkit validate
```

Quick reference: `docs/dm-toolkit-cheatsheet.md`

## Completion
Generate scripts:
```bash
dm completion powershell
dm completion bash
dm completion zsh
dm completion fish
```

Install automatically:
```bash
dm completion install
dm completion install --shell bash
dm completion install --shell zsh
dm completion install --shell fish
```

## Repository Layout
```text
.
|-- main.go
|-- internal/
|   |-- agent/
|   |-- app/
|   |-- doctor/
|   |-- filesearch/
|   |-- platform/
|   |-- plugins/
|   |-- renamer/
|   |-- systeminfo/
|   |-- toolkitgen/
|   `-- ui/
|-- tools/
|-- plugins/
|-- scripts/
|-- README.md
`-- README.txt
```
