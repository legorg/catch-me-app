#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 || $1 != --confirm-development ]]; then
  echo "Usage: bash scripts/seed-game.sh --confirm-development" >&2
  echo "Development databases only. Each run inserts a new synthetic game." >&2
  exit 2
fi

repository_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
# Git Bash needs a Windows host path, but must not rewrite /data inside Docker.
if command -v cygpath >/dev/null 2>&1; then
  repository_root="$(cygpath -m "$repository_root")"
fi

MSYS_NO_PATHCONV=1 exec docker compose --project-directory "$repository_root" -f "$repository_root/compose.yaml" \
  run --build --rm --no-deps devfixture -db /data/catch-me.sqlite -confirm-development
