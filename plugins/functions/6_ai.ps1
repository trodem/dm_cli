function dm_ai_analyze_dir {
    param([string]$Path)

    $files = Get-ChildItem -Recurse -File $Path

    foreach ($file in $files) {

        $code = Get-Content $file.FullName -Raw

        $prompt = @"
Analyze this file:

$file

Focus on:
- purpose
- bugs
- improvements

$code
"@

        $result = $prompt | ollama run deepseek-coder-v2:latest

        Write-Host "---- $file ----"
        Write-Host $result
    }
}
