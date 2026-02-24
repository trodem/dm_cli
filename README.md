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
dm completion
dm ps_profile
dm cp profile
dm -o ps_profile
dm -o profile
```

Group shortcuts:
- `-a`, `--add-alias` -> `alias add`
- `-t`, `--tools` -> `tools`
- `-p`, `--plugins` -> `plugins`
- `-o`, `--open` -> `open`
- `-r`, `--run-alias` -> `alias run`

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
- `--response-mode raw-first|llm-first` (default `raw-first`: show tool/plugin output; LLM recovery text appears only on errors)
- `-a`, `--as-powershell` (run prompt as direct PowerShell command, bypassing AI planner)
- `-f`, `--file <path>` (attach file as context, repeatable)
- `-s`, `--scope <prefix>` (limit catalog to a toolkit domain, e.g. `stibs`, `m365`, `docker`)
- `--json` (structured output, one-shot mode only)
- `--debug` (enable debug logging to stderr)

Examples:
```bash
dm ask "spiegami questo errore"
dm ask --provider auto "cerca i file pdf in Downloads"
dm ask -a "Get-Location"
dm ask --json "trova file recenti in Downloads"
dm ask -f config.json "analizza questo file"
dm ask -f main.go -f go.mod "confronta questi file"
dm ask --scope stibs "stato del database"
```

Interactive `dm ask` commands:
- `/cd <path>` (or `cd <path>`) to change current working directory
- `/pwd` (or `pwd`) to show current working directory
- `/help` (or `help`)
- `/status` (or `status`)
- `/reset` (or `reset`) to clear session context
- `/clear` (or `clear`, `cls`)
- `/exit` (or `exit`, `quit`)

Note: commit-message prompts automatically switch to `llm-first` so the final commit subject is always shown.

## Aliases
Store simple command aliases in `dm.aliases.json` (and automatically sync them to `$PROFILE`):

```bash
dm alias add d "cd C:\Users\Demtro\Downloads"
dm alias add ll "Get-ChildItem -Force"
dm alias ls
dm alias run d
dm alias run ll
dm alias sync
dm alias rm d
```

`dm alias run` executes the stored command using the same PowerShell path used by `dm ask -a`.
`dm alias sync` forces a full rewrite of the managed alias block in `$PROFILE`.

Config path priority:
1. `DM_AGENT_CONFIG`
2. `dm.agent.json` next to executable
3. `~/.config/dm/agent.json`

OpenAI key can also be set with `OPENAI_API_KEY`.

### Self-evolving agent
When the agent receives a request that no existing plugin or tool can handle, it can propose creating a new PowerShell function on the fly. The flow:
1. Agent detects no matching plugin exists and proposes `create_function`.
2. User confirms whether to proceed.
3. A specialized builder agent generates the function code following toolkit conventions.
4. The generated code is shown for approval before writing.
5. The function is added to an existing toolkit or a new one is created.
6. The plugin catalog is refreshed and the new function is executed.

This feature is not available in `--json` mode.

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
dm tools clean
dm tools system
dm tools read
dm tools grep
dm tools diff
```

Tool aliases:
- `search/s`
- `rename/r`
- `recent/rec`
- `clean/c`
- `system/sys/htop`
- `read/f/cat/view`
- `grep/g/find/rg`
- `diff/d`

## Plugins
Standalone toolkit layout:
- `plugins/<Name>_Toolkit.ps1` (top-level toolkits)
- `plugins/<Domain>/<Name>_Toolkit.ps1` (domain-scoped toolkits)

### Toolkits

| Toolkit | Prefix | Area |
|---|---|---|
| System | `sys_*` | OS, processes, clipboard, network diagnostics, WiFi |
| FileSystem | `fs_path_*` | Windows special paths, navigation |
| Docker | `dc_*` | Docker Compose orchestration |
| Browser | `browser_*` | Browser launch, close, localhost |
| Excel | `xls_*` | Excel operations |
| Text | `txt_*` | Encoding, hashing, UUID, JSON, URL |
| Network | `net_*` | HTTP requests, download, SSL certs, speed test |
| Winget | `pkg_*` | Software install, update, search via winget |
| Archive | `arc_*` | Zip/tar.gz create, extract, list |
| Scheduler | `sched_*` | Windows Task Scheduler management |
| Toolkit Manager | `tk_*` | List, create, scaffold, validate toolkits |
| Help | `help_*` | Runtime introspection, intent search, quickref, env vars, prerequisites |
| Start Dev | `start_*` | Launch development tools |
| M365 Auth | `m365_*` | Microsoft 365 authentication and session |
| SharePoint | `spo_*` | SharePoint Online lists, items, files, sites |
| Power Automate | `flow_*` | Power Automate flow management |
| Power Apps | `pa_*` | Power Apps environment and app management |
| KVP Star Site | `kvpstar_*` | KVP Star SharePoint site operations |
| Star IBS Apps | `star_ibs_*` | Star IBS SharePoint applications |
| STIBS App | `stibs_app_*` | STIBS application inspection and monitoring |
| STIBS DB | `stibs_db_*` | STIBS MariaDB database analytics |
| STIBS Docker | `stibs_docker_*` | STIBS Docker stack management |

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

## Development

Run before pushing:
```powershell
.\scripts\pre-push.ps1
```

Or rely on the git pre-push hook (installed via `.git/hooks/pre-push`), which runs:
1. `go test ./...`
2. `golangci-lint run`

Skip with `git push --no-verify`.

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
