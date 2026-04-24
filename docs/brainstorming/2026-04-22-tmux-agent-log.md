# tmux-agent-log 設計ブレインストーミング

- 作成日: 2026-04-22
- ステータス: **WIP（ブレインストーミング中）**
- 本ドキュメントは superpowers:brainstorming スキルに基づき、最終 spec 作成前の合意形成用メモです。

---

## 1. 課題（What）

tmux ベースの並列 AI 開発において、以下がボトルネックになっている。

- 各セッションで「何をしているか」が把握できない
- 差分（git diff）は見えるが「なぜそうなったか」が分からない
- AI 生成コードのため、思考プロセスが追えず構造化できない
- どのプロセスがどこで動いているか分からない

完了通知は別仕組みで解決済み。**進行中の状態と意味の把握**が残課題。

## 2. 問題の本質（Why）

> **差分に「意味」と「時間軸」が存在しない**

現状の情報構造:

| 情報源 | 何を示すか |
|--------|------------|
| git diff | 結果（What changed） |
| tmux | 実行状態（What is running） |

欠けているもの:

- AI の判断（Why）
- 変更の意図（Intent）
- ステップの連続性（History）

結果として「変更のストーリー」が存在せず、脳内で再構築できない。

## 3. 解決方針（How）

**差分に意味を与え、変更をストーリーとして扱う観察層を追加する**。tmux はそのまま、Claude Code はそのまま。不足している「思考の記録・意図の構造化・差分との接続」だけを足す。

### 3.1 スコープと他ツールとの関係

本プロジェクトは **tmux 内で動く AI agent（主に Claude Code）の「観察層」** に機能を絞る。

- ✅ **含める**: 構造化履歴・goal 管理・multi-base diff・行単位コメント・review → Claude 送り返し
- ❌ **含めない**: セッション起動管理・worktree 管理・Docker サンドボックス・サイドバー表示・エージェント起動

ユーザーは普段通りに tmux ペインで `claude` を起動する。本ツールは hooks 経由で黙って observable なレイヤーを構築する。

### 3.2 OSS 設計原則

tmux / Claude Code ユーザーは設定を強くカスタムしている前提で、**UX を押し付けない**:

- **No forced keybindings** — `.tmux.conf` を自動で書き換えない。examples として提示するのみ
- **No forced hook install** — `~/.claude/settings.json` を自動で変更しない。`install-hooks` サブコマンドで明示的に OPT-IN
- **No forced UI mode** — popup / split-pane / 新規 window / fzf 合成 / CLI 専用 / 外部ターミナル、どれで動かしても機能する設計
- **CLI primitive first** — すべての read 系は `--format tsv|jsonl|json`、write 系はスクリプトから呼べる形
- **fzf / delta / bat / gum 等と自然合成可能** — stdout に流せる、stdin を受け取れる

既存 dispatcher（AoE 等）への依存もせず、**単体で完結して動く**。既存ツールとの関係は付録 A を参照。

---

## 4. 合意済み決定事項

ブレインストーミング中に確定した項目。

### 4.1 対象範囲

| 項目 | 決定 | 備考 |
|------|------|------|
| 対象 AI | **Claude Code のみ（MVP）** | 将来の拡張性を持つアダプタ構造を維持 |
| 対象プロジェクト | **git 管理されたリポジトリ** | 非 git は起動時に警告のみ、no-op |
| クライアント | **tmux 内に閉じる（エミュレータ非依存）** | Ghostty / WezTerm 等には依存しない |
| 実装言語 | **Go 単一バイナリ（純 Go）** | hook / tail viewer / popup TUI / CLI をすべて内蔵。shell スクリプト不使用。`tmux` / `git` 以外のランタイム依存なし。OSS 配布時は `brew install` / `go install` で完結 |

### 4.2 ログ収集方式

