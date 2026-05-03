package errlog_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yusei-wy/tmux-agent-log/internal/errlog"
)

func TestRecordAndRead(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	require.NoError(t, errlog.Record("hook/turn-end", "git-diff-failed", "session-abc", "boom"))
	entries, err := errlog.Read()
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
	require.NoError(t, errlog.Record("c", "e", "s", "err"))
	require.NoError(t, errlog.Clear())
	_, err := os.Stat(filepath.Join(dir, "tmux-agent-log", "errors.jsonl"))
	require.True(t, os.IsNotExist(err))
}
