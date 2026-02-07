# tellme

Small personal CLI to jump to folders, run project commands, and search a knowledge base.

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
go build -o tellme.exe .
```

## Configuration
`tellme.json` is loaded from the executable directory.

### Packs (recommended)
Each pack is a folder that contains everything for a domain/project:
```
packs/<name>/pack.json
packs/<name>/knowledge/
```

Example `tellme.json`:
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
tellme --profile work list jumps
```

### Cache
`tellme` writes a cache file in the executable directory:
- `.tellme.cache.json` (default)
- `.tellme.cache.<profile>.json` (profile)

Disable with:
```bash
tellme --no-cache list jumps
```

## Usage
```bash
tellme help
tellme aliases
tellme --pack docker list jumps
tellme -p docker list jumps
tellme list jumps
tellme add jump <name> <path>
tellme --pack docker add jump <name> <path>
tellme pack new <name>
tellme pack list
tellme pack info <name>
tellme pack use <name>
tellme pack current
tellme pack unset
tellme validate
tellme plugin list
tellme plugin run <name> [args...]
tellme run <alias>
tellme find <query>
tellme <project> <action>
tellme <name>
```

Notes:
- If `--pack` is not provided, `tellme add` writes to `packs/core/pack.json`.
- You can set a default pack with `tellme pack use <name>`.

Interactive target:
```bash
tellme <name>
```
`<name>` can be a `jump` alias or a `project` name.

## Project Actions
Project actions are defined under `projects.<name>.commands`.

Example:
```bash
tellme app test
```

## Search
Searches all `.md` files under `search.knowledge`:
```bash
tellme find golang
```
If `rg` (ripgrep) is installed, it is used automatically for faster search.
If you use packs, pass `--pack <name>` so search uses that pack knowledge folder.

## Plugins
Place scripts in `plugins/`:
- Windows: `.ps1`, `.cmd`, `.bat`, `.exe`
- Linux/mac: `.sh` (or executable files)

Run:
```bash
tellme plugin list
tellme plugin run <name> [args...]
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
