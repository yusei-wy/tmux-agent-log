package storage_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/yusei-wy/tmux-agent-log/internal/storage"
)

func TestReadEvents(t *testing.T) {
	cases := []struct {
		name       string
		events     []storage.Event
		filterTurn string
		wantIDs    []string
	}{
		{
			name: "matching turn id returns events in append order",
			events: []storage.Event{
				{ID: "e1", TurnID: "t1", TS: time.Unix(1, 0).UTC(), Tool: "Read", Phase: storage.EventPhasePre},
				{ID: "e2", TurnID: "t1", TS: time.Unix(2, 0).UTC(), Tool: "Read", Phase: storage.EventPhasePost, Success: true},
			},
			filterTurn: "t1",
			wantIDs:    []string{"e1", "e2"},
		},
		{
			name: "non-matching turn ids are filtered out",
			events: []storage.Event{
				{ID: "e1", TurnID: "t1"},
				{ID: "e2", TurnID: "t2"},
			},
			filterTurn: "t2",
			wantIDs:    []string{"e2"},
		},
		{
			name: "empty filter returns all events",
			events: []storage.Event{
				{ID: "e1", TurnID: "t1"},
				{ID: "e2", TurnID: "t2"},
			},
			filterTurn: "",
			wantIDs:    []string{"e1", "e2"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "events.jsonl")
			for _, e := range tc.events {
				require.NoError(t, storage.AppendEvent(p, e))
			}

			got, err := storage.ReadEvents(p, tc.filterTurn)
			require.NoError(t, err)

			ids := make([]string, len(got))
			for i, e := range got {
				ids[i] = e.ID
			}

			require.Equal(t, tc.wantIDs, ids)
		})
	}
}
