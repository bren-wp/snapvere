$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$Version = (Get-Content (Join-Path $Root "VERSION") -Raw).Trim()
$Dist = Join-Path $Root "dist"
$PayloadApp = Join-Path $Root "cmd/setup/payload-app.exe"
$PayloadUninstall = Join-Path $Root "cmd/setup/payload-uninstall.exe"
New-Item -ItemType Directory -Force -Path $Dist | Out-Null
function Remove-Payloads {
  Remove-Item $PayloadApp,$PayloadUninstall -Force -ErrorAction SilentlyContinue
}
function Build-Arch([string]$GoArch,[string]$Label) {
  $env:GOOS="windows"; $env:GOARCH=$GoArch; $env:CGO_ENABLED="0"
  go vet -unsafeptr=false ./cmd/snapvera ./cmd/uninstall
  go vet ./internal/...
  go build -buildvcs=false -trimpath -ldflags "-H windowsgui -s -w -buildid= -X main.buildMode=portable" -o (Join-Path $Dist "Snapvera-$Version-windows-$Label-portable.exe") (Join-Path $Root "cmd/snapvera")
  go build -buildvcs=false -trimpath -ldflags "-H windowsgui -s -w -buildid= -X main.buildMode=installed" -o (Join-Path $Dist "Snapvera-$Version-windows-$Label.exe") (Join-Path $Root "cmd/snapvera")
  go build -buildvcs=false -trimpath -ldflags "-H windowsgui -s -w -buildid=" -o (Join-Path $Dist "Snapvera-$Version-windows-$Label-uninstall.exe") (Join-Path $Root "cmd/uninstall")
  try {
    Copy-Item (Join-Path $Dist "Snapvera-$Version-windows-$Label.exe") $PayloadApp -Force
    Copy-Item (Join-Path $Dist "Snapvera-$Version-windows-$Label-uninstall.exe") $PayloadUninstall -Force
    go vet -unsafeptr=false ./cmd/setup
    go build -buildvcs=false -trimpath -ldflags "-H windowsgui -s -w -buildid=" -o (Join-Path $Dist "Snapvera-$Version-windows-$Label-setup.exe") (Join-Path $Root "cmd/setup")
  } finally {
    Remove-Payloads
  }
}
Set-Location $Root
$unformatted = gofmt -l cmd internal
if ($unformatted) { throw "gofmt required: $($unformatted -join ', ')" }
go test ./internal/...
python tools/verify_release.py .
try {
  Build-Arch "amd64" "x64"
  Build-Arch "arm64" "arm64"
} finally {
  Remove-Payloads
}
python tools/verify_windows.py dist
Write-Host "Snapvera Windows build completed."
