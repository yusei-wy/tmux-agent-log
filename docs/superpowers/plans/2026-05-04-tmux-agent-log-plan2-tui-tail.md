# tmux-agent-log Plan 2: v0.1 完成 (TUI + tail viewer) 実装計画

> **エージェント実装者向け:** 必須サブスキル: `superpowers:subagent-driven-development`（推奨）または `superpowers:executing-plans` を使ってタスク単位で実装する。各 step はチェックボックス（`- [ ]`）で進捗管理する。

**位置づけ:** spec §0.4 の Version 戦略における **v0.1 (Core)** の **Phase B**。Plan 1（CLI コア）が完了している前提で、対話 TUI と tail viewer を追加し、**v0.1 リリース可能** な状態にする。

**目的:** Plan 1 で構築した JSONL ストアと CLI の上に、bubbletea ベースの対話 TUI（timeline / diff [累積 read-only] / overview の 3 タブ）と fsnotify ベースの tail viewer を構築する。これにより spec §0.3 の v0.1 完了条件 2（構造化表示）が満たされ、v0.1 完成。

**v0.1 全体の完了条件**（spec §0.3、再掲）:
1. 構造化保存（hook 経由 JSONL 永続化）— **Plan 1 で完了**
2. 構造化表示（TUI + `tail` + 各種 read 系 CLI）— **本計画 (Plan 2) で完成**
3. コメント送信（`s` で send-keys、OSC 52 fallback）— Plan 1 で CLI 経由は完了、本計画で TUI 経由を追加

**アーキテクチャ:** bubbletea (Elm 風 Model-Update-View) で TUI を構築。各タブは独立した Model として実装し、root Model がタブ切替を制御する。tail viewer は fsnotify でファイル変更を watch し、最新 N turn を再描画する。Plan 1 で作った `storage` / `git` / `tmux` パッケージを再利用し、新規ロジックは `internal/tui` と `internal/tail` に局所化する。`os` / `exec` 系の呼び出しは adapter 層 (`tail` / `tui`) の中だけに留める。

**追加技術スタック:**
- `github.com/charmbracelet/bubbletea` — Elm 風 TUI フレームワーク
- `github.com/charmbracelet/lipgloss` — スタイル定義
- `github.com/charmbracelet/bubbles` — 既製コンポーネント（viewport, list, textarea）
- `github.com/fsnotify/fsnotify` — ファイル変更通知

**ソース spec:** [`docs/superpowers/specs/2026-04-23-tmux-agent-log-design.md`](../specs/2026-04-23-tmux-agent-log-design.md) — 特に §0.3 (v0.1 完了条件) / §0.4 (Version 戦略) / §5.0 (引き算の哲学) / §5.3 (行コメント) / §5.4 (送り返し) / §9 (TUI レイアウト) / §11.1 (v0.1 含める)

**前提（Plan 1 で完了済み）:**
- `internal/storage/`: `ReadTurns` / `ReadComments` / `UnsentComments` / `ReadSessionMeta` / `AppendComment` / `MarkCommentsSent` / `DeleteComment` / `ReadTurnDiff`
- `internal/git/`: `DiffSince(cwd, baseSHA)` / `Run(cwd, args...)`
- `internal/tmux/`: `SendToPane(pane, prompt) → SendResult` (OSC 52 fallback 内蔵)
- `internal/config/`: `StateDir` / `SessionDir` / `ProjectSlug` / `ErrorsPath`
- `internal/cli/`: `findSessionDir(sessionID)` ヘルパー
- `internal/format/`: 共通の出力フォーマッタ
- `internal/errlog/`: 失敗の記録先

**この計画 (Plan 2) のスコープ外:**

*v0.2（後続 Plan）:*
- per-turn diff モード + liveness マーカー
- `vs-main` モード切替（ただし show-diff CLI には Plan 1 の時点で `--base=main` がある）
- `blame.json` / `liveness.json` の増分更新
- line → turn blame footer + intent

*v0.3（後続 Plan）:*
- chroma による言語別 syntax highlight
- overview タブ（goal 横断ビュー）の本格実装（v0.1 は最低限のグルーピング表示のみ）
- `narrate` CLI（goal 単位の変更ストーリー Markdown 出力）

*v0.4（保留）:*
- コメント入力時の `@<file>` / `/<skill>` 等の補完、Markdown プレビュー

*永久 non-goal*（spec §11.3 / §5.0 引き算の哲学）:
- `R` Refine（再生成依頼動線）
- `E` 直接編集 / hunk revert / discard 系
- AI コメント生成補助

**記述言語:** 本計画で作成するコード・コメント・テスト・コミットメッセージ・ドキュメントはすべて **日本語** で記述する。ただし Go のテスト名（`TestXxx`）・関数名・型名など言語仕様上英語が必要な箇所は英語のままとする。

**コミット粒度:** Task ごとに独立した commit。コミットメッセージは Conventional Commits (`feat:`, `test:`, `refactor:` 等)、本文 1〜2 文で why を書く（CLAUDE.md / .claude/rules/git.md 参照）。

---

## ファイル構成（本計画で作成）

```
tmux-agent-log/
├── internal/
│   ├── cli/
│   │   ├── tui.go              # 新規: 'tui' サブコマンド
│   │   └── tail.go             # 新規: 'tail' サブコマンド
│   ├── tui/                    # 新規パッケージ
│   │   ├── style.go            # lipgloss style 定義（共有）
│   │   ├── keys.go             # キーバインド定義（共有）
│   │   ├── model.go            # root Model（タブ切替・サイズ・ステータス）
│   │   ├── timeline.go         # timeline タブ（turn 一覧 + 詳細）
│   │   ├── timeline_test.go
│   │   ├── diffview.go         # diff タブ（累積 read-only、行コメント）
│   │   ├── diffview_test.go
│   │   ├── overview.go         # overview タブ（goal 横断、簡易版）
│   │   ├── overview_test.go
│   │   ├── parse.go            # unified diff parser（TUI 内部）
│   │   ├── parse_test.go
│   │   ├── colorize.go         # ANSI 色分け（追加=緑/削除=赤/context=通常）
│   │   ├── colorize_test.go
│   │   ├── loader.go           # storage → view-ready 変換
│   │   └── loader_test.go
│   ├── tail/                   # 新規パッケージ
│   │   ├── tail.go             # fsnotify + 再描画ループ
│   │   ├── render.go           # 表示フォーマット
│   │   └── tail_test.go
│   └── ... (既存)
├── scripts/smoke.sh            # 更新（tui/tail を追加検証）
└── README.md                   # 更新（tui/tail 起動例）
```

---

## 共有データモデル

`internal/tui/parse.go` に置く unified diff の表現。Plan 1 の `internal/storage/model.go` の型はそのまま再利用する。

```go
package tui

type LineKind int

const (
    LineContext LineKind = iota
    LineAdd
    LineRemove
    LineHunkHeader  // "@@ -... +... @@"
    LineFileHeader  // "diff --git ...", "--- ...", "+++ ..." など
)

type Line struct {
    Kind LineKind
    // 元のテキスト（先頭の +/-/space を含む生の行）
    Raw string
    // diff 内の行番号（OldNo: a 側、NewNo: b 側、0 は無効）
    OldNo int
    NewNo int
}

type Hunk struct {
    Header string  // "@@ -... +... @@" の生文字列
    Lines  []Line  // 内部の行（Header は含まない）
}

type FileDiff struct {
    OldPath string  // a/...（NewFile の場合は "/dev/null"）
    NewPath string  // b/...（DeletedFile の場合は "/dev/null"）
    Hunks   []Hunk
}
```

---

## Phase 0: 依存追加

### Task 1: bubbletea / lipgloss / bubbles / fsnotify を追加

**ファイル:**
- 修正: `go.mod`, `go.sum`

- [ ] **Step 1:** 依存を追加:

  ```bash
  go get github.com/charmbracelet/bubbletea@latest
  go get github.com/charmbracelet/lipgloss@latest
  go get github.com/charmbracelet/bubbles@latest
  go get github.com/fsnotify/fsnotify@latest
  ```

- [ ] **Step 2:** `go mod tidy` で依存を正規化:

  ```bash
  go mod tidy
  ```

  期待: `go.mod` に 4 つの新規 require 行、`go.sum` が更新される。

- [ ] **Step 3:** ビルドで衝突しないことを確認:

  ```bash
  mise run check
  ```

  期待: fmt → lint → test が全て通る（既存テストは Plan 1 のもの）。

- [ ] **Step 4:** コミット:

  ```bash
  git add go.mod go.sum
  git commit -m "chore: TUI 用に bubbletea/lipgloss/bubbles/fsnotify を追加"
  ```

---

## Phase 1: diff パーサー（TUI 内部、共有モジュール）

`internal/tui/parse.go` と `colorize.go` は純粋関数のみ。bubbletea / lipgloss には依存しない（lipgloss は `style.go` のみで使う）。

### Task 2: diff の data model

**ファイル:**
- 作成: `internal/tui/parse.go`

**Exports:**
- `LineKind`, `LineContext`, `LineAdd`, `LineRemove`, `LineHunkHeader`, `LineFileHeader`
- `Line`, `Hunk`, `FileDiff`

- [ ] **Step 1:** 型定義のみのファイルを作る（実装は次タスク）:

  ```go
  package tui

  type LineKind int

  const (
      LineContext LineKind = iota
      LineAdd
      LineRemove
      LineHunkHeader
      LineFileHeader
  )

  type Line struct {
      Kind  LineKind
      Raw   string
      OldNo int
      NewNo int
  }

  type Hunk struct {
      Header string
      Lines  []Line
  }

  type FileDiff struct {
      OldPath string
      NewPath string
      Hunks   []Hunk
  }
  ```

- [ ] **Step 2:** `go build ./internal/tui` がコンパイル通ることを確認。

- [ ] **Step 3:** コミット:

  ```bash
  git add internal/tui/parse.go
  git commit -m "feat(tui): diff parser のデータ型を追加"
  ```

### Task 3: unified diff parser

**ファイル:**
- 修正: `internal/tui/parse.go`
- 作成: `internal/tui/parse_test.go`

**Exports:**
- `func ParseUnified(raw string) []FileDiff`

