---
name: go-error-handling
description: Use when writing, modifying, or reviewing Go error handling in this project. このプロジェクトのエラーハンドリング規約 — `fmt.Errorf("...: %w", err)` で wrap（pkg/errors は使わない）、wrap message は layer の意図だけ書き path は重複させない（os.* err は既に path を含むため）、握りつぶし許容は 3 ケースのみ（出力系 / `fs.ErrNotExist` / read-only `defer Close()`）、エラーは 3 Tier（致命=`RunE` return / 継続可=`errlog.Record` + stderr 警告 / 無視可=blank assign）、`os.Exit` 直接呼出し禁止、`--debug` フラグ導入しない、sentinel error は YAGNI。TRIGGER when: error を return する / `fmt.Errorf` を書く / `%w` で wrap する / wrap message に path を入れようとする / `_, _ =` で破棄する / `errors.Is/As` を使う / sentinel error を定義する / panic / recover / `errlog.Record` を呼ぶ / `RunE` を書く / `pkg/errors` の導入を検討する。
---

# エラーハンドリング規約（このプロジェクト）

## 基本方針

CLI Go の王道に従う。**標準 `fmt.Errorf("...: %w", err)` で wrap して境界（`RunE`）まで持ち上げ、cobra に exit code を任せる。**

`pkg/errors` / `cockroachdb/errors` 等のサードパーティは**使わない**。Go 1.13 (2019) で `%w` が標準入りして以降、新規プロジェクトでは `fmt.Errorf` で十分。スタックトレースが本気で必要になったら、その時点で `cockroachdb/errors` を後付け検討する（YAGNI）。

> wrap チェーン（`load config: read /path: permission denied`）で「**どこで何をしようとして失敗したか**」が分かれば、CLI のトラブルシュートには十分。

## 必ず守る

1. **内部関数は `fmt.Errorf("<doing what>: %w", err)` で wrap して return する**
2. **「何をしようとしてた」を必ず添える**（`return err` を裸で投げない、`fmt.Errorf("%w", err)` 単独もダメ）
3. **`pkg/errors` を導入しない**
4. **sentinel error は実需が出てから**（`errors.Is/As` で識別したい具体ケースが出るまで作らない）
5. **`os.Exit` を直接呼ばない**（cobra に任せる）
6. **`--debug` / `-v` フラグを導入しない**（必要になってから）

## wrap の書き方

```go
// Bad: トレースもコンテキストも消える
if err := os.ReadFile(path); err != nil {
    return err
}

// Bad: コンテキストなし。チェーンに加える価値がない
if err := os.ReadFile(path); err != nil {
    return fmt.Errorf("%w", err)
}

// Bad: os.ReadFile のエラーは既に path を含む。wrap で path を重複させると
//      "load config /etc/foo.toml: open /etc/foo.toml: permission denied" のように冗長になる
if err := os.ReadFile(path); err != nil {
    return fmt.Errorf("load config %s: %w", path, err)
}

// Good: 「何をしようとしたか」だけ足す。path は os err 側にだけ残す
//      → "load config: open /etc/foo.toml: permission denied"
if err := os.ReadFile(path); err != nil {
    return fmt.Errorf("load config: %w", err)
}
```

### path / 引数を wrap message に書くか

| 状況 | 書く |
|---|---|
| os.ReadFile / os.WriteFile / os.MkdirAll / os.Open 等、path が **os err に含まれる** 場合 | **書かない**（"load config: %w"） |
| ドメイン ID で識別（turnID, sessionID, commentID 等） | **書く**（"mark comment %s sent: %w"）— 上位 err には載らないため |
| そもそも path が引数になっていない関数内エラー | 必要に応じて |

Go の `os` / `net` パッケージの多くは `*PathError` / `*OpError` で詳細を載せる。重複を避けるため wrap は **layer の意図** だけを書く。

`%w` チェーンは `errors.Is` / `errors.As` でアンラップできる。識別したい sentinel error が出てきたら追加する。

```go
var ErrConfigMissing = errors.New("config missing")

// 上位で
if errors.Is(err, ErrConfigMissing) {
    // 専用のメッセージや fallback
}
```

## 握りつぶしていい 3 ケース（これ以外は禁止）