| 項目 | 決定 |
|------|------|
| 基本戦略 | **完全パッシブ + 閲覧時要約（選択肢 4）** |
| 情報源 | Claude Code hooks + transcript JSONL（参照のみ、本文は複製しない） |
| AI 側への format 契約 | **なし**（CLAUDE.md 汚染なし、slash command 追加なし） |
| 要約 | 閲覧時にオンデマンドで実行（MVP では生プロンプト先頭 2 行を intent として表示、要約器は差し替え可能な interface に） |

### 4.3 step 粒度

| 項目 | 決定 |
|------|------|
| 粒度 | **ハイブリッド二層（選択肢 D）** |
| 外側（主語） | **ターン**（`UserPromptSubmit` → `Stop`） |
| 内側（イベント） | tool 呼び出しの時系列（Read / Edit / Write / Bash 等） |
| diff スナップショット境界 | **ターン境界**（turn 開始前・終了後） |

### 4.4 diff スナップショット

| 項目 | 決定 |
|------|------|
| 方式 | **A. git-native のみ（MVP）** |
| 記録方法 | セッション開始時 `base_sha` を記録 → 各ターン終了時 `git diff <base_sha>..HEAD` を保存 + unstaged 差分を別途保存 |
| 非 git プロジェクト | 起動時に 1 度警告、hook は no-op（拡張は将来） |
| セキュリティ | 保存ディレクトリは mode 0700、transcript 本体は複製せず参照のみ、redact フィルタは後付け可能 |

### 4.5 レビューモデル

**log は完全パッシブ（垂れ流し）。判断しない。review は人間が diff に対して行う**。

| 項目 | 決定 |
|------|------|
| 自動 review フラグ | **廃止**（heuristic 判定なし） |
| turn の status | `running / done / error` のみ（`review` は削除） |
| コメント粒度 | **行単位**（file + line_range に anchor） |
| コメント対象 | 現在の diff（session base SHA → HEAD）に現れる行のみ |
| 上書きされた行 | 自動的に diff から消えるので、コメント対象にならない |
| 送信 | 未送信コメントを束ねて Claude ペインへ 1 プロンプト |
| ブロッキング | Claude Code 自身の permission system に委ねる |

### 4.6 保存レイアウト

| 項目 | 決定 |
|------|------|
| 既定パス | `~/.local/state/tmux-agent-log/<project-slug>/` |
| 設定で変更可 | `~/.config/tmux-agent-log/config.toml` で上書き |
| クリアコマンド | `tmux-agent-log clear --session <name>` / `--all` / `--older-than 7d` |
| hook 設定 | `~/.claude/settings.json` にグローバルに記述（プロジェクトごとの設定は不要） |

### 4.7 tmux / UI 統合

UX は**ユーザー選択制**。ツールは以下を提供するのみ:

| 提供物 | 説明 |
|------|------|
| `tmux-agent-log tui` | フル機能の対話 TUI（bubbletea）。起動コンテキストを問わない |
| `tmux-agent-log tail` | tail viewer 単体。起動コンテキストを問わない |
| CLI プリミティブ | list-*/show-*/comment/goal/clear/export、全 machine-readable |
| `examples/tmux/` | popup / split-pane / 専用 window の .tmux.conf 例 |
| `examples/fzf/` | fzf 合成スクリプト例 |
| `install-hooks` サブコマンド | `~/.claude/settings.json` の対話的書き換え、差分プレビュー付き |

**強制しないもの**:
- キーバインド（ユーザーが examples から選んで手動で設定）
- popup の利用（選べるが強制しない）
- hook 設定（明示的に `install-hooks` を叩くまで何もしない）

### 4.8 エミュレータ OSC 機能

すべて opt-in 活用（対応エミュレータなら使う、非対応なら silently no-op）。

- **OSC 9**: review フラグ付き turn や長時間待ちに通知
- **OSC 8**: TUI 内のファイル名を cmd-click でエディタが開くリンクに
- **OSC 52**: turn 要約 / diff をクリップボードへ

---

## 5. Warp から学んだ追加要素

Warp Agents 3.0 の Code Review / Interactive Code Review から反映する 4 点。

### 5.1 `goal` を first-class 概念に（最重要）

