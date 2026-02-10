# dm

Small personal CLI to jump to folders, run project commands, and search a knowledge base.

## Index
- [Features](#features)
- [Requirements](#requirements)
- [Build](#build)
- [Configuration](#configuration)
- [Usage](#usage)
- [Project Actions](#project-actions)
- [Search](#search)
- [Plugins](#plugins)
- [Tools](#tools)
- [Help](#help)
- [Project Layout](#project-layout)
- [Changelog](#changelog)

## Features
- Jump to paths from aliases
- Run global aliases
- Run project actions
- Search markdown notes
- Interactive menu for targets
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
- `README.txt`
- `README.md`
- `LICENSE` (if present)

## Configuration
`dm` works without a config file. By default it loads:
`packs/*/pack.json`

If you want custom includes or profiles, create `dm.json`.

### Packs (recommended)
Each pack is a folder that contains everything for a domain/project:
```
packs/<name>/pack.json
packs/<name>/knowledge/
```

### Pack Explained (simple)
Think of a pack as a box with 4 things:

- `jump`: shortcuts to folders
- `run`: buttons to run commands
- `projects`: projects with their commands
- `search`: where to search notes

Example:
```json
{
  "schema_version": 1,
  "description": "Git workflows and notes",
  "summary": "Git aliases and project actions",
  "tags": ["git", "vcs"],
  "examples": [
    "dm -p git find rebase",
    "dm -p git run gs"
  ],
  "jump": { "api": "projects/api" },
  "run": { "gs": "git status" },
  "projects": {
    "git-tools": {
      "path": "projects/git-tools",
      "commands": { "gcommit": "git add . && git commit" }
    }
  },
  "search": { "knowledge": "packs/git/knowledge" }
}
```

Usage:
```bash
dm api
dm run gs
dm git-tools gcommit
dm -p git find branch
```

Optional `dm.json` example:
```json
{
  "include": ["packs/*/pack.json"]
}
```

Example `packs/docker/pack.json`:
```json
{
  "jump": { "docker": "E:/tools/docker" },
  "run": { "dps": "docker ps" },
  "projects": {
    "docker": {
      "path": "E:/projects/docker",
      "commands": {
        "up": "docker compose up -d"
      }
    }
  },
  "search": { "knowledge": "packs/docker/knowledge" }
}
```

Notes:
- Paths can be absolute or relative to the executable directory.
- Forward slashes are supported on Windows (`E:/...`).

### Split Config (includes)
You can include packs (or any config files) using `include` patterns.

### Profiles
Define profile-specific includes and optional search overrides:
```json
{
  "include": ["packs/*/pack.json"],
  "profiles": {
    "work": {
      "include": ["packs/work/pack.json"],
      "search": { "knowledge": "packs/work/knowledge" }
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
dm --pack docker list jumps
dm -p docker list jumps
dm list jumps
dm add jump <name> <path>
dm --pack docker add jump <name> <path>
dm pack new <name>
dm pack new <name> --description "..."
dm pack clone <src> <dst>
dm pack edit <name> --summary "..." --tag dev --example "dm -p <name> find <query>"
dm pack list
dm pack list --verbose
dm pack info <name>
dm pack doctor <name>
dm pack doctor <name> --json
dm pack use <name>
dm pack current
dm pack unset
dm validate
dm plugin list
dm plugin list --functions
dm plugin info <name>
dm plugin menu
dm plugin run <name> [args...]
dm <plugin> [args...]
dm run <alias>
dm find <query>
dm tools
dm tools <tool>
dm -t [tool]
dm -k [cmd]
dm -k <pack> <cmd...>
dm -g [cmd]
dm <project> <action>
dm <name>
```

Notes:
- Use `-p <pack>` or set a default pack with `dm pack use <name>`.
- Group shortcuts:
  - `-t` / `--tools` => `tools`
  - `-k` / `--packs` => `pack`
  - `-g` / `--plugins` => `plugin`
- Pack profile shortcut:
  - `dm -k <pack> <cmd...>` runs `<cmd...>` with `--pack <pack>` (example: `dm -k vim run vim`).
- Fallback dispatch:
  - `dm <name>` now tries `jump/project` first, then direct plugin execution.
  - If no plugin exists, `dm` returns an error (it does not auto-run search).

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

## Search
Searches all `.md` files under `search.knowledge`:
```bash
dm find golang
```
If `rg` (ripgrep) is installed, it is used automatically for faster search.
If you use packs, pass `--pack <name>` so search uses that pack knowledge folder.

## Tools
Interactive menu for file search, rename, quick notes, recent files, pack backup, clean empty folders, and system snapshot:
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

## Help
Help output is generated by Cobra (command-aware usage and flags).

Global:
```bash
dm help
dm help tools
dm help plugin
dm pack --help
dm pack <name> --help
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
dm -g
dm plugin list
dm plugin list --functions
dm plugin info <name>
dm plugin menu
dm plugin run <name> [args...]
dm <name> [args...]
```

Interactive plugin menu:
- `dm -g` (or `dm plugin`) opens a 2-level menu:
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

## Project Layout
```
.
|-- .github/workflows/ci.yml
|-- main.go
|-- packs
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
    |-- search
    |-- systeminfo
    |-- store
    `-- ui
```

## Changelog
- Unreleased: Start incremental migration to Cobra for CLI parsing while keeping legacy behavior and adding completion install for powershell/bash/zsh/fish.
- Unreleased: Add split config support via `include` and package refactor into `internal/`.
- Unreleased: Add profiles, plugins, cache, validation, and add/list commands.
- Unreleased: Add packs with per-pack knowledge.
- v0.1.0: Initial public version.
