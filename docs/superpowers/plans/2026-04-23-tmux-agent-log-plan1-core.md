# tmux-agent-log Plan 1: コア (MVP) 実装計画

> **エージェント実装者向け:** 必須サブスキル: `superpowers:subagent-driven-development`（推奨）または `superpowers:executing-plans` を使ってタスク単位で実装する。各 step はチェックボックス（`- [ ]`）で進捗管理する。

**目的:** `tmux-agent-log` の CLI コアを構築する。Claude Code の turn を構造化 JSONL として記録し、turn 境界で git diff のスナップショットを取り、CLI サブコマンド経由でレビュー / コメント / Claude への送り返しを可能にする。Go 単一バイナリ、OSS リリース可能な状態。

**アーキテクチャ:** Cobra サブコマンドを持つ単一バイナリ。hook サブコマンドは Claude Code hook の JSON を stdin から読み、JSONL に追記する（必ず exit 0）。ユーザー向けサブコマンドは JSONL ストアとセッションの git リポジトリを読み、machine-readable な出力（tsv / jsonl / json / table）を生成する。TUI と tail viewer は Plan 2 へ、examples とリリース基盤は Plan 3 へ先送り。

**技術スタック:**
- **Go 1.26+**
- `github.com/spf13/cobra` — サブコマンドルーター
- `github.com/BurntSushi/toml` — config
- `github.com/google/uuid` — id 生成
- `github.com/gofrs/flock` — ファイルロック（v0.12+ API: `TryLockContext`）
- `github.com/stretchr/testify` — assertion
- 外部ランタイム: `tmux` 3.2+、`git`

**ソース spec:** [`docs/superpowers/specs/2026-04-23-tmux-agent-log-design.md`](../specs/2026-04-23-tmux-agent-log-design.md)

**この計画のスコープ外**（spec §11 MVP 参照）:
- TUI（`tmux-agent-log tui`）— Plan 2
- fsnotify ベースの tail viewer — Plan 2
- `blame.json` / `liveness.json` の計算 — Plan 2
- syntax highlighting（chroma）— Plan 2
- `examples/`、GitHub Actions CI、GoReleaser — Plan 3

**記述言語:** 本計画で作成するコード・コメント・テスト・コミットメッセージ・ドキュメントはすべて **日本語** で記述する。ただし Go のテスト名（`TestXxx`）・関数名・型名など言語仕様上英語が必要な箇所は英語のままとする。

---

## ファイル構成（本計画で作成）

```
tmux-agent-log/
├── cmd/tmux-agent-log/main.go
├── internal/
│   ├── cli/           # cobra wiring (hook + ユーザーコマンド)
│   │   ├── root.go
│   │   ├── hook.go
│   │   ├── list.go            # list-sessions / list-turns / list-comments
│   │   ├── show.go            # show-session / show-turn / show-diff
│   │   ├── goal.go
│   │   ├── comment.go         # comment add/list/delete/send
│   │   ├── clear.go
│   │   ├── export.go
│   │   ├── config_cmd.go
│   │   ├── errors_cmd.go
│   │   └── install.go         # install-hooks / uninstall-hooks
│   ├── config/
│   │   ├── paths.go           # XDG ディレクトリ、project slug、session ディレクトリ
│   │   └── config.go          # TOML ロード + デフォルト
│   ├── storage/
│   │   ├── model.go           # SessionMeta, Turn, Event, Comment, ErrEntry
│   │   ├── jsonl.go           # append + read（flock + 壊れ行 skip）
│   │   ├── session.go         # meta.json CRUD
│   │   ├── turn.go            # open/close 合流読み
│   │   ├── event.go           # event 追記 / フィルタ読み
│   │   └── comment.go         # comment CRUD + sent マーク
│   ├── errlog/errlog.go       # errors.jsonl writer
│   ├── hook/
│   │   ├── hook.go            # stdin parse + recover + exit 0
│   │   ├── session_start.go
│   │   ├── turn_start.go
│   │   ├── tool.go            # tool-pre + tool-post（共通）
│   │   └── turn_end.go
│   ├── git/
│   │   ├── git.go             # exec ラッパー
│   │   ├── detect.go          # IsRepo / HeadSHA
│   │   └── diff.go            # DiffSince(base)
│   ├── tmux/
│   │   ├── tmux.go            # 環境変数 + pane 存在確認
│   │   └── sendkeys.go        # SendToPane + OSC 52 fallback
│   └── format/format.go       # tsv / jsonl / json / table
├── scripts/smoke.sh
├── go.mod / go.sum
├── .gitignore
├── README.md
└── docs/                       # 既存
    ├── brainstorming/2026-04-22-tmux-agent-log.md
    └── superpowers/
        ├── specs/2026-04-23-tmux-agent-log-design.md
        └── plans/2026-04-23-tmux-agent-log-plan1-core.md  ← 本ファイル
```

## 共有データモデル

すべての型は `internal/storage/model.go` に置く。以下のタスクからはこれらを参照する（再定義しない）。

```go
package storage

import "time"

type SessionMeta struct {
    ClaudeSessionID string    `json:"claude_session_id"`
    TmuxPane        string    `json:"tmux_pane,omitempty"`
    Cwd             string    `json:"cwd"`
    Goal            string    `json:"goal,omitempty"`
    BaseSHA         string    `json:"base_sha,omitempty"`
    GitTracked      bool      `json:"git_tracked"`
    StartedAt       time.Time `json:"started_at"`
    TranscriptPath  string    `json:"transcript_path,omitempty"`
}

type TurnOpen struct {
    ID                  string    `json:"id"`
    Phase               string    `json:"phase"` // 常に "open"
    StartedAt           time.Time `json:"started_at"`
    UserPromptPreview   string    `json:"user_prompt_preview,omitempty"`
    HeadSHAPre          string    `json:"head_sha_pre,omitempty"`
    TranscriptMessageID string    `json:"transcript_message_id,omitempty"`
}

type TurnClose struct {
    ID                      string    `json:"id"`
    Phase                   string    `json:"phase"` // 常に "close"
    EndedAt                 time.Time `json:"ended_at"`
    AssistantSummaryPreview string    `json:"assistant_summary_preview,omitempty"`
    HeadSHA                 string    `json:"head_sha,omitempty"`
    DiffPath                string    `json:"diff_path,omitempty"`
    Status                  string    `json:"status"` // "done" | "error"
    ErrorMessage            string    `json:"error_message,omitempty"`
}

// Turn は open ∪ close の合流ビュー。Status="open" は open レコードのみ存在する状態。
type Turn struct {
    ID                      string    `json:"id"`
    StartedAt               time.Time `json:"started_at"`
    EndedAt                 time.Time `json:"ended_at,omitempty"`
    UserPromptPreview       string    `json:"user_prompt_preview,omitempty"`
    AssistantSummaryPreview string    `json:"assistant_summary_preview,omitempty"`
    HeadSHAPre              string    `json:"head_sha_pre,omitempty"`
    HeadSHA                 string    `json:"head_sha,omitempty"`
    DiffPath                string    `json:"diff_path,omitempty"`
    Status                  string    `json:"status"` // "open" | "done" | "error"
    ErrorMessage            string    `json:"error_message,omitempty"`
    TranscriptMessageID     string    `json:"transcript_message_id,omitempty"`
}

type Event struct {
    ID           string    `json:"id"`
    TurnID       string    `json:"turn_id"`
    Ts           time.Time `json:"ts"`
    Tool         string    `json:"tool"`
    ArgsPreview  string    `json:"args_preview,omitempty"`
    Phase        string    `json:"phase"` // "pre" | "post"
    Success      bool      `json:"success,omitempty"`
    ErrorMessage string    `json:"error_message,omitempty"`
}

type Comment struct {
    ID        string    `json:"id"`
    File      string    `json:"file"`
    LineStart int       `json:"line_start"`
    LineEnd   int       `json:"line_end"`
    Text      string    `json:"text"`
    CreatedAt time.Time `json:"created_at"`
    SentAt    time.Time `json:"sent_at,omitempty"`
}

type ErrEntry struct {
    Ts          time.Time `json:"ts"`
    Component   string    `json:"component"`
    Event       string    `json:"event"`
    SessionID   string    `json:"session_id,omitempty"`
    ErrorString string    `json:"error"`
}
```

---

## Phase 0: ブートストラップ

### Task 1: git 初期化と最初のコミット

**ファイル:**
- 作成: `.gitignore`

- [ ] **Step 1:** `.gitignore` を作成:

  ```gitignore
  # ビルド成果物
  /tmux-agent-log
  /dist/

  # Go
  *.test
  *.out
  coverage.out
  coverage.html

  # エディタ / OS
  .vscode/
  .idea/
  *.swp
  .DS_Store
  ```

- [ ] **Step 2:** git を初期化し、既存の docs を初回コミット:

  ```bash
  git init
  git branch -M main
  git add .gitignore docs/
  git commit -m "chore: spec とブレインストーミングでリポジトリを初期化"
  ```

  期待: `main` に `.gitignore` + `docs/` を含む root commit が 1 つ。

### Task 2: Go モジュール

**ファイル:**
- 作成: `go.mod`, `go.sum`

- [ ] **Step 1:** モジュール初期化と依存追加:

  ```bash
  go mod init github.com/yusei-wy/tmux-agent-log
  go get github.com/spf13/cobra@latest
  go get github.com/BurntSushi/toml@latest
  go get github.com/google/uuid@latest
  go get github.com/gofrs/flock@latest
  go get github.com/stretchr/testify@latest
  ```

- [ ] **Step 2:** `go.mod` で `go 1.26` が宣言されていることを確認（`go get` が古いバージョンを書いた場合は手動で修正）。

- [ ] **Step 3:** コミット:

  ```bash
  git add go.mod go.sum
  git commit -m "chore: Go 1.26 モジュールを cobra/toml/uuid/flock/testify で初期化"
  ```

### Task 3: エントリポイントの足場

**ファイル:**
- 作成: `cmd/tmux-agent-log/main.go`
- 作成: `internal/cli/root.go`

**Exports:**
- `cli.Execute() error` — cobra root を実行する。`main` から呼ばれる。