ユーザーの「**目的ベースで**」という要件と合致。各セッションに 1 つの goal を宣言可能にし、タイムライン・横断ビューの主語を **セッション名ではなく goal** にする。

```sh
# セッション開始時
$ tmux-agent-log goal "2700 の認可バグ修正"

# 既存セッションに遡及
$ tmux-agent-log goal --current "DB マイグレーションの実装"

# goal を終了（完了マーク）
$ tmux-agent-log goal --done
```

横断ビューでの表示例:

```
 myproject    🎯 2700 の認可バグ修正         12 turns  [2 review]
 dotfiles   🎯 mise の設定を zsh から分離       3 turns
 nb         (no goal)                           5 turns
```

### 5.2 Diff の多重 base ビュー（2 モード設計）

**モード 1: 累積 diff**（session base → HEAD）
- 表示は**常に生きている行のみ**（diff の定義上、上書きされた行は出てこない）
- デフォルトビュー
- コメント可能

**モード 2: per-turn diff**（turn 別 patch）
- 個別 turn の diff を見る
- **liveness マーカー付き**: 後続 turn に上書きされた行は灰色＋斜体で表示、コメント不可
- 生きている行には通常通りコメント可能
- コメントは**現在の HEAD 位置に anchor**（turn 時点の行番号ではなく、comments.jsonl の構造は不変）

**追加モード（設定次第）**:
- **vs-main**: `main` ブランチとの diff（PR プレビュー相当）

切替: `b` キーでモード巡回、per-turn モードでは `shift-j/k` で turn を切替。

### 5.3 行単位のインラインコメント

Warp の inline comment を観察ツールとしてそのまま採用。**現在の diff（session base → HEAD）** に対して、特定の行範囲にフリーテキストのコメントを付けられる。

```
 @@ src/auth/middleware.go:42-48 @@
 +     if u == nil {
 +         return ErrUnauthenticated
 +     }
   ─ 💬 "ここの nil check は本当に必要? 後で要確認"
```

- 粒度: ファイル + 行範囲
- 対象: 現在の diff に現れる行のみ（上書きされた行は自動的に対象外）
- 永続化: `comments.jsonl`（`{ id, file, line_start, line_end, text, created_at, sent_at }`）
- UI: diff viewer で行選択 → `c` でコメント入力、`e` で編集、`d` で削除
- CLI: `tmux-agent-log comment add --file ... --line ... "text"` でも追加可能（オプション）

### 5.3b 補助: line → turn blame

diff の行にカーソルを合わせると、**その行を追加・変更した turn とその intent** をフッター表示:

```
@@ src/auth/middleware.go @@
+     if u == nil {
+         return ErrUnauthenticated
+     }
────────────────────────────────────────────────
Added in turn #7 (16:42)
Intent: "認可判定前に user=nil で panic する可能性があったため"
[↵] jump to timeline  [c] add comment
```

本プロジェクトの中核的な価値。**「差分に意味と時間軸がない」** 問題への直接解。

実装: 各 turn の patch を順に走査し、現在の diff に残っている各 `file:line` を最後に touch した turn を記録。

### 5.4 ★ レビュー結果を Claude session に送り返す（核心機能）

Warp の最も優れた点。tmux では **`tmux send-keys` 標準装備で簡単に実装可能**。

動作フロー:

1. diff view で行にコメントを付ける（複数可）
2. `s` キー → 固定テンプレートで即送信、`S` キー → `$EDITOR` でプレビュー編集して送信
3. `tmux send-keys -t <claude_pane> -l "<プロンプト>" \; send-keys Enter` で該当ペインに流し込む
4. 送信済みコメントは `sent_at` をセットして半透明表示（二重送信防止）
5. Claude は通常のユーザー入力として受け取り、次 turn で反映

**送信プロンプトのテンプレート（固定、MVP）**:

```
以下のレビューコメントを反映してください:

- src/auth/middleware.go:44-46
  ここの nil check は本当に必要?

- src/routes/api.go:108-112
  エラーハンドリングが単純すぎる

- src/db/query.go:45-52
  このループは O(n²) になっているのでは?

(反映後、関連テストを実行して結果を報告してください)
```