- [ ] **Step 1:** 失敗するテストを書く（external test package）:

  ```go
  package tui_test

  import (
      "testing"

      "github.com/stretchr/testify/require"

      "github.com/yusei-wy/tmux-agent-log/internal/tui"
  )

  func TestParseUnified(t *testing.T) {
      cases := []struct {
          name string
          in   string
          want []tui.FileDiff
      }{
          {
              name: "空入力",
              in:   "",
              want: nil,
          },
          {
              name: "1 ファイル / 1 hunk / 追加と削除",
              in: "diff --git a/foo.go b/foo.go\n" +
                  "--- a/foo.go\n" +
                  "+++ b/foo.go\n" +
                  "@@ -1,3 +1,4 @@\n" +
                  " package foo\n" +
                  "+\n" +
                  "+func Bar() {}\n" +
                  "-var x int\n",
              want: []tui.FileDiff{{
                  OldPath: "a/foo.go",
                  NewPath: "b/foo.go",
                  Hunks: []tui.Hunk{{
                      Header: "@@ -1,3 +1,4 @@",
                      Lines: []tui.Line{
                          {Kind: tui.LineContext, Raw: " package foo", OldNo: 1, NewNo: 1},
                          {Kind: tui.LineAdd, Raw: "+", OldNo: 0, NewNo: 2},
                          {Kind: tui.LineAdd, Raw: "+func Bar() {}", OldNo: 0, NewNo: 3},
                          {Kind: tui.LineRemove, Raw: "-var x int", OldNo: 2, NewNo: 0},
                      },
                  }},
              }},
          },
          {
              name: "新規ファイル",
              in: "diff --git a/new.go b/new.go\n" +
                  "new file mode 100644\n" +
                  "--- /dev/null\n" +
                  "+++ b/new.go\n" +
                  "@@ -0,0 +1,1 @@\n" +
                  "+package new\n",
              want: []tui.FileDiff{{
                  OldPath: "/dev/null",
                  NewPath: "b/new.go",
                  Hunks: []tui.Hunk{{
                      Header: "@@ -0,0 +1,1 @@",
                      Lines: []tui.Line{
                          {Kind: tui.LineAdd, Raw: "+package new", OldNo: 0, NewNo: 1},
                      },
                  }},
              }},
          },
      }
      for _, tc := range cases {
          t.Run(tc.name, func(t *testing.T) {
              got := tui.ParseUnified(tc.in)
              require.Equal(t, tc.want, got)
          })
      }
  }
  ```

- [ ] **Step 2:** テスト実行で「`ParseUnified` が無い」コンパイルエラーになることを確認:

  ```bash
  go test ./internal/tui/...
  ```

- [ ] **Step 3:** `internal/tui/parse.go` に最小実装を追加:

  ```go
  // ParseUnified は git diff 形式（unified diff）の生テキストを構造化する。
  // 行番号は header `@@ -a,b +c,d @@` から復元する。
  func ParseUnified(raw string) []FileDiff {
      if raw == "" {
          return nil
      }
      lines := strings.Split(raw, "\n")
      var files []FileDiff
      var cur *FileDiff
      var hunk *Hunk
      var oldNo, newNo int

      flushHunk := func() {
          if hunk != nil && cur != nil {
              cur.Hunks = append(cur.Hunks, *hunk)
              hunk = nil
          }
      }
      flushFile := func() {
          flushHunk()
          if cur != nil {
              files = append(files, *cur)
              cur = nil
          }
      }

      for _, ln := range lines {
          switch {
          case strings.HasPrefix(ln, "diff --git "):
              flushFile()
              cur = &FileDiff{}
          case strings.HasPrefix(ln, "--- "):
              if cur == nil {
                  cur = &FileDiff{}
              }
              cur.OldPath = strings.TrimPrefix(ln, "--- ")
          case strings.HasPrefix(ln, "+++ "):
              if cur == nil {
                  cur = &FileDiff{}
              }
              cur.NewPath = strings.TrimPrefix(ln, "+++ ")
          case strings.HasPrefix(ln, "@@"):
              flushHunk()
              hunk = &Hunk{Header: ln}
              oldNo, newNo = parseHunkStart(ln)
          case hunk != nil:
              switch {
              case strings.HasPrefix(ln, "+"):
                  hunk.Lines = append(hunk.Lines, Line{Kind: LineAdd, Raw: ln, NewNo: newNo})
                  newNo++
              case strings.HasPrefix(ln, "-"):
                  hunk.Lines = append(hunk.Lines, Line{Kind: LineRemove, Raw: ln, OldNo: oldNo})
                  oldNo++
              case ln == "" || strings.HasPrefix(ln, " "):
                  // 末尾の空行は出力しない
                  if ln == "" {
                      continue
                  }
                  hunk.Lines = append(hunk.Lines, Line{Kind: LineContext, Raw: ln, OldNo: oldNo, NewNo: newNo})
                  oldNo++
                  newNo++
              }
          }
      }
      flushFile()
      return files
  }

  // "@@ -a,b +c,d @@" → (a, c) を返す。
  func parseHunkStart(header string) (int, int) {
      // header 例: "@@ -42,7 +42,10 @@ funcName"
      open := strings.Index(header, "-")
      space := strings.Index(header, " +")
      end := strings.Index(header, " @@")
      if open < 0 || space < 0 || end < 0 {
          return 0, 0
      }
      a := header[open+1 : space]
      b := header[space+2 : end]
      if i := strings.Index(a, ","); i >= 0 {
          a = a[:i]
      }
      if i := strings.Index(b, ","); i >= 0 {
          b = b[:i]
      }
      return atoi(a), atoi(b)
  }

  func atoi(s string) int {
      n := 0
      for _, c := range s {
          if c < '0' || c > '9' {
              return n
          }
          n = n*10 + int(c-'0')
      }
      return n
  }
  ```

  必要に応じて先頭に `import "strings"` を追加。

- [ ] **Step 4:** テスト実行で全部通ることを確認:

  ```bash
  go test ./internal/tui/... -run TestParseUnified -v
  ```

  期待: 3 サブテスト全部 PASS。

- [ ] **Step 5:** コミット:

  ```bash
  git add internal/tui/parse.go internal/tui/parse_test.go
  git commit -m "feat(tui): unified diff parser を追加"
  ```

### Task 4: ANSI 色分け

**ファイル:**
- 作成: `internal/tui/colorize.go`
- 作成: `internal/tui/colorize_test.go`

**Exports:**
- `func RenderLine(l Line) string` — lipgloss でスタイル済みの 1 行を返す

- [ ] **Step 1:** 失敗するテスト:

  ```go
  package tui_test

  import (
      "strings"
      "testing"

      "github.com/yusei-wy/tmux-agent-log/internal/tui"
  )

  func TestRenderLine(t *testing.T) {
      cases := []struct {
          name      string
          line      tui.Line
          mustHave  string  // ANSI コード断片（例: "\x1b[32m"）
          rawText   string  // 行本体（色コード除外）
      }{
          {
              name:     "追加行は緑",
              line:     tui.Line{Kind: tui.LineAdd, Raw: "+func Bar() {}"},
              mustHave: "\x1b[",   // ANSI が出ていれば OK
              rawText:  "+func Bar() {}",
          },
          {
              name:     "削除行は赤",
              line:     tui.Line{Kind: tui.LineRemove, Raw: "-var x int"},
              mustHave: "\x1b[",
              rawText:  "-var x int",
          },
          {
              name:     "context は装飾なし",
              line:     tui.Line{Kind: tui.LineContext, Raw: " package foo"},
              mustHave: "",
              rawText:  " package foo",
          },
      }
      for _, tc := range cases {
          t.Run(tc.name, func(t *testing.T) {
              got := tui.RenderLine(tc.line)
              if !strings.Contains(got, tc.rawText) {
                  t.Fatalf("行本体が含まれない: got=%q want substring=%q", got, tc.rawText)
              }
              if tc.mustHave != "" && !strings.Contains(got, tc.mustHave) {
                  t.Fatalf("ANSI コードが含まれない: got=%q", got)
              }
          })
      }
  }
  ```

- [ ] **Step 2:** テスト実行で fail することを確認:

  ```bash
  go test ./internal/tui/... -run TestRenderLine
  ```

- [ ] **Step 3:** `internal/tui/colorize.go` に実装:

  ```go
  package tui

  import "github.com/charmbracelet/lipgloss"

  var (
      addStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))   // 緑
      removeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))   // 赤
      headerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
  )

  // RenderLine は diff の 1 行を ANSI 色付きで整形する。
  // 言語別 syntax HL は v0.3 で chroma を導入してから。
  func RenderLine(l Line) string {
      switch l.Kind {
      case LineAdd:
          return addStyle.Render(l.Raw)
      case LineRemove:
          return removeStyle.Render(l.Raw)
      case LineHunkHeader, LineFileHeader:
          return headerStyle.Render(l.Raw)
      default:
          return l.Raw
      }
  }
  ```

- [ ] **Step 4:** テスト通過確認:

  ```bash
  go test ./internal/tui/... -run TestRenderLine -v
  ```

  期待: 3 サブテスト全部 PASS。

- [ ] **Step 5:** コミット:

  ```bash
  git add internal/tui/colorize.go internal/tui/colorize_test.go
  git commit -m "feat(tui): diff 1 行の ANSI 色分けを追加 (追加=緑/削除=赤)"
  ```

---

## Phase 2: tail viewer

`tail` は単発描画ではなく、ファイル更新を fsnotify で受け取って再描画する常駐プロセス。

### Task 5: 現セッション解決ヘルパー

**ファイル:**
- 作成: `internal/tail/tail.go`（最初の skeleton）

**Exports:**
- `func ResolveCurrent() (sessionDir string, err error)` — `TMUX_AGENT_LOG_SESSION` 環境変数（hook が export 済み）または `--session` から sessionDir を解決

- [ ] **Step 1:** スケルトンと doc コメント:

  ```go
  // Package tail はライブで JSONL を追従する tail viewer を提供する。
  // fsnotify で events.jsonl / turns.jsonl / comments.jsonl の変更を watch し、
  // lipgloss で最新 N turn を整形して再描画する。
  package tail

  import (
      "errors"
      "os"
      "path/filepath"

      "github.com/yusei-wy/tmux-agent-log/internal/config"
  )

  // ResolveCurrent は現在のセッション dir を解決する。
  //   1. --session 引数（呼び出し側が引き渡す）
  //   2. 環境変数 TMUX_AGENT_LOG_SESSION_DIR（hook が export 済み）
  //   3. cwd から ProjectSlug を計算 → 最新 session を返す
  // のいずれかが見つかった時点で返す。
  func ResolveCurrent(explicitSession string) (string, error) {
      if explicitSession != "" {
          return findSessionByID(explicitSession)
      }
      if v := os.Getenv("TMUX_AGENT_LOG_SESSION_DIR"); v != "" {
          return v, nil
      }
      cwd, err := os.Getwd()
      if err != nil {
          return "", err
      }
      return latestSessionFromCwd(cwd)
  }

  func findSessionByID(id string) (string, error) {
      // 全プロジェクトを舐めて id 一致を探す（list-sessions と同じ走査）
      state, err := config.StateDir()
      if err != nil {
          return "", err
      }
      projects := filepath.Join(state, "projects")
      entries, err := os.ReadDir(projects)
      if err != nil {
          return "", err
      }
      for _, p := range entries {
          if !p.IsDir() {
              continue
          }
          candidate := filepath.Join(projects, p.Name(), "sessions", id)
          if _, err := os.Stat(candidate); err == nil {
              return candidate, nil
          }
      }
      return "", errors.New("session が見つからない: " + id)
  }

  func latestSessionFromCwd(cwd string) (string, error) {
      state, err := config.StateDir()
      if err != nil {
          return "", err
      }
      sessDir := filepath.Join(state, "projects", config.ProjectSlug(cwd), "sessions")
      entries, err := os.ReadDir(sessDir)
      if err != nil {
          return "", err
      }
      var latest string
      var latestMTime int64
      for _, e := range entries {
          if !e.IsDir() {
              continue
          }
          info, err := e.Info()
          if err != nil {
              continue
          }
          if t := info.ModTime().UnixNano(); t > latestMTime {
              latestMTime = t
              latest = filepath.Join(sessDir, e.Name())
          }
      }
      if latest == "" {
          return "", errors.New("このプロジェクトに session がない: " + cwd)
      }
      return latest, nil
  }
  ```