- [ ] **Step 1:** `cmd/tmux-agent-log/main.go`:

  ```go
  package main

  import (
      "os"

      "github.com/yusei-wy/tmux-agent-log/internal/cli"
  )

  func main() {
      if err := cli.Execute(); err != nil {
          os.Exit(1)
      }
  }
  ```

- [ ] **Step 2:** `internal/cli/root.go`:

  ```go
  package cli

  import "github.com/spf13/cobra"

  var rootCmd = &cobra.Command{
      Use:   "tmux-agent-log",
      Short: "tmux 内で動く Claude Code のための構造化履歴レイヤー",
      Long:  "Claude Code セッションを構造化 JSONL で記録し、turn ごとの diff を取り、行コメントを agent に送り返せる。",
  }

  // Execute は cmd/tmux-agent-log/main.go から呼ばれるエントリポイント。
  func Execute() error {
      return rootCmd.Execute()
  }
  ```

- [ ] **Step 3:** ビルドして `--help` で配線確認:

  ```bash
  go build -o /tmp/tmux-agent-log ./cmd/tmux-agent-log
  /tmp/tmux-agent-log --help
  ```

  期待: cobra が生成したヘルプが表示される。

- [ ] **Step 4:** コミット:

  ```bash
  git add cmd/ internal/cli/root.go
  git commit -m "feat: cobra root command で CLI エントリポイントの足場を作成"
  ```

---

## Phase 1: Config とパス

### Task 4: XDG パスと project slug

**ファイル:**
- 作成: `internal/config/paths.go` と `paths_test.go`

**Exports:**
- `StateDir() (string, error)` — `$XDG_STATE_HOME/tmux-agent-log` または `~/.local/state/tmux-agent-log`
- `ConfigDir() (string, error)` — `$XDG_CONFIG_HOME/tmux-agent-log` または `~/.config/tmux-agent-log`
- `ProjectSlug(cwd string) string` — `"<basename>-<8hex>"`。hex は `sha256(cwd)` の先頭 4 byte
- `SessionDir(cwd, claudeSessionID string) (string, error)` — `StateDir/projects/<slug>/sessions/<uuid>`
- `ErrorsPath() (string, error)` — `StateDir/errors.jsonl`

**振る舞い契約（テスト）:**

```go
package config

import (
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/require"
)

func TestStateDirDefaultsToXDG(t *testing.T) {
    t.Setenv("XDG_STATE_HOME", "/tmp/xdg-state")
    dir, err := StateDir()
    require.NoError(t, err)
    require.Equal(t, "/tmp/xdg-state/tmux-agent-log", dir)
}

func TestStateDirFallbackToHome(t *testing.T) {
    t.Setenv("XDG_STATE_HOME", "")
    t.Setenv("HOME", "/tmp/myhome")
    dir, err := StateDir()
    require.NoError(t, err)
    require.Equal(t, "/tmp/myhome/.local/state/tmux-agent-log", dir)
}

func TestProjectSlug(t *testing.T) {
    slug := ProjectSlug("/Users/alice/src/myproject")
    require.Contains(t, slug, "myproject-")
    require.Len(t, slug, len("myproject-")+8)
}

func TestSessionDir(t *testing.T) {
    t.Setenv("XDG_STATE_HOME", "/tmp/xdg-state")
    got, err := SessionDir("/Users/alice/src/myproject", "abc-123")
    require.NoError(t, err)
    want := filepath.Join("/tmp/xdg-state/tmux-agent-log/projects", ProjectSlug("/Users/alice/src/myproject"), "sessions", "abc-123")
    require.Equal(t, want, got)
}
```

**実装メモ:**
- `os.Getenv("XDG_STATE_HOME")` / `XDG_CONFIG_HOME` / `HOME` を使う
- XDG 環境変数と `HOME` の両方が未設定なら明確なエラーを返す
- `ProjectSlug`: `filepath.Base(cwd)` → `/` を `_` に置換 → `-` + `hex.EncodeToString(sha256(cwd)[:4])` を追加

**コミット:** `feat(config): XDG 準拠の state/config パスと project slug を追加`

### Task 5: TOML config ローダー

**ファイル:**
- 作成: `internal/config/config.go` と `config_test.go`

**Exports:**

```go
const DefaultSendEditorCommand = "${EDITOR:-nvim}"

type Config struct {
    SendEditorCommand    string `toml:"send_editor_command"`
    DisableOSC52Fallback bool   `toml:"disable_osc52_fallback"`
    StateDirOverride     string `toml:"state_dir"` // 絶対パス、XDG を上書き
}

func Load() (Config, error)
```

**振る舞い契約（テスト）:**

```go
package config

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/require"
)

func TestLoadReturnsDefaultsWhenFileMissing(t *testing.T) {
    t.Setenv("XDG_CONFIG_HOME", t.TempDir())
    cfg, err := Load()
    require.NoError(t, err)
    require.Equal(t, DefaultSendEditorCommand, cfg.SendEditorCommand)
    require.False(t, cfg.DisableOSC52Fallback)
}

func TestLoadReadsTOMLOverrides(t *testing.T) {
    dir := t.TempDir()
    t.Setenv("XDG_CONFIG_HOME", dir)
    path := filepath.Join(dir, "tmux-agent-log", "config.toml")
    require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
    body := "send_editor_command = \"vim\"\ndisable_osc52_fallback = true\n"
    require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

    cfg, err := Load()
    require.NoError(t, err)
    require.Equal(t, "vim", cfg.SendEditorCommand)
    require.True(t, cfg.DisableOSC52Fallback)
}
```

**実装メモ:**
- ファイルが存在しなければ `Load()` はデフォルトを返す（エラーなし）
- `BurntSushi/toml.Decode` を使う
- TOML で `send_editor_command` が空文字列の場合も `DefaultSendEditorCommand` にフォールバック

**コミット:** `feat(config): TOML config ローダーをデフォルト付きで追加`

---

## Phase 2: ストレージ基盤

### Task 6: データモデル

**ファイル:**
- 作成: `internal/storage/model.go`

- [ ] 上記の「共有データモデル」セクションの型定義を `internal/storage/model.go` にコピーする。テスト不要（コンパイルのみ）。

- [ ] `go build ./...` が通ることを確認。

**コミット:** `feat(storage): データモデル型を追加`

### Task 7: flock 付き JSONL 追記 + 読取

**ファイル:**
- 作成: `internal/storage/jsonl.go` と `jsonl_test.go`

**Exports:**

```go
func AppendJSONL(path string, v interface{}) error
func AppendRaw(path string, line []byte) error
func ReadJSONL(path string, fn func(raw []byte) error) error

const flockTimeout = 500 * time.Millisecond
```

**振る舞い契約（テスト）:**

```go
package storage

import (
    "encoding/json"
    "path/filepath"
    "sync"
    "testing"

    "github.com/stretchr/testify/require"
)

type sampleRec struct {
    ID   string `json:"id"`
    Text string `json:"text"`
}

func TestAppendAndReadLines(t *testing.T) {
    path := filepath.Join(t.TempDir(), "a.jsonl")
    require.NoError(t, AppendJSONL(path, sampleRec{ID: "a", Text: "hello"}))
    require.NoError(t, AppendJSONL(path, sampleRec{ID: "b", Text: "world"}))

    var got []sampleRec
    require.NoError(t, ReadJSONL(path, func(raw []byte) error {
        var r sampleRec
        if err := json.Unmarshal(raw, &r); err != nil {
            return nil
        }
        got = append(got, r)
        return nil
    }))
    require.Equal(t, []sampleRec{{ID: "a", Text: "hello"}, {ID: "b", Text: "world"}}, got)
}

func TestReadSkipsCorruptedLines(t *testing.T) {
    path := filepath.Join(t.TempDir(), "b.jsonl")
    require.NoError(t, AppendRaw(path, []byte(`{"id":"a"}`)))
    require.NoError(t, AppendRaw(path, []byte(`NOT JSON`)))
    require.NoError(t, AppendRaw(path, []byte(`{"id":"b"}`)))

    good := 0
    corrupted := 0
    require.NoError(t, ReadJSONL(path, func(raw []byte) error {
        var r sampleRec
        if err := json.Unmarshal(raw, &r); err != nil {
            corrupted++
            return nil
        }
        good++
        return nil
    }))
    require.Equal(t, 2, good)
    require.Equal(t, 1, corrupted)
}

func TestConcurrentAppendsSerialize(t *testing.T) {
    path := filepath.Join(t.TempDir(), "c.jsonl")
    var wg sync.WaitGroup
    const n = 50
    for i := 0; i < n; i++ {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            require.NoError(t, AppendJSONL(path, sampleRec{ID: "x"}))
        }(i)
    }
    wg.Wait()
    total := 0
    require.NoError(t, ReadJSONL(path, func(raw []byte) error { total++; return nil }))
    require.Equal(t, n, total)
}
```

**実装メモ:**
- `AppendRaw` はシリアライズ済み行を受け取り `<line>\n` を書く
- 親ディレクトリは存在しなければ mode `0o700` で作成
- ロックは sidecar `<path>.lock`、`gofrs/flock` v0.12+ の `TryLockContext(ctx, 20*time.Millisecond)` を 500 ms デッドラインで使う
- ファイルは `os.O_CREATE|O_WRONLY|O_APPEND`、mode `0o600` でオープン
- `ReadJSONL` は 1 MiB バッファの `bufio.myproject` を使う。ファイルが存在しなければ `nil` を返す（コールバックは呼ばない）
- `AppendJSONL` は `v` を標準ライブラリで marshal して `AppendRaw` に委譲

**コミット:** `feat(storage): flock 直列化付きの JSONL 追記+読取を追加`

### Task 8: session meta

**ファイル:**
- 作成: `internal/storage/session.go` と `session_test.go`

**Exports:**

```go
func MetaFile(sessionDir string) string
func WriteSessionMeta(sessionDir string, m SessionMeta) error
func ReadSessionMeta(sessionDir string) (SessionMeta, error)
func UpdateSessionGoal(sessionDir, goal string) error
```

**振る舞い契約（テスト）:**

