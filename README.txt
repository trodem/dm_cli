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
General help:
  dm help

Tools menu:
  dm tools
  dm tools system

Run aliases and project actions:
  dm run <alias>
  dm <project> <action>

Plugins:
  dm plugins list
  dm plugins run <name>

Notes
-----
- dm.json is optional (custom includes/profiles)
- Set NO_COLOR=1 to disable ANSI colors