- [ ] **Step 2:** ビルド確認:

  ```bash
  go build ./internal/tail/...
  ```

- [ ] **Step 3:** コミット:

  ```bash
  git add internal/tail/tail.go
  git commit -m "feat(tail): 現セッションの解決ヘルパーを追加"
  ```

### Task 6: tail 表示 render

**ファイル:**
- 作成: `internal/tail/render.go`
- 作成: `internal/tail/tail_test.go`

**Exports:**
- `type RenderInput struct { ... }`
- `func Render(in RenderInput) string`

- [ ] **Step 1:** テスト先行（spec §9.1 のレイアウトに合わせる）:

  ```go
  package tail_test

  import (
      "strings"
      "testing"
      "time"

      "github.com/stretchr/testify/require"

      "github.com/yusei-wy/tmux-agent-log/internal/storage"
      "github.com/yusei-wy/tmux-agent-log/internal/tail"
  )

  func TestRender(t *testing.T) {
      now := time.Date(2026, 5, 4, 16, 42, 0, 0, time.UTC)
      in := tail.RenderInput{
          Goal: "2700 認可バグ修正",
          Turns: []storage.Turn{
              {ID: "turn-0007", StartedAt: now, Status: storage.TurnStatusDone, UserPromptPreview: "nil チェック"},
              {ID: "turn-0008", StartedAt: now.Add(time.Minute), Status: storage.TurnStatusDone, UserPromptPreview: "go test"},
          },
          UnsentComments: 2,
      }
      got := tail.Render(in)

      // ゴールが先頭に来る
      require.Contains(t, got, "2700 認可バグ修正")
      // 各 turn ID が表示される
      require.Contains(t, got, "turn-0007")
      require.Contains(t, got, "turn-0008")
      // 未送信コメント数が表示される
      require.Contains(t, got, "2")
      // フッター（comments: ...）が含まれる
      require.True(t, strings.Contains(got, "comments"), "comments フッターが必要: %q", got)
  }
  ```

- [ ] **Step 2:** fail 確認:

  ```bash
  go test ./internal/tail/...
  ```

- [ ] **Step 3:** `internal/tail/render.go` を作成:

  ```go
  package tail

  import (
      "fmt"
      "strings"

      "github.com/charmbracelet/lipgloss"

      "github.com/yusei-wy/tmux-agent-log/internal/storage"
  )

  // RenderInput は Render 1 回分の状態。読み手は loader が組み立てる。
  type RenderInput struct {
      Goal           string
      Turns          []storage.Turn // 新しい順、最大 N 件
      UnsentComments int
  }

  var (
      goalStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3"))
      doneStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
      errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
      footStyle  = lipgloss.NewStyle().Faint(true)
  )

  func Render(in RenderInput) string {
      var b strings.Builder
      if in.Goal != "" {
          fmt.Fprintf(&b, "%s\n\n", goalStyle.Render("🎯 "+in.Goal))
      }
      for _, t := range in.Turns {
          status := string(t.Status)
          switch t.Status {
          case storage.TurnStatusDone:
              status = doneStyle.Render("✓")
          case storage.TurnStatusError:
              status = errStyle.Render("✗")
          case storage.TurnStatusOpen:
              status = "·"
          }
          line := fmt.Sprintf("%s  %s  %s  %s",
              t.ID,
              t.StartedAt.Format("15:04"),
              status,
              t.UserPromptPreview,
          )
          b.WriteString(line)
          b.WriteByte('\n')
      }
      b.WriteByte('\n')
      fmt.Fprintf(&b, "%s\n", footStyle.Render(fmt.Sprintf("comments: %d unsent", in.UnsentComments)))
      return b.String()
  }
  ```

- [ ] **Step 4:** テスト通過:

  ```bash
  go test ./internal/tail/... -v
  ```

- [ ] **Step 5:** コミット:

  ```bash
  git add internal/tail/render.go internal/tail/tail_test.go
  git commit -m "feat(tail): tail 表示の render 関数を追加 (spec §9.1)"
  ```

### Task 7: fsnotify 駆動の tail ループ

**ファイル:**
- 修正: `internal/tail/tail.go`

**Exports:**
- `func Run(ctx context.Context, sessionDir string, w io.Writer, maxTurns int) error`

- [ ] **Step 1:** `Run` を追加:

  ```go
  import (
      "context"
      "io"
      "path/filepath"
      "time"

      "github.com/fsnotify/fsnotify"

      "github.com/yusei-wy/tmux-agent-log/internal/storage"
  )

  // Run は session dir を fsnotify で watch し、変更があれば再描画する。
  // ctx キャンセルで終了する。出力は w に対して ANSI clear screen + render の繰り返し。
  func Run(ctx context.Context, sessionDir string, w io.Writer, maxTurns int) error {
      watcher, err := fsnotify.NewWatcher()
      if err != nil {
          return err
      }
      defer func() { _ = watcher.Close() }()

      // session dir を watch すれば配下の jsonl 変更も拾える
      if err := watcher.Add(sessionDir); err != nil {
          return err
      }

      redraw := func() error {
          in, err := loadRenderInput(sessionDir, maxTurns)
          if err != nil {
              return err
          }
          // ANSI: home + clear-from-cursor で簡易再描画
          if _, err := io.WriteString(w, "\x1b[H\x1b[2J"); err != nil {
              return err
          }
          _, err = io.WriteString(w, Render(in))
          return err
      }

      // 初回描画
      if err := redraw(); err != nil {
          return err
      }

      // デバウンス: 50ms 内の連続 event を 1 回にまとめる
      const debounce = 50 * time.Millisecond
      var timer *time.Timer

      for {
          select {
          case <-ctx.Done():
              return nil
          case ev, ok := <-watcher.Events:
              if !ok {
                  return nil
              }
              if ev.Op&(fsnotify.Write|fsnotify.Create) == 0 {
                  continue
              }
              if timer != nil {
                  timer.Stop()
              }
              timer = time.AfterFunc(debounce, func() { _ = redraw() })
          case err, ok := <-watcher.Errors:
              if !ok {
                  return nil
              }
              return err
          }
      }
  }

  func loadRenderInput(sessionDir string, maxTurns int) (RenderInput, error) {
      meta, err := storage.ReadSessionMeta(sessionDir)
      if err != nil {
          return RenderInput{}, err
      }
      turns, err := storage.ReadTurns(filepath.Join(sessionDir, "turns.jsonl"))
      if err != nil {
          return RenderInput{}, err
      }
      // 新しい順に
      if len(turns) > maxTurns {
          turns = turns[len(turns)-maxTurns:]
      }
      // 反転して新しい順に
      for i, j := 0, len(turns)-1; i < j; i, j = i+1, j-1 {
          turns[i], turns[j] = turns[j], turns[i]
      }
      unsent, err := storage.UnsentComments(filepath.Join(sessionDir, "comments.jsonl"))
      if err != nil {
          return RenderInput{}, err
      }
      return RenderInput{Goal: meta.Goal, Turns: turns, UnsentComments: len(unsent)}, nil
  }
  ```

- [ ] **Step 2:** ビルド確認:

  ```bash
  go build ./internal/tail/...
  ```

- [ ] **Step 3:** 簡易 e2e テストを追加（fsnotify の発火を検証）:

  ```go
  func TestRunWatchesSessionDir(t *testing.T) {
      dir := t.TempDir()
      meta := storage.SessionMeta{ClaudeSessionID: "test", Cwd: dir, Goal: "g", StartedAt: time.Now()}
      require.NoError(t, storage.WriteSessionMeta(dir, meta))

      var buf bytes.Buffer
      ctx, cancel := context.WithCancel(context.Background())
      done := make(chan error, 1)
      go func() { done <- tail.Run(ctx, dir, &buf, 5) }()

      // 少し待って turn を追記
      time.Sleep(80 * time.Millisecond)
      turnsPath := filepath.Join(dir, "turns.jsonl")
      require.NoError(t, storage.AppendTurnOpen(turnsPath, storage.TurnOpen{
          ID: "turn-0001", StartedAt: time.Now(),
      }))
      time.Sleep(150 * time.Millisecond)

      cancel()
      <-done

      out := buf.String()
      require.Contains(t, out, "turn-0001")
      require.Contains(t, out, "g")  // goal
  }
  ```

  必要な import: `bytes`, `context`, `path/filepath`, `time`, `github.com/yusei-wy/tmux-agent-log/internal/storage`。

- [ ] **Step 4:** テスト通過:

  ```bash
  go test ./internal/tail/... -race -v
  ```

- [ ] **Step 5:** コミット:

  ```bash
  git add internal/tail/tail.go internal/tail/tail_test.go
  git commit -m "feat(tail): fsnotify 駆動の再描画ループを追加"
  ```

### Task 8: `tail` CLI サブコマンド

**ファイル:**
- 作成: `internal/cli/tail.go`

**Exports:**
- `tail` cobra サブコマンド（init で root に登録）

- [ ] **Step 1:** ファイル作成:

  ```go
  package cli

  import (
      "context"
      "os"
      "os/signal"
      "syscall"

      "github.com/spf13/cobra"

      "github.com/yusei-wy/tmux-agent-log/internal/tail"
  )

  func init() {
      rootCmd.AddCommand(tailCmd())
  }

  func tailCmd() *cobra.Command {
      var sessionID string
      var maxTurns int
      cmd := &cobra.Command{
          Use:   "tail",
          Short: "現セッションの最新 turn をライブ表示する",
          RunE: func(cmd *cobra.Command, args []string) error {
              sessionDir, err := tail.ResolveCurrent(sessionID)
              if err != nil {
                  return err
              }
              ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
              defer cancel()
              return tail.Run(ctx, sessionDir, cmd.OutOrStdout(), maxTurns)
          },
      }
      cmd.Flags().StringVar(&sessionID, "session", "", "セッション ID（未指定なら環境変数 / cwd から推定）")
      cmd.Flags().IntVar(&maxTurns, "max-turns", 8, "表示する直近 turn の最大数")
      return cmd
  }
  ```

- [ ] **Step 2:** ビルド + `--help` 確認:

  ```bash
  go build -o /tmp/tmux-agent-log ./cmd/tmux-agent-log
  /tmp/tmux-agent-log tail --help
  ```

  期待: `tail` の help が出る。

- [ ] **Step 3:** コミット:

  ```bash
  git add internal/cli/tail.go
  git commit -m "feat(cli): tail サブコマンドを追加"
  ```

---

## Phase 3: TUI 基盤

> **実装ノート — bubbletea の通信パターン:**
>
> 子 Model（timeline / diffview / overview）は親 Model を直接書き換えない。慣例:
> - 子の `Update` は `(子モデル, tea.Cmd)` を返す。親への参照は持たない
> - 共通キーバインドだけ親から渡す（引数 `keys Keys`）
> - 子 → 親の通知は **custom msg を tea.Cmd で発行** し、親の Update でハンドルする
>   - `statusMsg(string)`: ステータスバー文字列を更新
>   - `dataReloadedMsg(SessionData)`: コメント追加・送信・削除後の再読込通知
>
> 値レシーバの Update では `root.status = ...` のような親への代入は反映されないため、必ず msg 経由にすること。