```go
package storage

import (
    "path/filepath"
    "testing"
    "time"

    "github.com/stretchr/testify/require"
)

func TestWriteAndReadSession(t *testing.T) {
    dir := t.TempDir()
    m := SessionMeta{ClaudeSessionID: "abc", Cwd: "/tmp/r", GitTracked: true, StartedAt: time.Unix(100, 0).UTC()}
    require.NoError(t, WriteSessionMeta(dir, m))
    got, err := ReadSessionMeta(dir)
    require.NoError(t, err)
    require.Equal(t, m, got)
}

func TestReadSessionMissingReturnsError(t *testing.T) {
    _, err := ReadSessionMeta(filepath.Join(t.TempDir(), "missing"))
    require.Error(t, err)
}

func TestUpdateSessionGoal(t *testing.T) {
    dir := t.TempDir()
    require.NoError(t, WriteSessionMeta(dir, SessionMeta{ClaudeSessionID: "abc"}))
    require.NoError(t, UpdateSessionGoal(dir, "2700"))
    got, err := ReadSessionMeta(dir)
    require.NoError(t, err)
    require.Equal(t, "2700", got.Goal)
}
```

**実装メモ:**
- `MetaFile` は `<sessionDir>/meta.json` を返す
- `WriteSessionMeta` は `sessionDir` を mode `0o700` で作成、`json.MarshalIndent`（2 スペース）で marshal、mode `0o600` で書込み
- `UpdateSessionGoal` = read → mutate → write

**コミット:** `feat(storage): session meta の read/write/update を追加`

### Task 9: turn 追記 + 合流読取

**ファイル:**
- 作成: `internal/storage/turn.go` と `turn_test.go`

**Exports:**

```go
func AppendTurnOpen(path string, t TurnOpen) error
func AppendTurnClose(path string, t TurnClose) error
func ReadTurns(path string) ([]Turn, error)
```

**振る舞い契約（テスト）:**

```go
package storage

import (
    "path/filepath"
    "testing"
    "time"

    "github.com/stretchr/testify/require"
)

func TestOpenCloseMerge(t *testing.T) {
    p := filepath.Join(t.TempDir(), "turns.jsonl")
    require.NoError(t, AppendTurnOpen(p, TurnOpen{ID: "t1", StartedAt: time.Unix(100, 0).UTC(), UserPromptPreview: "hi"}))
    require.NoError(t, AppendTurnClose(p, TurnClose{ID: "t1", EndedAt: time.Unix(200, 0).UTC(), HeadSHA: "def", Status: "done", DiffPath: "diffs/t1.patch"}))
    turns, err := ReadTurns(p)
    require.NoError(t, err)
    require.Len(t, turns, 1)
    require.Equal(t, "done", turns[0].Status)
    require.Equal(t, "def", turns[0].HeadSHA)
    require.Equal(t, "hi", turns[0].UserPromptPreview)
}

func TestOpenWithoutCloseKeepsStatusOpen(t *testing.T) {
    p := filepath.Join(t.TempDir(), "turns.jsonl")
    require.NoError(t, AppendTurnOpen(p, TurnOpen{ID: "t2", StartedAt: time.Unix(1, 0).UTC()}))
    turns, err := ReadTurns(p)
    require.NoError(t, err)
    require.Equal(t, "open", turns[0].Status)
}

func TestReadTurnsOrderedByStartedAt(t *testing.T) {
    p := filepath.Join(t.TempDir(), "turns.jsonl")
    require.NoError(t, AppendTurnOpen(p, TurnOpen{ID: "b", StartedAt: time.Unix(200, 0).UTC()}))
    require.NoError(t, AppendTurnOpen(p, TurnOpen{ID: "a", StartedAt: time.Unix(100, 0).UTC()}))
    turns, err := ReadTurns(p)
    require.NoError(t, err)
    require.Equal(t, []string{"a", "b"}, []string{turns[0].ID, turns[1].ID})
}
```

**実装メモ:**
- 両 appender は `AppendJSONL` 呼出し前に `Phase` を `"open"` / `"close"` に強制セット
- `ReadTurns`: 各 raw 行を stream し、まず `{"id","phase"}` を peek。ID で合流: open は `StartedAt`, `UserPromptPreview`, `HeadSHAPre`, `TranscriptMessageID` を埋める。close は `EndedAt`, `AssistantSummaryPreview`, `HeadSHA`, `DiffPath`, `Status`, `ErrorMessage` を埋める
- 初期化時に `Status = "open"` を入れておくことで open 単独を正しく報告
- 壊れた行 / id なし行は skip
- `StartedAt` 昇順で sort して返す

**コミット:** `feat(storage): turn の open/close 追記と合流読取を追加`

### Task 10: event 追記 + フィルタ読取

**ファイル:**
- 作成: `internal/storage/event.go` と `event_test.go`

**Exports:**

```go
func AppendEvent(path string, e Event) error
func ReadEvents(path, turnID string) ([]Event, error)
```

**振る舞い契約（テスト）:**

```go
package storage

import (
    "path/filepath"
    "testing"
    "time"

    "github.com/stretchr/testify/require"
)

func TestAppendAndReadEvents(t *testing.T) {
    p := filepath.Join(t.TempDir(), "events.jsonl")
    require.NoError(t, AppendEvent(p, Event{ID: "e1", TurnID: "t1", Ts: time.Unix(1, 0).UTC(), Tool: "Read", Phase: "pre"}))
    require.NoError(t, AppendEvent(p, Event{ID: "e2", TurnID: "t1", Ts: time.Unix(2, 0).UTC(), Tool: "Read", Phase: "post", Success: true}))
    got, err := ReadEvents(p, "t1")
    require.NoError(t, err)
    require.Len(t, got, 2)
    require.Equal(t, "e1", got[0].ID)
    require.Equal(t, "e2", got[1].ID)
}

func TestReadEventsFiltersByTurn(t *testing.T) {
    p := filepath.Join(t.TempDir(), "events.jsonl")
    require.NoError(t, AppendEvent(p, Event{ID: "e1", TurnID: "t1"}))
    require.NoError(t, AppendEvent(p, Event{ID: "e2", TurnID: "t2"}))
    got, err := ReadEvents(p, "t2")
    require.NoError(t, err)
    require.Len(t, got, 1)
    require.Equal(t, "e2", got[0].ID)
}
```

**実装メモ:**
- `ReadEvents` は `ReadJSONL` で stream、各行を decode、`turnID` でフィルタ（空文字列なら filter なし）、ファイル順を保つ

**コミット:** `feat(storage): event 追記とフィルタ読取を追加`

### Task 11: comment CRUD（append-only）

**ファイル:**
- 作成: `internal/storage/comment.go` と `comment_test.go`

**Exports:**

```go
func AppendComment(path string, c Comment) error
func MarkCommentsSent(path string, ids []string, ts time.Time) error
func DeleteComment(path, id string) error
func ReadComments(path string) ([]Comment, error)
func UnsentComments(path string) ([]Comment, error)
```

**振る舞い契約（テスト）:**

```go
package storage

import (
    "path/filepath"
    "testing"
    "time"

    "github.com/stretchr/testify/require"
)

func TestAppendAndReadComments(t *testing.T) {
    p := filepath.Join(t.TempDir(), "c.jsonl")
    require.NoError(t, AppendComment(p, Comment{ID: "c1", File: "a.go", LineStart: 10, LineEnd: 12, Text: "foo", CreatedAt: time.Unix(1, 0).UTC()}))
    require.NoError(t, AppendComment(p, Comment{ID: "c2", File: "b.go", LineStart: 20, LineEnd: 20, Text: "bar", CreatedAt: time.Unix(2, 0).UTC()}))
    got, err := ReadComments(p)
    require.NoError(t, err)
    require.Len(t, got, 2)
    require.Equal(t, "c1", got[0].ID)
    require.Equal(t, "c2", got[1].ID)
}

func TestMarkCommentSent(t *testing.T) {
    p := filepath.Join(t.TempDir(), "c.jsonl")
    require.NoError(t, AppendComment(p, Comment{ID: "c1", File: "a.go", LineStart: 1, LineEnd: 1, Text: "x", CreatedAt: time.Unix(1, 0).UTC()}))
    require.NoError(t, MarkCommentsSent(p, []string{"c1"}, time.Unix(2, 0).UTC()))
    got, err := ReadComments(p)
    require.NoError(t, err)
    require.Equal(t, time.Unix(2, 0).UTC(), got[0].SentAt)
}

func TestDeleteComment(t *testing.T) {
    p := filepath.Join(t.TempDir(), "c.jsonl")
    require.NoError(t, AppendComment(p, Comment{ID: "c1", File: "a", LineStart: 1, LineEnd: 1, Text: "x"}))
    require.NoError(t, AppendComment(p, Comment{ID: "c2", File: "a", LineStart: 2, LineEnd: 2, Text: "y"}))
    require.NoError(t, DeleteComment(p, "c1"))
    got, err := ReadComments(p)
    require.NoError(t, err)
    require.Len(t, got, 1)
    require.Equal(t, "c2", got[0].ID)
}

func TestUnsentComments(t *testing.T) {
    p := filepath.Join(t.TempDir(), "c.jsonl")
    require.NoError(t, AppendComment(p, Comment{ID: "c1", File: "a", LineStart: 1, LineEnd: 1, Text: "x", CreatedAt: time.Unix(1, 0).UTC()}))
    require.NoError(t, AppendComment(p, Comment{ID: "c2", File: "a", LineStart: 2, LineEnd: 2, Text: "y", CreatedAt: time.Unix(2, 0).UTC()}))
    require.NoError(t, MarkCommentsSent(p, []string{"c1"}, time.Unix(3, 0).UTC()))
    got, err := UnsentComments(p)
    require.NoError(t, err)
    require.Len(t, got, 1)
    require.Equal(t, "c2", got[0].ID)
}
```

**実装メモ:**
- append-only: 変更は新規レコードに `deleted` または `set_sent` のタグを付けて追加。非公開 `commentRecord` 型で `Comment` をラップし `Deleted bool` と `SetSent time.Time` を追加
- `AppendComment` は `CreatedAt` がゼロ値なら `time.Now().UTC()` を打刻
- `MarkCommentsSent` は id ごとに 1 レコード追記（`SetSent=ts`）
- `DeleteComment` は `Deleted=true` のレコードを追記
- `ReadComments` は id でレコードをまとめる: `deleted` セットを管理し、そのセット内 id のレコードは skip。非削除レコードは初出時にゼロ値の merged 構造体に詰め、以降のレコードは `SetSent` が非ゼロのときだけ `SentAt` を更新。`CreatedAt` 昇順で sort
- `UnsentComments` = `ReadComments` の `SentAt.IsZero()` フィルタ

