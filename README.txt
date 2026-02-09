dm CLI - Quick Start
====================

This package contains:
- dm.exe (or dm on non-Windows)
- install.ps1
- README.md
- README.txt
- packs/
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
General help:
  dm help

Packs:
  dm pack list
  dm pack info <name>
  dm pack use <name>
  dm pack current

Tools menu:
  dm tools
  dm tools system

Search:
  dm find <query>
  dm -p <pack> find <query>

Run aliases and project actions:
  dm run <alias>
  dm <project> <action>

Plugins:
  dm plugin list
  dm plugin run <name>

Notes
-----
- By default, packs are loaded from: packs/<name>/pack.json
- dm.json is optional (custom includes/profiles)
- Set NO_COLOR=1 to disable ANSI colors
