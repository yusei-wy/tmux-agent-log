---
name: go-testing
description: Use when writing, modifying, refactoring, splitting, merging, or reviewing Go test code (*_test.go) in this project. このプロジェクトの Go テスト規約 — table-driven 形式の使い方と「適さない兆候」（split すべき条件）、t.Run サブテスト、external test package（package X_test）convention、testify (require/assert) 使用方針、カバレッジの考え方を含む。TRIGGER when: テスト関数を新規追加する / 既存テストを修正する / table-driven 化を検討する / テストを split / merge する / カバレッジ gap を議論する / *_test.go ファイルを開いて編集する。
---

# Go テスト規約（このプロジェクト）

## 基本方針

Go のテストは **table-driven 形式**（`cases` スライス + `t.Run(tc.name, ...)`）を基本とする。Effective Go と Go 標準ライブラリの慣習に従う。

## 必ず守る

1. 同一関数の挙動を **2 ケース以上、同形の input/output で**テストする場合は table-driven に統一
2. 各ケースは `name` フィールドを持ち、`t.Run(tc.name, func(t *testing.T) { ... })` でサブテスト化（個別 PASS が出るよう）
3. エラー経路は `wantErr bool` フィールドで同一テーブル内に混ぜる
4. external test package（`package X_test`）を使う
5. private 関数のテストが必要な場合は **`export_test.go` で test 用に export** する（`package X` 内部テストにしない）
6. testify の `require` / `assert` を使ってよい。失敗時に subsequent assertion を止めたいなら `require`、続行したいなら `assert`
7. 既存の非 table-driven テストは、依頼があれば書き換える。無断で巻き込まない

## table-driven にしてはいけない兆候（強制 split）

以下のいずれかが出たら **個別関数に分離** する:

- ループ内に `if/else` での **setup 分岐** がある（cases の同質性が崩壊）
- ループ内に `if/else` での **assertion 分岐 / 早 return** がある（同一 success/failure path の同居崩壊）
- 同じ assertion で押し込められるが、SUT で踏むコードパスが異なる（例: `panic` 復帰 vs `error` ロギング）
- setup 関数が完全に case-specific で uniform な act/assert を持たない
- ケースが 2 つだけで、片方の setup が他方より大幅に長く（数倍）非対称

table の boilerplate より「**何をテストしているか**」を関数名で示す価値が勝る場合は分離する。

## 良い table-driven の例

```go
func TestParseLineRange(t *testing.T) {
    cases := []struct {
        name    string
        in      string
        start   int
        end     int
        wantErr bool
    }{
        {name: "single line", in: "44", start: 44, end: 44},
        {name: "range", in: "44-46", start: 44, end: 46},
        {name: "end before start returns error", in: "46-44", wantErr: true},
        {name: "non-numeric returns error", in: "abc", wantErr: true},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            s, e, err := parseLineRange(tc.in)
            if tc.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            require.Equal(t, tc.start, s)
            require.Equal(t, tc.end, e)
        })
    }
}
```

## 分離すべき例

```go
// Bad: panic と error は同じ「return 0」だが SUT で踏むコードパスが異なる
func TestRunWithRecover(t *testing.T) {
    cases := []struct{ ... }{
        {name: "panic", fn: func() error { panic("boom") }},
        {name: "error", fn: func() error { return errors.New("bad") }},
    }
    // ...
}

// Good: 関数名で何が起きるかを伝える
func TestRunWithRecoverReturnsZeroOnPanic(t *testing.T) { ... }
func TestRunWithRecoverReturnsZeroOnError(t *testing.T) { ... }
```

## private 関数のテスト（`export_test.go` パターン）

private 関数（小文字始まり）をテストする際は、内部テスト（`package X` のままテストファイルを書く）ではなく、**`export_test.go`** で test 用に export する。Go 標準ライブラリ（`net/http` 等）が広く使う慣習で、外部テストパッケージ（`package X_test`）の一貫性を保ちながら private にもアクセスできる。

### 仕組み

`*_test.go` で終わるファイルは **test 時のみコンパイル** される。`export_test.go` を `package X` で書くと、test 時だけ private シンボルを export できる。

### 例

```go
// internal/storage/export_test.go
package storage

// test 用に private を export する。
// このファイルは _test.go なので production binary には含まれない。
var (
    ParseLineRange = parseLineRange
    BuildKey       = buildKey
)

// 型のエイリアス
type ExportedConfig = config
```

```go
// internal/storage/storage_test.go
package storage_test

import (
    "testing"

    "github.com/yusei-wy/tmux-agent-log/internal/storage"
)

func TestParseLineRange(t *testing.T) {
    got, err := storage.ParseLineRange("44-46")
    // ...
}
```

### メソッドの export

メソッドはエイリアスできないので、wrapper 関数を定義する:

```go
// export_test.go
package storage

func (s *Store) ExportedDoSomething() error { return s.doSomething() }
```

または test 用 helper を生やす:

```go
func ExportedDoSomethingOn(s *Store) error { return s.doSomething() }
```

### 例外: cobra コマンドの内部テスト

`cli/` パッケージのように **「cobra Command の組立そのものを test するために rootCmd へ直接アクセスしたい」** ケースは、現状 `package cli` 内部テストを許容している（`hook_test.go` 等）。
将来 Phase B で `runX(opts)` 関数化が完了したら、`runX` を `export_test.go` で export する形に揃える。

## 出力モードを混ぜる場合の許容パターン

assert する形が「exact bytes」と「部分一致」の 2 系統あるとき、構造体に両フィールドを持たせて
sentinel で切り替えるのは許容（例: `format/format_test.go` の `TestWrite`）:

```go
cases := []struct {
    name         string
    want         string   // exact bytes; "" でスキップ
    wantContains []string // 部分一致
    wantErr      bool
}{ ... }
```

ただし、これが 3 系統以上になったら個別関数に分離した方が読みやすい。

## カバレッジ確認

```bash
go test -cover ./...                         # パッケージ別の数値
go test -coverprofile=cov.out ./...
go tool cover -func=cov.out                  # 関数別
go tool cover -html=cov.out                  # ブラウザで色付き表示
```

カバレッジ数値は目安。重要なのは **エラー経路と境界ケースに 1 case ずつ当てる** こと。
数値追求のためにテストを書かない。