**コミット:** `feat(storage): append-only な update/delete マーカー付きの comment CRUD を追加`

### Task 12: errors ロギング

**ファイル:**
- 作成: `internal/errlog/errlog.go` と `errlog_test.go`

**Exports:**

```go
func Record(component, event, sessionID, errMsg string) error
func Read() ([]storage.ErrEntry, error)
func Clear() error
```

**振る舞い契約（テスト）:**

```go
package errlog

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/require"
)

func TestRecordAndRead(t *testing.T) {
    t.Setenv("XDG_STATE_HOME", t.TempDir())
    require.NoError(t, Record("hook/turn-end", "git-diff-failed", "session-abc", "boom"))
    entries, err := Read()
    require.NoError(t, err)
    require.Len(t, entries, 1)
    require.Equal(t, "hook/turn-end", entries[0].Component)
    require.Equal(t, "git-diff-failed", entries[0].Event)
    require.Equal(t, "session-abc", entries[0].SessionID)
    require.Equal(t, "boom", entries[0].ErrorString)
}

func TestClearRemovesFile(t *testing.T) {
    dir := t.TempDir()
    t.Setenv("XDG_STATE_HOME", dir)
    require.NoError(t, Record("c", "e", "s", "err"))
    require.NoError(t, Clear())
    _, err := os.Stat(filepath.Join(dir, "tmux-agent-log", "errors.jsonl"))
    require.True(t, os.IsNotExist(err))
}
```

**実装メモ:**
- 内部で `config.ErrorsPath()` と `storage.AppendJSONL` を使う
- `Clear` は `errors.jsonl` と `errors.jsonl.lock`（あれば）を削除。ファイルが存在しなくてもエラーにしない

**コミット:** `feat(errlog): errors.jsonl の record/read/clear を追加`

---

## Phase 3: Git 連携

### Task 13: git ラッパー + リポジトリ検出

**ファイル:**
- 作成: `internal/git/git.go`、`internal/git/detect.go`、`detect_test.go`

**Exports:**

```go
// git.go
func Run(dir string, args ...string) (string, error)
type Error struct {
    Args     []string
    Stderr   string
    ExitCode int
}
func (e *Error) Error() string

// detect.go
func IsRepo(dir string) (bool, error)
func HeadSHA(dir string) (string, error)
```

**振る舞い契約（テスト）:**

```go
package git

import (
    "os/exec"
    "testing"

    "github.com/stretchr/testify/require"
)

func TestIsRepoReturnsFalseForNonRepo(t *testing.T) {
    ok, err := IsRepo(t.TempDir())
    require.NoError(t, err)
    require.False(t, ok)
}

func TestIsRepoReturnsTrueForRepo(t *testing.T) {
    dir := t.TempDir()
    require.NoError(t, exec.Command("git", "-C", dir, "init").Run())
    ok, err := IsRepo(dir)
    require.NoError(t, err)
    require.True(t, ok)
}

func TestHeadSHAOfFreshRepoIsEmpty(t *testing.T) {
    dir := t.TempDir()
    require.NoError(t, exec.Command("git", "-C", dir, "init").Run())
    sha, err := HeadSHA(dir)
    require.NoError(t, err)
    require.Equal(t, "", sha)
}

func TestHeadSHAAfterCommit(t *testing.T) {
    dir := t.TempDir()
    for _, args := range [][]string{
        {"init"},
        {"config", "user.email", "a@b"},
        {"config", "user.name", "a"},
        {"commit", "--allow-empty", "-m", "x"},
    } {
        require.NoError(t, exec.Command("git", append([]string{"-C", dir}, args...)...).Run())
    }
    sha, err := HeadSHA(dir)
    require.NoError(t, err)
    require.Len(t, sha, 40)
}
```

**実装メモ:**
- `Run` は `git -C <dir> <args...>` を shell out し、stdout/stderr を捕捉。stdout は末尾改行を trim して返す。non-zero exit なら `&Error{Args, Stderr, ExitCode}` を返す
- `IsRepo` = `git rev-parse --is-inside-work-tree` の trim 結果が `"true"` か。`*Error` 結果は `(false, nil)` を返す（リポジトリでないことは hard error ではない）
- `HeadSHA` = `git rev-parse HEAD`。stderr に `"unknown revision"` が含まれる場合（初回コミット前）は `""` と `nil` を返す

**コミット:** `feat(git): git ラッパーと IsRepo/HeadSHA を追加`

### Task 14: diff 生成

**ファイル:**
- 作成: `internal/git/diff.go` と `diff_test.go`

**Exports:**

```go
// emptyTreeHash は git の正規 empty-tree SHA。
const emptyTreeHash = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

func DiffSince(dir, base string) (string, error)
```

**振る舞い契約（テスト）:**

```go
package git

import (
    "os"
    "os/exec"
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/require"
)

func setupRepo(t *testing.T) string {
    t.Helper()
    dir := t.TempDir()
    for _, c := range [][]string{
        {"init"},
        {"config", "user.email", "a@b"},
        {"config", "user.name", "a"},
    } {
        require.NoError(t, exec.Command("git", append([]string{"-C", dir}, c...)...).Run())
    }
    return dir
}

func TestDiffIncludesCommittedChanges(t *testing.T) {
    dir := setupRepo(t)
    require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello\n"), 0o644))
    require.NoError(t, exec.Command("git", "-C", dir, "add", "a.txt").Run())
    require.NoError(t, exec.Command("git", "-C", dir, "commit", "-m", "c1").Run())
    base, _ := HeadSHA(dir)

    require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("world\n"), 0o644))
    require.NoError(t, exec.Command("git", "-C", dir, "add", "a.txt").Run())
    require.NoError(t, exec.Command("git", "-C", dir, "commit", "-m", "c2").Run())

    diff, err := DiffSince(dir, base)
    require.NoError(t, err)
    require.Contains(t, diff, "-hello")
    require.Contains(t, diff, "+world")
}

func TestDiffIncludesUnstagedChanges(t *testing.T) {
    dir := setupRepo(t)
    require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hi\n"), 0o644))
    require.NoError(t, exec.Command("git", "-C", dir, "add", "a.txt").Run())
    require.NoError(t, exec.Command("git", "-C", dir, "commit", "-m", "c1").Run())
    base, _ := HeadSHA(dir)

    require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("changed\n"), 0o644))
    diff, err := DiffSince(dir, base)
    require.NoError(t, err)
    require.Contains(t, diff, "-hi")
    require.Contains(t, diff, "+changed")
}

func TestDiffSinceEmptyBaseUsesEmptyTree(t *testing.T) {
    dir := setupRepo(t)
    require.NoError(t, os.WriteFile(filepath.Join(dir, "new.txt"), []byte("hello\n"), 0o644))
    diff, err := DiffSince(dir, "")
    require.NoError(t, err)
    require.Contains(t, diff, "+hello")
}
```

**実装メモ:**
- `DiffSince`: `base == ""` のとき `emptyTreeHash` を代入
- コマンド: `git diff --no-color -U3 <base> --`。これで committed + unstaged の変更を 1 回で取得

**コミット:** `feat(git): committed+unstaged diff を返す DiffSince(base) を追加`

---

## Phase 4: tmux 連携

### Task 15: tmux 環境変数 + pane 存在確認

**ファイル:**
- 作成: `internal/tmux/tmux.go` と `tmux_test.go`

**Exports:**

```go
func IsInsideTmux() bool
func CurrentPane() string
func PaneExists(paneID string) (bool, error)

// 非公開（テストと sendkeys.go から使う）:
func paneExistsWithSocket(socket, paneID string) (bool, error)
```

**振る舞い契約（テスト）:**

```go
package tmux

import (
    "os/exec"
    "strings"
    "testing"

    "github.com/stretchr/testify/require"
)

func TestIsInsideTmuxDetectsEnv(t *testing.T) {
    t.Setenv("TMUX", "/tmp/tmux-1000/default,123,0")
    require.True(t, IsInsideTmux())
}

func TestIsInsideTmuxFalseWhenUnset(t *testing.T) {
    t.Setenv("TMUX", "")
    require.False(t, IsInsideTmux())
}

func TestPaneExists(t *testing.T) {
    if _, err := exec.LookPath("tmux"); err != nil {
        t.Skip("tmux not installed")
    }
    sock := t.TempDir() + "/tmux.sock"
    _ = exec.Command("tmux", "-S", sock, "kill-server").Run()
    require.NoError(t, exec.Command("tmux", "-S", sock, "new-session", "-d", "-s", "t", "sleep", "5").Run())
    defer exec.Command("tmux", "-S", sock, "kill-server").Run()

    out, err := exec.Command("tmux", "-S", sock, "list-panes", "-t", "t", "-F", "#{pane_id}").Output()
    require.NoError(t, err)
    paneID := strings.TrimSpace(string(out))

    ok, err := paneExistsWithSocket(sock, paneID)
    require.NoError(t, err)
    require.True(t, ok)

    ok, err = paneExistsWithSocket(sock, "%9999")
    require.NoError(t, err)
    require.False(t, ok)
}
```

**実装メモ:**
- `IsInsideTmux`: `os.Getenv("TMUX") != ""`
- `CurrentPane`: `os.Getenv("TMUX_PANE")` を返す
- `paneExistsWithSocket`: `tmux [-S socket] list-panes -a -F '#{pane_id}'` を実行。改行で split して比較。`tmux` のエラー（サーバーなし等）は `(false, nil)` で返す
- `PaneExists` は `paneExistsWithSocket("", paneID)` に委譲

**コミット:** `feat(tmux): tmux 環境変数の検出と pane 存在確認を追加`

### Task 16: send-keys と OSC 52 fallback

**ファイル:**
- 作成: `internal/tmux/sendkeys.go` と `sendkeys_test.go`

**Exports:**

