# Install bhuppi on Windows.
# bhuppi.cmd goes on PATH. bhuppi.exe stays in %LOCALAPPDATA%\bhuppi\lib
# so PATHEXT does not prefer the .exe over the .cmd (folders go must cd).
# Usage: .\install.ps1
#        .\install.ps1 --go-decision     # prints use|fetch and does not download

$ErrorActionPreference = 'Stop'
$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $Root
$GoVer = '1.22.12'

function Get-GoDecision {
  $cmd = Get-Command go -ErrorAction SilentlyContinue
  if (-not $cmd) { return 'fetch' }
  $ver = & go env GOVERSION 2>$null
  if ($ver -match '^go1\.(\d+)') {
    if ([int]$Matches[1] -ge 22) { return 'use' }
  }
  return 'fetch'
}

if ($args -contains '--go-decision') {
  Get-GoDecision
  exit 0
}

function Ensure-Go {
  if ((Get-GoDecision) -eq 'use') {
    Write-Host "bhuppi install: using $(Get-Command go | Select-Object -ExpandProperty Source)"
    return
  }
  $local = Join-Path $env:LOCALAPPDATA "bhuppi\go\bin\go.exe"
  if (Test-Path $local) {
    $env:PATH = "$(Split-Path $local);$env:PATH"
    if ((Get-GoDecision) -eq 'use') {
      Write-Host "bhuppi install: using $local"
      return
    }
  }
  $arch = if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') { 'arm64' } else { 'amd64' }
  $url = "https://go.dev/dl/go$GoVer.windows-$arch.zip"
  $tmp = Join-Path $env:TEMP "bhuppi-go-$GoVer.zip"
  Write-Host "bhuppi install: fetching $url"
  Invoke-WebRequest -Uri $url -OutFile $tmp
  $dest = Join-Path $env:LOCALAPPDATA 'bhuppi'
  if (Test-Path (Join-Path $dest 'go')) { Remove-Item -Recurse -Force (Join-Path $dest 'go') }
  New-Item -ItemType Directory -Force -Path $dest | Out-Null
  Expand-Archive -Path $tmp -DestinationPath $dest -Force
  Remove-Item $tmp -Force
  $env:PATH = "$(Join-Path $dest 'go\bin');$env:PATH"
  if ((Get-GoDecision) -ne 'use') {
    throw "Go $GoVer is not usable after fetch"
  }
}

function Install-GitIfNeeded {
  $candidates = @(
    (Join-Path $env:ProgramFiles 'Git\usr\bin\ls.exe'),
    (Join-Path ${env:ProgramFiles(x86)} 'Git\usr\bin\ls.exe')
  )
  foreach ($c in $candidates) {
    if ($c -and (Test-Path $c)) { return }
  }
  $winget = Get-Command winget -ErrorAction SilentlyContinue
  if (-not $winget) {
    Write-Host "bhuppi install: Git usr\bin is missing and winget is not on PATH"
    return
  }
  Write-Host "bhuppi install: Unix tools missing; installing Git.Git"
  & winget install --id Git.Git -e --source winget --accept-package-agreements --accept-source-agreements
}

Ensure-Go
$env:CGO_ENABLED = '0'
$lib = Join-Path $env:LOCALAPPDATA 'bhuppi\lib'
$binDir = Join-Path $env:LOCALAPPDATA 'bhuppi\bin'
New-Item -ItemType Directory -Force -Path $lib, $binDir | Out-Null
Write-Host 'bhuppi install: building'
& go build -trimpath -ldflags '-s -w' -o (Join-Path $lib 'bhuppi.exe') .\cmd\bhuppi
Copy-Item -Force (Join-Path $Root 'share\bhuppi.cmd') (Join-Path $binDir 'bhuppi.cmd')
Copy-Item -Force (Join-Path $Root 'share\bhuppi-shell.ps1') (Join-Path $env:LOCALAPPDATA 'bhuppi\bhuppi-shell.ps1')

$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if ($userPath -notlike "*$binDir*") {
  [Environment]::SetEnvironmentVariable('Path', "$binDir;$userPath", 'User')
  $env:PATH = "$binDir;$env:PATH"
  Write-Host "bhuppi install: added $binDir to the user PATH (new windows pick it up)"
}

$profileDir = Split-Path -Parent $PROFILE
if (-not (Test-Path $profileDir)) { New-Item -ItemType Directory -Force -Path $profileDir | Out-Null }
if (-not (Test-Path $PROFILE)) { New-Item -ItemType File -Path $PROFILE | Out-Null }
$marker = '# >>> bhuppi >>>'
$hook = Join-Path $env:LOCALAPPDATA 'bhuppi\bhuppi-shell.ps1'
$text = Get-Content -Raw $PROFILE
if ($text -notlike "*$marker*") {
  Add-Content -Path $PROFILE -Value @"

$marker
. '$hook'
# <<< bhuppi <<<
"@
}

Install-GitIfNeeded
Write-Host "bhuppi install: $(Join-Path $binDir 'bhuppi.cmd')"
Write-Host 'try: bhuppi help'