**送信先ペインが閉じている場合**: エラー画面にせず、プロンプト本文をクリップボード（OSC 52）にコピーして手で貼れるようにフォールバック。

Warp との差別化ポイント:

- **tmux 内であれば Claude Code 以外（Codex CLI / Aider / Gemini CLI）にも送れる** — agent 非依存
- **エミュレータ非依存** — Ghostty / WezTerm / iTerm2 どれでも動く
- **新規 IPC 不要** — tmux 標準機能のみ

### 5.5 Syntax highlight

`alecthomas/chroma` で Go 側から ANSI 出力。tmux を経由してそのまま表示される。ファイル拡張子から言語検出、テーマは設定可能（デフォルトは nord / cyberdream などユーザーの tmux テーマに寄せる）。

---

## 6. コンポーネント構成（ドラフト）

```
┌───────────────────────────────────────────────┐
│ tmux-agent-log (Go single binary)             │
│                                               │
│  ├─ hook-handler   ← Claude Code hooks から   │
│  │                   subprocess で呼ばれる    │
│  │                                            │
│  ├─ tail-viewer    ← 常設ペインで             │
│  │                   直近 turn を 1 行要約追尾 │
│  │                                            │
│  ├─ popup-tui      ← prefix+a で全画面        │
│  │                   (bubbletea + lipgloss)   │
│  │    ├ timeline                              │
│  │    ├ diff viewer (3 base + syntax HL)      │
│  │    └ overview (全セッション横断)           │
│  │                                            │
│  └─ cli            ← goal / note / clear /    │
│                      attach / export / config │
└───────────────────────────────────────────────┘
         ↑                              ↓
  ~/.claude/settings.json        ~/.local/state/tmux-agent-log/
    (hook config)                    ├ projects/<slug>/sessions/<uuid>/
                                     │   ├ meta.json
                                     │   ├ turns.jsonl
                                     │   ├ events.jsonl
                                     │   ├ comments.jsonl
                                     │   └ diffs/<turn_id>.patch
                                     ├ registry.json
                                     └ config.toml
```

---

## 7. データモデル（ドラフト・要詳細設計）

### 7.1 session meta

```jsonc
// meta.json
{
  "claude_session_id": "a1b2c3d4-...",
  "tmux_pane": "%42",
  "cwd": "/Users/.../myproject",
  "goal": "2700 認可バグ修正",   // 単なる文字列、空なら null
  "base_sha": "abcd1234",
  "started_at": "..."
}
```

### 7.2 turn

```
{
  "id": "turn-0001",
  "started_at": "...",
  "ended_at": "...",
  "user_prompt_preview": "...",   // 先頭 2 行
  "assistant_summary_preview": "...",
  "head_sha": "ef567890",
  "diff_path": "diffs/turn-0001.patch",
  "status": "done | error"        // review は削除
}
```

### 7.3 event（turn 内の tool 呼び出し）

```
{
  "id": "evt-00042",
  "turn_id": "turn-0001",
  "ts": "...",
  "tool": "Edit",
  "args_preview": "src/auth/middleware.go ... (truncated)",
  "success": true
}
```

### 7.4 comment（行単位インラインコメント）

```
{
  "id": "cmt-xyz",
  "file": "src/auth/middleware.go",
  "line_start": 44,
  "line_end": 46,
  "text": "ここの nil check は本当に必要?",
  "created_at": "...",
  "sent_at": null                  // send 後にタイムスタンプ
}
```

**goal は独立エンティティにしない**（meta.json の string フィールドのみ）。

### 7.5 storage の append-only ルール

turns.jsonl / events.jsonl / comments.jsonl はすべて **append-only**。既存行を書き換えない。

1 つの turn について open / close を別レコードで追記:

```jsonl
{"id":"turn-0007","phase":"open","started_at":"16:42","prompt_preview":"..."}
{"id":"turn-0007","phase":"close","ended_at":"16:43","head_sha":"ef567","status":"done"}
```