```go
type SendResultKind int

const (
    SendResultOK SendResultKind = iota
    SendResultFallbackClipboard
    SendResultFailed
)

type SendResult struct {
    Kind SendResultKind
    Err  error
}

func SendToPane(paneID, text string) SendResult
func SendToPaneWithSocket(socket, paneID, text string) SendResult

// 非公開（テストから使う）:
func sendToPaneWithWriters(socket, paneID, text string, clipboard io.Writer, stderr io.Writer) SendResult
```

**振る舞い契約（テスト）:**

```go
package tmux

import (
    "bytes"
    "os/exec"
    "strings"
    "testing"
    "time"

    "github.com/stretchr/testify/require"
)

func TestSendKeysToValidPane(t *testing.T) {
    if _, err := exec.LookPath("tmux"); err != nil {
        t.Skip("tmux not installed")
    }
    sock := t.TempDir() + "/tmux.sock"
    _ = exec.Command("tmux", "-S", sock, "kill-server").Run()
    require.NoError(t, exec.Command("tmux", "-S", sock, "new-session", "-d", "-s", "t", "cat").Run())
    defer exec.Command("tmux", "-S", sock, "kill-server").Run()

    out, _ := exec.Command("tmux", "-S", sock, "list-panes", "-t", "t", "-F", "#{pane_id}").Output()
    paneID := strings.TrimSpace(string(out))

    res := SendToPaneWithSocket(sock, paneID, "hello world")
    require.Equal(t, SendResultOK, res.Kind)

    time.Sleep(100 * time.Millisecond)
    captured, _ := exec.Command("tmux", "-S", sock, "capture-pane", "-t", paneID, "-p").Output()
    require.Contains(t, string(captured), "hello world")
}

func TestSendKeysFallsBackOnMissingPane(t *testing.T) {
    clip := &bytes.Buffer{}
    res := sendToPaneWithWriters("", "%9999", "hello", clip, nil)
    require.Equal(t, SendResultFallbackClipboard, res.Kind)
    require.True(t, strings.HasPrefix(clip.String(), "\x1b]52;c;"))
}
```

**実装メモ:**
- pane が存在しなければ `clipboard` に OSC 52 シーケンスを書く: `"\x1b]52;c;" + base64(text) + "\x07"` → `SendResultFallbackClipboard` を返す
- 存在すれば: `tmux [-S sock] send-keys -t <pane> -l <text>` の後に `tmux [-S sock] send-keys -t <pane> Enter`。エラーがあれば `SendResultFailed` に `Err` をセット
- `SendToPane` と `SendToPaneWithSocket` は `sendToPaneWithWriters(socket, pane, text, os.Stdout, os.Stderr)` に委譲

**コミット:** `feat(tmux): OSC 52 クリップボード fallback 付きの send-to-pane を追加`

---

## Phase 5: hook ハンドラ

### Task 17: 共通の hook I/O

**ファイル:**
- 作成: `internal/hook/hook.go` と `hook_test.go`

**Exports:**

```go
func ReadInput(r io.Reader, v interface{}) error
func RunWithRecover(fn func() error) int
```

**振る舞い契約（テスト）:**

```go
package hook

import (
    "bytes"
    "errors"
    "testing"

    "github.com/stretchr/testify/require"
)

type sampleIn struct {
    SessionID string `json:"session_id"`
    Extra     string `json:"extra"`
}

func TestReadInputParsesJSON(t *testing.T) {
    var s sampleIn
    require.NoError(t, ReadInput(bytes.NewBufferString(`{"session_id":"abc","extra":"x"}`), &s))
    require.Equal(t, "abc", s.SessionID)
    require.Equal(t, "x", s.Extra)
}

func TestReadInputIgnoresUnknownFields(t *testing.T) {
    var s sampleIn
    require.NoError(t, ReadInput(bytes.NewBufferString(`{"session_id":"abc","new_field":{"nested":true}}`), &s))
    require.Equal(t, "abc", s.SessionID)
}

func TestRunWithRecoverReturnsZeroOnPanic(t *testing.T) {
    require.Equal(t, 0, RunWithRecover(func() error { panic("boom") }))
}

func TestRunWithRecoverReturnsZeroOnError(t *testing.T) {
    require.Equal(t, 0, RunWithRecover(func() error { return errors.New("bad") }))
}
```

**実装メモ:**
- `ReadInput` は `json.NewDecoder(r).Decode(v)` を使う（unknown fields はデフォルトで許容）
- `RunWithRecover`: `defer recover()` で panic を `errlog.Record("hook","panic","",fmt.Sprintf("%v",r))` に記録、stderr に出力。`fn` を呼び、non-nil error なら `errlog` に記録 + stderr 出力。常に 0 を返す

**コミット:** `feat(hook): 共通の stdin parse と panic 復帰ランナーを追加`

### Task 18: session-start hook

**ファイル:**
- 作成: `internal/hook/session_start.go` と `session_start_test.go`

**Exports:**

```go
func RunSessionStart(stdin io.Reader) error
```

**テストヘルパー（後続の hook テストでも使う）**: `session_start_test.go` に追加:

```go
package hook

import (
    "os/exec"
    "testing"

    "github.com/stretchr/testify/require"
)

func setupGitRepo(t *testing.T) string {
    t.Helper()
    dir := t.TempDir()
    for _, c := range [][]string{
        {"init"},
        {"config", "user.email", "a@b"},
        {"config", "user.name", "a"},
        {"commit", "--allow-empty", "-m", "base"},
    } {
        require.NoError(t, exec.Command("git", append([]string{"-C", dir}, c...)...).Run())
    }
    return dir
}
```

**振る舞い契約（テスト）:**

```go
func TestSessionStartCreatesMeta(t *testing.T) {
    t.Setenv("XDG_STATE_HOME", t.TempDir())
    cwd := setupGitRepo(t)
    require.NoError(t, RunSessionStart(bytes.NewBufferString(
        `{"session_id":"abc","cwd":"`+cwd+`","transcript_path":"/tmp/t.jsonl"}`)))

    sDir, _ := config.SessionDir(cwd, "abc")
    meta, err := storage.ReadSessionMeta(sDir)
    require.NoError(t, err)
    require.Equal(t, "abc", meta.ClaudeSessionID)
    require.Equal(t, cwd, meta.Cwd)
    require.True(t, meta.GitTracked)
    require.Len(t, meta.BaseSHA, 40)
    require.NotZero(t, meta.StartedAt)
}

func TestSessionStartNonGitSetsFlag(t *testing.T) {
    t.Setenv("XDG_STATE_HOME", t.TempDir())
    cwd := t.TempDir()
    require.NoError(t, RunSessionStart(bytes.NewBufferString(`{"session_id":"def","cwd":"`+cwd+`"}`)))

    sDir, _ := config.SessionDir(cwd, "def")
    meta, _ := storage.ReadSessionMeta(sDir)
    require.False(t, meta.GitTracked)
    require.Equal(t, "", meta.BaseSHA)
}
```

（`bytes`、`config`、`storage` の import を追加すること）

**実装メモ:**
- ローカル struct `{SessionID, Cwd, TranscriptPath}` に decode
- `SessionID` か `Cwd` が空なら no-op
- session ディレクトリは `config.SessionDir` で解決
- `git.IsRepo(cwd)` を呼び、true なら `git.HeadSHA(cwd)` を `BaseSHA` に
- `storage.SessionMeta` を書込み: `StartedAt = time.Now().UTC()`、`TmuxPane = os.Getenv("TMUX_PANE")`

**コミット:** `feat(hook): session-start を追加（base SHA 付き meta.json を書込み）`

### Task 19: turn-start hook

**ファイル:**
- 作成: `internal/hook/turn_start.go` と `turn_start_test.go`

**Exports:**

```go
func RunTurnStart(stdin io.Reader) error
// 非公開ヘルパー、パッケージ内で共有:
func previewFirstLines(s string, max int) string
```

**振る舞い契約（テスト）:**

```go
func TestTurnStartAppendsOpenRecord(t *testing.T) {
    t.Setenv("XDG_STATE_HOME", t.TempDir())
    cwd := setupGitRepo(t)

    require.NoError(t, RunSessionStart(bytes.NewBufferString(`{"session_id":"abc","cwd":"`+cwd+`"}`)))
    require.NoError(t, RunTurnStart(bytes.NewBufferString(`{"session_id":"abc","cwd":"`+cwd+`","prompt":"do the thing"}`)))

    sDir, _ := config.SessionDir(cwd, "abc")
    turns, err := storage.ReadTurns(filepath.Join(sDir, "turns.jsonl"))
    require.NoError(t, err)
    require.Len(t, turns, 1)
    require.Equal(t, "open", turns[0].Status)
    require.Equal(t, "do the thing", turns[0].UserPromptPreview)
    require.NotEmpty(t, turns[0].HeadSHAPre)
}

func TestPreviewFirstLinesTruncatesAndLimitsLines(t *testing.T) {
    require.Equal(t, "line1\nline2", previewFirstLines("line1\nline2\nline3", 100))
    require.Equal(t, "abc…", previewFirstLines("abcdef", 3))
}
```

**実装メモ:**
- `{SessionID, Cwd, Prompt}` に decode
- session meta を読み、`GitTracked` なら `HeadSHAPre` を取得
- ID = `"turn-" + uuid.NewString()`
- `UserPromptPreview = previewFirstLines(prompt, 400)`
- `previewFirstLines`: 先頭 2 行までを保持（`strings.SplitN(s,"\n",3)` で split して再結合）、その後 `max` 文字に切り詰め、超過時は `"…"` を付ける
- `storage.AppendTurnOpen` で追記

**コミット:** `feat(hook): turn-start を追加（user prompt preview 付きの open turn）`

### Task 20: tool-pre と tool-post hook

**ファイル:**
- 作成: `internal/hook/tool.go` と `tool_test.go`

**Exports:**

```go
func RunToolPre(stdin io.Reader) error
func RunToolPost(stdin io.Reader) error
```

**振る舞い契約（テスト）:**

