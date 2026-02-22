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
  - keep command flows split by concern (for example `ask.go`, `plugin_menu.go`, `profile_ops.go`)
  - `ask.go` — main agent loop and action handlers (run_plugin, run_tool, create_function, answer)
  - `ask_cache.go` — decision cache (deduplicates identical agent requests)
  - `ask_catalog.go` — builds plugin and tool catalogs for the agent prompt
  - `ask_helpers.go` — argument formatting, display helpers, mandatory-param pre-check, token budget trimming, file context builder
  - `ask_stream.go` — streaming answer buffering with markdown post-processing
  - `ask_risk.go` — risk assessment, toolkit safety parsing, confirmation prompts
  - `ask_output.go` — TTY and JSON output renderers for agent responses (humanized step descriptions, risk display)
  - `ask_toolkit_writer.go` — file writing helpers for the toolkit builder (append function, update index, create new toolkit)
  - `signal.go` — Ctrl+C signal handler, temp file cleanup on interrupt
- AI agent logic: `internal/agent/`
  - `agent.go` — planner agent (decides action: answer, run_plugin, run_tool, create_function)
  - `toolkit_builder.go` — builder agent that generates PowerShell functions following toolkit conventions
- Plugin engine: `internal/plugins/`
  - `plugins.go` — types, public API (List, GetInfo, Run, RunWithOutput, RunWithOutputAgent), plugin discovery
  - `plugins_parse.go` — PowerShell function/help/param parsing, toolkit safety metadata
  - `plugins_exec.go` — execution logic (PowerShell function bridge, script runner, platform dispatch)
  - `cache.go` — entry list and info caching with file-stamp invalidation
- Config files:
  - `dm.json` (optional root includes)
  - `config/*.json` (optional included fragments)
- Plugin files:
  - `plugins/*.ps1` (standalone toolkit files with public command functions)
  - Domain-scoped toolkits go in subfolders (e.g. `plugins/STIBS/`, `plugins/M365/`)

## Code Style
- Keep ASCII-only in source files unless necessary.
- Keep functions small and single-purpose.
- Avoid duplication; reuse helpers in `internal/`.
- Use `internal/` packages for new functionality.

## Config Rules
- Use `include` in `dm.json` for scale.
- Split by domain using included config fragments (for example `config/work.json`, `config/home.json`).
- Keep paths either absolute or relative to the executable directory.

## Testing
- If you add parsing logic, add unit tests in the same package.
- If you add or change Cobra commands/flags, update tests in `internal/app/`.
- For performance-sensitive changes, add or update benchmarks where applicable:
  - `go test ./tools -run ^$ -bench Benchmark -benchmem`
  - `go test ./internal/plugins -run ^$ -bench Benchmark -benchmem`
  - optional real-path benchmark input via `DM_BENCH_BASE` (and optional `DM_BENCH_NAME`, `DM_BENCH_EXT`)

## CLI Conventions
- Use Cobra native help/usage output; do not add custom global help printers.
- Keep command docs in Cobra metadata (`Use`, `Short`, `Long`, `Example`).
- Keep group shortcuts aligned across legacy/Cobra parsing:
  - `-t` / `--tools` -> `tools`
  - `-p` / `--plugins` -> `plugins`
  - `-o` / `--open` -> `open`
- Tools should be invocable both as:
  - `dm tools <name>`
  - `dm -t <name>`
- Keep tool aliases consistent (`search/s`, `rename/r`, `note/n`, `recent/rec`, `backup/b`, `clean/c`).
- For tools that request `Base path`, default to current working directory.
- Plugin menu UX:
  - `dm -p` / `dm plugins` should open the interactive plugin menu.
  - Keep plugin navigation two-level:
    1. plugin file selection
    2. function selection inside the chosen file
  - Support both number and letter shortcuts in plugin menu selections.
  - Keep `h <n|letter>` for function help and `x` for exit/back.
  - After function execution/help in menu, pause with an explicit "press enter to continue" prompt.
- Keep legacy plugin commands working:
  - `dm plugins list`
  - `dm plugins list --functions`
  - `dm plugins info <name>`
  - `dm plugins run <name> [args...]`

## PowerShell Plugin Conventions
- Store public PowerShell plugin commands in `plugins/*.ps1`.
- Each toolkit file must be fully standalone — all helpers, guards, and config defined internally.
- Use `Set-StrictMode -Version Latest` and `$ErrorActionPreference = "Stop"` in plugin `.ps1` files.
- Public plugin function names must be explicit and domain-prefixed (for example `sys_*`, `git_*`, `stibs_db_*`).
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
- Validate plugin help blocks before finalizing changes:
  - `go run ./scripts/check_plugin_help.go`