- TUI / CLI 読み込み時に同じ id の行を合流させる（最後のレコードが勝つ）
- 並行書込みで壊れない、途中クラッシュ耐性あり
- flock で同一ファイルへの書込みを直列化

### 7.6 blame cache と liveness map

turn-end hook 時に以下を増分更新:

**blame.json**: 現在の HEAD の各行 → 最後にその行を touch した turn_id

```jsonc
{
  "src/auth/middleware.go:44": "turn-0007",
  "src/auth/middleware.go:45": "turn-0007"
}
```

**liveness.json**: 各 turn の diff の各行 → 生死 + 現在位置

```jsonc
{
  "turn-0008": {
    "src/auth/middleware.go": {
      "42-44": { "alive": false, "overwritten_by": "turn-0009" },
      "45":    { "alive": true,  "head_line": 46 }
    }
  }
}
```

TUI の per-turn diff 表示で liveness.json を引いて生死を可視化。

---

## 8. Hook 配線案（ドラフト）

`~/.claude/settings.json` に追加するイメージ:

```json
{
  "hooks": {
    "SessionStart":      [{"matcher": "*", "command": "tmux-agent-log hook session-start"}],
    "UserPromptSubmit":  [{"matcher": "*", "command": "tmux-agent-log hook turn-start"}],
    "PreToolUse":        [{"matcher": "*", "command": "tmux-agent-log hook tool-pre"}],
    "PostToolUse":       [{"matcher": "*", "command": "tmux-agent-log hook tool-post"}],
    "Stop":              [{"matcher": "*", "command": "tmux-agent-log hook turn-end"}]
  }
}
```

各 hook は `stdin` で JSON を受け取り、該当 JSONL に追記する。起動時間はおよそ 10–30ms（Go バイナリ）。

---

## 9. TUI レイアウト（ドラフト・ASCII）

### 9.1 常設ペイン（tail viewer）

```
╭── tmux-agent-log tail ─────────────────────╮
│ 🎯 2700 の認可バグ修正                  │
│                                            │
│ #5 Bash  go test ./auth/...        ✓       │
│ #6 Edit  src/auth/middleware.go    +12 -3  │
│ #7 Edit  src/auth/middleware.go    +4  -1  │
│ #8 Bash  go test ./auth/...        ✓       │
│                                            │
│ [review] 2 未確認 / note 1                 │
╰────────────────────────────────────────────╯
```

### 9.2 Popup TUI

```
╭── tmux-agent-log ─────────────────────────────────────────╮
│ [timeline] [diff] [overview]          myproject 🎯 2700 │
├───────────────────────────────────────────────────────────┤
│                                                           │
│  > #7  16:42  Edit  src/auth/middleware.go    [review]    │
│        ├ intent:  "nil チェックを追加"                     │
│        ├ reason:  "認可判定前に user=nil で panic する可  │
│        │          能性があったため"                        │
│        └ events:  Read×2, Edit×1                          │
│                                                           │
│    #8  16:43  Bash  go test ./auth/...         ✓          │
│    #9  16:45  Edit  src/routes/api.go          +21 -4     │
│                                                           │
│  ─────────────────────────────────────────────────────    │
│  [j/k] move  [enter] detail  [c] note  [s] send-to-claude │
│  [g] goal    [o] overview    [/] filter                   │
╰───────────────────────────────────────────────────────────╯
```

### 9.3 Diff view（3 base 切替）

```
╭── tmux-agent-log > diff ──────────────────────────────────╮
│ [turn-diff] goal-diff | vs-main                           │
│ turn #7 の変更  src/auth/middleware.go   +4 -1            │
├───────────────────────────────────────────────────────────┤
│  @@ -42,7 +42,10 @@                                       │
│   func authorize(u *User, r *Request) error {             │
│ +     if u == nil {                                       │
│ +         return ErrUnauthenticated                       │
│ +     }                                                   │
│       if !u.HasRole(r.Scope) {                            │
│           return ErrForbidden                             │
│       }                                                   │
├───────────────────────────────────────────────────────────┤
│  💬 "ここの nil check は本当に必要? 後で要確認"            │
│  [c] edit note   [d] delete note                          │
╰───────────────────────────────────────────────────────────╯
```

