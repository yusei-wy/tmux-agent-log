# tmux-agent-log

tmux 内で動く Claude Code セッションのための構造化履歴レイヤー。

**ステータス:** Plan 1 のスコープのみ（CLI プリミティブ）。対話 TUI、tail viewer、配布成果物は Plan 2 / 3 で対応する。

## できること

- 各 Claude Code turn を append-only JSONL + 各 turn の git diff として記録
- 現在の diff に anchor された行コメントを追加し、`tmux send-keys` で 1 つのプロンプトとして Claude に送り返す（送信先 pane 消失時は OSC 52 クリップボード fallback）
- Go 単一バイナリで、`fzf` / `delta` / `bat` / `jq` 等の shell ワークフローと自然合成可能な CLI プリミティブを提供

## インストール（ソースビルド）

```bash
go install github.com/yusei-wy/tmux-agent-log/cmd/tmux-agent-log@latest
tmux-agent-log install-hooks      # ~/.claude/settings.json への opt-in 編集
```

前提: tmux 3.2+、git、ビルド時に Go 1.26+。

## クイック使用例

```sh
# tmux ペインで普段通り Claude Code を起動
claude

# goal を設定（任意）
tmux-agent-log goal --session <uuid> "認可バグ修正"

# セッションの diff と turn を確認
tmux-agent-log list-turns --session <uuid>
tmux-agent-log show-diff --base session <uuid>

# file:line に anchor したコメントを追加
tmux-agent-log comment add --session <uuid> \
    --file src/auth/middleware.go --line 44-46 \
    --text "nil check は本当に必要?"

# レビュープロンプトをプレビューしてから Claude に送信
tmux-agent-log comment send --session <uuid> --preview
tmux-agent-log comment send --session <uuid>

# PR 説明用に Markdown サマリーを export
tmux-agent-log export --session <uuid> --format md
```

## Spec

完全な設計、スコープ、non-goals は [`docs/superpowers/specs/2026-04-23-tmux-agent-log-design.md`](docs/superpowers/specs/2026-04-23-tmux-agent-log-design.md) を参照。

## ステータスとロードマップ

- **Plan 1（このリリース）: CLI コア** — hooks、storage、comment send、install-hooks
- **Plan 2:** 対話 TUI（timeline / diff / overview タブ）、tail viewer、blame + liveness map
- **Plan 3:** tmux/fzf/shell 連携の examples、GitHub Actions CI、GoReleaser + Homebrew tap

## ライセンス

MIT（予定。初回公開リリース前に確定）。
