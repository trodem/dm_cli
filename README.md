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
- `--provider auto|ollama|openai` (default `auto`)
- `--model <name>` (override model for selected provider)
- `--base-url <url>` (override provider base URL)
- `--confirm-tools` (ask confirmation before running plugin/function selected by the agent)

Interactive mode:
- `dm ask` opens a persistent ask prompt
- It exits only with explicit commands: `/exit`, `exit`, `quit`

Response header:
- each answer prints `[provider | model]` before the text

Plugin tool-use:
- the agent receives the full plugin/function catalog
- when useful it can choose `run_plugin` and `dm` will execute that plugin/function
- plugin execution is shown in output with selected plugin name and args

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
