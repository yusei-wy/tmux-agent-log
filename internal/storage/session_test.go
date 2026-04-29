package storage_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yusei-wy/tmux-agent-log/internal/storage"
)

func TestWriteAndReadSession(t *testing.T) {
	dir := t.TempDir()
	m := storage.SessionMeta{
		ClaudeSessionID: "abc",
		Cwd:             "/tmp/r",
		GitTracked:      true,
		StartedAt:       time.Unix(100, 0).UTC(),
	}
	require.NoError(t, storage.WriteSessionMeta(dir, m))
	got, err := storage.ReadSessionMeta(dir)
	require.NoError(t, err)
	require.Equal(t, m, got)
}

func TestReadSessionMissingReturnsError(t *testing.T) {
	_, err := storage.ReadSessionMeta(filepath.Join(t.TempDir(), "missing"))
	require.Error(t, err)
}

func TestUpdateSessionGoal(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, storage.WriteSessionMeta(dir, storage.SessionMeta{ClaudeSessionID: "abc"}))
	require.NoError(t, storage.UpdateSessionGoal(dir, "2700"))
	got, err := storage.ReadSessionMeta(dir)
	require.NoError(t, err)
	require.Equal(t, "2700", got.Goal)
}
