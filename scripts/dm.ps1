Param(
  [Parameter(ValueFromRemainingArguments = $true)]
  [string[]]$Args
)

$here = Split-Path -Parent $MyInvocation.MyCommand.Path
& "$here\..\dm.exe" @Args
