DM CLI - Quick Setup (Windows)

1) Keep this folder together:
   - dm.exe
   - packs\
   - (optional) plugins\

2) Run from this folder:
   - Open PowerShell here
   - Type: .\dm.exe

3) If you want to type "dm" from anywhere (optional):
   - Add this folder to the PATH environment variable
   - Open a new terminal
   - Type: dm

How to add PATH (PowerShell, current user):
1) Open PowerShell in the dm folder.
2) Run this command:

   $dir = (Get-Location).Path
   [Environment]::SetEnvironmentVariable("Path", $env:Path + ";" + $dir, "User")
   Write-Host "Added to PATH. Open a new terminal."

3) Close and reopen the terminal, then type: dm

Basic Usage
-----------
Show help:
  dm help

List packs:
  dm pack list

Set active pack:
  dm pack use <name>

Show pack info:
  dm pack info <name>

Show pack help:
  dm pack help <name>

Tools menu:
  dm tools

Search knowledge in a pack:
  dm -p <pack> find <query>

Run a command alias:
  dm run <alias>

Run a project action:
  dm <project> <action>

Plugins
-------
List plugins:
  dm plugin list

Run a plugin:
  dm plugin run <name>

Example (open Paint):
  dm plugin run paint

Notes
-----
- dm.json is optional. If present, it is used for custom includes/profiles.
- Packs are loaded from: packs\<name>\pack.json
- Quick notes are stored in: packs\<pack>\knowledge\inbox.md

Notes:
- dm.json is optional. If present, it is used for custom includes/profiles.
- Packs are loaded from: packs\<name>\pack.json
