#!/usr/bin/env bash
# Builds the self-contained desktop binary: web remote first (embedded), then the Go binary.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

"$ROOT/scripts/build-web.sh"

echo "==> Building desktop binary (Go)…"
cd "$ROOT/desktop"
mkdir -p bin
go build -o bin/letzplay ./cmd/letzplay
echo "==> Built desktop/bin/letzplay"
echo "    Run it:  ./desktop/bin/letzplay --admin-password <pw> --guest-password <pw>"
