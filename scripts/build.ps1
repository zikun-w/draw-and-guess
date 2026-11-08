param([ValidateSet('amd64', 'arm64')][string]$Architecture = 'amd64')
$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
$releasePath = Join-Path $projectRoot 'dist'
$goBinary = Join-Path $projectRoot '.tools\go\bin\go.exe'
if (!(Test-Path -LiteralPath $goBinary)) { $goBinary = (Get-Command go -ErrorAction Stop).Source }
New-Item -ItemType Directory -Force -Path $releasePath | Out-Null
Push-Location (Join-Path $projectRoot 'frontend')
try {
  npm ci
  if ($LASTEXITCODE -ne 0) { throw 'npm ci failed. Stop the frontend dev server before packaging.' }
  npm run build
  if ($LASTEXITCODE -ne 0) { throw 'Frontend build failed.' }
} finally { Pop-Location }
$previousOS, $previousArch, $previousCGO = $env:GOOS, $env:GOARCH, $env:CGO_ENABLED
Push-Location (Join-Path $projectRoot 'backend')
try {
  $env:GOOS = 'linux'; $env:GOARCH = $Architecture; $env:CGO_ENABLED = '0'
  & $goBinary build -trimpath -ldflags '-s -w' -o (Join-Path $releasePath 'drawguess') ./cmd/server
  if ($LASTEXITCODE -ne 0) { throw 'Backend build failed.' }
} finally { $env:GOOS = $previousOS; $env:GOARCH = $previousArch; $env:CGO_ENABLED = $previousCGO; Pop-Location }
# Copy each build into its own release directory to avoid stale frontend assets.
$bundlePath = Join-Path $releasePath ('release-' + (Get-Date -Format 'yyyyMMdd-HHmmss') + '-' + $Architecture)
New-Item -ItemType Directory -Force -Path $bundlePath | Out-Null
Copy-Item -LiteralPath (Join-Path $releasePath 'drawguess') -Destination $bundlePath
Copy-Item -LiteralPath (Join-Path $projectRoot 'frontend\dist') -Destination (Join-Path $bundlePath 'frontend') -Recurse
Copy-Item -LiteralPath (Join-Path $projectRoot 'deploy') -Destination $bundlePath -Recurse
Write-Output "Debian release directory: $bundlePath"