### Task 9: lipgloss スタイル定義

**ファイル:**
- 作成: `internal/tui/style.go`

**Exports:**
- `var TabActiveStyle, TabInactiveStyle, BorderStyle, StatusBarStyle, FaintStyle lipgloss.Style`

- [ ] **Step 1:** 共通スタイルを定義:

  ```go
  package tui

  import "github.com/charmbracelet/lipgloss"

  var (
      // タブ
      TabActiveStyle = lipgloss.NewStyle().
          Bold(true).
          Foreground(lipgloss.Color("15")).
          Background(lipgloss.Color("4")).
          Padding(0, 2)

      TabInactiveStyle = lipgloss.NewStyle().
          Faint(true).
          Padding(0, 2)

      // パネル境界
      BorderStyle = lipgloss.NewStyle().
          Border(lipgloss.RoundedBorder()).
          BorderForeground(lipgloss.Color("8"))

      // フォーカスされたパネル
      FocusedBorderStyle = BorderStyle.BorderForeground(lipgloss.Color("4"))

      // ステータスバー
      StatusBarStyle = lipgloss.NewStyle().
          Background(lipgloss.Color("0")).
          Foreground(lipgloss.Color("7")).
          Padding(0, 1)

      FaintStyle = lipgloss.NewStyle().Faint(true)

      // コメント sent 済の半透明
      SentCommentStyle = lipgloss.NewStyle().Faint(true)
  )
  ```

- [ ] **Step 2:** ビルド確認:

  ```bash
  go build ./internal/tui/
  ```

- [ ] **Step 3:** コミット:

  ```bash
  git add internal/tui/style.go
  git commit -m "feat(tui): lipgloss スタイルの共通定義を追加"
  ```

### Task 10: キーバインド定義

**ファイル:**
- 作成: `internal/tui/keys.go`

**Exports:**
- `type Keys struct { ... }` — bubbles/key.Binding を集約
- `var DefaultKeys Keys`

- [ ] **Step 1:** 中身:

  ```go
  package tui

  import "github.com/charmbracelet/bubbles/key"

  type Keys struct {
      Quit     key.Binding
      NextTab  key.Binding
      PrevTab  key.Binding
      Up       key.Binding
      Down     key.Binding
      Left     key.Binding
      Right    key.Binding
      Enter    key.Binding
      Comment  key.Binding // c
      Send     key.Binding // s
      Delete   key.Binding // d
      Help     key.Binding // ?
  }

  var DefaultKeys = Keys{
      Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
      NextTab: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next tab")),
      PrevTab: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev tab")),
      Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
      Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
      Left:    key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "left")),
      Right:   key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "right")),
      Enter:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
      Comment: key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "add comment")),
      Send:    key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "send to claude")),
      Delete:  key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete comment")),
      Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
  }
  ```

- [ ] **Step 2:** ビルド:

  ```bash
  go build ./internal/tui/
  ```

- [ ] **Step 3:** コミット:

  ```bash
  git add internal/tui/keys.go
  git commit -m "feat(tui): キーバインドの集約定義を追加"
  ```

### Task 11: loader（storage → view-ready 変換）

**ファイル:**
- 作成: `internal/tui/loader.go`
- 作成: `internal/tui/loader_test.go`

**Exports:**
- `type SessionData struct { Meta storage.SessionMeta; Turns []storage.Turn; Comments []storage.Comment }`
- `func LoadSession(sessionDir string) (SessionData, error)`

- [ ] **Step 1:** 失敗するテスト:

  ```go
  package tui_test

  import (
      "path/filepath"
      "testing"
      "time"

      "github.com/stretchr/testify/require"

      "github.com/yusei-wy/tmux-agent-log/internal/storage"
      "github.com/yusei-wy/tmux-agent-log/internal/tui"
  )

  func TestLoadSession(t *testing.T) {
      dir := t.TempDir()
      meta := storage.SessionMeta{ClaudeSessionID: "s1", Goal: "g", Cwd: dir, StartedAt: time.Now()}
      require.NoError(t, storage.WriteSessionMeta(dir, meta))
      require.NoError(t, storage.AppendTurnOpen(filepath.Join(dir, "turns.jsonl"), storage.TurnOpen{
          ID: "t1", StartedAt: time.Now(),
      }))
      require.NoError(t, storage.AppendComment(filepath.Join(dir, "comments.jsonl"), storage.Comment{
          ID: "c1", File: "f.go", LineStart: 1, LineEnd: 1, Text: "hi",
      }))

      data, err := tui.LoadSession(dir)
      require.NoError(t, err)
      require.Equal(t, "g", data.Meta.Goal)
      require.Len(t, data.Turns, 1)
      require.Equal(t, "t1", data.Turns[0].ID)
      require.Len(t, data.Comments, 1)
      require.Equal(t, "c1", data.Comments[0].ID)
  }
  ```

- [ ] **Step 2:** fail 確認:

  ```bash
  go test ./internal/tui/... -run TestLoadSession
  ```

- [ ] **Step 3:** 実装:

  ```go
  package tui

  import (
      "path/filepath"

      "github.com/yusei-wy/tmux-agent-log/internal/storage"
  )

  type SessionData struct {
      Dir      string
      Meta     storage.SessionMeta
      Turns    []storage.Turn
      Comments []storage.Comment
  }

  // LoadSession は session ディレクトリから 1 セッション分のデータをまとめて読む。
  func LoadSession(sessionDir string) (SessionData, error) {
      meta, err := storage.ReadSessionMeta(sessionDir)
      if err != nil {
          return SessionData{}, err
      }
      turns, err := storage.ReadTurns(filepath.Join(sessionDir, "turns.jsonl"))
      if err != nil {
          return SessionData{}, err
      }
      comments, err := storage.ReadComments(filepath.Join(sessionDir, "comments.jsonl"))
      if err != nil {
          return SessionData{}, err
      }
      return SessionData{Dir: sessionDir, Meta: meta, Turns: turns, Comments: comments}, nil
  }
  ```

- [ ] **Step 4:** テスト通過:

  ```bash
  go test ./internal/tui/... -run TestLoadSession -v
  ```

- [ ] **Step 5:** コミット:

  ```bash
  git add internal/tui/loader.go internal/tui/loader_test.go
  git commit -m "feat(tui): SessionData loader を追加"
  ```

### Task 12: root Model（タブ枠だけ）

**ファイル:**
- 作成: `internal/tui/model.go`

**Exports:**
- `type Tab int` (`TabTimeline`, `TabDiff`, `TabOverview`)
- `type Model struct { ... }`
- `func New(data SessionData) Model`
- `func (m Model) Init() tea.Cmd`
- `func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)`
- `func (m Model) View() string`

- [ ] **Step 1:** ファイル作成（タブだけ機能、各タブは仮表示）:

  ```go
  package tui

  import (
      "fmt"

      tea "github.com/charmbracelet/bubbletea"
      "github.com/charmbracelet/bubbles/key"
      "github.com/charmbracelet/lipgloss"
  )

  type Tab int

  const (
      TabTimeline Tab = iota
      TabDiff
      TabOverview
  )

  func (t Tab) String() string {
      switch t {
      case TabTimeline:
          return "timeline"
      case TabDiff:
          return "diff"
      case TabOverview:
          return "overview"
      }
      return ""
  }

  // 子 Model から親へ通知するための custom msg。Phase 3 冒頭の実装ノート参照。
  type statusMsg string
  type dataReloadedMsg SessionData

  type Model struct {
      keys     Keys
      data     SessionData
      tab      Tab
      width    int
      height   int
      timeline timelineModel
      diff     diffviewModel
      overview overviewModel
      status   string
  }

  func New(data SessionData) Model {
      return Model{
          keys:     DefaultKeys,
          data:     data,
          tab:      TabTimeline,
          timeline: newTimeline(data),
          diff:     newDiffview(data),
          overview: newOverview(data),
      }
  }

  func (m Model) Init() tea.Cmd {
      return nil
  }

  func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
      switch msg := msg.(type) {
      case tea.WindowSizeMsg:
          m.width = msg.Width
          m.height = msg.Height
      case statusMsg:
          m.status = string(msg)
          return m, nil
      case dataReloadedMsg:
          m.data = SessionData(msg)
          m.timeline.data = m.data
          m.diff.data = m.data
          m.overview.data = m.data
          return m, nil
      case tea.KeyMsg:
          switch {
          case key.Matches(msg, m.keys.Quit):
              return m, tea.Quit
          case key.Matches(msg, m.keys.NextTab):
              m.tab = (m.tab + 1) % 3
              return m, nil
          case key.Matches(msg, m.keys.PrevTab):
              m.tab = (m.tab + 2) % 3
              return m, nil
          }
      }
      // 現在のタブに転送（共通キーは Keys だけ渡す）
      var cmd tea.Cmd
      switch m.tab {
      case TabTimeline:
          m.timeline, cmd = m.timeline.Update(msg, m.keys)
      case TabDiff:
          m.diff, cmd = m.diff.Update(msg, m.keys, m.width, m.height)
      case TabOverview:
          m.overview, cmd = m.overview.Update(msg, m.keys)
      }
      return m, cmd
  }

  func (m Model) View() string {
      // タブヘッダ
      tabs := []string{"timeline", "diff", "overview"}
      var rendered []string
      for i, name := range tabs {
          if Tab(i) == m.tab {
              rendered = append(rendered, TabActiveStyle.Render(name))
          } else {
              rendered = append(rendered, TabInactiveStyle.Render(name))
          }
      }
      header := lipgloss.JoinHorizontal(lipgloss.Top, rendered...)

      // 本体
      var body string
      switch m.tab {
      case TabTimeline:
          body = m.timeline.View(m.width, m.height)
      case TabDiff:
          body = m.diff.View(m.width, m.height)
      case TabOverview:
          body = m.overview.View(m.width, m.height)
      }

      // ステータスバー
      goal := m.data.Meta.Goal
      if goal == "" {
          goal = "(no goal)"
      }
      sbar := m.status
      if sbar == "" {
          sbar = fmt.Sprintf("%s | %s | [tab] switch [q] quit", goal, m.tab)
      }
      status := StatusBarStyle.Width(m.width).Render(sbar)

      return lipgloss.JoinVertical(lipgloss.Left, header, body, status)
  }
  ```

  ※ `timelineModel` / `diffviewModel` / `overviewModel` と `newTimeline` / `newDiffview` / `newOverview` は次タスクで実装する skeleton として、ここでは型エラーになる前提。