### 9.4 Overview（横断ビュー）

```
╭── tmux-agent-log > overview ──────────────────────────────╮
│  session    goal                       turns   review note│
│                                                           │
│  myproject    🎯 2700 認可バグ修正     12      2      1  │
│  dotfiles   🎯 mise 設定分離              3      0      0  │
│  nb         (no goal)                     5      0      0  │
│  portal     🎯 API rate limit 導入        7      1      3  │
│                                                           │
│  [enter] open session   [n] new goal                      │
╰───────────────────────────────────────────────────────────╯
```

---

## 10. CLI 設計（プリミティブ中心）

```sh
# インタラクティブ UI（起動コンテキスト不問）
tmux-agent-log tui                                # フル TUI
tmux-agent-log tail                               # tail viewer 単体

# 読み取り系（全て --format tsv|jsonl|json|table 対応）
tmux-agent-log list-sessions  [--goal <str>] [--format ...]
tmux-agent-log list-turns     [--session <id>] [--format ...]
tmux-agent-log list-comments  [--session <id>] [--unsent] [--format ...]
tmux-agent-log show-session   <session-id>
tmux-agent-log show-turn      <turn-id> [--with-diff]
tmux-agent-log show-diff      <session-id> [--base session|turn|main]
tmux-agent-log blame          <file> <line>

# 書き込み系
tmux-agent-log goal           "<title>"            # 現セッションの goal 設定
tmux-agent-log comment add    --file F --line S-E --text "..."
tmux-agent-log comment send   [--preview] [--editor]
tmux-agent-log comment list   [--unsent]
tmux-agent-log comment delete <id>

# ライフサイクル
tmux-agent-log clear          --session <id> | --older-than 7d | --all
tmux-agent-log export         --session <id> --format md

# セットアップ（明示的 OPT-IN）
tmux-agent-log install-hooks        [--dry]       # ~/.claude/settings.json を対話的に編集
tmux-agent-log uninstall-hooks                    # 設定を元に戻す

# 設定
tmux-agent-log config         show | path | edit

# 内部（Claude Code hooks から呼ばれる）
tmux-agent-log hook           <event>
```

全 read 系コマンドが machine-readable 出力をサポートすることで、**fzf / jq / delta / bat / gum 等と自然に合成可能**。

---

## 11. MVP スコープ（提案・未確定）

**含める**:

- [x] hook 配線（SessionStart / UserPromptSubmit / PreToolUse / PostToolUse / Stop）
- [x] session meta / turn / event / comment の JSONL 永続化
- [x] git-native diff スナップショット（turn 境界）
- [x] `tmux-agent-log tail` サブコマンド（tail viewer 単体、起動コンテキスト不問）
- [x] `tmux-agent-log tui` サブコマンド（timeline / diff / overview の 3 タブ、起動コンテキスト不問）
- [x] `install-hooks` / `uninstall-hooks` サブコマンド（~/.claude/settings.json の OPT-IN 編集）
- [x] `examples/tmux/` に popup / split-pane / 専用 window の設定例
- [x] `examples/fzf/` に fzf 合成スクリプト例
- [x] 全 read 系 CLI の `--format tsv|jsonl|json|table` 対応
- [x] diff の 2 モード切替（累積 / per-turn）、per-turn モードは liveness マーカー付き
- [x] vs-main モード（PR プレビュー相当、オプション）
- [x] blame.json / liveness.json の増分更新（turn-end hook）
- [x] **行単位インラインコメント**（diff viewer 上で行選択 → `c` で入力）
- [x] **line → turn blame**（diff 行にカーソル → 追加した turn と intent をフッター表示）
- [x] ★ send-to-claude（`s` 固定テンプレ即送信 / `S` $EDITOR でプレビュー編集）
- [x] 送信済みコメントの半透明表示（sent_at）
- [x] 送信先ペイン消失時は OSC 52 クリップボードに fallback
- [x] goal 設定 CLI（`goal "<title>"`）
- [x] clear CLI
- [x] syntax highlight（chroma）

