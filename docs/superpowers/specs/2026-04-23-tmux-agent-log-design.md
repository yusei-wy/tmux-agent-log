# tmux-agent-log 設計仕様書

- **作成日:** 2026-04-23（ブレインストーミング開始 2026-04-22、version 戦略改訂 2026-05-04）
- **ステータス:** Final（version 戦略確定、v0.1 から実装開始）
- **ソースのブレインストーミング**: [`docs/brainstorming/2026-04-22-tmux-agent-log.md`](../../brainstorming/2026-04-22-tmux-agent-log.md)
- 本ドキュメントは `superpowers:brainstorming` セッションで合意した決定事項をまとめたもの。`superpowers:writing-plans` への入力となる。

---

## 0. プロジェクトの要点

実装中にブレないための北極星。詳細は §3〜§11 を参照。

### 0.1 目的 (Goal)

AI agent が tmux 内で行った作業を **構造化されたログ** として残し、後から振り返って **頭の中で再構築できる** 状態を作る。"成果物だけが残って思考が消える" 問題を解消し、ログを起点に Claude session へ **直接フィードバック** できるようにする。

具体的には次の 3 点を達成する:

1. **振り返り可能性** — 各セッションで AI が何をしたか、その意図とともに辿れる
2. **構造化された理解** — ログを追うことで「変更のストーリー」を脳内で再構築できる
3. **直接フィードバック** — ログを見ながらコメントを付け、そのまま Claude session に送り返せる

### 0.2 制約 (Constraints) — 引き算の哲学

- **観察と意味付けに専念** — 書込・取消は git に委ねる。直接編集 / hunk revert / discard all は持たない（§5.0 参照）
- **不満は必ず言語化** — `R` Refine 等の "再ガチャ" 動線は提供せず、コメントによる言語化を強制する（§5.0 参照）
- **全部やろうとしない** — orchestration / worktree 管理 / Docker サンドボックスはやらない（§3.1 参照）
- **UX を強制しない** — TUI の起動方法・キーバインド・ショートカットはユーザー任せ。`examples/` として提示するのみで、`.tmux.conf` も `~/.claude/settings.json` も自動書き換えしない（§3.2 参照）
- **上書き済みコードへのコメントは不可** — 現在の diff（session base → HEAD）に存在しない行はコメント対象にならない。後続 turn で書き換えられた行は自動的に diff から消える（§4.5 参照）

### 0.3 完了条件 (v0.1 リリース)

v0.1 リリース可能と判断する 3 条件:

1. **構造化保存** — Claude Code hook 経由で `session / turn / event / comment` が JSONL として `~/.local/state/tmux-agent-log/` 配下にローカル保存されている
2. **構造化表示** — `tmux-agent-log tui`(timeline / diff [累積 read-only] / overview) と `tail` / `list-*` / `show-*` 系 CLI が、保存された JSONL を読み込んで表示できる
3. **コメント送信** — diff viewer 上で行にコメントを付け、`s` キーで Claude pane に `tmux send-keys` 経由で送り返せる（送信先 pane 消失時は OSC 52 クリップボード fallback）

詳細な v0.1 スコープは §11、各機能の決定根拠は §4 / §5 を参照。

### 0.4 Version 戦略

完了主義（各 version で「使える状態」で区切る）:

| v | テーマ | 主機能 | 完了条件 |
|---|---|---|---|
| **v0.1** | Core | hook 配線 / JSONL / timeline / diff (read-only 累積) / 行コメント / 送信 / `goal` / `tail` / `install-hooks` | 自分の作業ログを見て、行コメントを Claude に送れる |
| **v0.2** | Semantic | per-turn diff + liveness marker / `vs-main` / blame footer + intent | 行 → turn → intent の縦串が見える |
| **v0.3** | Storytelling | overview タブ (goal 横断) / `narrate` CLI (Markdown 出力) / chroma による syntax HL | 振り返って頭の中で再構築できる |
| ~~v0.4~~ | ~~Comment Authoring~~ | **保留** — 実使用 + Claude / Ghostty の動向で再判断（付録 B 参照） | — |

v0.2 以降は v0.1 を使ってみてから優先度を実体験で再評価する。

---

## 1. 課題（What）

tmux ベースの並列 AI 開発において、以下がボトルネックになっている。

- 各セッションで「何をしているか」が把握できない
- 差分（`git diff`）は見えるが「なぜそうなったか」が分からない
- AI 生成コードのため、思考プロセスが追えず構造化できない
- どのプロセスがどこで動いているか分からない

完了通知は別仕組みで解決済み。残課題は **進行中の状態と意味の把握**。

## 2. 問題の本質（Why）

> **差分に「意味」と「時間軸」が存在しない**

現状の情報構造:

| 情報源 | 何を示すか |
|--------|------------|
| `git diff` | 結果（What changed） |
| `tmux` | 実行状態（What is running） |

欠けているもの:

- AI の判断（Why）
- 変更の意図（Intent）
- ステップの連続性（History）

結果として「変更のストーリー」が存在せず、脳内で再構築できない。

## 3. 解決方針（How）

差分に意味を与え、変更をストーリーとして扱う **観察層** を追加する。tmux はそのまま、Claude Code はそのまま。不足している「思考の記録・意図の構造化・差分との接続」だけを足す。

### 3.1 スコープと他ツールとの関係

本プロジェクトは **tmux 内で動く AI agent（主に Claude Code）の観察層** に機能を絞る。

- ✅ **含める:** 構造化履歴・goal 管理・multi-base diff・行単位コメント・review → Claude 送り返し
- ❌ **含めない:** セッション起動管理・worktree 管理・Docker サンドボックス・サイドバー表示・エージェント起動