## Toolkit Construction Rules
Toolkits are domain-specific PowerShell `.ps1` files under `plugins/`.
Each toolkit groups related functions behind a shared prefix.

### File Layout
Every toolkit file MUST follow this top-to-bottom structure:

1. **Header banner** (bordered with `=` lines):
   - Toolkit name with `(standalone)` tag and one-line purpose (e.g. `# SYSTEM TOOLKIT – Local system & network operations (standalone)`).
   - Explicit `Safety:` line describing the risk profile (e.g. `# Safety: Read-only — no destructive operations.` or `# Safety: Non-destructive defaults. Kill/restart require -Force or confirmation.`).
   - Entry point prefix (e.g. `# Entry point: sys_*`).
   - Exhaustive `FUNCTIONS` index listing every public function in the file.
2. **Strict mode block** (immediately after header):
   ```
   Set-StrictMode -Version Latest
   $ErrorActionPreference = "Stop"
   ```
3. **Internal helpers section** (optional, prefixed with `_`).
4. **Public functions**, grouped by logical sections separated with `# ------` or `# ======` dividers and a section title.

### Naming
- **File name**: `<Name>_Toolkit.ps1` (PascalCase name, underscore, `Toolkit`).
  Use a numeric prefix for ordering when needed (e.g. `3_System_Toolkit.ps1`).
  Domain-scoped toolkits go in subfolders (e.g. `plugins/STIBS/`, `plugins/M365/`).
- **Public functions**: `<prefix>_<action>` all lowercase (e.g. `sys_uptime`, `git_status`, `browser_open`).
  The prefix must be short, unique across all toolkits, and domain-descriptive.
- **Private helpers**: start with `_` (e.g. `_assert_git_repo`, `_dc`, `_wifi_profiles`).
  Private helpers are invisible to the plugin menu and the AI agent.
- **Module-level variables** in toolkit files must use `$script:` scope when they are file-local constants.

### Help Blocks
Every function (public AND private) MUST have a comment-based help block directly above it:
```
<#
.SYNOPSIS
One-line summary of what the function does.
.DESCRIPTION
More detail if the synopsis alone is not enough.
.PARAMETER ParamName
Description of the parameter.
.EXAMPLE
prefix_action -ParamName value
#>
```
- `SYNOPSIS` and at least one `EXAMPLE` are mandatory.
- `DESCRIPTION` is required when the synopsis is not self-explanatory.
- `PARAMETER` is required for every declared parameter.
- The `.EXAMPLE` must show actual invocation syntax (e.g. `dm prefix_action -Param value`).
- Do NOT use placeholder help like "Invoke X" / "Helper/command function for X" in new code.

### Parameters And Return Values
- Mark mandatory parameters with `[Parameter(Mandatory = $true)]` (always use the explicit form with spaces).
- Use `[Parameter(Position = N)]` for positional convenience parameters.
- Use `[ValidateSet(...)]` when the parameter domain is closed (e.g. service names).
- **Preferred parameter declaration style** — use the multi-line form with each attribute on its own line:
  ```
  param(
      [Parameter(Mandatory = $true)]
      [string]$Host,

      [ValidateSet("json", "text")]
      [string]$Format = "text",

      [switch]$Force
  )
  ```
  The single-line form `param([Parameter(Mandatory = $true)][string]$Name)` is acceptable for private helpers with a single parameter, but public functions with multiple parameters should always use multi-line for readability.
- Return structured `[pscustomobject]@{ ... }` objects instead of raw strings when the output has multiple fields.
  This allows pipeline processing, AI consumption, and consistent formatting.

### PowerShell Coding Style
- Use `Test-Path -LiteralPath` instead of `Test-Path -Path` to prevent wildcard expansion on paths containing brackets or special characters.
- Place `$null` on the left side of equality comparisons (`$null -eq $x`, not `$x -eq $null`) to avoid unexpected collection filtering.
- Never use `Write-Host` for function output — return `[pscustomobject]` or plain values so output is pipeline-friendly and consumable by AI agents.
- Use `throw` for error conditions, not `Write-Host` with a return. Thrown errors surface clearly in agent logs.
- Use `[Environment]::GetEnvironmentVariable($Name)` (not `$env:`) inside `_env_or_default` helpers for reliable null detection.
- Prefer `Start-Process -FilePath` over direct invocation (`& exe`) when launching external GUI applications.

