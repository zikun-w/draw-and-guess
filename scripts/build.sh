#!/usr/bin/env bash
set -euo pipefail
project_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
architecture="${1:-amd64}"
case "$architecture" in amd64|arm64) ;; *) echo '架构应为 amd64 或 arm64' >&2; exit 1 ;; esac
go_binary="${GO_BINARY:-go}"
release_path="$project_root/dist/release-$(date +%Y%m%d-%H%M%S)-$architecture"
mkdir -p "$release_path"
cd "$project_root/frontend"
npm ci
npm run build
cp -R dist "$release_path/frontend"
cp -R "$project_root/deploy" "$release_path/deploy"
cd "$project_root/backend"
CGO_ENABLED=0 GOOS=linux GOARCH="$architecture" "$go_binary" build -trimpath -ldflags='-s -w' -o "$release_path/drawguess" ./cmd/server
printf 'Debian 发布目录：%s\n' "$release_path"
