---
name: go-packaging
description: Use when designing, creating, modifying, or refactoring Go package structure in this project. このプロジェクトのパッケージ設計規約 — anti-pattern（util/common/helpers 禁止、層分け禁止）、cli/ の責務（cobra 配線専任、長い RunE は同一パッケージ内 runX 関数へ外出し）、format/ への表示整形集約、新規パッケージを切る判断基準を含む。TRIGGER when: internal/ 配下に新ファイルを作る / 新パッケージを検討する / 既存パッケージを refactor する / import を整理する / どこにコードを置くか迷う / 共通ヘルパーを切り出したくなる / cli/ サブコマンドを追加・修正する。
---

# パッケージ設計規約（このプロジェクト）

## 基本方針

**少数の大きいパッケージ**を正とする。Go コミュニティの慣習（Effective Go / 標準ライブラリの net/http 等、1 パッケージで 50+ ファイル）に従う。

> "A little copying is better than a little dependency" — Rob Pike

## 必ず守る

1. **`util` / `common` / `helpers` / `misc` パッケージを作らない**（何の責務も表していない anti-pattern）
2. **層（layer）でパッケージを切らない**（`app/`, `service/`, `model/`, `repository/` のような層分けは Java 由来の過剰設計）
3. **ドメインで切る**（`storage/`, `git/`, `tmux/`, `format/` のように責務 1 単語）
4. **`internal/` で十分カプセル化されている**。さらに細分化しない
5. **新規パッケージは「他からも呼ばれる」+「独立してテストしたい」が両方真**のときだけ作る
6. **パッケージ名は短い小文字 1 単語**

## このプロジェクトの責務マップ

```
cmd/tmux-agent-log/main.go    起動だけ（cli.Execute() を呼ぶのみ）
internal/
  cli/                         cobra 配線。サブコマンド単位で 1 ファイル
  hook/                        Claude Code hook ハンドラ（書き込みパス）
  storage/                     JSONL append-only + flock のデータ層
  config/                      XDG パス + TOML config
  format/                      表示整形（Time / JSONIndent / Write）の集約
  git/                         git CLI ラッパー
  tmux/                        tmux send-keys + OSC 52 fallback
  errlog/                      errors.jsonl の record / read / clear
```

## cli パッケージの規約

`cli/` は **cobra 配線専任**。RunE が 20 行を超えたら同一パッケージ内に `runX(opts)` 関数として外出し（kubectl / gh の慣習）:

```go
// cli/comment.go
func newCommentAddCmd() *cobra.Command {
    var opts commentAddOptions
    cmd := &cobra.Command{
        Use: "add",
        RunE: func(cmd *cobra.Command, args []string) error {
            return runCommentAdd(cmd, opts)   // RunE は 1 行
        },
    }
    cmd.Flags()...
    return cmd
}

type commentAddOptions struct { ... }

func runCommentAdd(cmd *cobra.Command, opts commentAddOptions) error {
    // ロジック本体（cobra 抜きでテスト可能）
}
```

利点: ① cobra 抜きで `runCommentAdd` を直接テストできる、② cobra 構築部だけを目で追える、③ 将来 TUI/SDK から同じ `runX` を呼べる。

## ファイル名規約

- `cli/comment.go` / `cli/install.go` のように **責務をファイル名で表現**
- `util.go` / `helpers.go` / `common.go` を作らない
- 同一ファイル内の並び: cobra 構築関数 → options 型 → handler 関数（gh CLI 慣習）

## 表示整形は format/ に集約

`writeJSONIndent` のような表示整形関数を cli パッケージのプライベートに置かず、`format.JSONIndent` として `internal/format/` に集約する。`format.Time`, `format.Write` も同様。

理由: 表示整形は cli から呼ぶが本質的にレイアウト責務。format/ にあれば他の subcommand や将来の TUI からも使える。

## 新規パッケージを切る判断

| 状況 | 判断 |
|------|------|
| 1 箇所からしか呼ばれない | **作らない**。呼び出し側ファイルに置く |
| 複数箇所から呼ばれるが小さい（数十行） | 既存の近いパッケージに置く（例: format/） |
| 複数箇所 + 独立した責務 + 独立してテストしたい | **新規パッケージ可** |
| settings.json 編集を将来 TUI からも使う見込みあり | `internal/claudesettings/` 化の検討対象（今は cli/install.go のままで OK） |

迷ったら **作らない側に倒す**。後で必要になったら切り出すのは簡単、最初から細かく切ったものを統合するのは難しい。

## ドメイン外のものを切り出した先例

- `cli/util.go` の `formatInt` → 削除（dead code）
- `cli/sessiondir.go` の `formatTime` → `internal/format/time.go` の `format.Time`
- `cli/show.go` の `writeJSONIndent` → `internal/format/json.go` の `format.JSONIndent`
- `cli/sessiondir.go` の `findSessionDir` → `cli/session.go`（パッケージは cli のまま、ファイル名で意図表現）

## 参照

- Effective Go: https://go.dev/doc/effective_go
- Google Go Style Guide: https://google.github.io/styleguide/go/
- Dave Cheney "Practical Go": https://dave.cheney.net/practical-go/presentations/qcon-china.html
- Uber Go Style Guide: https://github.com/uber-go/guide/blob/master/style.md