**含めない（拡張）**:

- [ ] 要約器（Haiku 等で intent/reason を 1 行要約）: interface は切るが初期実装は「ユーザープロンプト先頭 2 行」
- [ ] 非 git プロジェクト対応（file-tracking / shadow commit）
- [ ] turn 単位の note（行コメントで代替）
- [ ] review 自動フラグ（heuristic 判定）
- [ ] OSC 9 通知
- [ ] Claude Code 以外の agent アダプタ
- [ ] TPM プラグイン wrapper
- [ ] redact フィルタ（API キー等のマスク）
- [ ] web UI / 外部サーバ
- [ ] コメントテンプレートの config 上書き（固定テンプレで MVP）

---

## 12. エラー処理ポリシー

### 原則

1. **hook は絶対に Claude Code を止めない**: 何があっても exit 0
2. **整合性 > リアルタイム性**: 書込み競合は flock 待ち（最大 500ms）、タイムアウト時は記録して諦める
3. **壊れた JSONL 行は読み飛ばす**: 行単位 parse、decode 失敗行は skip + カウント
4. **未知の hook event は無視**: 将来の Claude Code 拡張に対して silent no-op

### 主要な失敗モード

| シナリオ | 対応 |
|---|---|
| git リポジトリでない | meta に `git: false` 記録、以降 diff 生成は skip、JSONL 追記は継続 |
| git diff が空 or タイムアウト | diff_path=null で turn close、errors.jsonl に記録 |
| JSONL 書込み失敗（disk full 等） | stderr + errors.jsonl、hook は exit 0 |
| JSONL 行が壊れている | 読取側で skip、TUI に「N 件破損」インジケータ |
| target tmux pane が消えている | OSC 52 クリップボード fallback |
| stdin JSON が不正 | errors.jsonl、hook は exit 0 |
| transcript_path が読めない | ref=null で turn 保存、TUI で「参照不可」表示 |
| flock タイムアウト | 該当 hook 分 lost、errors.jsonl 記録 |
| tail viewer クラッシュ | recover + エラー表示 + 数秒後に自動再起動 |

### errors.jsonl

```jsonl
{"ts":"...","component":"hook/turn-end","event":"git-diff-failed","session_id":"...","error":"..."}
```

- 場所: `~/.local/state/tmux-agent-log/errors.jsonl`
- TUI 起動時にバナーで件数通知
- `tmux-agent-log errors list` / `errors clear` で操作可能

### hook 出力ポリシー

- **stdout**: 何も出さない（Claude Code が hook stdout を拾う仕様のため）
- **stderr**: エラー時のみ 1 行
- 通常動作情報はすべて errors.jsonl / debug log ファイルへ

---

## 13. テスト戦略

### レイヤー別テスト

| レイヤー | 種別 | 手段 | 重点 |
|---|---|---|---|
| storage（JSONL 永続化） | 単体 | Go table-driven | append-only 契約、flock 競合、壊れた行 skip、open/close 合流 |
| hook handler | 単体 | fixture JSON を stdin | 各 event 受理、未知フィールド無視、exit code 常に 0 |
| git 連携 | integration | tempdir で git init | diff 生成、base_sha 管理、非 git fallback |
| liveness / blame | 単体 | 時系列 patch 流入 | 上書き検知、行番号シフト |
| tmux send-keys | integration | `tmux new-session -d` | 実送信 + `capture-pane`、pane 消失で OSC 52 fallback |
| TUI (bubbletea) | golden | `teatest` | キーバインド、タブ、コメント、送信プレビュー |
| CLI | 単体 + e2e | table test | フォーマット出力、exit code、引数 |
| install-hooks | 単体 | temp HOME | dry run、merge、uninstall の復帰 |

### 重点回帰テスト