ユーザーは普段通りに tmux ペインで `claude` を起動する。本ツールは Claude Code hooks 経由で黙って observable なレイヤーを構築する。

### 3.2 OSS 設計原則

tmux / Claude Code ユーザーは設定を強くカスタムしている前提で、**UX を押し付けない**:

- **No forced keybindings** — `.tmux.conf` を自動で書き換えない。examples として提示するのみ
- **No forced hook install** — `~/.claude/settings.json` を自動で変更しない。`install-hooks` サブコマンドで明示的に OPT-IN
- **No forced UI mode** — popup / split-pane / 新規 window / fzf 合成 / CLI 専用 / 外部ターミナル、どれで動かしても機能する設計
- **CLI primitive first** — すべての read 系は `--format tsv|jsonl|json`、write 系はスクリプトから呼べる形
- **fzf / delta / bat / gum 等と自然合成可能** — clean stdout / stdin

既存 dispatcher（AoE 等）への依存もせず、**単体で完結して動く**。既存ツールとの関係は付録 A を参照。

---

## 4. 合意済み決定事項

ブレインストーミング中に確定した項目。

### 4.1 対象範囲

| 項目 | 決定 | 備考 |
|------|------|------|
| 対象 AI | **Claude Code のみ（MVP）** | 将来の拡張性を持つアダプタ構造を維持 |
| 対象プロジェクト | **git 管理されたリポジトリ** | 非 git プロジェクトは起動時に警告のみ、hooks は no-op |
| クライアント | **tmux 内に閉じる（エミュレータ非依存）** | Ghostty / WezTerm / iTerm2 等には依存しない |
| 実装言語 | **Go 単一バイナリ（純 Go）** | hook handler / tail viewer / TUI / CLI をすべて内蔵。shell スクリプト不使用。`tmux` / `git` 以外のランタイム依存なし。OSS 配布時は `brew install` / `go install` で完結 |

### 4.2 ログ収集方式

| 項目 | 決定 |
|------|------|
| 戦略 | **完全パッシブ + 閲覧時要約（選択肢 4）** |
| 情報源 | Claude Code hooks + transcript JSONL（参照のみ、本文は複製しない） |
| AI 側への format 契約 | **なし** — CLAUDE.md 汚染なし、slash command 追加なし |
| 要約 | 閲覧時にオンデマンド実行（MVP は intent としてユーザープロンプトの先頭 2 行を表示。要約器は差し替え可能な interface） |

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
| 方式 | **git-native のみ（MVP）** |
| 記録方法 | セッション開始時に `base_sha` を記録。各ターン終了時に `git diff <base_sha>..HEAD` + unstaged 差分を保存 |
| 非 git プロジェクト | 起動時に 1 度警告、hook は no-op（拡張は将来） |
| セキュリティ | 保存ディレクトリは mode `0700`、transcript 本体は複製せず参照のみ、redact フィルタは後付け可能 |

### 4.5 レビューモデル

**log は完全パッシブ（垂れ流し）。判断しない。review は人間が diff に対して行単位で行う。**

| 項目 | 決定 |
|------|------|
| 自動 review フラグ | **廃止** — heuristic 判定なし |
| turn の status | `running / done / error` のみ（`review` は削除） |
| コメント粒度 | **行単位**（`file + line_range` に anchor） |
| コメント対象 | 現在の diff（session base SHA → HEAD）に現れる行のみ |
| 上書きされた行 | 自動的に diff から消えるので、コメント対象にならない |
| 送信 | 未送信コメントを束ねて Claude ペインへ 1 プロンプトで送る |
| ブロッキング | Claude Code 自身の permission system に委ねる |

### 4.6 保存レイアウト

| 項目 | 決定 |
|------|------|
| 既定パス | `~/.local/state/tmux-agent-log/<project-slug>/` |
| 設定で変更可 | `~/.config/tmux-agent-log/config.toml` で上書き |
| クリアコマンド | `tmux-agent-log clear --session <id>` / `--all` / `--older-than 7d` |
| hook 設定 | `~/.claude/settings.json` にグローバルに記述 — プロジェクトごとの設定は不要 |

### 4.7 tmux / UI 統合

UX は **ユーザー選択制**。ツールは以下のみを提供する:

| 提供物 | 説明 |
|------|------|
| `tmux-agent-log tui` | フル機能の対話 TUI（bubbletea）。起動コンテキストを問わない |
| `tmux-agent-log tail` | tail viewer 単体。起動コンテキストを問わない |
| CLI プリミティブ | `list-*` / `show-*` / `comment` / `goal` / `clear` / `export` — すべて machine-readable |
| `examples/tmux/` | popup / split-pane / 専用 window の `.tmux.conf` 例 |
| `examples/fzf/` | fzf 合成スクリプト例 |
| `install-hooks` サブコマンド | `~/.claude/settings.json` の対話的書き換え、差分プレビュー付き |

**強制しないもの:**
- キーバインド（ユーザーが examples から選んで手動で設定）
- popup の利用（選べるが強制しない）
- hook 設定（明示的に `install-hooks` を叩くまで何もしない）

### 4.8 ターミナル OSC 機能

すべて opt-in: 対応エミュレータなら使う、非対応なら silently no-op。

- **OSC 9:** review トリガーで通知。*（先送り — review フラグを廃止したため発火条件がなくなった）*
- **OSC 8:** TUI 内のファイル名を `file://` ハイパーリンクに。Ghostty / iTerm2 / WezTerm で cmd-click でエディタが開く
- **OSC 52:** turn 要約 / diff のシステムクリップボード yank。`tmux set-clipboard on` 設定で透過動作

---

