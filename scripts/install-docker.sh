#!/usr/bin/env bash
set -euo pipefail

# Backward-compatible entrypoint
exec bash "$(dirname "$0")/install.sh" "$@"
