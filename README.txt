dm CLI - Quick Start
====================

This package contains:
- dm.exe (or dm on non-Windows)
- install.ps1
- README.md
- README.txt
- plugins/

Install (recommended on Windows)
--------------------------------
1) Open PowerShell in this folder.
2) Run:
   .\install.ps1
   # default target: %LOCALAPPDATA%\Programs\dm-cli
3) Open a new terminal.
4) Verify:
   dm --help

Manual run without installation
-------------------------------
From this folder:
  .\dm.exe --help

Useful commands
---------------
General:
  dm help
  dm doctor
  dm validate
  dm ps_profile
  dm cp profile
  dm -o ps_profile
  dm -o profile

Tools:
  dm tools
  dm tools system
  dm -t search

Plugins:
  dm -p
  dm plugins list
  dm plugins list --functions
  dm plugins menu
  dm plugins run <name>

Toolkit generator (built-in):
  dm toolkit
  dm toolkit new --name MSWord --prefix word --category office
  dm toolkit add --file plugins/functions/office/MSWord_Toolkit.ps1 --prefix word --func export_pdf --param InputPath --param OutputPath --confirm
  dm toolkit validate

Agent:
  dm ask
  dm ask "spiegami questo errore"

Build with explicit version (instead of default dev)
----------------------------------------------------
  go build -ldflags "-X cli/internal/app.Version=v0.2.0" -o dm.exe .

Scripts
-------
- scripts/release.ps1: build + package release artifacts
- scripts/check_plugin_help.go: validate plugin help blocks
- scripts/smoke_plugins.ps1: plugin smoke checks

Notes
-----
- Splash shows Version and executable build time.
- dm.json is optional (custom includes/profiles).
- Set NO_COLOR=1 to disable ANSI colors.
