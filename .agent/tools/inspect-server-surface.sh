#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root"

echo "Core boot and integration files:"
find . -maxdepth 3 -type f | sort | grep -E '(^\./main\.go$|^\./README\.md$|^\./Makefile$|^\./src/(server|config|database|synch|websocket)/|^\./docs/)' || true

echo
echo "Key routing and runtime symbols:"
rg -n \
  "serve\\(|Route\\(|Main\\(|NewDeliveryWorker|NewSchedulerWorker|SYNC_BACKEND|DatabaseURL|workspace|websocket|migrate:down|migrate:status" \
  main.go src docs README.md || true
