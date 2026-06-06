#!/usr/bin/env bash
# Builds the shared React remote once and copies the bundle into BOTH servers:
#   - android/app/src/main/assets/web  (served by the Android Ktor server)
#   - desktop/internal/webui/dist       (embedded into the Go binary)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "==> Building web remote (Vite)…"
cd "$ROOT/web"
npm install
npm run build

copy_bundle() {
  local dest="$1"
  rm -rf "$dest"
  mkdir -p "$dest"
  cp -R "$ROOT/web/dist/." "$dest/"
  echo "    copied -> $dest"
}

echo "==> Distributing bundle…"
copy_bundle "$ROOT/android/app/src/main/assets/web"
copy_bundle "$ROOT/desktop/internal/webui/dist"

echo "==> Done. Rebuild the Android app / Go binary to pick up the new remote."