## 5. Warp から学んだ要素と引き算の哲学

Warp Agents 3.0 の Code Review / Interactive Code Review から本ツールに採用した点と、**意図的に採らなかった点** をまとめる。

### 5.0 引き算の哲学（最重要）

Warp の機能のうち、以下は **意図的に削除**。流行を作るには完全再現ではなく、制約こそが哲学になる。

| Warp の機能 | 削る理由 |
|---|---|
| `R` Refine（コメント書かずに再生成依頼） | **不満は言語化を強制**。再ガチャ動線は agent に同じ失敗を繰り返させる |
| `E` 直接編集（diff を panel 内で書換） | tmux pane でエディタを開けば済む。書込は **git の責務** |
| hunk 単位 revert | `git checkout -p` で済む |
| Discard all / file 単位 discard | `git restore` で済む |
| AI コメント生成補助 | コメント自体はユーザーの思考。代行しない |

→ 本ツールは **観察と意味付けに専念**。書込・取消は外部委譲（git / エディタ / Claude）。

以下、Warp の Code Review / Interactive Code Review から **採用** した 4 点。

### 5.1 `goal` を first-class 概念に（最重要）

ユーザーの「目的ベースで」という要件と合致。各セッションに 1 つの goal を宣言可能にし、タイムライン・横断ビューの主語を **セッション名ではなく goal** にする。

```sh
# 現セッションの goal を設定 / 更新
$ tmux-agent-log goal "2700 認可バグ修正"

# 現在の goal を表示
$ tmux-agent-log goal
```

- goal はセッションの `meta.json` 内の単一文字列フィールド（§4.5 / §7.1 参照）
- first-class エンティティとしてはモデル化しない。完了概念もなし（`--done` なし）
- 同じ文字列を複数セッションで使える。横断ビューでは自然にグループ化される

横断ビューの表示例:

```
 myproject    🎯 2700 認可バグ修正            12 turns
 dotfiles   🎯 mise の設定を zsh から分離       3 turns
 nb         (no goal)                          5 turns
```

### 5.2 Diff の多重 base ビュー（version 配分付き）

**v0.1 — 累積 diff**（session base → HEAD）
- 表示は **常に生きている行のみ**（diff の定義上、上書きされた行は出てこない）
- デフォルトビュー、コメント可能、**read-only**

**v0.2 — per-turn diff**（turn 別 patch）+ liveness マーカー
- 個別 turn の patch を見る
- **liveness マーカー:** 後続 turn で上書きされた行は灰色 + 斜体で表示、コメント不可
- 生きている行には通常通りコメント可能。コメントは **現在の HEAD 位置に anchor**（turn 時点の行番号ではなく、`comments.jsonl` の構造は不変）

**v0.2 — `vs-main`** — `main` ブランチとの diff（PR プレビュー相当）

操作: `b` でモード巡回、per-turn モードでは `shift-j/k` で turn 切替。

### 5.3 行単位のインラインコメント（v0.1）

Warp の inline comment 機能を観察用に特化。**現在の diff（session base → HEAD）** に対して、特定の行範囲にフリーテキストのコメントを付けられる。

```
 @@ src/auth/middleware.go:42-48 @@
 +     if u == nil {
 +         return ErrUnauthenticated
 +     }
   ─ 💬 "ここの nil check は本当に必要? 後で要確認"
```

- **粒度:** ファイル + 行範囲
- **対象:** 現在の diff に現れる行のみ（上書きされた行は自動的に対象外）
- **永続化:** `comments.jsonl` — `{ id, file, line_start, line_end, text, created_at, sent_at }`
- **UI:** diff viewer で行選択 → `c` でコメント入力、`e` で編集、`d` で削除
- **CLI:** `tmux-agent-log comment add --file ... --line ... --text "..."`（任意）

### 5.3b 補助: line → turn blame（v0.2）

diff の行にカーソルを合わせると、**その行を追加・変更した turn とその intent** をフッター表示:

```
@@ src/auth/middleware.go @@
+     if u == nil {
+         return ErrUnauthenticated
+     }
────────────────────────────────────────────────
Added in turn #7 (16:42)
Intent: "認可判定前に user=nil で panic する可能性があったため"
[↵] jump to timeline   [c] add comment
```

本プロジェクトの中核的な価値。**「差分に意味と時間軸がない」** 問題への直接解。

実装: 各 turn の patch を順に走査し、現在の diff に残っている各 `file:line` を最後に touch した turn を記録。

### 5.4 ★ レビュー結果を Claude session に送り返す（v0.1 核心機能）

Warp の最も優れた点。tmux では `tmux send-keys` 標準装備で簡単に実装可能。

動作フロー:

1. diff view で行にコメントを付ける（複数可）
2. `s` — 固定テンプレートで即送信。`S` — `$EDITOR` でプレビュー編集してから送信
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

**送信先ペインが消えている場合:** エラー画面にせず、プロンプト本文を OSC 52 でシステムクリップボードにコピーしてフォールバック。

tmux 上で Warp より優れる点:

- **tmux 内で動く任意の agent に送れる**（Claude Code / Codex CLI / Aider / Gemini CLI / …） — agent 非依存
- **エミュレータ非依存** — Ghostty / WezTerm / iTerm2 どれでも動く
- **新規 IPC 不要** — tmux 標準機能のみ

**意図的に持たない動線（§5.0 参照）**:
固定テンプレ送信のみ。「`R` Refine」のような曖昧な再生成依頼は提供しない。不満は必ずコメントで言語化する。

### 5.5 Syntax highlight

