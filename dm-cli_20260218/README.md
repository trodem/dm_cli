# dm

Small personal CLI to jump to folders and run project commands.

## Index
- [Features](#features)
- [Requirements](#requirements)
- [Build](#build)
- [Configuration](#configuration)
- [Usage](#usage)
- [Project Actions](#project-actions)
- [Plugins](#plugins)
- [Tools](#tools)
- [Help](#help)
- [Project Layout](#project-layout)
- [Changelog](#changelog)

## Features
- Jump to paths from aliases
- Run global aliases
- Run project actions
- Interactive menu for targets
- Splash shows version, executable build time, and current time
- Config splitting with includes and profiles
- Plugins (scripts) support
- Validation and list/add commands

## Requirements
- Go 1.24+

## Build
```bash
go build -o dm.exe .
```

Local release package (Windows PowerShell):
```powershell
.\scripts\release.ps1 -Version v0.2.0
```
All standard targets (windows/linux/darwin):
```powershell
.\scripts\release.ps1 -Version v0.2.0 -AllTargets
```
This produces:
- `dist/dm-v0.2.0-windows-amd64/`
- `dist/dm-v0.2.0-windows-amd64.zip`
- `dist/dm-v0.2.0-windows-amd64.zip.sha256`
Each artifact folder/zip includes:
- `dm` or `dm.exe`
- `install.ps1`
- `dm.agent.json` (if present)
- `README.txt`
- `README.md`
- `LICENSE` (if present)

## Configuration
`dm` works without a config file. If `dm.json` is missing, CLI starts with an empty config.

If you want custom includes or profiles, create `dm.json`.

Example `dm.json`:
```json
{
  "jump": { "api": "projects/api" },
  "run": { "gs": "git status" },
  "projects": {
    "git-tools": {
      "path": "projects/git-tools",
      "commands": { "gcommit": "git add . && git commit" }
    }
  }
}
```

Usage:
```bash
dm api
dm run gs
dm git-tools gcommit
```

Notes:
- Paths can be absolute or relative to the executable directory.
- Forward slashes are supported on Windows (`E:/...`).

### Split Config (includes)
You can include config files using `include` patterns.

### Profiles
Define profile-specific includes:
```json
{
  "include": ["config/*.json"],
  "profiles": {
    "work": {
      "include": ["config/work.json"]
    }
  }
}
```

Use it with:
```bash
dm --profile work list jumps
```

### Cache
`dm` writes a cache file in the executable directory:
- `.dm.cache.json` (default)
- `.dm.cache.<profile>.json` (profile)

Disable with:
```bash
dm --no-cache list jumps
```

## Usage
```bash
dm help
dm aliases
dm list jumps
dm add jump <name> <path>
dm ps_profile
dm cp profile
dm -o ps_profile
dm -o profile
dm validate
dm plugins list
dm plugins list --functions
dm plugins info <name>
dm plugins menu
dm plugins run <name> [args...]
dm <plugin> [args...]
dm doctor
dm doctor --json
dm ask <prompt...>
dm ask --provider ollama --model deepseek-coder-v2:latest "spiegami questo errore"
dm ask
dm run <alias>
dm tools
dm tools <tool>
dm -t [tool]
dm -p [cmd]
dm -o [cmd]
dm <project> <action>
dm <name>
```

Notes:
- Group shortcuts:
  - `-t` / `--tools` => `tools`
  - `-p` / `--plugins` => `plugins`
  - `-o` / `--open` => `open`
- Fallback dispatch:
- `dm <name>` now tries `jump/project` first, then direct plugin execution.
  - If no plugin exists, `dm` returns an error.

## AI Agent
`dm ask` uses AI with this priority:
- Ollama first: model `deepseek-coder-v2:latest`
- OpenAI fallback: if Ollama is unavailable

Runtime overrides:
- `--provider openai|auto|ollama` (default `openai`)
- `--model <name>` (override model for selected provider)
- `--base-url <url>` (override provider base URL)
- `--confirm-tools` (default `true`, ask confirmation before running plugin/function/tool selected by the agent)
- `--no-confirm-tools` (disable confirmation)
- `--risk-policy strict|normal|off` (default `normal`)
  - `strict`: always ask confirmation before agent actions
  - `normal`: ask for high-risk actions even with `--no-confirm-tools`
  - `off`: use only `--confirm-tools` / `--no-confirm-tools`

Interactive mode:
- `dm ask` opens a persistent ask prompt
- It exits only with explicit commands: `/exit`, `exit`, `quit`
- At startup, `dm ask` resolves provider once (Ollama first, OpenAI fallback) and keeps a fixed prompt for the session: `ask(provider,model)>`
- The interactive prompt label is highlighted in yellow

Response header:
- each answer prints `[provider | model]` before the text

Plugin tool-use:
- the agent receives the full plugin/function catalog
- when useful it can choose `run_plugin` and `dm` will execute that plugin/function
- plugin execution is shown in output with selected plugin name and args

Tools tool-use:
- the agent also receives the tools catalog (`search`, `rename`, `note`, `recent`, `backup`, `clean`, `system`)
- when useful it can choose `run_tool` and `dm` will execute that tool flow
- for `search`, the agent can pass non-interactive params (`base`, `ext`, `name`, `sort`, `limit`)
- for `rename`, the agent can pass params (`base`, `from`, `to`, `name`, `case_sensitive`); missing required values are asked interactively
- for `recent`, the agent can pass non-interactive params (`base`, `limit`)
- for `clean`, the agent can pass non-interactive params (`base`, `apply`)
- for paged results (`search`/`recent`), `dm ask` prompts automatically to continue with next page
- for `rename`, file changes are applied only after explicit confirmation

Example:
```bash
dm ask "devo cercare file pdf nella cartella Downloads"
dm ask "mostrami i 30 file piu recenti in Desktop"
dm ask --confirm-tools "pulisci le cartelle vuote in Downloads"
```

Safe execution example:
```bash
dm ask --confirm-tools "riavvia backend dev"
```

User config is loaded outside the repository from:
- `DM_AGENT_CONFIG` (optional override)
- default: `dm.agent.json` next to `dm.exe`

Example:
```json
{
  "ollama": {
    "base_url": "http://127.0.0.1:11434",
    "model": "deepseek-coder-v2:latest"
  },
  "openai": {
    "api_key": "sk-...",
    "model": "gpt-4o-mini"
  }
}
```

You can also provide the OpenAI key with environment variable:
- `OPENAI_API_KEY`

## Doctor
Run diagnostics for agent setup and runtime dependencies:
```bash
dm doctor
dm doctor --json
```
Checks include:
- agent config file loading (`dm.agent.json`)
- Ollama reachability and selected model
- OpenAI key presence/config
- plugin/function discovery
- common user paths used by tools

Interactive target:
```bash
dm <name>
```
`<name>` can be a `jump` alias or a `project` name.

## Project Actions
Project actions are defined under `projects.<name>.commands`.

Example:
```bash
dm app test
```

## Tools
Interactive menu for file search, rename, quick notes, recent files, folder backup, clean empty folders, and system snapshot:
```bash
dm tools
dm -t
```
The interactive menus and some command outputs use ANSI colors when supported by the terminal.
Set `NO_COLOR=1` to disable colors.

For tools that ask `Base path`, the default is the current working directory.

Run a specific tool directly:
```bash
dm tools search
dm tools s
dm -t search
dm -t s
dm tools system
dm tools sys
dm tools htop
```

Architecture note:
- Tools are built into `dm` (Go code under `tools/`), so they run without external tool files next to `dm.exe`.
- Plugins are external runtime extensions loaded from `plugins/` next to the executable.

## Help
Help output is generated by Cobra (command-aware usage and flags).

Global:
```bash
dm help
dm help tools
dm help plugins
```

Shell completion (PowerShell):
```powershell
dm completion powershell > $HOME\Documents\PowerShell\dm-completion.ps1
```
Then load it from your profile:
```powershell
. $HOME\Documents\PowerShell\dm-completion.ps1
```

Generate completion for other shells:
```bash
dm completion bash
dm completion zsh
dm completion fish
```

One-step install:
```powershell
dm completion install
dm completion install --shell bash
dm completion install --shell zsh
dm completion install --shell fish
```

## Plugins
Place scripts in `plugins/`:
- Windows: `.ps1`, `.cmd`, `.bat`, `.exe`
- Linux/mac: `.sh` (or executable files)
Recommended layout:
- `plugins/variables.ps1`
- `plugins/functions/*.ps1`

Run:
```bash
dm -p
dm plugins list
dm plugins list --functions
dm plugins info <name>
dm plugins menu
dm plugins run <name> [args...]
dm <name> [args...]
```

Interactive plugin menu:
- `dm -p` (or `dm plugins`) opens a 2-level menu:
  - select plugin file by number/letter
  - then select function by number/letter
  - `h <n|letter>` shows function help
  - `x` exits (from function list it goes back; from file list it closes the menu)

For cross-shell use, provide both plugin variants when needed:
- `plugins/<name>.ps1` for PowerShell
- `plugins/<name>.sh` for Bash/sh

PowerShell profile function bridge:
- `dm <function_name>` can invoke functions declared in PowerShell source files under `plugins/` (recursive: `.ps1`, `.psm1`, `.txt`).
- `dm help <function_name>` shows detailed info when the function is discovered as plugin bridge entry.
- CI runs `PSScriptAnalyzer` in non-blocking mode and validates help blocks for all discovered functions.
- Git function quick reference: `docs/git-cheatsheet.md` (generated from `plugins/functions/git.ps1`).

Toolkit generator (`dm-toolkit-gen`) for external use:
- Build:
```powershell
go build -o dist/dm-toolkit-gen.exe ./cmd/dm-toolkit-gen
```
- Optional placement next to runtime plugins:
```powershell
Copy-Item dist/dm-toolkit-gen.exe plugins/dm-toolkit-gen.exe -Force
```
- Commands:
```powershell
.\plugins\dm-toolkit-gen.exe init --name MSWord --prefix word --category office
.\plugins\dm-toolkit-gen.exe add --file plugins/functions/office/MSWord_Toolkit.ps1 --prefix word --func export_pdf --param InputPath --param OutputPath --confirm --require-helper _confirm_action --require-var DM_WORD_TEMPLATE=normal.dotm
.\plugins\dm-toolkit-gen.exe validate
```
- Behavior:
  - auto-detects repo root from current directory or executable location (use `--repo` to override)
  - can ensure shared helpers in `plugins/utils.ps1`
  - can ensure shared variables in `plugins/variables.ps1` (managed region `dm-toolkit-gen:variables`)
- Quick reference: `docs/dm-toolkit-gen-cheatsheet.md`

## Project Layout
```
.
|-- .github/workflows/ci.yml
|-- main.go
|-- plugins
|   |-- variables.ps1
|   `-- functions
|-- tools
|-- scripts
`-- internal
    |-- app
    |-- config
    |-- plugins
    |-- platform
    |-- runner
    |-- systeminfo
    `-- ui
```

## Changelog
- Unreleased: Start incremental migration to Cobra for CLI parsing while keeping legacy behavior and adding completion install for powershell/bash/zsh/fish.
- Unreleased: Add split config support via `include` and package refactor into `internal/`.
- Unreleased: Add profiles, plugins, cache, validation, and add/list commands.
- Unreleased: Consolidate configuration around `dm.json` + plugins/tools.
- v0.1.0: Initial public version.

Recent updates (2026-02-18):
- `02c8c22` Refactor app modules and optimize tool paging:
  - split `internal/app` into focused files (`ask`, `plugin menu`, `profile ops`)
  - add in-process caches for plugin catalog/info and paged `search`/`recent` flows
  - add benchmark suites for `tools` and `internal/plugins`
- `f2d847a` Harden cache invalidation and paging tests:
  - add mtime-aware invalidation for plugin cache entries
  - add tests for cache invalidation and paging cache behavior (TTL, eviction, loader error)
- `274ec5f` Add real-path benchmarks for tools:
  - add `DM_BENCH_BASE` benchmarks for real filesystem workloads
  - document real-path benchmark commands in README

## Performance Benchmarks
Local benchmark snapshot (Windows, Ryzen 9 5900X):

Run:
```bash
go test ./tools -run ^$ -bench Benchmark -benchmem
go test ./internal/plugins -run ^$ -bench Benchmark -benchmem
```

Real-path benchmarks (optional):
```powershell
$env:DM_BENCH_BASE="E:\path\to\folder"
# optional filters for search benchmark:
$env:DM_BENCH_NAME="report"
$env:DM_BENCH_EXT="pdf"
go test ./tools -run ^$ -bench "Benchmark(SearchFindRealPath|RecentCollectSortedRealPath)$" -benchmem
```

Highlights:
- `search` full scan (`BenchmarkSearchFind`): ~42.6 ms/op
- `search` paging cache hit (`BenchmarkSearchPagingCacheHit`): ~5.6 us/op
- `recent` full collect+sort (`BenchmarkRecentCollectSorted`): ~42.9 ms/op
- `recent` paging cache hit (`BenchmarkRecentPagingCacheHit`): ~45.8 us/op
- `plugins` list cold (`BenchmarkListEntriesWithFunctionsCold`): ~2.83 ms/op
- `plugins` list warm (`BenchmarkListEntriesWithFunctionsWarm`): ~3.18 us/op
- `plugins` info cold (`BenchmarkGetInfoFunctionCold`): ~2.64 ms/op
- `plugins` info warm (`BenchmarkGetInfoFunctionWarm`): ~0.305 us/op

Note:
- Cold benchmarks measure filesystem scanning and parsing cost.
- Warm benchmarks measure in-process cache reuse in the same `dm` process.
