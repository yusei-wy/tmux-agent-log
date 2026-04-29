package storage_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yusei-wy/tmux-agent-log/internal/storage"
)

func TestAppendAndReadEvents(t *testing.T) {
	p := filepath.Join(t.TempDir(), "events.jsonl")
	require.NoError(t, storage.AppendEvent(p, storage.Event{ID: "e1", TurnID: "t1", Ts: time.Unix(1, 0).UTC(), Tool: "Read", Phase: storage.EventPhasePre}))
	require.NoError(t, storage.AppendEvent(p, storage.Event{ID: "e2", TurnID: "t1", Ts: time.Unix(2, 0).UTC(), Tool: "Read", Phase: storage.EventPhasePost, Success: true}))
	got, err := storage.ReadEvents(p, "t1")
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, "e1", got[0].ID)
	require.Equal(t, "e2", got[1].ID)
}

func TestReadEventsFiltersByTurn(t *testing.T) {
	p := filepath.Join(t.TempDir(), "events.jsonl")
	require.NoError(t, storage.AppendEvent(p, storage.Event{ID: "e1", TurnID: "t1"}))
	require.NoError(t, storage.AppendEvent(p, storage.Event{ID: "e2", TurnID: "t2"}))
	got, err := storage.ReadEvents(p, "t2")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "e2", got[0].ID)
}

func TestReadEventsEmptyTurnIDReturnsAll(t *testing.T) {
	p := filepath.Join(t.TempDir(), "events.jsonl")
	require.NoError(t, storage.AppendEvent(p, storage.Event{ID: "e1", TurnID: "t1"}))
	require.NoError(t, storage.AppendEvent(p, storage.Event{ID: "e2", TurnID: "t2"}))
	got, err := storage.ReadEvents(p, "")
	require.NoError(t, err)
	require.Len(t, got, 2)
}
