# AGENTS

Repository guidelines for automated agents.

## Scope
- Target: Go CLI tool `dm`.
- Keep changes focused and minimal.
- Prefer incremental, reversible edits.

## Structure
- Entry point: `main.go`
- Core logic: `internal/`
- Tools: `tools/` (interactive utilities)
- CLI command wiring: `internal/app/` (Cobra-based)
- Config files:
  - `dm.json` (optional root includes)
  - `packs/*/pack.json`
  - `packs/*/knowledge/`
- Plugin files:
  - `plugins/variables.ps1` (shared variables + private helper functions)
  - `plugins/functions/*.ps1` (public command functions)

## Code Style
- Keep ASCII-only in source files unless necessary.
- Keep functions small and single-purpose.
- Avoid duplication; reuse helpers in `internal/`.
- Use `internal/` packages for new functionality.

## Config Rules
- Use `include` in `dm.json` for scale.
- Split by domain using packs:
  - `packs/<name>/pack.json`
  - `packs/<name>/knowledge/`
- Keep paths either absolute or relative to the executable directory.

## Testing
- If you add parsing logic, add unit tests in the same package.
- If you add or change Cobra commands/flags, update tests in `internal/app/`.

## CLI Conventions
- Use Cobra native help/usage output; do not add custom global help printers.
- Keep command docs in Cobra metadata (`Use`, `Short`, `Long`, `Example`).
- Keep group shortcuts aligned across legacy/Cobra parsing:
  - `-t` / `--tools` -> `tools`
  - `-k` / `--packs` -> `pack`
  - `-g` / `--plugins` -> `plugin`
- Tools should be invocable both as:
  - `dm tools <name>`
  - `dm -t <name>`
- Keep tool aliases consistent (`search/s`, `rename/r`, `note/n`, `recent/rec`, `backup/b`, `clean/c`).
- For tools that request `Base path`, default to current working directory.
- Plugin menu UX:
  - `dm -g` / `dm plugin` should open the interactive plugin menu.
  - Keep plugin navigation two-level:
    1. plugin file selection
    2. function selection inside the chosen file
  - Support both number and letter shortcuts in plugin menu selections.
  - Keep `h <n|letter>` for function help, `b` for back, `q` for quit.
  - After function execution/help in menu, pause with an explicit "press enter to continue" prompt.
- Keep legacy plugin commands working:
  - `dm plugin list`
  - `dm plugin list --functions`
  - `dm plugin info <name>`
  - `dm plugin run <name> [args...]`

## PowerShell Plugin Conventions
- Store public PowerShell plugin commands in `plugins/functions/*.ps1`.
- Keep shared variables and helper utilities in `plugins/variables.ps1`.
- Use `Set-StrictMode -Version Latest` and `$ErrorActionPreference = "Stop"` in plugin `.ps1` files.
- Public plugin function names must be explicit and domain-prefixed (for example `g_*`, `stibs_*`).
- Private helper functions must start with `_` so they are not exposed as CLI commands.
- Every public function must include comment-based help block immediately above the function:
  - `SYNOPSIS`
  - `DESCRIPTION`
  - at least one `EXAMPLE`
  - add `PARAMETER` entries when parameters exist
- Prefer safety defaults for destructive actions:
  - require explicit switch/confirmation for high-risk operations
  - do not add wrappers for destructive Git commands like `reset --hard` unless explicitly requested
- Use guard helpers (for example command/path checks) before calling external tools.
- Regenerate docs when Git plugin functions change:
  - `go run ./scripts/gen_git_cheatsheet`
- Validate plugin help blocks before finalizing changes:
  - `go run ./scripts/check_plugin_help.go`

## Menu And Output Styling
- Use shared color helpers from `internal/ui/pretty.go` (for example `Accent`, `Warn`, `Muted`, `Prompt`) for interactive menus.
- Apply the same style across all interactive menus in the project (tools, target actions, plugin menu, and future menus).
- Keep prompts colorized and explicit (for example `Select option >`, `Args (optional) >`).
- Use `Prompt(...)` for user input questions, `Warn(...)` for cancellations, and `Error(...)` for invalid selections.
- Preserve readability:
  - highlight primary selectable labels
  - keep synopsis/secondary hints muted
  - avoid excessive decoration that reduces scanability
- Respect `NO_COLOR` behavior (color output must remain optional).

## Build And Lint
- Prefer `go test ./...` before changes are finalized.
- If you format code, use `gofmt` on touched files only.
- Do not introduce new dependencies without justification.
- If build commands are slow, state that you did not run them.

## Commits
- Use short, imperative messages (e.g., "Refactor config loader").
- Avoid bundling unrelated changes in a single commit.
- Do not amend or rewrite history unless explicitly asked.

## Output
- CLI output should remain human-friendly and minimal.

## Documentation
- Update `README.md` when behavior, configuration, or structure changes.
- Specifically update it when:
  - CLI commands or flags change.
  - Config schema changes (new keys, removed keys, behavior changes).
  - Default paths or output formats change.