- [ ] **Step 2:** 各タブの skeleton を別ファイルで作る（実装は後タスク）:

  `internal/tui/timeline.go`:

  ```go
  package tui

  import tea "github.com/charmbracelet/bubbletea"

  type timelineModel struct {
      data SessionData
      idx  int
  }

  func newTimeline(d SessionData) timelineModel { return timelineModel{data: d} }
  func (m timelineModel) Update(msg tea.Msg, keys Keys) (timelineModel, tea.Cmd) { return m, nil }
  func (m timelineModel) View(width, height int) string { return "(timeline placeholder)" }
  ```

  同様に `diffview.go`:

  ```go
  package tui

  import tea "github.com/charmbracelet/bubbletea"

  type diffviewModel struct {
      data SessionData
  }

  func newDiffview(d SessionData) diffviewModel { return diffviewModel{data: d} }
  func (m diffviewModel) Update(msg tea.Msg, keys Keys, width, height int) (diffviewModel, tea.Cmd) { return m, nil }
  func (m diffviewModel) View(width, height int) string { return "(diff placeholder)" }
  ```

  `overview.go`:

  ```go
  package tui

  import tea "github.com/charmbracelet/bubbletea"

  type overviewModel struct {
      data SessionData
  }

  func newOverview(d SessionData) overviewModel { return overviewModel{data: d} }
  func (m overviewModel) Update(msg tea.Msg, keys Keys) (overviewModel, tea.Cmd) { return m, nil }
  func (m overviewModel) View(width, height int) string { return "(overview placeholder)" }
  ```

- [ ] **Step 3:** ビルド確認:

  ```bash
  go build ./internal/tui/
  ```

  期待: コンパイル通る。

- [ ] **Step 4:** コミット:

  ```bash
  git add internal/tui/model.go internal/tui/timeline.go internal/tui/diffview.go internal/tui/overview.go
  git commit -m "feat(tui): root Model とタブ skeleton を追加"
  ```

### Task 13: `tui` CLI サブコマンド

**ファイル:**
- 作成: `internal/cli/tui.go`

- [ ] **Step 1:** 実装:

  ```go
  package cli

  import (
      tea "github.com/charmbracelet/bubbletea"
      "github.com/spf13/cobra"

      "github.com/yusei-wy/tmux-agent-log/internal/tail"
      "github.com/yusei-wy/tmux-agent-log/internal/tui"
  )

  func init() {
      rootCmd.AddCommand(tuiCmd())
  }

  func tuiCmd() *cobra.Command {
      var sessionID string
      cmd := &cobra.Command{
          Use:   "tui",
          Short: "対話 TUI を起動する (timeline / diff / overview)",
          RunE: func(cmd *cobra.Command, args []string) error {
              sessionDir, err := tail.ResolveCurrent(sessionID)
              if err != nil {
                  return err
              }
              data, err := tui.LoadSession(sessionDir)
              if err != nil {
                  return err
              }
              p := tea.NewProgram(tui.New(data), tea.WithAltScreen())
              _, err = p.Run()
              return err
          },
      }
      cmd.Flags().StringVar(&sessionID, "session", "", "セッション ID（未指定なら推定）")
      return cmd
  }
  ```

- [ ] **Step 2:** ビルド + `--help`:

  ```bash
  go build -o /tmp/tmux-agent-log ./cmd/tmux-agent-log
  /tmp/tmux-agent-log tui --help
  ```

- [ ] **Step 3:** コミット:

  ```bash
  git add internal/cli/tui.go
  git commit -m "feat(cli): tui サブコマンドを追加 (placeholder UI)"
  ```

---

## Phase 4: timeline タブ

### Task 14: timeline Model 本実装

**ファイル:**
- 修正: `internal/tui/timeline.go`
- 作成: `internal/tui/timeline_test.go`

**機能:**
- 上下キーで turn 一覧を移動
- 右ペインに選択中 turn の prompt preview / 状態 / diff_path を表示
- spec §9.2 のレイアウトに合わせる

- [ ] **Step 1:** テスト先行:

  ```go
  package tui_test

  import (
      "strings"
      "testing"
      "time"

      tea "github.com/charmbracelet/bubbletea"

      "github.com/yusei-wy/tmux-agent-log/internal/storage"
      "github.com/yusei-wy/tmux-agent-log/internal/tui"
  )

  func TestTimelineRendersTurns(t *testing.T) {
      data := tui.SessionData{
          Meta: storage.SessionMeta{Goal: "g"},
          Turns: []storage.Turn{
              {ID: "t1", StartedAt: time.Now(), Status: storage.TurnStatusDone, UserPromptPreview: "prompt 1"},
              {ID: "t2", StartedAt: time.Now(), Status: storage.TurnStatusOpen, UserPromptPreview: "prompt 2"},
          },
      }
      m := tui.New(data)
      next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
      m = next.(tui.Model)
      view := m.View()
      if !strings.Contains(view, "t1") || !strings.Contains(view, "t2") {
          t.Fatalf("turn ID が表示されていない: %q", view)
      }
      if !strings.Contains(view, "prompt 1") {
          t.Fatalf("選択中の prompt が表示されない: %q", view)
      }
  }
  ```

- [ ] **Step 2:** fail 確認:

  ```bash
  go test ./internal/tui/... -run TestTimelineRendersTurns
  ```

- [ ] **Step 3:** `internal/tui/timeline.go` を本実装:

  ```go
  package tui

  import (
      "fmt"
      "strings"

      tea "github.com/charmbracelet/bubbletea"
      "github.com/charmbracelet/bubbles/key"
      "github.com/charmbracelet/lipgloss"

      "github.com/yusei-wy/tmux-agent-log/internal/storage"
  )

  type timelineModel struct {
      data SessionData
      idx  int
  }

  func newTimeline(d SessionData) timelineModel {
      return timelineModel{data: d, idx: len(d.Turns) - 1} // 末尾（最新）にカーソル
  }

  func (m timelineModel) Update(msg tea.Msg, keys Keys) (timelineModel, tea.Cmd) {
      if k, ok := msg.(tea.KeyMsg); ok {
          switch {
          case key.Matches(k, keys.Up):
              if m.idx > 0 {
                  m.idx--
              }
          case key.Matches(k, keys.Down):
              if m.idx < len(m.data.Turns)-1 {
                  m.idx++
              }
          }
      }
      return m, nil
  }

  func (m timelineModel) View(width, height int) string {
      if len(m.data.Turns) == 0 {
          return FaintStyle.Render("(まだ turn がない)")
      }
      // 左: turn list
      var listLines []string
      for i, t := range m.data.Turns {
          marker := " "
          if i == m.idx {
              marker = ">"
          }
          line := fmt.Sprintf("%s %s  %s  %s", marker, t.ID, t.StartedAt.Format("15:04"), t.UserPromptPreview)
          if i == m.idx {
              line = TabActiveStyle.Render(line)
          }
          listLines = append(listLines, line)
      }
      list := strings.Join(listLines, "\n")

      // 右: 詳細
      cur := m.data.Turns[m.idx]
      detail := turnDetail(cur)

      // 横並び
      listW := width / 2
      detailW := width - listW
      return lipgloss.JoinHorizontal(lipgloss.Top,
          lipgloss.NewStyle().Width(listW).Render(list),
          lipgloss.NewStyle().Width(detailW).Render(detail),
      )
  }

  func turnDetail(t storage.Turn) string {
      var b strings.Builder
      fmt.Fprintf(&b, "ID: %s\n", t.ID)
      fmt.Fprintf(&b, "Started: %s\n", t.StartedAt.Format("2006-01-02 15:04:05"))
      if !t.EndedAt.IsZero() {
          fmt.Fprintf(&b, "Ended: %s\n", t.EndedAt.Format("2006-01-02 15:04:05"))
      }
      fmt.Fprintf(&b, "Status: %s\n", t.Status)
      if t.DiffPath != "" {
          fmt.Fprintf(&b, "Diff: %s\n", t.DiffPath)
      }
      b.WriteString("\nPrompt:\n")
      b.WriteString(t.UserPromptPreview)
      return b.String()
  }
  ```

- [ ] **Step 4:** テスト通過:

  ```bash
  go test ./internal/tui/... -v
  ```

- [ ] **Step 5:** コミット:

  ```bash
  git add internal/tui/timeline.go internal/tui/timeline_test.go
  git commit -m "feat(tui): timeline タブの本実装 (turn 一覧 + 詳細)"
  ```

---

## Phase 5: diff タブ（累積、read-only）

このタブは v0.1 の中で最も中核。spec §5.3 / §5.4 / §11.1 を満たす。

### Task 15: 累積 diff loader

**ファイル:**
- 修正: `internal/tui/loader.go`
- 修正: `internal/tui/loader_test.go`

**Exports:**
- `func LoadCumulativeDiff(sessionDir string) ([]FileDiff, error)`

`SessionMeta.Cwd` と `BaseSHA` から `git.DiffSince` を呼んで unified diff を取得し、`ParseUnified` で構造化する。

- [ ] **Step 1:** テスト先行（git リポジトリを tempdir に作る）:

  ```go
  func TestLoadCumulativeDiff(t *testing.T) {
      // 一時 repo 初期化
      repo := t.TempDir()
      run := func(args ...string) {
          cmd := exec.Command("git", args...)
          cmd.Dir = repo
          require.NoError(t, cmd.Run())
      }
      run("init", "-q")
      run("config", "user.email", "t@example.com")
      run("config", "user.name", "t")
      require.NoError(t, os.WriteFile(filepath.Join(repo, "f.go"), []byte("package f\n"), 0o600))
      run("add", ".")
      run("commit", "-q", "-m", "init")

      // base sha 取得
      base, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
      require.NoError(t, err)
      baseSHA := strings.TrimSpace(string(base))

      // 編集
      require.NoError(t, os.WriteFile(filepath.Join(repo, "f.go"), []byte("package f\nvar X = 1\n"), 0o600))

      // session dir 準備
      sDir := t.TempDir()
      meta := storage.SessionMeta{ClaudeSessionID: "s1", Cwd: repo, BaseSHA: baseSHA, GitTracked: true, StartedAt: time.Now()}
      require.NoError(t, storage.WriteSessionMeta(sDir, meta))

      diffs, err := tui.LoadCumulativeDiff(sDir)
      require.NoError(t, err)
      require.Len(t, diffs, 1)
      require.Contains(t, diffs[0].NewPath, "f.go")
  }
  ```

  必要な import: `os`, `os/exec`, `path/filepath`, `strings`, `time`。

- [ ] **Step 2:** fail 確認:

  ```bash
  go test ./internal/tui/... -run TestLoadCumulativeDiff
  ```

- [ ] **Step 3:** `internal/tui/loader.go` に追加:

  ```go
  import (
      "github.com/yusei-wy/tmux-agent-log/internal/git"
  )

  // LoadCumulativeDiff は session base SHA から HEAD までの累積 diff を構造化して返す。
  // git_tracked=false の session では空 slice を返す（エラーではない）。
  func LoadCumulativeDiff(sessionDir string) ([]FileDiff, error) {
      meta, err := storage.ReadSessionMeta(sessionDir)
      if err != nil {
          return nil, err
      }
      if !meta.GitTracked || meta.BaseSHA == "" {
          return nil, nil
      }
      raw, err := git.DiffSince(meta.Cwd, meta.BaseSHA)
      if err != nil {
          return nil, err
      }
      return ParseUnified(raw), nil
  }
  ```