- **v0.1**: 最低限の diff 色分け（追加 = 緑、削除 = 赤、context = 通常）。言語別 HL は持たない
- **v0.3**: `alecthomas/chroma/v2` で Go 側から ANSI を直接出力。tmux はそのまま通す。ファイル拡張子から言語検出、テーマは設定可能（デフォルトはユーザーの tmux テーマに寄せる: 例 nord / cyberdream）

---

## 6. コンポーネント構成

```
┌───────────────────────────────────────────────┐
│ tmux-agent-log（Go 単一バイナリ）              │
│                                               │
│  ├─ hook handler   ← Claude Code hooks から   │
│  │                   subprocess で呼ばれる    │
│  │                                            │
│  ├─ tail viewer    ← 直近 turn を 1 行要約で  │
│  │                   tail 表示                │
│  │                                            │
│  ├─ popup TUI      ← オンデマンドで全画面 TUI │
│  │                   （bubbletea + lipgloss） │
│  │    ├ timeline                              │
│  │    ├ diff viewer (3 base + syntax HL)      │
│  │    └ overview (全セッション横断)           │
│  │                                            │
│  └─ cli            ← goal / comment / clear / │
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

## 7. データモデル

### 7.1 session meta

```jsonc
// meta.json
{
  "claude_session_id": "a1b2c3d4-...",
  "tmux_pane": "%42",
  "cwd": "/Users/.../myproject",
  "goal": "2700 認可バグ修正",   // 単なる文字列、空でもよい
  "base_sha": "abcd1234",
  "started_at": "...",
  "git_tracked": true,
  "transcript_path": "..."
}
```

### 7.2 turn

```jsonc
{
  "id": "turn-0001",
  "started_at": "...",
  "ended_at": "...",
  "user_prompt_preview": "...",        // 先頭 2 行
  "assistant_summary_preview": "...",
  "head_sha_pre": "abc...",
  "head_sha": "def...",
  "diff_path": "diffs/turn-0001.patch",
  "status": "open | done | error"      // `review` はなし
}
```

### 7.3 event（turn 内の tool 呼び出し）

```jsonc
{
  "id": "evt-00042",
  "turn_id": "turn-0001",
  "ts": "...",
  "tool": "Edit",
  "args_preview": "src/auth/middleware.go ... (truncated)",
  "phase": "pre | post",
  "success": true
}
```

### 7.4 comment（行単位インラインコメント）

```jsonc
{
  "id": "cmt-xyz",
  "file": "src/auth/middleware.go",
  "line_start": 44,
  "line_end": 46,
  "text": "ここの nil check は本当に必要?",
  "created_at": "...",
  "sent_at": null
}
```

**`goal` は独立エンティティとしてモデル化しない**（`meta.json` の string フィールドのみ）。

### 7.5 storage の append-only ルール

`turns.jsonl` / `events.jsonl` / `comments.jsonl` はすべて **append-only**。既存レコードを書き換えない。

1 つの turn は open / close の 2 レコードで記録する:

```jsonl
{"id":"turn-0007","phase":"open","started_at":"16:42","prompt_preview":"..."}
{"id":"turn-0007","phase":"close","ended_at":"16:43","head_sha":"ef567","status":"done"}
```

- 読取側（TUI / CLI）は同じ id の行を合流させる（最後のレコードが勝つ）
- 並行書込みで壊れない、途中クラッシュ耐性あり
- 同一ファイルへの並行書込みは `flock` で直列化

### 7.6 blame cache と liveness map

turn-end hook 時に増分更新。

**`blame.json`** — 現在の HEAD の各行 → 最後にその行を touch した turn:

```jsonc
{
  "src/auth/middleware.go:44": "turn-0007",
  "src/auth/middleware.go:45": "turn-0007"
}
```

**`liveness.json`** — 各 turn の diff の各行 → 生死 + 現在位置:

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

TUI の per-turn diff モードは `liveness.json` を引いてマーカーを描画する。

---

## 8. Hook 配線

### 追加内容

`~/.claude/settings.json` に以下のスタンザを追加する（**自動インストールはしない**。§3.2 のとおり `install-hooks` で OPT-IN）:

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

### インストールフロー

ユーザーは以下のいずれかを実行:

```sh
tmux-agent-log install-hooks          # 差分表示 → y/n プロンプト → 書込み
tmux-agent-log install-hooks --dry    # 差分表示のみ
tmux-agent-log uninstall-hooks        # 自前のエントリだけクリーンに削除
```

`settings.json` に既存の hook がある場合、自前のエントリは各イベント配列に追記マージされる。`install-hooks` は冪等（重複しない）。

各 hook は stdin から JSON を受け取り、該当 JSONL に追記する。起動時間は ~10–30 ms（Go バイナリ）。stdin が壊れていても exit 0（§12 エラー処理）。

---

## 9. TUI レイアウト（ASCII モックアップ）

TUI は起動コンテキスト不問（§3.2 / §4.7: popup / split-pane / 新規 window どれでも動く）。下記モックアップは意図を示すためのもの。

### 9.1 tail viewer（`tmux-agent-log tail`）

```
╭── tmux-agent-log tail ─────────────────────╮
│ 🎯 2700 認可バグ修正                    │
│                                            │
│ #5 Bash  go test ./auth/...        ✓       │
│ #6 Edit  src/auth/middleware.go    +12 -3  │
│ #7 Edit  src/auth/middleware.go    +4  -1  │
│ #8 Bash  go test ./auth/...        ✓       │
│                                            │
│ comments: 0 unsent                          │
╰────────────────────────────────────────────╯
```

### 9.2 TUI — timeline タブ（`tmux-agent-log tui`）

```
╭── tmux-agent-log ─────────────────────────────────────────╮
│ [timeline] diff  overview             myproject 🎯 2700 │
├───────────────────────────────────────────────────────────┤
│                                                           │
│  > #7  16:42  Edit  src/auth/middleware.go    +4 -1       │
│        ├ intent:  "nil チェックを追加"                     │
│        ├ reason:  "認可判定前の panic 防止"                │
│        └ events:  Read×2, Edit×1                          │
│                                                           │
│    #8  16:43  Bash  go test ./auth/...         ✓          │
│    #9  16:45  Edit  src/routes/api.go          +21 -4     │
│                                                           │
│  ─────────────────────────────────────────────────────    │
│  [tab] switch tab  [j/k] move  [enter] drill down         │
│  [s] send comments   [/] filter   [q] quit                │
╰───────────────────────────────────────────────────────────╯
```

### 9.3 TUI — diff タブ（2 モード）

**累積 diff（session base → HEAD、デフォルト）:**

```
╭── tmux-agent-log > diff ──────────────────────────────────╮
│ timeline [diff] overview     base: [session] per-turn vs-main │
├───────────────────────────────────────────────────────────┤
│  Files                    │  src/auth/middleware.go       │
│  ▶ middleware.go +16 -3   │  ────────────────────────     │
│    api.go        +21 -4   │  @@ -42,7 +42,10 @@           │
│                           │   func authorize(u *User) {   │
│                           │ +     if u == nil {           │
│                           │ +         return ErrUnauth    │
│                           │ +     }                       │
│                           │                               │
│                           │  Added in turn #7 (16:42)     │
│                           │  Intent: "panic 防止"         │
├───────────────────────────────────────────────────────────┤
│  [c] comment on line(s)  [b] cycle base  [s] send  [q]    │
╰───────────────────────────────────────────────────────────╯
```

**per-turn diff（liveness マーカー付き）:**

```
╭── tmux-agent-log > diff ──────────────────────────────────╮
│ timeline [diff] overview   base: session [per-turn #8] vs-main │
├───────────────────────────────────────────────────────────┤
│  turn #8 @ 16:18  Edit src/auth/middleware.go             │
│                                                           │
│  @@ -42,7 +42,10 @@                                       │
│   func authorize(u *User) error {                         │
│ +    if u == nil || u.ID == 0 {        ░░ overwritten #9  │
│ +        return ErrUnauthenticated     ░░ overwritten #9  │
│ +    }                                  ░░ overwritten #9  │
│      if !u.HasRole(r.Scope) {              ← alive        │
│          return ErrForbidden                              │
│                                                           │
├───────────────────────────────────────────────────────────┤
│  [c] comment (alive only)  [shift-j/k] prev/next turn     │
╰───────────────────────────────────────────────────────────╯
```

### 9.4 TUI — overview タブ（横断ビュー）

```
╭── tmux-agent-log > overview ──────────────────────────────╮
│  by goal  [by session]                                    │
│                                                           │
│  🎯 2700 認可バグ修正                                  │
│    myproject/<uuid-1>     12 turns   last 5m ago  🟢         │
│    myproject/<uuid-2>      6 turns   last 2h ago  🔵         │
│  🎯 DB マイグレーション                                    │
│    myproject/<uuid-3>      3 turns   last 10m ago 🟢         │
│  (no goal)                                                 │
│    nb/<uuid-4>           5 turns   last 1d ago             │
│                                                           │
│  [enter] open session   [/] filter                        │
╰───────────────────────────────────────────────────────────╯
```

---

## 10. CLI 設計（プリミティブ中心）

```sh
# 対話 UI（起動コンテキスト不問）
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
tmux-agent-log goal           "<title>"
tmux-agent-log comment add    --file F --line S-E --text "..."
tmux-agent-log comment send   [--preview] [--editor]
tmux-agent-log comment list   [--unsent]
tmux-agent-log comment delete <id>

# ライフサイクル
tmux-agent-log clear          --session <id> | --older-than 7d | --all
tmux-agent-log export         --session <id> --format md

# セットアップ（明示的 OPT-IN）
tmux-agent-log install-hooks        [--dry]
tmux-agent-log uninstall-hooks

# 設定 / 診断
tmux-agent-log config         show | path | edit
tmux-agent-log errors         list | clear

# 内部（Claude Code hooks から呼ばれる）
tmux-agent-log hook           <event>
```

全 read 系コマンドが machine-readable 出力をサポートすることで、**fzf / jq / delta / bat / gum** などと自然に合成可能。

---

## 11. v0.1 スコープ（確定）

v0.2 以降のスコープは §0.4、v0.4 保留分は付録 B、引き算した項目は §5.0 を参照。

### 11.1 v0.1 含める

- [x] hook 配線（SessionStart / UserPromptSubmit / PreToolUse / PostToolUse / Stop）
- [x] session meta / turn / event / comment の JSONL 永続化
- [x] git-native diff スナップショット（turn 境界）
- [x] `tmux-agent-log tail`（tail viewer 単体、起動コンテキスト不問）
- [x] `tmux-agent-log tui`(timeline / diff [累積 read-only] / overview の 3 タブ)
- [x] `install-hooks` / `uninstall-hooks`（`~/.claude/settings.json` の OPT-IN 編集）
- [x] `examples/tmux/` に popup / split-pane / 専用 window のスニペット
- [x] `examples/fzf/` に合成スクリプト
- [x] 全 read 系 CLI の `--format tsv|jsonl|json|table` 対応
- [x] **累積 diff のみ**（session base → HEAD、コメント可、read-only）
- [x] **行単位インラインコメント**（diff viewer で行選択 → `c`）
- [x] ★ `send-to-claude`（`s` で固定テンプレ即送信、`S` で `$EDITOR` プレビュー編集）
- [x] 送信済みコメントの半透明表示（`sent_at` 経由）
- [x] 送信先 pane 消失時の OSC 52 クリップボード fallback
- [x] `goal` CLI（`goal "<title>"`）
- [x] `clear` CLI
- [x] 最低限の diff 色分け（追加 = 緑、削除 = 赤、context = 通常）

### 11.2 v0.1 含めない（後続バージョンへ移動）

*v0.2 で追加（Semantic）:*
- [ ] per-turn diff モード + liveness マーカー
- [ ] `vs-main` モード（PR プレビュー）
- [ ] `blame.json` / `liveness.json` の増分更新（turn-end hook）
- [ ] **line → turn blame footer**（フッターで turn と intent を表示）

*v0.3 で追加（Storytelling）:*
- [ ] overview タブ（goal 横断ビュー）の本格実装
- [ ] `narrate` CLI（goal 単位の変更ストーリー Markdown 出力）
- [ ] chroma による言語別 syntax highlight チューニング

*v0.4 で再検討（Comment Authoring、保留 — 付録 B）:*
- [ ] コメント入力時の `@<file>` `@<dir>/` `/<skill>` `/<command>` 補完
- [ ] Markdown プレビュー画面

### 11.3 永久 non-goal（引き算の哲学 §5.0）

- [ ] `R` Refine 動線（コメントを書かずに再生成依頼）
- [ ] `E` 直接編集（diff を panel 内で書換）
- [ ] hunk 単位 revert / discard all / file 単位 discard
- [ ] AI コメント生成補助

### 11.4 永久 non-goal（観察層スコープ外）

*ツール機能:*
- [ ] エージェント orchestration — dispatcher（AoE / dmux / agtx）の責務
- [ ] git worktree / Docker サンドボックス管理
- [ ] ブロッキング実行（PreToolUse 拒否 — Claude Code の permission system に委ねる）
- [ ] 自動 review フラグ（heuristic）— 行コメントで人間が判断する
- [ ] AI による intent / reason 要約器（v0.1 は user prompt 先頭 2 行。要約器は interface として後で差し替え可能）
- [ ] 非 git プロジェクトの file-tracking diff（起動時警告で graceful-degrade）
- [ ] shadow commit（全ツリー snapshot）— サイズと安全性のリスクが高すぎる
- [ ] hunk 単位のコメント分割（v0.1 はファイル + 行範囲のみ）
- [ ] ユーザー設定可能なコメントテンプレート（v0.1 は組込みテンプレート）
- [ ] turn 単位の note（行コメントで代替可能）
- [ ] Claude Code 以外の agent アダプタ（拡張可能な構造は維持、v0.1 は Claude Code のみ）
- [ ] OSC 9 通知（review フラグ廃止により発火条件なし）
- [ ] プロジェクト横断 goal（goal はプロジェクトスコープ）
- [ ] secret redaction フィルタ（v0.1 は `0700` 保護のみ。redaction はアドオン）
- [ ] transcript 本文の複製（`transcript_path` + message id 参照のみ）
- [ ] リアルタイムコラボ / マルチユーザー対応
- [ ] リモートアクセス / チーム共有

*UX:*
- [ ] `.tmux.conf` の自動編集
- [ ] `~/.claude/settings.json` の自動編集（`install-hooks` の OPT-IN を必須）
- [ ] 強制デフォルトキーバインド
- [ ] popup を前提とする TUI（起動コンテキスト不問）
- [ ] マウス操作（キーボード優先、後で追加可能）
- [ ] TPM プラグイン wrapper（examples で代替）
- [ ] Web UI / 外部サーバ

*プラットフォーム:*
- [ ] ネイティブ Windows（WSL2 経由の暗黙対応のみ）
- [ ] 古い tmux（`display-popup` は tmux 3.2+ 必須）

*セキュリティ:*
- [ ] マルチユーザーアクセス制御（`~/.local/state/tmux-agent-log/` の `0700` のみ）
- [ ] ログ暗号化（同一ユーザー前提の plaintext）
- [ ] 監査ログの改ざん検知

---

## 12. エラー処理ポリシー

### 原則

1. **hook は絶対に Claude Code を止めない** — 何があっても exit 0
2. **整合性 > リアルタイム性** — 書込み競合は `flock` 待ち（最大 500 ms）。タイムアウト時は記録して諦める
3. **壊れた JSONL 行は読み飛ばす** — 行単位 parse、decode 失敗行は skip + カウント
4. **未知の hook event は無視** — 将来の Claude Code 拡張に対して silent no-op

### 主な失敗モード

| シナリオ | 対応 |
|---|---|
| git リポジトリでない | `meta.git_tracked = false`、以降 diff 生成は skip、JSONL 追記は継続 |
| `git diff` が空 / タイムアウト | `diff_path = null` で turn close、`errors.jsonl` に記録 |
| JSONL 書込み失敗（disk full 等） | stderr + `errors.jsonl`、hook は exit 0 |
| 壊れた JSONL 行 | 読取側で skip、TUI に「N 件破損」インジケータ |
| target tmux pane が消えている | OSC 52 クリップボード fallback |
| stdin JSON が不正 | `errors.jsonl`、hook は exit 0 |
| `transcript_path` が読めない | `transcript_ref = null` で turn 保存、TUI で「参照不可」表示 |
| flock タイムアウト | 該当 hook 分 lost、`errors.jsonl` 記録 |
| tail viewer クラッシュ | recover + エラー表示 + 数秒後に自動再起動 |

### `errors.jsonl`

```jsonl
{"ts":"...","component":"hook/turn-end","event":"git-diff-failed","session_id":"...","error":"..."}
```

- 場所: `~/.local/state/tmux-agent-log/errors.jsonl`
- TUI 起動時にバナーで件数通知
- `tmux-agent-log errors list` / `errors clear` で操作

### hook 出力ポリシー

- **stdout:** 何も出さない（Claude Code が hook stdout を拾うため、書込みが干渉する可能性あり）
- **stderr:** エラー時のみ 1 行
- それ以外は `errors.jsonl` / debug log ファイルへ

---

## 13. テスト戦略

### レイヤー別テスト

| レイヤー | 種別 | 手段 | 重点 |
|---|---|---|---|
| storage（JSONL） | 単体 | Go table-driven | append-only 契約、flock 競合、壊れた行 skip、open/close 合流 |
| hook handler | 単体 | stdin fixture JSON | 各 event 受理、未知フィールド無視、exit code 常に 0 |
| git 連携 | integration | `tempdir` + `git init` | diff 生成、`base_sha` 管理、非 git fallback |
| liveness / blame | 単体 | 時系列 patch 流入 | 上書き検知、行番号シフト |
| tmux send-keys | integration | `tmux new-session -d` で一時 socket | 実送信 + `capture-pane` 検証、pane 消失で OSC 52 fallback |
| TUI（bubbletea） | golden | `teatest` | キーバインド、タブ切替、コメント editor、send プレビュー |
| CLI | 単体 + e2e | table test + `go test` | 出力フォーマット、exit code、引数検証 |
| install-hooks | 単体 | temp `HOME` | dry-run、既存 settings との merge、`uninstall` の clean 復帰 |

### 重点回帰テスト（必ず通す）

1. 並行 hook 10 発で `events.jsonl` に 10 件残る
2. 壊れた JSONL 行を混ぜても読取継続（skip + count）
3. turn-open 後に hook を `kill -9` しても整合性保持、TUI で `status=open`
4. `git diff` 空 → close レコードで `diff_path = null`
5. 存在しない pane に send-keys → OSC 52 fallback（エラー終了しない）
6. 非 git プロジェクト: 起動時警告、JSONL 追記は動作、TUI ではコメント disabled
7. turn A で追加 → turn B で上書き → turn A の diff で該当行が `alive: false`
8. HEAD の生きている行に blame → turn A の intent 文字列を返す

### CI と配布

- GitHub Actions マトリクス: Ubuntu + macOS、`go test -race`、`golangci-lint`
- GoReleaser で `darwin/linux` × `amd64/arm64` を GitHub Releases に publish
- Homebrew tap を GoReleaser が更新
- Windows native の CI なし（WSL2 暗黙サポート）

### カバレッジ目標

- storage / hook: **80% 以上**
- git 連携: **70% 以上**
- TUI: **50% 以上**（主要遷移を golden で）
- CLI: **70% 以上**

---

## 14. 設計決定の総括

ブレインストーミングで確定した主要決定:

1. ✅ **対象 AI:** Claude Code（v0.1）。アダプタ seam は維持
2. ✅ **ログ収集:** 完全パッシブ + 閲覧時要約（AI 側への format 契約なし）
3. ✅ **step 粒度:** ハイブリッド二層（外側 = turn、内側 = tool 呼び出し）
4. ✅ **diff スナップショット:** git-native のみ（非 git は graceful-degrade）
5. ✅ **レビューモデル:** 人間駆動の行単位コメント、自動フラグなし
6. ✅ **保存先:** `~/.local/state/tmux-agent-log/`（XDG 準拠、上書き可）
7. ✅ **session ↔ log 紐付け:** Claude `session_id`（UUID）が primary key
8. ✅ **goal:** session meta の plain string フィールド（first-class エンティティではない）
9. ✅ **diff base:** 3 モード — 累積（v0.1）/ per-turn 付き liveness（v0.2）/ vs-main（v0.2）
10. ✅ **send-to-Claude:** `s` で固定テンプレ送信、`S` で `$EDITOR` プレビュー、OSC 52 fallback
11. ✅ **UI:** 起動コンテキスト不問（popup / split-pane / window 全対応）、OSS の "強制しない" 原則
12. ✅ **実装:** Go 単一バイナリ、ランタイム依存は `tmux` + `git` のみ
13. ✅ **エラー処理:** hook は常に exit 0、壊れた JSONL は skip
14. ✅ **overview の再生成:** TUI 起動時に lazy compute（fsnotify なし、SQLite 移行は将来検討）
15. ✅ **テスト:** レイヤー別に 単体 / integration / golden、CI は Linux + macOS
16. ✅ **引き算の哲学:** Refine / 直接編集 / hunk revert / discard / AI コメント生成補助は永久 non-goal（§5.0）
17. ✅ **Version 戦略:** v0.1 Core → v0.2 Semantic → v0.3 Storytelling、v0.4 は保留（§0.4 / 付録 B）

---

## 15. 次のステップ

1. 本 spec のレビュー → 承認
2. **v0.1 の実装計画** を `superpowers:writing-plans` で作成
3. `superpowers:executing-plans` または `subagent-driven-development` で v0.1 実装
4. v0.1 を自己 dogfooding（自分の作業で 2〜4 週間使う）
5. v0.2 / v0.3 の優先度を実体験で再評価
6. v0.4 を再判断（Claude Code / Ghostty の動向次第）

---

## 付録 A: 既存ツール調査

### A.1 初回調査（2026-04-22）

観察 / 履歴系で近い既存ツール:

| ツール | 言語 | 近さ | 本プロジェクトに対して欠けている要素 |
|---|---|---|---|
| [Agent of Empires](https://github.com/njbrake/agent-of-empires) | Rust | ★★★★★ | 構造化履歴（turn/event/intent/reason）、goal、multi-base diff、turn-level note、review → agent 送信 |
| [claude-code-log](https://github.com/daaain/claude-code-log) | Python | ★★★☆☆ | tmux 統合、diff 扱い、コメント、goal、send-back |
| [multi-agent-observability](https://github.com/disler/claude-code-hooks-multi-agent-observability) | Python + Bun + Vue | ★★★☆☆ | tmux-native、diff、コメント、send-back（Web ベース） |
| [Claudoscope](https://github.com/cordwainersmith/Claudoscope) | macOS native | ★★★☆☆ | クロスプラットフォーム、tmux 統合、send-back |
| [workmux](https://github.com/raine/workmux) / [dmux](https://dmux.ai/) / [agtx](https://github.com/fynnfluegge/agtx) | — | ★★☆☆☆ | 観察層としての機能（これらは orchestrator、役割が違う） |
| [eyes-on-claude-code](https://github.com/joe-re/eyes-on-claude-code) | — | ★★☆☆☆ | tmux 統合、goal、構造化履歴 |

### A.2 追加調査（2026-05-04）

| ツール | 言語 | 近さ | 本プロジェクトに対して欠けている要素 |
|---|---|---|---|
| [revdiff](https://revdiff.com/) | 不明 | ★★★★★ | goal、構造化履歴（turn/event/intent）、per-turn liveness、後追いタイムライン、常時パッシブ収集 |
| [agentdiff](https://github.com/sunilmallya/agentdiff) | Python | ★★★★☆ | TUI、行コメント、agent 送り返し、tmux 統合、goal、per-turn liveness |
| [agentblame](https://github.com/mesa-dot-dev/agentblame) | TS | ★★★☆☆ | TUI、行コメント、send-back、tmux、per-turn liveness |
| [Plannotator](https://plannotator.ai/) | 不明 | ★★★☆☆ | tmux 統合、構造化履歴、後追いタイムライン |
| [opcode](https://github.com/winfunc/opcode) | Tauri (Rust+TS) | ★★★☆☆ | tmux ネイティブ、行コメント、send-back |
| [agents-observe](https://github.com/simple10/agents-observe) | TS | ★★★☆☆ | 行コメント、send-back、tmux |
| [Claude Code 公式 `/diff`](https://blog.vincentqiao.com/en/posts/claude-code-diff/) | (内蔵) | ★★★☆☆ | 行コメント、blame、goal、send-back、liveness |
| [Selvedge](https://glama.ai/mcp/servers/masondelan/selvedge) | MCP | ★★☆☆☆ | TUI、tmux、行コメント、send-back |
| [review-for-agent](https://github.com/Waraq-Labs/review-for-agent) | Go+TS | ★★☆☆☆ | tmux ネイティブ、構造化履歴、自動 send-back |

### A.3 結論（更新後）

revdiff が「行コメント + send-back + tmux ネイティブ + multi-base diff + blame」を既に提供。agentdiff は「Claude Code hook + 行レベル intent + git notes」を実装済み。Claude Code 公式は per-turn diff を内蔵している。

**それでも本プロジェクトの差別化は依然として有効**:
- **goal-first 横断ビュー** — 他になし
- **観察層 + レビュー機能の統合** — 同じデータモデルで両立しているツールはない
- **per-turn diff の liveness marker** — 公式にも revdiff にもない
- **後追い再構築（narrate）** — 「変更のストーリー」を Markdown 出力するツールは未確認

位置付け:
- AoE（Rust）: dispatcher として採用、本プロジェクトと併用
- revdiff / agentdiff: 実装着手前に試用し、本プロジェクトとの併用 / 役割分担を確認
- それ以外: 参考にするが置き換えない

---

## 付録 B: 将来検討事項

### B.1 v0.4 — Comment Authoring（保留）

コメント入力時に Claude Code 同等の補完を提供する案:

| 接頭辞 | 補完元 | 用途 |
|---|---|---|
| `@<path>` | `git ls-files` + 修正行のあるファイル優先 | ファイル参照 |
| `@<dir>/` | ディレクトリ列挙 | 範囲指定 |
| `/<skill>` | `~/.claude/skills/*/SKILL.md` | スキル呼び出し |
| `/<command>` | `~/.claude/commands/*.md` + プロジェクト固有 | コマンド呼び出し |
| `#<turn>` | 現セッションの turn-id | turn 参照 |
| `!<comment-id>` | 既存 unsent コメント | コメント引用 |

**保留理由**:
1. 「Claude 相当」を目指すと vim mode 等の差で **逆に迷うユーザー** が出る
2. Claude Code / Ghostty 側の進化で **不要になる可能性**
3. v0.1〜v0.3 を使ってみないと **本当に必要か** 判断できない

**判断時期**: v0.3 リリース後、3 ヶ月使用したタイミングで再評価。

### B.2 引き算した項目への一貫した代替手段

§5.0 で削除した機能は、すべて既存ツールで代替可能:

| 削除した動線 | 代替 |
|---|---|
| Refine（再生成） | コメント `c` で言語化 → `s` 送信（強制動線） |
| 直接編集 | tmux pane でエディタ起動 / Claude に依頼 |
| hunk revert | `git checkout -p <file>` |
| discard all | `git restore .` / `git stash` |
| AI コメント生成補助 | Claude pane で「このコードの懸念点を挙げて」と聞く |

ユーザーが代替手段を選ぶ自由を確保することで、ツールの責務を狭く保つ。

---

*本ドキュメントはブレインストーミングセッションから導出された確定設計である。v0.1 から `superpowers:writing-plans` への入力となる。*
