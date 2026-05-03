package hook

import (
	"bytes"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yusei-wy/tmux-agent-log/internal/config"
	"github.com/yusei-wy/tmux-agent-log/internal/storage"
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
