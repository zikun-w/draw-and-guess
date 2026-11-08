param([switch]$Backend, [switch]$Frontend)
$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
if ($Backend -eq $Frontend) { throw 'Choose -Backend or -Frontend; run each service in its own terminal.' }
if ($Frontend) {
  Push-Location (Join-Path $projectRoot 'frontend')
  try { npm run dev } finally { Pop-Location }
} else {
  $goBinary = Join-Path $projectRoot '.tools\go\bin\go.exe'
  if (!(Test-Path -LiteralPath $goBinary)) { $goBinary = (Get-Command go -ErrorAction Stop).Source }
  Push-Location (Join-Path $projectRoot 'backend')
  try { & $goBinary run ./cmd/server } finally { Pop-Location }
}