### Guard Helpers
- Call `_assert_command_available -Name <tool>` before invoking external CLI tools (docker, git, netsh, m365, etc.).
- Call `_assert_path_exists -Path <path>` before reading/writing paths that may not exist.
- Define these helpers as private `_` functions inside the toolkit itself (each toolkit carries its own copy).

### Safety Defaults
- Default to non-destructive, read-only behavior.
- Destructive or state-changing operations must require an explicit `-Force` switch or an interactive confirmation via `_confirm_action`.
- Never perform irreversible side effects without either confirmation or `-Force`.

### Standalone Requirement
Every toolkit MUST be fully self-contained with **zero** cross-file dependencies:
- A toolkit file must **not** call functions defined in any other `.ps1` file — neither other toolkits nor shared files like `utils.ps1` or `variables.ps1`.
- Each toolkit must define its own private guard helpers (`_assert_command_available`, `_assert_path_exists`, `_confirm_action`) and config loaders internally.
- Configuration values (paths, credentials, URLs) must be loaded via a local `_env_or_default` helper so they remain overridable through environment variables.
- This guarantees that each `.ps1` file can be loaded, tested, and reasoned about in complete isolation — both by humans and AI agents — with no hidden dependencies.

### Toolkit Anti-Patterns
Do NOT create toolkits that:
- Contain personal shell aliases or trivial wrappers around built-in cmdlets (e.g. `function c { Clear-Host }`).
- Mix unrelated domains under one file without a coherent prefix.
- Use functions without a domain prefix — every public function must follow the `<prefix>_<action>` convention.
- Rely on bare `$variable` references defined outside the file.
- Use `Format-Table`, `Format-List`, or `Write-Host` for structured output — return `[pscustomobject]` instead.
- Contain Italian (or other non-English) help blocks — all documentation must be in English.

Every function in a toolkit must provide meaningful value to an AI agent or automated workflow.
If a function is only useful as a personal keyboard shortcut, it does not belong in a toolkit.

### Prefix Registry
Active prefixes — do not reuse these when creating new toolkits:

| Prefix | Toolkit | Path |
|---|---|---|
| `sys_*` | System Toolkit | `plugins/3_System_Toolkit.ps1` |
| `git_*` | Git Toolkit | `plugins/4_Git_Toolkit.ps1` |
| `fs_path_*` | FileSystem Path Toolkit | `plugins/2_FileSystem_Toolkit.ps1` |
| `browser_*` | Browser Toolkit | `plugins/Browser_Toolkit.ps1` |
| `start_*` | Start Dev Toolkit | `plugins/Start_Dev_Toolkit.ps1` |
| `help_*` | Help Toolkit | `plugins/Help_Toolkit.ps1` |
| `stibs_db_*` | STIBS DB Toolkit | `plugins/STIBS/STIBS_DB_Toolkit.ps1` |
| `dc_*` | Docker Toolkit | `plugins/Docker_Toolkit.ps1` |
| `stibs_docker_*` | STIBS Docker Toolkit | `plugins/STIBS/STIBS_Docker_Toolkit.ps1` |
| `kvpstar_*` | KVP Star Site Toolkit | `plugins/M365/KVP_Star_Site_Toolkit.ps1` |
| `star_ibs_*` | Star IBS Applications Toolkit | `plugins/M365/Star_IBS_Applications_Toolkit.ps1` |
| `txt_*` | Text Toolkit | `plugins/Text_Toolkit.ps1` |
| `stibs_app_*` | STIBS App Toolkit | `plugins/STIBS/STIBS_App_Toolkit.ps1` |
| `xls_*` | Excel Toolkit | `plugins/Excel_Toolkit.ps1` |

Update this table when adding or removing toolkits.
Toolkits auto-generated by the `dm ask` builder agent should be reviewed and their prefixes added here after acceptance.

### Checklist For New Toolkits
1. Choose a unique prefix not already used by any existing toolkit (see Prefix Registry above).
2. Create the file manually following the header banner template.
3. Add `Set-StrictMode` + `$ErrorActionPreference` immediately after the header.
4. Verify the toolkit is truly standalone — no calls to functions in any other `.ps1` file.
5. Include private guard helpers (`_assert_command_available`, `_assert_path_exists`) and config loaders inside the toolkit.
   - Copy helpers from an existing toolkit — then **review every parameter name** to ensure it matches (e.g. `_assert_path_exists` uses `$Path`, NOT `$Name`).
