# dm-toolkit-gen Cheat Sheet

## 1) Build / Install

```powershell
# build + install in plugins/
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\setup_toolkit_gen.ps1
```

```powershell
# build only
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\setup_toolkit_gen.ps1 -SkipInstall
```

```powershell
# build + tests
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\setup_toolkit_gen.ps1 -RunTests
```

## 2) Create a New Toolkit

```powershell
.\plugins\dm-toolkit-gen.exe init --name MSWord --prefix word --category office
```

Creates:

`plugins/functions/office/MSWord_Toolkit.ps1`

## 3) Add a New Function

```powershell
.\plugins\dm-toolkit-gen.exe add --file plugins/functions/office/MSWord_Toolkit.ps1 --prefix word --func export_pdf --param InputPath --param OutputPath --confirm
```

## 4) Validate Toolkits

```powershell
.\plugins\dm-toolkit-gen.exe validate
```

## 5) Useful Variants

```powershell
# ensure shared helper exists in plugins/utils.ps1
.\plugins\dm-toolkit-gen.exe add --file plugins/functions/office/MSWord_Toolkit.ps1 --prefix word --func open --require-helper _assert_path_exists --param Path
```

```powershell
# ensure shared variable exists in plugins/variables.ps1
.\plugins\dm-toolkit-gen.exe add --file plugins/functions/office/MSWord_Toolkit.ps1 --prefix word --func export_default --require-var DM_WORD_TEMPLATE=normal.dotm
```

## 6) If exe is not in plugins/

```powershell
.\dist\dm-toolkit-gen.exe init --repo . --name MSWord --prefix word --category office
```
