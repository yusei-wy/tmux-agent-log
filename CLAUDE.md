# tmux-agent-log

tmux + AI Agent 開発の認知負荷を軽減する Go 製 CLI ツール。

## スコープ

- **Plan 1（現在）**: CLI コア（hooks / storage / comment send / install-hooks）
- Plan 2 / 3: 対話 TUI、tail viewer、配布成果物（未着手）

詳細設計は `docs/superpowers/specs/2026-04-23-tmux-agent-log-design.md` を参照。

## 開発方針

- 常に根拠を意識する
- 不確定な部分は先に PoC として検証する

## ツールチェーン

すべて `mise` 管理。直接 `go install` / `brew install` で入れたバイナリは使わない。

```bash
mise install                      # 初回 / .mise.toml 更新時
mise exec -- lefthook install     # 初回のみ pre-commit フック配置
```

日常コマンド:

```bash
mise run fmt          # gofumpt + goimports で整形
mise run lint         # golangci-lint
mise run lint:fix     # golangci-lint --fix
mise run test         # go test ./...
mise run check        # fmt → lint → test
```

複合処理（fmt + lint + test 等）は必ず `.mise.toml` の `[tasks]` に集約し、フック・CI・手動実行はすべて `mise run <task>` を呼ぶだけの薄いラッパーにする。lefthook や hook スクリプト内に gofumpt / golangci-lint の引数を直書きしない。
