---
name: go-testing
description: Use when writing, modifying, refactoring, splitting, merging, or reviewing Go test code (*_test.go). Go テスト規約 — テスト対象の選び方（t-wada 流: テストは投資、書く前に問う）、table-driven 形式と「適さない兆候」（split すべき条件）、t.Run サブテスト、external test package（package X_test）convention、private 関数は export_test.go で export、testify (require/assert) と go-cmp の使用方針を含む。TRIGGER when: テスト関数を新規追加する / 既存テストを修正する / テストすべきか迷う / table-driven 化を検討する / テストを split / merge する / カバレッジ gap を議論する / *_test.go ファイルを開いて編集する。
---

# Go テスト規約

## テスト対象の選び方（書く前に問う）

t-wada 流: **テストは投資**。目的はカバレッジ数値ではなく、**変更を恐れずに行うための確信**。

- **テストする**: 自分の分岐ロジック / 境界値・エッジケース / 自分が return する error / 過去のバグ（regression）/ 並行性で守る不変条件
- **テストしない**: ライブラリ・フレームワーク呼出の薄いラッパー / trivial getter・setter・定数 / 他テストが透過的にカバー済み / 実装詳細

判定: **「このテストが落ちたら何が壊れるか」を 1 文で言えないなら書かない**。

カバレッジ数値は追わない。低カバレッジ関数は上の判定を先に当てる。

## 必ず守る

1. 同一関数の挙動を **2 ケース以上、同形の input/output で**テストする場合は table-driven に統一
2. 各ケースは `name` フィールドを持ち、`t.Run(tc.name, ...)` でサブテスト化
3. エラー経路は `wantErr bool` で同一テーブル内に混ぜる
4. external test package（`package X_test`）を使う
5. private 関数のテストが必要な場合は **`export_test.go` で test 用に export** する（`package X` 内部テストにしない）
6. assertion は **testify の `require` / `assert` を使う**（停止が必要なら require、続行なら assert）。`if got != want { t.Errorf(...) }` を新規に書かない
7. 複合構造体 / slice / map の等価比較で diff が読みづらい場合は **`github.com/google/go-cmp/cmp`** を使う（`require.Empty(t, cmp.Diff(want, got))` パターン）。scalar や string の単純比較は testify のみで十分
8. 既存の非 table-driven テストは依頼があれば書き換える。無断で巻き込まない

## table-driven にしてはいけない兆候（強制 split）

以下のいずれかが出たら **個別関数に分離** する:

- ループ内で `if/else` での setup 分岐（cases 同質性の崩壊）
- ループ内で `if/else` での assertion 分岐 / 早 return（success/failure path の同居崩壊）
- 同じ assertion で押し込めても、SUT で踏むコードパスが異なる（例: `panic` 復帰 vs `error` ロギング）
- setup が完全に case-specific で uniform な act/assert を持たない
- 2 ケースのみで setup の長さが数倍非対称

「**何をテストしているか**」を関数名で直接示す価値が勝る場合は分離する。

## private 関数のテスト（`export_test.go` パターン）

`*_test.go` ファイルは test 時のみコンパイルされるため、`export_test.go` を `package X` で書けば test 時だけ private シンボルを export できる。Go 標準ライブラリ（`net/http` 等）の慣習。

```go
// pkg/foo/export_test.go
package foo

var ParseLineRange = parseLineRange     // 関数のエイリアス
type ExportedConfig = config            // 型のエイリアス
```

外部テスト（`package foo_test`）から `foo.ParseLineRange(...)` で呼べる。
メソッドはエイリアスできないので wrapper 関数を生やす。

フレームワーク（cobra のコマンドツリー等）に直接アクセスせざるを得ない真の理由がある場合のみ `package X` 内部テストを検討する。まずは `runX(opts)` のような関数を切り出して `export_test.go` で export する道を優先する。

## 出力モードを混ぜる場合の許容

assert が「exact bytes」と「部分一致」の 2 系統あるとき、構造体に両フィールドを持たせて sentinel で切り替えるのは許容（例: `want string` / `wantContains []string` の 2 フィールドを持ち、片方だけ埋めて assert 側で振り分ける）。3 系統以上になったら分離する。