1. 並行 hook 10 発で events.jsonl が全部残る
2. 壊れた JSONL 行を混ぜても読取継続
3. turn open 後に hook kill → 整合性保持、TUI で status=open 表示
4. git diff 空 → diff_path=null で close
5. 存在しない pane に send-keys → OSC 52 fallback
6. non-git project → 警告 + JSONL 追記のみ、TUI ではコメント disabled
7. turn A 追加 → turn B 上書き → turn A の該当行 `alive: false`
8. HEAD の行に blame → turn A の intent を返す

### CI / 配布

- GitHub Actions: Ubuntu + macOS マトリクス、`go test -race`、`golangci-lint`
- GoReleaser で darwin/linux の amd64/arm64 バイナリを GitHub Release に
- Homebrew tap を GoReleaser が更新
- Windows native は CI しない（WSL2 暗黙サポート）

### カバレッジ目標

- storage / hook: **80%+**
- git 連携: **70%+**
- TUI: **50%+**（golden で主要遷移）
- CLI: **70%+**

---

## 14. 解決済み / 残項目

### 解決済み

1. ✅ **tmux session ↔ log directory の紐付け** — Claude Code hook の `session_id`（UUID）を primary key、保存先 `projects/<slug>/sessions/<uuid>/`、`TMUX_PANE` は `meta.json` に副次情報として記録
2. ✅ **send-to-claude の形式** — `s` 固定テンプレ即送 / `S` $EDITOR プレビュー編集、送信済みは半透明、ペイン消失時は OSC 52 fallback
3. ✅ **review モデル** — 自動フラグ廃止、log は垂れ流し、review は人間が行単位コメントで行う
4. ✅ **goal の永続範囲** — session meta の string フィールドのみ、独立エンティティなし

### 残項目（設計セクション提示時に提案して確定）

5. **overview ビューの再生成タイミング** — hook 書込み時 / ファイル watcher / TUI 開いた時のどれか
6. **requirement / non-goals の明文化** — "MVP で含めないこと"を明確化
7. **テスト戦略** — hook のユニットテスト / TUI のゴールデンテスト / git 操作の integration test

---

## 15. 次のステップ

1. 本ドキュメントを `docs/superpowers/specs/2026-04-23-tmux-agent-log-design.md` に最終 spec として昇格
2. superpowers:writing-plans スキルで実装計画を作成
3. superpowers:executing-plans または subagent-driven-development で実装

---

## 付録 A: 既存ツール調査（2026-04-22）

観察・履歴系で近い既存ツール:

| ツール | 言語 | 近さ | 本プロジェクトに対して欠けている要素 |
|---|---|---|---|
| [Agent of Empires](https://github.com/njbrake/agent-of-empires) | Rust | ★★★★★ | 構造化履歴（turn/event/intent/reason）、goal、multi-base diff、turn-level note、review → agent 送信 |
| [claude-code-log](https://github.com/daaain/claude-code-log) | Python | ★★★☆☆ | tmux 統合、diff 扱い、note、goal、send-back |
| [multi-agent-observability](https://github.com/disler/claude-code-hooks-multi-agent-observability) | Python + Bun + Vue | ★★★☆☆ | tmux-native、diff、note、send-back（Web ベース） |
| [Claudoscope](https://github.com/cordwainersmith/Claudoscope) | macOS native | ★★★☆☆ | クロスプラットフォーム、tmux 統合、send-back |
| [workmux](https://github.com/raine/workmux) / [dmux](https://dmux.ai/) / [agtx](https://github.com/fynnfluegge/agtx) | — | ★★☆☆☆ | 観察層としての機能（これらは orchestrator、役割が違う） |
| [eyes-on-claude-code](https://github.com/joe-re/eyes-on-claude-code) | — | ★★☆☆☆ | tmux 統合、goal、構造化履歴 |

**結論**: tmux-native observation + goal first + multi-base diff + turn-level note + send-review-back の**組合せ**を持つツールは存在しない。差別化は成立する。

位置付け:
- AoE（Rust）: dispatcher として採用、本プロジェクトと併用
- それ以外: 参考にするが置き換えない

---

*本ドキュメントは brainstorming 段階の合意メモ。確定 spec ではない。*