```go
func TestToolPreAndPostAppendEvents(t *testing.T) {
    t.Setenv("XDG_STATE_HOME", t.TempDir())
    cwd := setupGitRepo(t)
    require.NoError(t, RunSessionStart(bytes.NewBufferString(`{"session_id":"s1","cwd":"`+cwd+`"}`)))
    require.NoError(t, RunTurnStart(bytes.NewBufferString(`{"session_id":"s1","cwd":"`+cwd+`","prompt":"p"}`)))

    sDir, _ := config.SessionDir(cwd, "s1")
    turns, _ := storage.ReadTurns(filepath.Join(sDir, "turns.jsonl"))
    turnID := turns[0].ID

    require.NoError(t, RunToolPre(bytes.NewBufferString(
        `{"session_id":"s1","cwd":"`+cwd+`","turn_id":"`+turnID+`","tool_name":"Read","tool_input":{"file_path":"/a"}}`)))
    require.NoError(t, RunToolPost(bytes.NewBufferString(
        `{"session_id":"s1","cwd":"`+cwd+`","turn_id":"`+turnID+`","tool_name":"Read","tool_response":{"success":true}}`)))

    events, _ := storage.ReadEvents(filepath.Join(sDir, "events.jsonl"), turnID)
    require.Len(t, events, 2)
    require.Equal(t, "pre", events[0].Phase)
    require.Equal(t, "post", events[1].Phase)
    require.True(t, events[1].Success)
}
```

**実装メモ:**
- 共有 decoder 構造体（非公開）: `SessionID, Cwd, TurnID, ToolName, ToolInput json.RawMessage, ToolResponse{Success bool, Error string}`
- `RunToolPre` / `RunToolPost` は private な `runToolHook(stdin, phase)` に転送
- `TurnID` が空なら、`ReadTurns` を逆順に走査して最初に出てくる `Status == "open"` の turn を選ぶ
- event id = `"evt-" + uuid.NewString()`
- `ArgsPreview`: `ToolInput` を文字列化して 200 文字で切り詰め、超過時は `"…"` を付ける

**コミット:** `feat(hook): tool-pre と tool-post の event 追記を追加`

### Task 21: turn-end hook

**ファイル:**
- 作成: `internal/hook/turn_end.go` と `turn_end_test.go`

**Exports:**

```go
func RunTurnEnd(stdin io.Reader) error
```

**振る舞い契約（テスト）:**

```go
func TestTurnEndClosesTurnAndWritesDiff(t *testing.T) {
    t.Setenv("XDG_STATE_HOME", t.TempDir())
    cwd := setupGitRepo(t)
    require.NoError(t, RunSessionStart(bytes.NewBufferString(`{"session_id":"s1","cwd":"`+cwd+`"}`)))
    require.NoError(t, RunTurnStart(bytes.NewBufferString(`{"session_id":"s1","cwd":"`+cwd+`","prompt":"p"}`)))

    require.NoError(t, os.WriteFile(filepath.Join(cwd, "hello.txt"), []byte("hi\n"), 0o644))
    require.NoError(t, exec.Command("git", "-C", cwd, "add", "hello.txt").Run())
    require.NoError(t, exec.Command("git", "-C", cwd, "commit", "-m", "t1").Run())

    require.NoError(t, RunTurnEnd(bytes.NewBufferString(`{"session_id":"s1","cwd":"`+cwd+`"}`)))

    sDir, _ := config.SessionDir(cwd, "s1")
    turns, _ := storage.ReadTurns(filepath.Join(sDir, "turns.jsonl"))
    require.Equal(t, "done", turns[0].Status)
    require.NotEmpty(t, turns[0].HeadSHA)
    require.NotEmpty(t, turns[0].DiffPath)

    raw, _ := os.ReadFile(filepath.Join(sDir, turns[0].DiffPath))
    require.Contains(t, string(raw), "+hi")
}

func TestTurnEndEmptyDiffClosesWithNullPath(t *testing.T) {
    t.Setenv("XDG_STATE_HOME", t.TempDir())
    cwd := setupGitRepo(t)
    require.NoError(t, RunSessionStart(bytes.NewBufferString(`{"session_id":"s2","cwd":"`+cwd+`"}`)))
    require.NoError(t, RunTurnStart(bytes.NewBufferString(`{"session_id":"s2","cwd":"`+cwd+`","prompt":"p"}`)))
    require.NoError(t, RunTurnEnd(bytes.NewBufferString(`{"session_id":"s2","cwd":"`+cwd+`"}`)))

    sDir, _ := config.SessionDir(cwd, "s2")
    turns, _ := storage.ReadTurns(filepath.Join(sDir, "turns.jsonl"))
    require.Equal(t, "done", turns[0].Status)
    require.Equal(t, "", turns[0].DiffPath)
}
```

**実装メモ:**
- `{SessionID, Cwd}` に decode
- meta と turns を読込み、最も新しい `Status == "open"` の turn を見つける。なければ no-op
- `meta.GitTracked` なら `git.DiffSince(cwd, openTurn.HeadSHAPre)` を呼ぶ。`strings.TrimSpace` 後に非空なら `<sessionDir>/diffs/<turn_id>.patch`（mode `0o600`、ディレクトリは `0o700`）に書出し、相対パスを記録
- `git.HeadSHA(cwd)` で `HeadSHA` を取得
- `Status = "done"` の `TurnClose` を追記

**コミット:** `feat(hook): turn-end を追加（per-turn diff patch 付きで turn を close）`

---

## Phase 6: CLI 配線

### Task 22: hook サブコマンド

**ファイル:**
- 作成: `internal/cli/hook.go` と `hook_test.go`

**Exports:**
- `rootCmd` に `hook` サブコマンドツリーを追加: 子は `session-start`, `turn-start`, `tool-pre`, `tool-post`, `turn-end`

**配線パターン（このとおり再現）:**

```go
package cli

import (
    "io"

    "github.com/spf13/cobra"

    "github.com/yusei-wy/tmux-agent-log/internal/hook"
)

func init() {
    hookCmd := &cobra.Command{
        Use:   "hook",
        Short: "Claude Code hook エンドポイント（agent から呼ばれる、人間は直接使わない）",
    }
    hookCmd.AddCommand(mkHook("session-start", hook.RunSessionStart))
    hookCmd.AddCommand(mkHook("turn-start", hook.RunTurnStart))
    hookCmd.AddCommand(mkHook("tool-pre", hook.RunToolPre))
    hookCmd.AddCommand(mkHook("tool-post", hook.RunToolPost))
    hookCmd.AddCommand(mkHook("turn-end", hook.RunTurnEnd))
    rootCmd.AddCommand(hookCmd)
}

// mkHook は stdin を runner にパイプする cobra サブコマンドを作る。
// hook.RunWithRecover が error / panic を errors.jsonl に飲み込み、
// 必ず exit code 0 を返す。
func mkHook(name string, runner func(io.Reader) error) *cobra.Command {
    return &cobra.Command{
        Use: name,
        RunE: func(cmd *cobra.Command, args []string) error {
            hook.RunWithRecover(func() error { return runner(cmd.InOrStdin()) })
            return nil
        },
    }
}
```

**実装メモ:**
- cobra は `RunE` が nil を返せば自動的に exit 0。`RunWithRecover` は外側からは常に nil を生むため hook サブコマンドは必ず成功する — これが spec の「hooks は Claude Code を止めない」ルールの実装形
- `RunE` 内で `os.Exit` を呼ばない。cobra のクリーンアップを bypass し、`rootCmd.Execute()` を直接呼ぶテストを混乱させるため

**振る舞い契約（テスト）:**

```go
package cli

import (
    "bytes"
    "os/exec"
    "testing"

    "github.com/stretchr/testify/require"
)

func TestHookSubcommandSessionStart(t *testing.T) {
    t.Setenv("XDG_STATE_HOME", t.TempDir())
    cwd := t.TempDir()
    require.NoError(t, exec.Command("git", "-C", cwd, "init").Run())

    rootCmd.SetArgs([]string{"hook", "session-start"})
    rootCmd.SetIn(bytes.NewBufferString(`{"session_id":"abc","cwd":"` + cwd + `"}`))
    rootCmd.SetOut(new(bytes.Buffer))
    rootCmd.SetErr(new(bytes.Buffer))
    require.NoError(t, rootCmd.Execute())
}
```

手動確認も:

```bash
go build -o /tmp/tal ./cmd/tmux-agent-log
echo '{"session_id":"x","cwd":"/tmp"}' | /tmp/tal hook session-start
echo $?   # 0
```

**コミット:** `feat(cli): hook サブコマンド（session-start/turn-{start,end}/tool-{pre,post}）を配線`

### Task 23: 出力フォーマットパッケージ

**ファイル:**
- 作成: `internal/format/format.go` と `format_test.go`

**Exports:**

```go
func Write(w io.Writer, fmtName string, columns []string, rows [][]string) error
```

サポートする format: `"tsv" | "jsonl" | "json" | "table"`。未知ならエラー。

**振る舞い契約（テスト）:**

```go
package format

import (
    "bytes"
    "testing"

    "github.com/stretchr/testify/require"
)

func TestTSV(t *testing.T) {
    buf := &bytes.Buffer{}
    require.NoError(t, Write(buf, "tsv", []string{"id", "name"}, [][]string{{"1", "alice"}, {"2", "bob"}}))
    require.Equal(t, "1\talice\n2\tbob\n", buf.String())
}

func TestTable(t *testing.T) {
    buf := &bytes.Buffer{}
    require.NoError(t, Write(buf, "table", []string{"id", "name"}, [][]string{{"1", "alice"}, {"22", "bob"}}))
    require.Contains(t, buf.String(), "id")
    require.Contains(t, buf.String(), "alice")
}

func TestJSONL(t *testing.T) {
    buf := &bytes.Buffer{}
    require.NoError(t, Write(buf, "jsonl", []string{"id", "name"}, [][]string{{"1", "alice"}, {"2", "bob"}}))
    require.Equal(t, "{\"id\":\"1\",\"name\":\"alice\"}\n{\"id\":\"2\",\"name\":\"bob\"}\n", buf.String())
}

func TestUnknownFormatErrors(t *testing.T) {
    err := Write(&bytes.Buffer{}, "xml", []string{"a"}, [][]string{{"b"}})
    require.Error(t, err)
}
```

**実装メモ:**
- `tsv`: 各行を `\t` で join、改行終端
- `table`: `text/tabwriter` を 2 スペース padding で。columns header → rows
- `jsonl` / `json`: 順序付き JSON object を手動で構築する（`map[string]string` 経由ではキー順が保てない）。実装: 各行ごとに `{` から始め、`json.Marshal(col)` / `:` / `json.Marshal(val)` をカンマ区切りで書く、最後に `}`。`jsonl` は 1 行 1 row、`json` は全行を `[...]` でラップして末尾改行

