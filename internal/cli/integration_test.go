package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yusei-wy/tmux-agent-log/internal/cli"
	"github.com/yusei-wy/tmux-agent-log/internal/hook"
)

func initGitRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	for _, c := range [][]string{
		{"init"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "test"},
		{"commit", "--allow-empty", "-m", "initial"},
	} {
		require.NoError(t, exec.Command("git", append([]string{"-C", dir}, c...)...).Run())
	}

	return dir
}

func mustRunCLI(t *testing.T, args ...string) string {
	t.Helper()

	out, err := cli.RunCLI(args...)
	require.NoError(t, err, "RunCLI(%v)", args)

	return out
}

func hookJSON(t *testing.T, fields map[string]string) *bytes.Reader {
	t.Helper()

	b, err := json.Marshal(fields)
	require.NoError(t, err)

	return bytes.NewReader(b)
}

// hook が書いた JSONL を CLI で読めることの smoke test。
func TestIntegrationHookToCLI(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("TMUX_PANE", "")

	cwd := initGitRepo(t)

	const sid = "integ-sess-001"

	require.NoError(t, hook.RunSessionStart(hookJSON(t, map[string]string{
		"session_id": sid, "cwd": cwd,
	})))

	require.NoError(t, hook.RunTurnStart(hookJSON(t, map[string]string{
		"session_id": sid, "cwd": cwd, "prompt": "add hello feature",
	})))

	require.NoError(t, os.WriteFile(filepath.Join(cwd, "hello.go"),
		[]byte("package main\n"), 0o644))
	require.NoError(t, exec.Command("git", "-C", cwd, "add", "hello.go").Run())
	require.NoError(t, exec.Command("git", "-C", cwd, "commit", "-m", "add hello").Run())

	require.NoError(t, hook.RunTurnEnd(hookJSON(t, map[string]string{
		"session_id": sid, "cwd": cwd,
	})))

	out := mustRunCLI(t, "list-sessions", "--format", "tsv")
	require.Contains(t, out, sid)

	out = mustRunCLI(t, "list-turns", "--session", sid, "--format", "jsonl")

	var row map[string]string
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(out)), &row))
	require.Equal(t, "done", row["status"])
	require.Contains(t, row["prompt_preview"], "add hello feature")
}

// 累積 diff (session base → HEAD) と増分 diff (turn 単位) の区別を検証する。
func TestIntegrationMultiTurnDiff(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("TMUX_PANE", "")

	cwd := initGitRepo(t)

	const sid = "integ-multi-turn"

	require.NoError(t, hook.RunSessionStart(hookJSON(t, map[string]string{
		"session_id": sid, "cwd": cwd,
	})))

	// Turn 1: create file
	require.NoError(t, hook.RunTurnStart(hookJSON(t, map[string]string{
		"session_id": sid, "cwd": cwd, "prompt": "create hello.go",
	})))

	require.NoError(t, os.WriteFile(filepath.Join(cwd, "hello.go"),
		[]byte("package main\n\nfunc hello() {}\n"), 0o644))
	require.NoError(t, exec.Command("git", "-C", cwd, "add", "hello.go").Run())
	require.NoError(t, exec.Command("git", "-C", cwd, "commit", "-m", "create hello").Run())

	require.NoError(t, hook.RunTurnEnd(hookJSON(t, map[string]string{
		"session_id": sid, "cwd": cwd,
	})))

	// Turn 2: modify file
	require.NoError(t, hook.RunTurnStart(hookJSON(t, map[string]string{
		"session_id": sid, "cwd": cwd, "prompt": "add world func",
	})))

	require.NoError(t, os.WriteFile(filepath.Join(cwd, "hello.go"),
		[]byte("package main\n\nfunc hello() {}\n\nfunc world() {}\n"), 0o644))
	require.NoError(t, exec.Command("git", "-C", cwd, "add", "hello.go").Run())
	require.NoError(t, exec.Command("git", "-C", cwd, "commit", "-m", "add world").Run())

	require.NoError(t, hook.RunTurnEnd(hookJSON(t, map[string]string{
		"session_id": sid, "cwd": cwd,
	})))

	// Cumulative: session base → HEAD contains both functions
	out := mustRunCLI(t, "show-diff", sid, "--base", "session")
	require.Contains(t, out, "+func hello()")
	require.Contains(t, out, "+func world()")

	// Incremental: turn 2 diff contains only world
	turnsOut := mustRunCLI(t, "list-turns", "--session", sid, "--format", "jsonl")
	lines := strings.Split(strings.TrimSpace(turnsOut), "\n")
	require.Len(t, lines, 2)

	var turn2 map[string]string
	require.NoError(t, json.Unmarshal([]byte(lines[1]), &turn2))

	out = mustRunCLI(t, "show-diff", sid, "--base", "turn", "--turn", turn2["id"])
	require.Contains(t, out, "+func world()")
	require.NotContains(t, out, "+func hello()")
}