- [ ] **Step 4:** テスト通過:

  ```bash
  go test ./internal/tui/... -run TestLoadCumulativeDiff -v
  ```

- [ ] **Step 5:** コミット:

  ```bash
  git add internal/tui/loader.go internal/tui/loader_test.go
  git commit -m "feat(tui): 累積 diff loader を追加 (git base → HEAD)"
  ```

### Task 16: diff タブの基本表示（左ファイル一覧 + 右 diff）

**ファイル:**
- 修正: `internal/tui/diffview.go`
- 作成: `internal/tui/diffview_test.go`

**機能:**
- 左ペイン: `LoadCumulativeDiff` の `[]FileDiff` を一覧
- 右ペイン: 選択中ファイルの全 hunk を色分けして表示（read-only）
- ←/→ でファイル切替、↑/↓ で右ペインを 1 行ずつスクロール

- [ ] **Step 1:** テスト:

  ```go
  func TestDiffviewRendersFiles(t *testing.T) {
      // LoadCumulativeDiff と同じセットアップ（重複だが TDD のため再掲）
      repo := t.TempDir()
      // ... (Task 15 と同じ手順) ...

      sDir := t.TempDir()
      // ... meta 書き込み ...

      data, err := tui.LoadSession(sDir)
      require.NoError(t, err)
      m := tui.New(data)
      // diff タブに切替
      next, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
      m = next.(tui.Model)
      next, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
      m = next.(tui.Model)
      view := m.View()
      require.Contains(t, view, "f.go")
  }
  ```

  ※ Task 15 と同様の git tempdir セットアップが必要。共通の helper 関数 `setupGitRepo(t) (sessionDir string)` を test ファイル内に置くと DRY。

- [ ] **Step 2:** fail 確認 + `internal/tui/diffview.go` を本実装:

  ```go
  package tui

  import (
      "fmt"
      "strings"

      tea "github.com/charmbracelet/bubbletea"
      "github.com/charmbracelet/bubbles/key"
      "github.com/charmbracelet/bubbles/viewport"
      "github.com/charmbracelet/lipgloss"
  )

  type diffviewModel struct {
      data     SessionData
      diffs    []FileDiff
      diffsErr error
      fileIdx  int
      viewport viewport.Model
      ready    bool
  }

  func newDiffview(d SessionData) diffviewModel {
      diffs, err := LoadCumulativeDiff(d.Dir)
      return diffviewModel{data: d, diffs: diffs, diffsErr: err}
  }

  func (m diffviewModel) Update(msg tea.Msg, keys Keys, width, height int) (diffviewModel, tea.Cmd) {
      switch msg := msg.(type) {
      case tea.WindowSizeMsg:
          listW := msg.Width / 3
          if !m.ready {
              m.viewport = viewport.New(msg.Width-listW, msg.Height-3)
              m.ready = true
          } else {
              m.viewport.Width = msg.Width - listW
              m.viewport.Height = msg.Height - 3
          }
          m.viewport.SetContent(m.renderCurrent())
      case tea.KeyMsg:
          switch {
          case key.Matches(msg, keys.Left):
              if m.fileIdx > 0 {
                  m.fileIdx--
                  m.viewport.SetContent(m.renderCurrent())
              }
          case key.Matches(msg, keys.Right):
              if m.fileIdx < len(m.diffs)-1 {
                  m.fileIdx++
                  m.viewport.SetContent(m.renderCurrent())
              }
          }
          var cmd tea.Cmd
          m.viewport, cmd = m.viewport.Update(msg)
          return m, cmd
      }
      return m, nil
  }

  func (m diffviewModel) View(width, height int) string {
      if m.diffsErr != nil {
          return errStyle().Render("diff 読込エラー: " + m.diffsErr.Error())
      }
      if len(m.diffs) == 0 {
          return FaintStyle.Render("(累積 diff なし — base SHA から変更なし、または git 管理外)")
      }
      // 左: ファイル一覧
      var lines []string
      for i, f := range m.diffs {
          marker := "  "
          if i == m.fileIdx {
              marker = "▶ "
          }
          line := marker + summarizeFile(f)
          if i == m.fileIdx {
              line = TabActiveStyle.Render(line)
          }
          lines = append(lines, line)
      }
      list := strings.Join(lines, "\n")

      listW := width / 3
      return lipgloss.JoinHorizontal(lipgloss.Top,
          lipgloss.NewStyle().Width(listW).Render(list),
          m.viewport.View(),
      )
  }

  func (m diffviewModel) renderCurrent() string {
      if m.fileIdx >= len(m.diffs) {
          return ""
      }
      f := m.diffs[m.fileIdx]
      var b strings.Builder
      fmt.Fprintf(&b, "%s\n%s\n\n",
          headerStyle.Render("--- "+f.OldPath),
          headerStyle.Render("+++ "+f.NewPath),
      )
      for _, h := range f.Hunks {
          b.WriteString(headerStyle.Render(h.Header))
          b.WriteByte('\n')
          for _, l := range h.Lines {
              b.WriteString(RenderLine(l))
              b.WriteByte('\n')
          }
          b.WriteByte('\n')
      }
      return b.String()
  }

  func summarizeFile(f FileDiff) string {
      add, del := 0, 0
      for _, h := range f.Hunks {
          for _, l := range h.Lines {
              switch l.Kind {
              case LineAdd:
                  add++
              case LineRemove:
                  del++
              }
          }
      }
      name := f.NewPath
      if name == "/dev/null" {
          name = f.OldPath
      }
      return fmt.Sprintf("%s +%d -%d", name, add, del)
  }

  func errStyle() lipgloss.Style {
      return lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
  }
  ```

- [ ] **Step 3:** テスト通過:

  ```bash
  go test ./internal/tui/... -v
  ```

- [ ] **Step 4:** コミット:

  ```bash
  git add internal/tui/diffview.go internal/tui/diffview_test.go
  git commit -m "feat(tui): diff タブの基本表示 (ファイル一覧 + 色付き diff)"
  ```

### Task 17: 行カーソルとコメント追加 (`c`)

**ファイル:**
- 修正: `internal/tui/diffview.go`

**機能:**
- 右ペインで行カーソル位置を保持（hunk 内の add/context/remove 行のうち、コメント可能な行 = `LineAdd` と `LineContext`）
- `c` で `$EDITOR` を起動し、保存内容を `storage.AppendComment` する
- 保存後、現セッションの comments を再読込

- [ ] **Step 1:** `diffviewModel` に行カーソルを追加:

  既存 struct に追加:

  ```go
  type diffviewModel struct {
      data     SessionData
      diffs    []FileDiff
      diffsErr error
      fileIdx  int
      lineIdx  int // 現ファイルの右ペイン内の行インデックス
      viewport viewport.Model
      ready    bool
  }
  ```

- [ ] **Step 2:** Update に↑↓ハンドラと `c` ハンドラを追加（Task 16 の Update スイッチ内に追記）:

  ```go
  case key.Matches(msg, keys.Up):
      if m.lineIdx > 0 {
          m.lineIdx--
          m.viewport.SetContent(m.renderCurrent())
      }
  case key.Matches(msg, keys.Down):
      total := totalLines(m.diffs[m.fileIdx])
      if m.lineIdx < total-1 {
          m.lineIdx++
          m.viewport.SetContent(m.renderCurrent())
      }
  case key.Matches(msg, keys.Comment):
      return m, m.startComment()
  ```

  `totalLines`:

  ```go
  func totalLines(f FileDiff) int {
      n := 0
      for _, h := range f.Hunks {
          n += len(h.Lines)
      }
      return n
  }
  ```

- [ ] **Step 3:** `startComment` 実装（`$EDITOR` を tea.ExecCommand 経由で起動）:

  ```go
  import (
      "os"
      "os/exec"
      "path/filepath"
      "time"
  )

  // commentSavedMsg はエディタ完了後に発行される。err != nil なら失敗。
  type commentSavedMsg struct{ err error }

  func (m diffviewModel) startComment() tea.Cmd {
      // 現在の行 → file/line を解決
      file, line, ok := m.currentAnchor()
      if !ok {
          return func() tea.Msg { return statusMsg("コメント可能な行ではない") }
      }
      tmp, err := os.CreateTemp("", "tmux-agent-log-comment-*.md")
      if err != nil {
          return func() tea.Msg { return statusMsg("tmpfile 作成失敗: " + err.Error()) }
      }
      _ = tmp.Close()

      editor := os.Getenv("EDITOR")
      if editor == "" {
          editor = "vi"
      }
      sessionDir := m.data.Dir
      c := exec.Command(editor, tmp.Name())
      return tea.ExecProcess(c, func(err error) tea.Msg {
          if err != nil {
              _ = os.Remove(tmp.Name())
              return commentSavedMsg{err: err}
          }
          body, rerr := os.ReadFile(tmp.Name())
          _ = os.Remove(tmp.Name())
          if rerr != nil {
              return commentSavedMsg{err: rerr}
          }
          text := strings.TrimSpace(string(body))
          if text == "" {
              return commentSavedMsg{} // 空 = キャンセル扱い
          }
          rec := storage.Comment{
              ID:        "cmt-" + time.Now().UTC().Format("20060102150405.000000"),
              File:      file,
              LineStart: line,
              LineEnd:   line,
              Text:      text,
              CreatedAt: time.Now().UTC(),
          }
          if err := storage.AppendComment(filepath.Join(sessionDir, "comments.jsonl"), rec); err != nil {
              return commentSavedMsg{err: err}
          }
          return commentSavedMsg{}
      })
  }

  func (m diffviewModel) currentAnchor() (string, int, bool) {
      if m.fileIdx >= len(m.diffs) {
          return "", 0, false
      }
      f := m.diffs[m.fileIdx]
      n := 0
      for _, h := range f.Hunks {
          for _, l := range h.Lines {
              if n == m.lineIdx {
                  if l.Kind == LineAdd || l.Kind == LineContext {
                      // file path から先頭 "b/" を除去
                      file := strings.TrimPrefix(f.NewPath, "b/")
                      return file, l.NewNo, true
                  }
                  return "", 0, false
              }
              n++
          }
      }
      return "", 0, false
  }
  ```

- [ ] **Step 4:** Update に `commentSavedMsg` ハンドラを追加して、保存成功時にコメントリスト再読込通知（root が dataReloadedMsg を受けて反映）:

  ```go
  case commentSavedMsg:
      if msg.err != nil {
          return m, func() tea.Msg { return statusMsg("コメント保存失敗: " + msg.err.Error()) }
      }
      data, err := LoadSession(m.data.Dir)
      if err != nil {
          return m, func() tea.Msg { return statusMsg("再読込失敗: " + err.Error()) }
      }
      m.data = data
      unsent := countUnsent(data.Comments)
      return m, tea.Batch(
          func() tea.Msg { return dataReloadedMsg(data) },
          func() tea.Msg { return statusMsg(fmt.Sprintf("コメント追加: %d 件 unsent", unsent)) },
      )
  ```

  `countUnsent`:

  ```go
  func countUnsent(cs []storage.Comment) int {
      n := 0
      for _, c := range cs {
          if c.SentAt.IsZero() {
              n++
          }
      }
      return n
  }
  ```

