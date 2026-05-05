package storage_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/yusei-wy/tmux-agent-log/internal/storage"
)

func TestAppendAndReadComments(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.jsonl")
	require.NoError(t, storage.AppendComment(p, storage.Comment{ID: "c1", File: "a.go", LineStart: 10, LineEnd: 12, Text: "foo", CreatedAt: time.Unix(1, 0).UTC()}))
	require.NoError(t, storage.AppendComment(p, storage.Comment{ID: "c2", File: "b.go", LineStart: 20, LineEnd: 20, Text: "bar", CreatedAt: time.Unix(2, 0).UTC()}))
	got, err := storage.ReadComments(p)
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, "c1", got[0].ID)
	require.Equal(t, "c2", got[1].ID)
}

func TestMarkCommentSent(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.jsonl")
	require.NoError(t, storage.AppendComment(p, storage.Comment{ID: "c1", File: "a.go", LineStart: 1, LineEnd: 1, Text: "x", CreatedAt: time.Unix(1, 0).UTC()}))
	require.NoError(t, storage.MarkCommentsSent(p, []string{"c1"}, time.Unix(2, 0).UTC()))
	got, err := storage.ReadComments(p)
	require.NoError(t, err)
	require.NotNil(t, got[0].SentAt)
	require.Equal(t, time.Unix(2, 0).UTC(), *got[0].SentAt)
}

func TestDeleteComment(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.jsonl")
	require.NoError(t, storage.AppendComment(p, storage.Comment{ID: "c1", File: "a", LineStart: 1, LineEnd: 1, Text: "x"}))
	require.NoError(t, storage.AppendComment(p, storage.Comment{ID: "c2", File: "a", LineStart: 2, LineEnd: 2, Text: "y"}))
	require.NoError(t, storage.DeleteComment(p, "c1"))
	got, err := storage.ReadComments(p)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "c2", got[0].ID)
}

func TestUnsentComments(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.jsonl")
	require.NoError(t, storage.AppendComment(p, storage.Comment{ID: "c1", File: "a", LineStart: 1, LineEnd: 1, Text: "x", CreatedAt: time.Unix(1, 0).UTC()}))
	require.NoError(t, storage.AppendComment(p, storage.Comment{ID: "c2", File: "a", LineStart: 2, LineEnd: 2, Text: "y", CreatedAt: time.Unix(2, 0).UTC()}))
	require.NoError(t, storage.MarkCommentsSent(p, []string{"c1"}, time.Unix(3, 0).UTC()))
	got, err := storage.UnsentComments(p)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "c2", got[0].ID)
}
