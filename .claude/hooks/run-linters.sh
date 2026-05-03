#!/usr/bin/env bash
set -euo pipefail

input=$(cat)
file=$(printf '%s' "$input" | jq -r '.tool_input.file_path // empty')

[[ -z "$file" || ! -f "$file" || "$file" != *.go ]] && exit 0
[[ -f "${CLAUDE_PROJECT_DIR}/go.mod" ]] || exit 0

cd "$CLAUDE_PROJECT_DIR"
mise run "fmt:files" "$file" >/dev/null 2>&1 || true
