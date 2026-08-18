#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="$(tr -d '\r\n' < "$ROOT/VERSION")"
mkdir -p "$ROOT/dist"
cleanup_payloads(){
  rm -f "$ROOT/cmd/setup/payload-app.exe" "$ROOT/cmd/setup/payload-uninstall.exe"
}
trap cleanup_payloads EXIT
build_arch(){
  local goarch="$1" label="$2"
  local base=(-buildvcs=false -trimpath)
  GOOS=windows GOARCH="$goarch" CGO_ENABLED=0 go vet -unsafeptr=false ./cmd/snapvera ./cmd/uninstall
  GOOS=windows GOARCH="$goarch" CGO_ENABLED=0 go vet ./internal/...
  local ld_common='-H windowsgui -s -w -buildid='
  GOOS=windows GOARCH="$goarch" CGO_ENABLED=0 go build "${base[@]}" -ldflags="$ld_common -X main.buildMode=portable" -o "$ROOT/dist/Snapvera-$VERSION-windows-$label-portable.exe" "$ROOT/cmd/snapvera"
  GOOS=windows GOARCH="$goarch" CGO_ENABLED=0 go build "${base[@]}" -ldflags="$ld_common -X main.buildMode=installed" -o "$ROOT/dist/Snapvera-$VERSION-windows-$label.exe" "$ROOT/cmd/snapvera"
  GOOS=windows GOARCH="$goarch" CGO_ENABLED=0 go build "${base[@]}" -ldflags="$ld_common" -o "$ROOT/dist/Snapvera-$VERSION-windows-$label-uninstall.exe" "$ROOT/cmd/uninstall"
  cp "$ROOT/dist/Snapvera-$VERSION-windows-$label.exe" "$ROOT/cmd/setup/payload-app.exe"
  cp "$ROOT/dist/Snapvera-$VERSION-windows-$label-uninstall.exe" "$ROOT/cmd/setup/payload-uninstall.exe"
  GOOS=windows GOARCH="$goarch" CGO_ENABLED=0 go vet -unsafeptr=false ./cmd/setup
  GOOS=windows GOARCH="$goarch" CGO_ENABLED=0 go build "${base[@]}" -ldflags="$ld_common" -o "$ROOT/dist/Snapvera-$VERSION-windows-$label-setup.exe" "$ROOT/cmd/setup"
  cleanup_payloads
}
cd "$ROOT"
test -z "$(gofmt -l cmd internal)"
go test ./internal/...
python3 "$ROOT/tools/verify_release.py" "$ROOT"
build_arch amd64 x64
build_arch arm64 arm64
python3 "$ROOT/tools/verify_windows.py" "$ROOT/dist"
echo "Windows release build completed."
