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
