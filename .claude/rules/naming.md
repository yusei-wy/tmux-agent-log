# 命名規約

## 関数名にスコープを反映する

特定コマンド専用のヘルパーは、どのコマンドに属するか名前で分かるようにする。

- Bad: `confirmAll` (clear 専用なのに汎用的)
- Good: `confirmClearAll`

## 標準ライブラリと被る名前を避ける

拡張ラッパーには違いが分かる名前をつける。

- Bad: `parseDuration` (`time.ParseDuration` と紛らわしい)
- Good: `parseDurationWithDays`

## 暗黙の振る舞いを引数で明示する

関数内にハードコードされた定数（行数、リトライ回数など）は、呼び出し側で意図が読めるよう引数に出す。

- Bad: `previewFirstLines(s, maxLen)` — 「何行？」が名前にもシグネチャにもない
- Good: `promptPreview(s, maxLines, maxLen)` — 呼び出し側 `promptPreview(s, 2, 400)` で意図が明示される

## ファクトリー関数の命名を揃える

Cobra コマンドファクトリーは `xxxCmd` で統一する。

- Bad: `mkHook`
- Good: `makeHookCmd`
