# tmux-agent-log

tmux + AI Agent 開発の認知負荷を軽減する Go 製 CLI ツール。

## スコープ

- **Plan 1（現在）**: CLI コア（hooks / storage / comment send / install-hooks）
- Plan 2 / 3: 対話 TUI、tail viewer、配布成果物（未着手）

詳細設計は `docs/superpowers/specs/2026-04-23-tmux-agent-log-design.md` を参照。

## 開発方針

- 常に根拠を意識する
- 不確定な部分は先に PoC として検証する