**コミット:** `feat(format): tsv/jsonl/json/table 出力 writer を追加`

### Task 24: CLI list-* コマンド

**ファイル:**
- 作成: `internal/cli/list.go`

**Exports（cobra 登録）:**
- `list-sessions` — 全プロジェクトの全セッション。columns: `session_id, project, goal, cwd, started_at`
- `list-turns` — セッションの turn 一覧。必須 flag `--session`。columns: `id, started_at, ended_at, status, diff_path, prompt_preview`
- `list-comments` — セッションのコメント。必須 flag `--session`、optional `--unsent`。columns: `id, file, line_start, line_end, text, sent_at`

3 つすべて `--format tsv|jsonl|json|table`（デフォルト `table`）対応。

**実装メモ:**
- パッケージレベルヘルパー `findSessionDir(sessionID string) (string, error)` を導入。`StateDir/projects/*/sessions/<id>` を走査し最初の一致を返す。なければ `os.ErrNotExist`
- 日付フォーマット: UTC で `"2006-01-02 15:04:05"`、ゼロ値なら空文字列
- 描画は `format.Write` に委譲

**テスト:** end-to-end smoke test（Task 30）に先送り。ビルド後の手動確認:

```bash
XDG_STATE_HOME=/tmp/tst /tmp/tal list-sessions --format tsv
```

**コミット:** `feat(cli): list-sessions/list-turns/list-comments を追加`

### Task 25: CLI show-* コマンド

**ファイル:**
- 作成: `internal/cli/show.go`

**Exports（cobra 登録）:**
- `show-session <session-id>` — `meta.json` を整形 JSON で出力
- `show-turn <turn-id>` — 必須 `--session`、optional `--with-diff`。turn を整形 JSON で出力。`--with-diff` で `DiffPath != ""` なら `--- diff ---` セパレータの後に diff patch を続ける
- `show-diff <session-id>` — 必須 flag `--base session|turn|main`（デフォルト `session`）、`--base=turn` のときは `--turn <id>` も必須。diff を stdout に書出し
  - `base=session`: `git.DiffSince(meta.Cwd, meta.BaseSHA)`
  - `base=turn`: per-turn `.patch` ファイルをそのまま読んで出力
  - `base=main`: `git.Run(meta.Cwd, "diff", "--no-color", "-U3", "main", "--")`

**実装メモ:**
- `findSessionDir` を再利用
- エラー: 未知の base → 既知集合を含むエラー、turn 不足 → エラー、diff 不在 → 空出力 + exit 0

**コミット:** `feat(cli): show-session/show-turn/show-diff を追加`

### Task 26: CLI goal / clear / export

**ファイル:**
- 作成: `internal/cli/goal.go`, `internal/cli/clear.go`, `internal/cli/export.go`

**Exports（cobra 登録）:**

**`goal [title]`** — 必須 flag `--session`。引数なしなら現在の goal を出力（空なら `(no goal)`）。引数 1 つなら `storage.UpdateSessionGoal`。

**`clear`** — 排他フラグ `--session <id>` / `--all` / `--older-than <duration>`:
- `--session`: `os.RemoveAll(findSessionDir(id))`
- `--all`: `os.RemoveAll(StateDir/projects)`
- `--older-than`: `time.ParseDuration` に加えて `7d`（日数）を解釈する小ヘルパーで parse、`projects/*/sessions/*` を走査し、mtime が cutoff より古いディレクトリを `RemoveAll`
- フラグなし → エラー

**`export --session <id> --format md`** — Markdown を stdout に書出し:
- H1: goal、空なら `"Session Export"`
- 箇条書きメタデータ: `session`, `cwd`, `base`
- H2 `## Turns` → 各 turn を H3 で（`started_at`、prompt preview、patch があれば ```diff ... ``` フェンス付きブロック）
- `--format` は MVP では `md` のみ受け付ける、それ以外はエラー

**実装メモ:**
- `parseDuration` のユニットテストを追加（`"7d" == 7*24h`、`"24h"`、`"abc"` → エラー）
- `clear --all` は **stdin が terminal のときだけ** 確認プロンプトを出す。スクリプト/CI ではスキップ。MVP では環境変数 `TMUX_AGENT_LOG_ASSUME_YES=1` で自動承認（テストはこれをセット）

**コミット:** `feat(cli): goal/clear/export サブコマンドを追加`

### Task 27: CLI comment コマンド

**ファイル:**
- 作成: `internal/cli/comment.go` と `comment_test.go`

**Exports（`comment` 配下のサブツリー）:**
- `comment add --session --file --line <n>|<s-e> --text "..."`
- `comment list --session [--unsent]`
- `comment delete --session <comment-id>`
- `comment send --session [--preview]`

**非公開ヘルパー:**

```go
func parseLineRange(s string) (start, end int, err error)
func renderSendPrompt(cs []storage.Comment) string
```

**振る舞い契約（純粋ヘルパーのテスト）:**

```go
package cli

import (
    "testing"
    "time"

    "github.com/stretchr/testify/require"

    "github.com/yusei-wy/tmux-agent-log/internal/storage"
)

func TestParseLineRange(t *testing.T) {
    cases := []struct {
        in      string
        start   int
        end     int
        wantErr bool
    }{
        {"44", 44, 44, false},
        {"44-46", 44, 46, false},
        {"46-44", 0, 0, true},
        {"abc", 0, 0, true},
    }
    for _, c := range cases {
        s, e, err := parseLineRange(c.in)
        if c.wantErr {
            require.Error(t, err, c.in)
            continue
        }
        require.NoError(t, err, c.in)
        require.Equal(t, c.start, s)
        require.Equal(t, c.end, e)
    }
}

func TestRenderSendPromptContainsAllEntries(t *testing.T) {
    cs := []storage.Comment{
        {File: "a.go", LineStart: 10, LineEnd: 12, Text: "foo", CreatedAt: time.Unix(1, 0).UTC()},
        {File: "b.go", LineStart: 30, LineEnd: 30, Text: "bar", CreatedAt: time.Unix(2, 0).UTC()},
    }
    out := renderSendPrompt(cs)
    require.Contains(t, out, "a.go:10-12")
    require.Contains(t, out, "foo")
    require.Contains(t, out, "b.go:30-30")
    require.Contains(t, out, "bar")
}
```

**実装メモ:**
- `parseLineRange`: `-` で split、整数 parse、`end < start` を reject
- `renderSendPrompt`: 開始行 `"以下のレビューコメントを反映してください:\n\n"`、コメントごとのブロック `"- <file>:<start>-<end>\n  <text>\n\n"`、終了行 `"(反映後、関連テストを実行して結果を報告してください)"`
- `comment add`: フラグ検証、`id = "cmt-" + uuid.NewString()`、`CreatedAt = time.Now().UTC()`
- `comment send`:
  - 未送信コメントを読む。なければ `"(no unsent comments)"` を出力して exit 0
  - `renderSendPrompt` でプロンプト構築
  - `--preview` → プロンプトのみ出力
  - それ以外は `tmux.SendToPane(meta.TmuxPane, prompt)` を呼ぶ。`SendResultOK` ならコメントを sent としてマーク。`SendResultFallbackClipboard` なら sent マークしつつ stderr に通知。`SendResultFailed` ならエラーを wrap して返す（sent マークしない）
- `comment list`: `ReadComments` / `UnsentComments`、コメントごとに 1 行: `<id>  <file>:<start>-<end> [sent]\n  <text>`

**コミット:** `feat(cli): comment add/list/delete/send サブコマンドを追加`

### Task 28: CLI config / errors サブコマンド

**ファイル:**
- 作成: `internal/cli/config_cmd.go`, `internal/cli/errors_cmd.go`

**Exports（cobra 登録）:**

- `config show` — `config.Load()` の結果を整形 JSON で出力
- `config path` — `config.toml` の絶対パスを出力
- `config edit` — `$EDITOR`（fallback `vi`）でファイルを開く。必要に応じてディレクトリ作成
- `errors list` — `errlog.Read()`、各エントリを 1 行 JSON で出力
- `errors clear` — `errlog.Clear()`

**実装メモ:** 既存パッケージ関数の薄いラッパー。新ロジックなし。

**コミット:** `feat(cli): config show/path/edit と errors list/clear を追加`

---

## Phase 7: install-hooks

### Task 29: install-hooks / uninstall-hooks

**ファイル:**
- 作成: `internal/cli/install.go` と `install_test.go`

**Exports:**
- `install-hooks [--dry]` — 自前の hook エントリを `~/.claude/settings.json` にマージ
- `uninstall-hooks` — 自前のエントリのみ削除

**非公開ヘルパー:**

```go
func installHooksTo(path, bin string) error
func uninstallHooksFrom(path, bin string) error

var hookEvents = []struct{
    Event string
    Sub   string
}{
    {"SessionStart", "session-start"},
    {"UserPromptSubmit", "turn-start"},
    {"PreToolUse", "tool-pre"},
    {"PostToolUse", "tool-post"},
    {"Stop", "turn-end"},
}
```

**振る舞い契約（テスト）:**

