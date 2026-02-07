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

## Configuration
`dm.json` is loaded from the executable directory.

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

Example `dm.json`:
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
dm pack list
dm pack info <name>
dm pack use <name>
dm pack current
dm pack unset
dm validate
dm plugin list
dm plugin run <name> [args...]
dm run <alias>
dm find <query>
dm <project> <action>
dm <name>
```

Notes:
- Use `-p <pack>` or set a default pack with `dm pack use <name>`.

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

## Plugins
Place scripts in `plugins/`:
- Windows: `.ps1`, `.cmd`, `.bat`, `.exe`
- Linux/mac: `.sh` (or executable files)

Run:
```bash
dm plugin list
dm plugin run <name> [args...]
```

## Project Layout
```
.
|-- .github/workflows/ci.yml
|-- main.go
|-- packs
`-- internal
    |-- app
    |-- config
    |-- plugins
    |-- platform
    |-- runner
    |-- search
    |-- store
    `-- ui
```

## Changelog
- Unreleased: Add split config support via `include` and package refactor into `internal/`.
- Unreleased: Add profiles, plugins, cache, validation, and add/list commands.
- Unreleased: Add packs with per-pack knowledge.
- v0.1.0: Initial public version.