- [ ] **Step 5:** ビルド確認:

  ```bash
  go build ./...
  ```

- [ ] **Step 6:** コミット:

  ```bash
  git add internal/tui/diffview.go
  git commit -m "feat(tui): 行カーソルと c でのコメント追加 ($EDITOR 連携)"
  ```

### Task 18: 行カーソルのハイライトとコメントマーカー表示

**ファイル:**
- 修正: `internal/tui/diffview.go`

**機能:**
- 現在の行を背景色でハイライト
- 既存コメントが anchor している行に `💬` マーカー
- 送信済みコメントは半透明（`SentCommentStyle`）

- [ ] **Step 1:** `renderCurrent` を行カーソル対応に書き換え:

  ```go
  func (m diffviewModel) renderCurrent() string {
      if m.fileIdx >= len(m.diffs) {
          return ""
      }
      f := m.diffs[m.fileIdx]
      cmtMap := commentsByLine(m.data.Comments, strings.TrimPrefix(f.NewPath, "b/"))

      cursor := lipgloss.NewStyle().Background(lipgloss.Color("8"))

      var b strings.Builder
      fmt.Fprintf(&b, "%s\n%s\n\n",
          headerStyle.Render("--- "+f.OldPath),
          headerStyle.Render("+++ "+f.NewPath),
      )
      n := 0
      for _, h := range f.Hunks {
          b.WriteString(headerStyle.Render(h.Header))
          b.WriteByte('\n')
          for _, l := range h.Lines {
              line := RenderLine(l)
              // コメントマーカー
              if cs, ok := cmtMap[l.NewNo]; ok && l.NewNo > 0 {
                  for _, c := range cs {
                      m := "💬 " + truncate(c.Text, 60)
                      if !c.SentAt.IsZero() {
                          m = SentCommentStyle.Render(m)
                      }
                      line = line + "\n  " + m
                  }
              }
              if n == m.lineIdx {
                  line = cursor.Render(line)
              }
              b.WriteString(line)
              b.WriteByte('\n')
              n++
          }
          b.WriteByte('\n')
      }
      return b.String()
  }

  func commentsByLine(cs []storage.Comment, file string) map[int][]storage.Comment {
      out := map[int][]storage.Comment{}
      for _, c := range cs {
          if c.File != file {
              continue
          }
          for line := c.LineStart; line <= c.LineEnd; line++ {
              out[line] = append(out[line], c)
          }
      }
      return out
  }

  func truncate(s string, n int) string {
      if len(s) <= n {
          return s
      }
      return s[:n] + "…"
  }
  ```

- [ ] **Step 2:** ビルド + 既存テスト通過:

  ```bash
  go test ./internal/tui/...
  ```

- [ ] **Step 3:** コミット:

  ```bash
  git add internal/tui/diffview.go
  git commit -m "feat(tui): 行カーソルのハイライトとコメントマーカー表示"
  ```

### Task 19: `s` でコメント送信（cli の renderSendPrompt と SendToPane を再利用）

**ファイル:**
- 修正: `internal/tui/diffview.go`
- 修正: `internal/cli/comment.go`（`renderSendPrompt` を export）

**機能:**
- `s` で未送信コメントをまとめて送信
- 結果はステータスバーに表示

- [ ] **Step 1:** `internal/cli/comment.go` の `renderSendPrompt` を別パッケージから使えるよう、`internal/storage` に移動するか別パッケージに切り出す。一番自然な場所は新規 `internal/sendprompt/sendprompt.go`。

  作成: `internal/sendprompt/sendprompt.go`:

  ```go
  // Package sendprompt は未送信コメント群から Claude に送る固定テンプレ文字列を組み立てる。
  // CLI と TUI の両方から呼ばれる純粋関数のみを置く。
  package sendprompt

  import (
      "fmt"
      "strings"

      "github.com/yusei-wy/tmux-agent-log/internal/storage"
  )

  // Render は未送信コメント群から Claude に送る固定テンプレを返す。spec §5.4 のテンプレ。
  func Render(cs []storage.Comment) string {
      var b strings.Builder
      b.WriteString("以下のレビューコメントを反映してください:\n\n")
      for _, c := range cs {
          fmt.Fprintf(&b, "- %s:%d-%d\n  %s\n\n", c.File, c.LineStart, c.LineEnd, c.Text)
      }
      b.WriteString("(反映後、関連テストを実行して結果を報告してください)")
      return b.String()
  }
  ```

- [ ] **Step 2:** `internal/cli/comment.go` の `renderSendPrompt` を `sendprompt.Render` に置換:

  ```go
  prompt := sendprompt.Render(unsent)
  ```

  既存の `func renderSendPrompt(...)` は削除。`internal/sendprompt` を import。

- [ ] **Step 3:** 既存 cli テストが通ることを確認:

  ```bash
  go test ./internal/cli/...
  ```

- [ ] **Step 4:** `internal/tui/diffview.go` に `s` / `S` ハンドラを追加:

  ```go
  import (
      "github.com/yusei-wy/tmux-agent-log/internal/sendprompt"
      "github.com/yusei-wy/tmux-agent-log/internal/tmux"
  )

  // Update のスイッチに追加:
  case key.Matches(msg, keys.Send):
      // s: 即送信
      return m, m.sendUnsent(false)
  case msg.String() == "S":
      // S: $EDITOR でプレビュー編集してから送信
      return m, m.sendUnsent(true)
  ```

  ```go
  type sendDoneMsg struct {
      sent int
      err  error
  }

  // sendUnsent は未送信コメント群を Claude pane に送る。
  // preview=true の場合は $EDITOR でプロンプトを開いて編集後に送信する。spec §5.4 の S キー対応。
  func (m diffviewModel) sendUnsent(preview bool) tea.Cmd {
      var unsent []storage.Comment
      for _, c := range m.data.Comments {
          if c.SentAt.IsZero() {
              unsent = append(unsent, c)
          }
      }
      if len(unsent) == 0 {
          return func() tea.Msg { return statusMsg("未送信コメントがない") }
      }
      pane := m.data.Meta.TmuxPane
      sessionDir := m.data.Dir
      ids := make([]string, 0, len(unsent))
      for _, c := range unsent {
          ids = append(ids, c.ID)
      }
      basePrompt := sendprompt.Render(unsent)

      send := func(prompt string) tea.Msg {
          res := tmux.SendToPane(pane, prompt)
          if res.Kind == tmux.SendResultFailed {
              return sendDoneMsg{err: res.Err}
          }
          path := filepath.Join(sessionDir, "comments.jsonl")
          if err := storage.MarkCommentsSent(path, ids, time.Now().UTC()); err != nil {
              return sendDoneMsg{err: err}
          }
          return sendDoneMsg{sent: len(unsent)}
      }

      if !preview {
          return func() tea.Msg { return send(basePrompt) }
      }

      // preview=true: 一時ファイルに書いて $EDITOR で開く
      tmp, err := os.CreateTemp("", "tmux-agent-log-send-*.md")
      if err != nil {
          return func() tea.Msg { return statusMsg("tmpfile 失敗: " + err.Error()) }
      }
      if _, werr := tmp.WriteString(basePrompt); werr != nil {
          _ = tmp.Close()
          _ = os.Remove(tmp.Name())
          return func() tea.Msg { return statusMsg("tmpfile 書込失敗: " + werr.Error()) }
      }
      _ = tmp.Close()

      editor := os.Getenv("EDITOR")
      if editor == "" {
          editor = "vi"
      }
      c := exec.Command(editor, tmp.Name())
      return tea.ExecProcess(c, func(err error) tea.Msg {
          if err != nil {
              _ = os.Remove(tmp.Name())
              return sendDoneMsg{err: err}
          }
          body, rerr := os.ReadFile(tmp.Name())
          _ = os.Remove(tmp.Name())
          if rerr != nil {
              return sendDoneMsg{err: rerr}
          }
          text := strings.TrimSpace(string(body))
          if text == "" {
              return statusMsg("送信キャンセル (空のプロンプト)")
          }
          return send(text)
      })
  }
  ```

- [ ] **Step 5:** Update に `sendDoneMsg` ハンドラを追加（statusMsg と dataReloadedMsg を返す）:

  ```go
  case sendDoneMsg:
      if msg.err != nil {
          return m, func() tea.Msg { return statusMsg("送信失敗: " + msg.err.Error()) }
      }
      data, err := LoadSession(m.data.Dir)
      if err != nil {
          return m, func() tea.Msg { return statusMsg("再読込失敗: " + err.Error()) }
      }
      m.data = data
      return m, tea.Batch(
          func() tea.Msg { return dataReloadedMsg(data) },
          func() tea.Msg { return statusMsg(fmt.Sprintf("送信成功: %d 件", msg.sent)) },
      )
  ```

- [ ] **Step 6:** ビルド + 既存テスト確認:

  ```bash
  go test ./...
  ```

- [ ] **Step 7:** コミット:

  ```bash
  git add internal/tui/diffview.go internal/cli/comment.go internal/sendprompt/sendprompt.go
  git commit -m "feat(tui): s で即送信、S で \$EDITOR プレビュー編集後に送信"
  ```

### Task 20: `d` でコメント削除

**ファイル:**
- 修正: `internal/tui/diffview.go`

**機能:**
- カーソル行に最も近い未送信コメントを論理削除
- 送信済みコメントは削除しない（appended-only ストアでは sent_at は残るが論理削除自体は可、ただし v0.1 では未送信のみ削除可能とする）

- [ ] **Step 1:** ハンドラ追加（Update スイッチに）:

  ```go
  case key.Matches(msg, keys.Delete):
      return m, m.deleteAtCursor()
  ```

  ```go
  type commentDeletedMsg struct{ err error }

  func (m diffviewModel) deleteAtCursor() tea.Cmd {
      file, line, ok := m.currentAnchor()
      if !ok {
          return func() tea.Msg { return statusMsg("削除可能な行ではない") }
      }
      var targetID string
      for _, c := range m.data.Comments {
          if c.SentAt.IsZero() && c.File == file && line >= c.LineStart && line <= c.LineEnd {
              targetID = c.ID
              break
          }
      }
      if targetID == "" {
          return func() tea.Msg { return statusMsg("この行には未送信コメントがない") }
      }
      sessionDir := m.data.Dir
      return func() tea.Msg {
          path := filepath.Join(sessionDir, "comments.jsonl")
          if err := storage.DeleteComment(path, targetID); err != nil {
              return commentDeletedMsg{err: err}
          }
          return commentDeletedMsg{}
      }
  }
  ```

- [ ] **Step 2:** Update に `commentDeletedMsg` ハンドラ:

  ```go
  case commentDeletedMsg:
      if msg.err != nil {
          return m, func() tea.Msg { return statusMsg("削除失敗: " + msg.err.Error()) }
      }
      data, err := LoadSession(m.data.Dir)
      if err != nil {
          return m, func() tea.Msg { return statusMsg("再読込失敗: " + err.Error()) }
      }
      m.data = data
      return m, tea.Batch(
          func() tea.Msg { return dataReloadedMsg(data) },
          func() tea.Msg { return statusMsg("コメント削除") },
      )
  ```