```go
package cli

import (
    "encoding/json"
    "os"
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/require"
)

func TestInstallHooksCreatesSettingsFile(t *testing.T) {
    home := t.TempDir()
    t.Setenv("HOME", home)
    target := filepath.Join(home, ".claude", "settings.json")
    require.NoError(t, installHooksTo(target, "tmux-agent-log"))

    raw, err := os.ReadFile(target)
    require.NoError(t, err)
    var settings map[string]interface{}
    require.NoError(t, json.Unmarshal(raw, &settings))
    hooks := settings["hooks"].(map[string]interface{})
    for _, k := range []string{"SessionStart", "UserPromptSubmit", "PreToolUse", "PostToolUse", "Stop"} {
        _, ok := hooks[k]
        require.True(t, ok, k)
    }
}

func TestInstallHooksIsIdempotent(t *testing.T) {
    home := t.TempDir()
    t.Setenv("HOME", home)
    target := filepath.Join(home, ".claude", "settings.json")
    require.NoError(t, installHooksTo(target, "tmux-agent-log"))
    require.NoError(t, installHooksTo(target, "tmux-agent-log"))

    raw, _ := os.ReadFile(target)
    var settings map[string]interface{}
    require.NoError(t, json.Unmarshal(raw, &settings))
    hooks := settings["hooks"].(map[string]interface{})
    require.Len(t, hooks["SessionStart"].([]interface{}), 1)
}

func TestUninstallRemovesOurHooks(t *testing.T) {
    home := t.TempDir()
    t.Setenv("HOME", home)
    target := filepath.Join(home, ".claude", "settings.json")
    require.NoError(t, installHooksTo(target, "tmux-agent-log"))
    require.NoError(t, uninstallHooksFrom(target, "tmux-agent-log"))

    raw, _ := os.ReadFile(target)
    var settings map[string]interface{}
    require.NoError(t, json.Unmarshal(raw, &settings))
    hooks, _ := settings["hooks"].(map[string]interface{})
    for _, k := range []string{"SessionStart", "UserPromptSubmit", "PreToolUse", "PostToolUse", "Stop"} {
        arr, _ := hooks[k].([]interface{})
        for _, e := range arr {
            m := e.(map[string]interface{})
            require.NotContains(t, m["command"].(string), "tmux-agent-log")
        }
    }
}
```

**実装メモ:**
- 両ヘルパーは `map[string]interface{}` で動かし、ユーザーが `settings.json` に追加した未知のフィールドを保持する
- `installHooksTo`:
  - ファイル存在時は `json.Unmarshal`、なければ空 map で開始
  - 既存の `"hooks"` map を取得（なければ作成）
  - 各 `hookEvents` エントリで、event の list に `{"matcher":"*","command":"<bin> hook <sub>"}` を追加。**ただし** 既存エントリの `command` に `<bin>` と `<sub>` の両方が含まれていない場合のみ
  - atomic に書込み: 2 スペース indent で marshal、`os.WriteFile(path+".tmp")` → `os.Rename`。親ディレクトリは `0o700`、ファイルは `0o600`
- `uninstallHooksFrom`: 同じ走査で `command` に `<bin>` を含むエントリを drop。結果リストが空なら `hooks` から該当キーを削除
- 上位 cobra コマンドは `os.Executable()` で `<bin>` を解決、エラー時は `"tmux-agent-log"` にフォールバック
- `--dry` は `+ <Event>: <bin> hook <sub>` 形式の行を stdout に出力するだけ（書込みなし）
- MVP では対話的確認なし。ユーザーが明示的にサブコマンドを実行している前提

**コミット:** `feat(cli): install-hooks/uninstall-hooks を追加（冪等な settings.json 編集）`

---

## Phase 8: end-to-end

### Task 30: スモークテスト

**ファイル:**
- 作成: `scripts/smoke.sh`

- [ ] **Step 1:** `scripts/smoke.sh` を作成:

  ```bash
  #!/usr/bin/env bash
  # End-to-end スモークテスト。tmux と git が PATH に必要。

  set -euo pipefail

  BIN_DIR=$(mktemp -d)
  BIN="$BIN_DIR/tmux-agent-log"
  STATE_DIR=$(mktemp -d)
  REPO=$(mktemp -d)

  trap 'rm -rf "$BIN_DIR" "$STATE_DIR" "$REPO"' EXIT

  export XDG_STATE_HOME="$STATE_DIR"
  export TMUX_AGENT_LOG_ASSUME_YES=1

  go build -o "$BIN" ./cmd/tmux-agent-log

  cd "$REPO"
  git init -q
  git config user.email a@b
  git config user.name a
  git commit --allow-empty -q -m base
  cd - >/dev/null

  echo "{\"session_id\":\"s1\",\"cwd\":\"$REPO\",\"transcript_path\":\"/tmp/t\"}" | "$BIN" hook session-start
  echo "{\"session_id\":\"s1\",\"cwd\":\"$REPO\",\"prompt\":\"add hello file\"}" | "$BIN" hook turn-start

  TURN_ID=$("$BIN" list-turns --session s1 --format tsv | cut -f1 | head -1)

  echo "{\"session_id\":\"s1\",\"cwd\":\"$REPO\",\"turn_id\":\"$TURN_ID\",\"tool_name\":\"Write\",\"tool_input\":{\"file_path\":\"hello.txt\"}}" | "$BIN" hook tool-pre

  echo "hello world" > "$REPO/hello.txt"
  (cd "$REPO" && git add hello.txt && git commit -q -m "add hello")

  echo "{\"session_id\":\"s1\",\"cwd\":\"$REPO\",\"turn_id\":\"$TURN_ID\",\"tool_name\":\"Write\",\"tool_response\":{\"success\":true}}" | "$BIN" hook tool-post
  echo "{\"session_id\":\"s1\",\"cwd\":\"$REPO\"}" | "$BIN" hook turn-end

  STATUS=$("$BIN" list-turns --session s1 --format tsv | cut -f4 | head -1)
  [[ "$STATUS" == "done" ]] || { echo "FAIL: expected status=done, got $STATUS" >&2; exit 1; }

  "$BIN" comment add --session s1 --file hello.txt --line 1 --text "why world?"
  "$BIN" comment send --session s1 --preview | grep -q "why world?" || { echo "FAIL: send --preview missing text"; exit 1; }

  "$BIN" export --session s1 --format md | grep -q "^# " || { echo "FAIL: export missing H1"; exit 1; }

  echo "SMOKE OK"
  ```

- [ ] **Step 2:** 実行:

  ```bash
  chmod +x scripts/smoke.sh
  ./scripts/smoke.sh
  ```

  期待: 最終行に `SMOKE OK`。

- [ ] **Step 3:** コミット:

  ```bash
  git add scripts/
  git commit -m "test: hook + CLI フロー全体をカバーする end-to-end スモークスクリプトを追加"
  ```

### Task 31: Plan 1 の README

**ファイル:**
- 作成: `README.md`

- [ ] **Step 1:** README を作成（簡潔に、全体像は spec を参照させる）:

  ```markdown
  # tmux-agent-log

  tmux 内で動く Claude Code セッションのための構造化履歴レイヤー。

  **ステータス:** Plan 1 のスコープのみ（CLI プリミティブ）。対話 TUI、tail viewer、配布成果物は Plan 2 / 3 で対応する。

  ## できること

  - 各 Claude Code turn を append-only JSONL + 各 turn の git diff として記録
  - 現在の diff に anchor された行コメントを追加し、`tmux send-keys` で 1 つのプロンプトとして Claude に送り返す（送信先 pane 消失時は OSC 52 クリップボード fallback）
  - Go 単一バイナリで、`fzf` / `delta` / `bat` / `jq` 等の shell ワークフローと自然合成可能な CLI プリミティブを提供

  ## インストール（ソースビルド）

  ```bash
  go install github.com/yusei-wy/tmux-agent-log/cmd/tmux-agent-log@latest
  tmux-agent-log install-hooks      # ~/.claude/settings.json への opt-in 編集
  ```

  前提: tmux 3.2+、git、ビルド時に Go 1.26+。

  ## クイック使用例

  ```sh
  # tmux ペインで普段通り Claude Code を起動
  claude

  # goal を設定（任意）
  tmux-agent-log goal --session <uuid> "2700 認可バグ修正"

  # セッションの diff と turn を確認
  tmux-agent-log list-turns --session <uuid>
  tmux-agent-log show-diff --base session <uuid>

  # file:line に anchor したコメントを追加
  tmux-agent-log comment add --session <uuid> \
      --file src/auth/middleware.go --line 44-46 \
      --text "nil check は本当に必要?"

  # レビュープロンプトをプレビューしてから Claude に送信
  tmux-agent-log comment send --session <uuid> --preview
  tmux-agent-log comment send --session <uuid>

  # PR 説明用に Markdown サマリーを export
  tmux-agent-log export --session <uuid> --format md
  ```

  ## Spec

  完全な設計、スコープ、non-goals は [`docs/superpowers/specs/2026-04-23-tmux-agent-log-design.md`](docs/superpowers/specs/2026-04-23-tmux-agent-log-design.md) を参照。

  ## ステータスとロードマップ

  - **Plan 1（このリリース）: CLI コア** — hooks、storage、comment send、install-hooks
  - **Plan 2:** 対話 TUI（timeline / diff / overview タブ）、tail viewer、blame + liveness map
  - **Plan 3:** tmux/fzf/shell 連携の examples、GitHub Actions CI、GoReleaser + Homebrew tap

  ## ライセンス

  MIT（予定。初回公開リリース前に確定）。
  ```

- [ ] **Step 2:** コミット:

  ```bash
  git add README.md
  git commit -m "docs: Plan 1 コア CLI 用の README を追加"
  ```

---

## セルフレビューチェックリスト

全 31 タスク完了後に確認:

- [ ] `go build ./...` が警告なしで成功する
- [ ] `go test -race ./...` が全部 green
- [ ] `./scripts/smoke.sh` が `SMOKE OK` を出力
- [ ] 手動: バイナリをインストール、`tmux-agent-log install-hooks` を実行、git リポジトリで `claude` セッションを開始、軽く対話してから:
  - `list-sessions` で新セッションが見える
  - `list-turns --session <id>` で turn が見える
  - コメント追加 + `comment send --preview` でレンダリング済みプロンプトが見える
  - `show-diff <id>` が有効な unified diff を出力
- [ ] `install-hooks` を 2 回実行しても重複しない（冪等）
- [ ] `uninstall-hooks` で hooks map から `tmux-agent-log` エントリが全部消える
- [ ] 全タスクがそれぞれ独立したコミットになっている

## 次の Plan

- **Plan 2（TUI）:** bubbletea ベースの対話 UI（timeline / diff / overview タブ）、chroma syntax highlighting、`blame.json` + `liveness.json` の計算、`tail` + `tui` サブコマンド
- **Plan 3（リリース）:** `examples/`（tmux / fzf / shell パターン）、GitHub Actions CI（Ubuntu + macOS）、Darwin/Linux × amd64/arm64 の GoReleaser、Homebrew tap、リリースワークフロー

---

*Plan 1 of 3。`docs/superpowers/specs/2026-04-23-tmux-agent-log-design.md` から `superpowers:writing-plans` で生成。*
