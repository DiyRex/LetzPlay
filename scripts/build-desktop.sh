#!/usr/bin/env bash
# Builds the self-contained desktop binary: web remote first (embedded), then the Go binary.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

"$ROOT/scripts/build-web.sh"

echo "==> Building desktop binary (Go)…"
cd "$ROOT/desktop"
mkdir -p bin
# Unset GOROOT: some setups export a GOROOT that points at a *different* Go version than the `go`
# on PATH (e.g. a stale `export GOROOT=$(brew --prefix go@1.25)/libexec`), which makes `go build`
# fail with "version X does not match go tool version Y". Each go binary knows its own GOROOT, so
# we drop the override to keep the build reliable regardless of the user's shell config.
env -u GOROOT go build -o bin/letzplay ./cmd/letzplay
echo "==> Built desktop/bin/letzplay"
echo "    Run it:  ./desktop/bin/letzplay        (reads desktop/.env — port 8090, passwords)"
echo "    Override:./desktop/bin/letzplay --port 9000 --admin-password <pw>"