- [ ] **Step 3:** ビルド + テスト:

  ```bash
  go test ./...
  ```

- [ ] **Step 4:** コミット:

  ```bash
  git add internal/tui/diffview.go
  git commit -m "feat(tui): d で未送信コメントを削除"
  ```

---

## Phase 6: overview タブ（簡易版）

spec §0.4 で「overview タブ」は v0.1 含みだが、§11.2 で「本格実装は v0.3」と振り分けた。v0.1 では「全 session を goal でグルーピングして表示するだけ」の最小版。

### Task 21: overview の loader と View

**ファイル:**
- 修正: `internal/tui/overview.go`
- 作成: `internal/tui/overview_test.go`

**機能:**
- 全 project の全 session を読み込み、goal で group by
- 各セッションの turn 数 / 最終更新時刻を表示
- ナビゲーション、フィルタは v0.3 で本格化

- [ ] **Step 1:** loader を追加（`internal/tui/loader.go`）:

  ```go
  type SessionRow struct {
      Project    string
      ID         string
      Goal       string
      Turns      int
      LastTouch  time.Time
  }

  func LoadAllSessions() ([]SessionRow, error) {
      state, err := config.StateDir()
      if err != nil {
          return nil, err
      }
      projects := filepath.Join(state, "projects")
      entries, err := os.ReadDir(projects)
      if err != nil {
          if errors.Is(err, fs.ErrNotExist) {
              return nil, nil
          }
          return nil, err
      }
      var out []SessionRow
      for _, p := range entries {
          if !p.IsDir() {
              continue
          }
          sessDir := filepath.Join(projects, p.Name(), "sessions")
          ses, err := os.ReadDir(sessDir)
          if err != nil {
              continue
          }
          for _, s := range ses {
              if !s.IsDir() {
                  continue
              }
              dir := filepath.Join(sessDir, s.Name())
              meta, err := storage.ReadSessionMeta(dir)
              if err != nil {
                  continue
              }
              turns, _ := storage.ReadTurns(filepath.Join(dir, "turns.jsonl"))
              info, _ := os.Stat(dir)
              var lastTouch time.Time
              if info != nil {
                  lastTouch = info.ModTime()
              }
              out = append(out, SessionRow{
                  Project:   p.Name(),
                  ID:        meta.ClaudeSessionID,
                  Goal:      meta.Goal,
                  Turns:     len(turns),
                  LastTouch: lastTouch,
              })
          }
      }
      sort.Slice(out, func(i, j int) bool { return out[i].LastTouch.After(out[j].LastTouch) })
      return out, nil
  }
  ```

  必要 import: `errors`, `io/fs`, `os`, `path/filepath`, `sort`, `time`, `github.com/yusei-wy/tmux-agent-log/internal/config`。

- [ ] **Step 2:** `internal/tui/overview.go` を本実装:

  ```go
  package tui

  import (
      "fmt"
      "strings"
      "time"

      tea "github.com/charmbracelet/bubbletea"
  )

  type overviewModel struct {
      data SessionData
      rows []SessionRow
      err  error
  }

  func newOverview(d SessionData) overviewModel {
      rows, err := LoadAllSessions()
      return overviewModel{data: d, rows: rows, err: err}
  }

  func (m overviewModel) Update(msg tea.Msg, keys Keys) (overviewModel, tea.Cmd) {
      return m, nil
  }

  func (m overviewModel) View(width, height int) string {
      if m.err != nil {
          return errStyle().Render("overview 読込エラー: " + m.err.Error())
      }
      if len(m.rows) == 0 {
          return FaintStyle.Render("(まだ session がない)")
      }
      // goal でグループ化
      groups := map[string][]SessionRow{}
      var goalOrder []string
      for _, r := range m.rows {
          key := r.Goal
          if key == "" {
              key = "(no goal)"
          }
          if _, ok := groups[key]; !ok {
              goalOrder = append(goalOrder, key)
          }
          groups[key] = append(groups[key], r)
      }
      var b strings.Builder
      for _, g := range goalOrder {
          fmt.Fprintf(&b, "%s\n", TabActiveStyle.Render("🎯 "+g))
          for _, r := range groups[g] {
              fmt.Fprintf(&b, "  %s/%s   %d turns   last %s\n",
                  r.Project, shortID(r.ID), r.Turns, ago(r.LastTouch))
          }
          b.WriteByte('\n')
      }
      return b.String()
  }

  func shortID(id string) string {
      if len(id) > 8 {
          return id[:8]
      }
      return id
  }

  func ago(t time.Time) string {
      d := time.Since(t)
      switch {
      case d < time.Minute:
          return "just now"
      case d < time.Hour:
          return fmt.Sprintf("%dm ago", int(d.Minutes()))
      case d < 24*time.Hour:
          return fmt.Sprintf("%dh ago", int(d.Hours()))
      default:
          return fmt.Sprintf("%dd ago", int(d.Hours()/24))
      }
  }
  ```

- [ ] **Step 3:** テスト:

  ```go
  func TestOverviewListsSessions(t *testing.T) {
      // XDG_STATE_HOME を tempdir に向ける
      stateRoot := t.TempDir()
      t.Setenv("XDG_STATE_HOME", stateRoot)

      sDir := filepath.Join(stateRoot, "tmux-agent-log", "projects", "p1", "sessions", "s1")
      require.NoError(t, os.MkdirAll(sDir, 0o700))
      require.NoError(t, storage.WriteSessionMeta(sDir, storage.SessionMeta{
          ClaudeSessionID: "s1", Goal: "fix-bug", Cwd: "/p1", StartedAt: time.Now(),
      }))

      data := tui.SessionData{Dir: sDir}
      m := tui.New(data)
      // overview タブへ (tab 2 回)
      next, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
      m = next.(tui.Model)
      next, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
      m = next.(tui.Model)
      next, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
      m = next.(tui.Model)

      view := m.View()
      require.Contains(t, view, "fix-bug")
      require.Contains(t, view, "p1")
  }
  ```

- [ ] **Step 4:** テスト通過:

  ```bash
  go test ./internal/tui/... -v
  ```

- [ ] **Step 5:** コミット:

  ```bash
  git add internal/tui/overview.go internal/tui/overview_test.go internal/tui/loader.go
  git commit -m "feat(tui): overview タブの簡易版 (goal グルーピング)"
  ```

---

## Phase 7: end-to-end と書き上げ

### Task 22: smoke スクリプトに tui / tail を追加

**ファイル:**
- 修正: `scripts/smoke.sh`

- [ ] **Step 1:** 既存 `scripts/smoke.sh` を読み、最後に追加:

  ```sh
  # tui --help と tail --help が落ちないこと
  $BIN tui --help > /dev/null
  $BIN tail --help > /dev/null

  # tui を 1 秒だけ起動して即終了（PTY が無い CI 環境では skip 想定）
  if [ -n "$TMUX_AGENT_LOG_SMOKE_PTY" ]; then
      timeout 1s $BIN tui --session "$SESSION" || true
  fi

  echo "  tui/tail SMOKE OK"
  ```

  既存の SMOKE OK 出力にぶら下げる位置に追加。

- [ ] **Step 2:** ローカルで実行:

  ```bash
  ./scripts/smoke.sh
  ```

  期待: 既存 SMOKE OK + `tui/tail SMOKE OK`。

- [ ] **Step 3:** コミット:

  ```bash
  git add scripts/smoke.sh
  git commit -m "test(smoke): tui / tail サブコマンドを smoke に追加"
  ```

### Task 23: README に tui / tail の節を追加

**ファイル:**
- 修正: `README.md`

- [ ] **Step 1:** 既存 README に節を追加:

  ```markdown
  ## tui — 対話 TUI

  bubbletea ベースの対話 UI。3 タブ: timeline / diff / overview。

  ```sh
  tmux-agent-log tui                        # 現プロジェクトの最新 session
  tmux-agent-log tui --session <id>         # 特定 session
  ```

  キーバインド:

  | キー | 動作 |
  |---|---|
  | `tab` / `shift+tab` | タブ切替 |
  | `j`/`k` または ↑↓ | カーソル上下 |
  | `h`/`l` または ←→ | diff タブでファイル切替 |
  | `c` | diff タブで行コメント追加 (`$EDITOR` 起動) |
  | `s` | 未送信コメントを固定テンプレで Claude pane に送信 |
  | `S` | `$EDITOR` でプレビュー編集してから送信 |
  | `d` | カーソル行のコメントを削除 |
  | `q` | 終了 |

  ## tail — ライブ tail viewer

  fsnotify で session ディレクトリを watch し、最新 N turn を再描画する。

  ```sh
  tmux-agent-log tail                       # 現プロジェクトの最新 session
  tmux-agent-log tail --session <id> --max-turns 12
  ```

  Ctrl+C で終了。
  ```

- [ ] **Step 2:** コミット:

  ```bash
  git add README.md
  git commit -m "docs: README に tui / tail の節を追加"
  ```

---

## セルフレビューチェックリスト

全 23 タスク完了後に確認:

- [ ] `go build ./...` が警告なしで成功する
- [ ] `go test -race ./...` が全部 green
- [ ] `mise run check` が成功する
- [ ] `./scripts/smoke.sh` が `SMOKE OK` を出力（tui/tail も含む）
- [ ] **手動 e2e:**
  - 既存 session で `tmux-agent-log tui` が起動し、3 タブが切り替えられる
  - timeline タブで turn を選択して詳細が表示される
  - diff タブで色付き diff が表示され、↑↓ でカーソル移動、←→ でファイル切替できる
  - diff タブで `c` を押すと `$EDITOR` が起動し、保存後にコメントが追加されている
  - diff タブで `s` を押すと現セッションの Claude pane にプロンプトが送られる
  - diff タブで `d` で未送信コメントを削除できる
  - overview タブで goal 別にセッションがグルーピングされて見える
  - `tmux-agent-log tail` が起動し、別セッションで turn 追加すると数秒以内に画面が更新される
- [ ] 全タスクがそれぞれ独立したコミットになっている

## 次の Plan

spec §0.4 / 引き算の哲学（§5.0）に沿った後続:

- **Plan 3（v0.2 — Semantic）:** per-turn diff モード + liveness マーカー、`vs-main` ブランチ切替、`blame.json` + `liveness.json` の増分更新（turn-end hook）、line → turn blame footer + intent 表示
- **Plan 4（v0.3 — Storytelling）:** overview タブの本格実装（フィルタ・ナビゲーション・カーソル）、`narrate` CLI（goal 単位の変更ストーリー Markdown 出力）、chroma による言語別 syntax highlight
- **Plan 5（配布）:** `examples/`（tmux popup / split-pane / 専用 window のスニペット、fzf 合成例）、GitHub Actions CI（Ubuntu + macOS）、Darwin/Linux × amd64/arm64 の GoReleaser、Homebrew tap
- **v0.4 — Comment Authoring（保留）:** spec 付録 B 参照。v0.3 リリース後に再判断

---

*Plan 2 of 5（v0.1 Phase B）。`docs/superpowers/specs/2026-04-23-tmux-agent-log-design.md` §0.4 / §11.1 / §5.3 / §5.4 / §9 から `superpowers:writing-plans` で生成。*
