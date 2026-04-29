#!/usr/bin/env bash
# End-to-end スモークテスト。tmux と git が PATH に必要。

set -euo pipefail

BIN_DIR=$(mktemp -d)
BIN="$BIN_DIR/tmux-agent-log"
STATE_DIR=$(mktemp -d)
REPO=$(mktemp -d)

trap 'rm -rf "$BIN_DIR" "$STATE_DIR" "$REPO"' EXIT

export XDG_STATE_HOME="$STATE_DIR"
export TMUX_AGENT_LOG_ASSUME_YES=1

go build -o "$BIN" ./cmd/tmux-agent-log

cd "$REPO"
git init -q
git config user.email a@b
git config user.name a
git commit --allow-empty -q -m base
cd - >/dev/null

echo "{\"session_id\":\"s1\",\"cwd\":\"$REPO\",\"transcript_path\":\"/tmp/t\"}" | "$BIN" hook session-start
echo "{\"session_id\":\"s1\",\"cwd\":\"$REPO\",\"prompt\":\"add hello file\"}" | "$BIN" hook turn-start

TURN_ID=$("$BIN" list-turns --session s1 --format tsv | cut -f1 | head -1)

echo "{\"session_id\":\"s1\",\"cwd\":\"$REPO\",\"turn_id\":\"$TURN_ID\",\"tool_name\":\"Write\",\"tool_input\":{\"file_path\":\"hello.txt\"}}" | "$BIN" hook tool-pre

echo "hello world" > "$REPO/hello.txt"
(cd "$REPO" && git add hello.txt && git commit -q -m "add hello")

echo "{\"session_id\":\"s1\",\"cwd\":\"$REPO\",\"turn_id\":\"$TURN_ID\",\"tool_name\":\"Write\",\"tool_response\":{\"success\":true}}" | "$BIN" hook tool-post
echo "{\"session_id\":\"s1\",\"cwd\":\"$REPO\"}" | "$BIN" hook turn-end

STATUS=$("$BIN" list-turns --session s1 --format tsv | cut -f4 | head -1)
[[ "$STATUS" == "done" ]] || { echo "FAIL: expected status=done, got $STATUS" >&2; exit 1; }

"$BIN" comment add --session s1 --file hello.txt --line 1 --text "why world?"
PREVIEW=$("$BIN" comment send --session s1 --preview)
[[ "$PREVIEW" == *"why world?"* ]] || { echo "FAIL: send --preview missing text" >&2; exit 1; }

EXPORT=$("$BIN" export --session s1 --format md)
[[ "$EXPORT" == "# "* ]] || { echo "FAIL: export missing H1" >&2; exit 1; }

echo "SMOKE OK"
