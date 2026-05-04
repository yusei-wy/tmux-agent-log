# 命名規約

## 関数名にスコープを反映する

特定コマンド専用のヘルパーは、どのコマンドに属するか名前で分かるようにする。

- Bad: `confirmAll` (clear 専用なのに汎用的)
- Good: `confirmClearAll`

## 標準ライブラリと被る名前を避ける

拡張ラッパーには違いが分かる名前をつける。

- Bad: `parseDuration` (`time.ParseDuration` と紛らわしい)
- Good: `parseDurationWithDays`

## ファクトリー関数の命名を揃える

Cobra コマンドファクトリーは `xxxCmd` で統一する。

- Bad: `mkHook`
- Good: `makeHookCmd`
