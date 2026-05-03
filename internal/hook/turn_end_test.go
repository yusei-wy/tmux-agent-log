package hook

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yusei-wy/tmux-agent-log/internal/config"
	"github.com/yusei-wy/tmux-agent-log/internal/storage"
)

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
	require.Equal(t, storage.TurnStatusDone, turns[0].Status)
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
	require.Equal(t, storage.TurnStatusDone, turns[0].Status)
	require.Equal(t, "", turns[0].DiffPath)
}