| ケース | 書き方 | 理由 |
|---|---|---|
| stdout/stderr への書込み | `_, _ = fmt.Fprintln(...)` | SIGPIPE 等。CLI は stdout が壊れたらそもそも何もできない |
| idempotent な削除の `fs.ErrNotExist` | `if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) { return err }` | 既に無いものを消そうとする操作は成功扱い |
| read-only ファイルの `defer Close()` | `defer f.Close()` | 読み取り専用のクローズ失敗で書込み破損は起きない |

書込み (`os.WriteFile` / `os.OpenFile` 後の Close) は **握りつぶさない**。`defer` ではなく明示的に `if err := f.Close(); err != nil { return ... }` で扱う。

それ以外のエラーは **必ず wrap して return**。

## エラーの 3 Tier

| Tier | 例 | 扱い |
|---|---|---|
| **致命** | 設定読込失敗、storage 書込失敗、必須引数欠落 | `RunE` から `return err`。cobra が `Error: <msg>` を stderr に出して exit 1 |
| **継続可** | hook 内の部分失敗、tmux pane 不在で clipboard fallback、個別 turn の処理失敗 | `errlog.Record` + stderr に **1 行警告**（`tmux-agent-log: warn: ...`）。処理は続行 |
| **無視可** | 上記の 3 ケース | blank assign |

### Tier 2 の典型パターン

```go
if err := doSomething(); err != nil {
    _ = errlog.Record(component, event, sessionID, err.Error())
    fmt.Fprintf(os.Stderr, "tmux-agent-log: warn: %v\n", err)
    // 続行
}
```

`errlog.Record` 自身の失敗は無視する（ログのログを取らない）。stderr の 1 行警告は `_, _ =` で破棄してもよい（出力系のため）。

## panic recover

- **hook 経路の最上位で `recover` + `errlog.Record`**（`internal/cli/hook.go` の現行パターン）
- それ以外では recover しない（cobra に任せる）
- panic は本来バグなので、recover するのは「他の hook 呼び出しを巻き込まないため」が目的

## cobra との分担

| 場所 | 責務 |
|---|---|
| 内部関数 (`hook` / adapter) | wrap して return |
| `cli/*.go` の `runX(opts)` | wrap して return |
| `cli/*.go` の `RunE` | `return runX(...)` のみ。エラー整形しない |
| `main.go` | cobra に任せる（`os.Exit` 呼出し禁止） |

cobra のデフォルトでは `RunE` が non-nil error を返すと:
1. `Error: <err.Error()>` を stderr に出力
2. usage を表示（`SilenceUsage = true` で抑止可能）
3. exit code 1 で終了

usage 表示は実行時エラーには邪魔なので、root cmd で `SilenceUsage: true` を設定するのが慣習。

## errlog の役割

| 流すもの | 流さないもの |
|---|---|
| Tier 2（継続可エラー） | Tier 1（致命）— cobra が stderr に出すので二重記録しない |
| hook 経路の panic recover | テスト失敗・ユーザー入力エラー |

## lint との整合

- `gosec G104` は global 除外を**維持**（出力系の `_, _ =` のため）
- `errcheck` は有効を**維持**（`_, _ =` は blank assign で通る、これが意図）
- `errorlint` は有効を**維持**（`%w` 強制 + `errors.Is/As` 強制）

## 既存コードからの移行方針（参考）

このプロジェクトには現状以下の不揃いがある。触ったタイミングで揃える（一斉移行はしない）:

- `internal/cli/` の `_, _ = fmt.Fprintln(...)` — **OK**（出力系の握りつぶし許容）
- `internal/cli/hook.go` の裸 `fmt.Fprintln(os.Stderr, ...)` — **OK**（出力系として `_, _ =` を付けるかは好みで揃える）
- `internal/format/format.go` の厳密チェック — `bytes.Buffer` への書込みは失敗しないため、書込み先の型に応じて握りつぶし or wrap を選ぶ
- 内部関数の裸 `return err` — 触るタイミングで wrap に置き換える

## 参照

- Go Blog "Working with Errors in Go 1.13": https://go.dev/blog/go1.13-errors
- Effective Go (Errors): https://go.dev/doc/effective_go#errors
- Dave Cheney "Don't just check errors, handle them gracefully": https://dave.cheney.net/2016/04/27/dont-just-check-errors-handle-them-gracefully
- Uber Go Style Guide (Error Wrapping): https://github.com/uber-go/guide/blob/master/style.md#error-wrapping
