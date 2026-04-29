package storage_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yusei-wy/tmux-agent-log/internal/storage"
)

func TestOpenCloseMerge(t *testing.T) {
	p := filepath.Join(t.TempDir(), "turns.jsonl")
	require.NoError(t, storage.AppendTurnOpen(p, storage.TurnOpen{
		ID:                "t1",
		StartedAt:         time.Unix(100, 0).UTC(),
		UserPromptPreview: "hi",
	}))
	require.NoError(t, storage.AppendTurnClose(p, storage.TurnClose{
		ID:       "t1",
		EndedAt:  time.Unix(200, 0).UTC(),
		HeadSHA:  "def",
		Status:   "done",
		DiffPath: "diffs/t1.patch",
	}))
	turns, err := storage.ReadTurns(p)
	require.NoError(t, err)
	require.Len(t, turns, 1)
	require.Equal(t, "done", turns[0].Status)
	require.Equal(t, "def", turns[0].HeadSHA)
	require.Equal(t, "hi", turns[0].UserPromptPreview)
}

func TestOpenWithoutCloseKeepsStatusOpen(t *testing.T) {
	p := filepath.Join(t.TempDir(), "turns.jsonl")
	require.NoError(t, storage.AppendTurnOpen(p, storage.TurnOpen{
		ID:        "t2",
		StartedAt: time.Unix(1, 0).UTC(),
	}))
	turns, err := storage.ReadTurns(p)
	require.NoError(t, err)
	require.Equal(t, "open", turns[0].Status)
}

func TestReadTurnsOrderedByStartedAt(t *testing.T) {
	p := filepath.Join(t.TempDir(), "turns.jsonl")
	require.NoError(t, storage.AppendTurnOpen(p, storage.TurnOpen{ID: "b", StartedAt: time.Unix(200, 0).UTC()}))
	require.NoError(t, storage.AppendTurnOpen(p, storage.TurnOpen{ID: "a", StartedAt: time.Unix(100, 0).UTC()}))
	turns, err := storage.ReadTurns(p)
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b"}, []string{turns[0].ID, turns[1].ID})
}
