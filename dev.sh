#!/bin/sh
set -eu
cd "$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
if command -v go >/dev/null 2>&1; then
  exec go run .
elif [ -x .tools/go/bin/go ]; then
  export GOCACHE="${GOCACHE:-${TMPDIR:-/tmp}/eagle-vet-gocache}"
  exec .tools/go/bin/go run .
else
  printf '%s\n' 'Go 1.22 or newer is required. Install Go from https://go.dev/dl/, then run ./dev.sh again.' >&2
  exit 1
fi