6. Write full help blocks on every function.
7. Return `[pscustomobject]` for multi-field outputs.
8. Gate destructive actions behind `-Force` or `_confirm_action`.
9. Update the `FUNCTIONS` index in the header when adding/removing functions.
10. Run `go run ./scripts/check_plugin_help.go` to verify help block parsing.
11. **Test every public function** before delivering:
    - Dot-source the toolkit and invoke each function with realistic arguments.
    - Example: `powershell -NoProfile -Command ". '.\plugins\My_Toolkit.ps1'; my_func -Param value"`
    - Verify the output is correct and no errors are thrown.
    - Do NOT skip this step — an untested function is a broken function.

### Common PowerShell Pitfalls
These mistakes have caused real bugs. Do not repeat them.

1. **StrictMode property access on XML nodes** — `$node.someAttribute` throws `PropertyNotFoundException` if the attribute does not exist. Always use `$node.GetAttribute("name")` instead, which safely returns an empty string.
2. **Copy-paste helper parameter names** — When copying `_assert_command_available` (which uses `$Name`) to create `_assert_path_exists` (which uses `$Path`), verify you renamed the parameter. Mismatched parameter names cause confusing errors at call sites.
3. **Multi-line SQL in `docker exec`** — PowerShell on Windows breaks multi-line strings passed to `docker exec ... sh -c "..."`. Always collapse SQL to a single line with `($Sql -replace '\r?\n', ' ').Trim()` before embedding.
4. **Variable expansion in bareword arguments** — `docker exec ... mysql -u$($cfg.User)` silently fails to expand. Use explicit argument variables: `$userArg = "--user=$($cfg.User)"`.
5. **`$null` comparisons** — Always put `$null` on the left: `$null -eq $x`, not `$x -eq $null`. The latter silently filters arrays instead of comparing.
6. **Switch parameters in help blocks** — Declare `[switch]$Force`, not `[bool]$Force`. Switch params are passed as `-Force` without a value.

## Agent Output Styling (`dm ask`)
- Show provider and model **once** at session start (`dm ask | openai/gpt-4o`), not per step.
- Move `Reason` and step counters to `--debug` only; users should not see internal planner reasoning.
- Describe steps in natural language: `> Running stibs_db_tables`, not `Plan step 1/4: plugin stibs_db_tables`.
- Show risk level only when **not** low. Use `Warn` (yellow) for MEDIUM, `Error` (red) for HIGH.
- Use concise confirm prompts: `Proceed? [Y/n]` for normal risk, `! Confirm? [y/N]` for high risk.
- Indent agent UI lines with two spaces for visual separation from plugin output.
- Write spinner frames to **stderr** (not stdout) to keep stdout pipe-clean.
- Suppress spinner when stdout/stderr are not terminals (piped mode).
- Final answer prints with a blank line before it for visual breathing room.
- Apply markdown rendering (`ui.RenderMarkdown`) to all answer paths: direct, streamed, partial, error-with-answer, canceled, loop-detected.
- Streamed answers buffer to `answerBuf` during streaming, then render markdown at completion via `Finish()`.
- `--file` / `-f` flag attaches file contents as context to the LLM prompt (repeatable, max 32KB per file).

## Menu And Output Styling
- Use shared color helpers from `internal/ui/pretty.go` (for example `Accent`, `Warn`, `Muted`, `Prompt`) for interactive menus.
- Use `internal/ui/spinner.go` (`ui.NewSpinner`, `Start`, `Stop`) for progress indication during long operations.
- Use `internal/ui/markdown.go` (`ui.RenderMarkdown`) to strip markdown syntax from LLM answers; always strips markers (`**`, `##`, `` ` ``, ` ``` `), adds ANSI styling only when color is supported.
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
- CI runs automatically on push/PR to `main` and `develop` via `.github/workflows/ci.yml` (vet, build, test with coverage and race detection).

## Commits
- Use short, imperative messages (e.g., "Refactor config loader").
- Avoid bundling unrelated changes in a single commit.
- Do not amend or rewrite history unless explicitly asked.

## Output
- CLI output should remain human-friendly and minimal.
- Splash should show `Version` and executable build time.

## Documentation
- Update `README.md` when behavior, configuration, or structure changes.
- Specifically update it when:
  - CLI commands or flags change.
  - Config schema changes (new keys, removed keys, behavior changes).
  - Default paths or output formats change.
