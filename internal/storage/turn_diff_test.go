package storage_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yusei-wy/tmux-agent-log/internal/storage"
)

func TestTurnDiffRelPath(t *testing.T) {
	require.Equal(t, filepath.Join("diffs", "t1.patch"), storage.TurnDiffRelPath("t1"))
}

func TestWriteTurnDiffCreatesDirAndFile(t *testing.T) {
	sDir := t.TempDir()
	require.NoError(t, storage.WriteTurnDiff(sDir, "t1", []byte("--- a/x\n+++ b/x\n")))

	body, err := os.ReadFile(filepath.Join(sDir, storage.TurnDiffRelPath("t1")))
	require.NoError(t, err)
	require.Equal(t, "--- a/x\n+++ b/x\n", string(body))
}

func TestReadTurnDiffRoundtrip(t *testing.T) {
	sDir := t.TempDir()
	require.NoError(t, storage.WriteTurnDiff(sDir, "t1", []byte("hello")))

	body, err := storage.ReadTurnDiff(sDir, "t1")
	require.NoError(t, err)
	require.Equal(t, "hello", string(body))
}

func TestReadTurnDiffMissingReturnsNil(t *testing.T) {
	body, err := storage.ReadTurnDiff(t.TempDir(), "absent")
	require.NoError(t, err)
	require.Nil(t, body)
}
