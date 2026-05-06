package storage_test

import (
	"encoding/json"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yusei-wy/tmux-agent-log/internal/storage"
)

type sampleRec struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

func TestAppendAndReadLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.jsonl")
	require.NoError(t, storage.AppendJSONL(path, sampleRec{ID: "a", Text: "hello"}))
	require.NoError(t, storage.AppendJSONL(path, sampleRec{ID: "b", Text: "world"}))

	var got []sampleRec

	require.NoError(t, storage.ReadJSONL(path, func(raw []byte) error {
		var r sampleRec
		if err := json.Unmarshal(raw, &r); err != nil {
			return nil
		}

		got = append(got, r)

		return nil
	}))
	require.Equal(t, []sampleRec{{ID: "a", Text: "hello"}, {ID: "b", Text: "world"}}, got)
}

func TestReadSkipsCorruptedLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "b.jsonl")
	require.NoError(t, storage.AppendRaw(path, []byte(`{"id":"a"}`)))
	require.NoError(t, storage.AppendRaw(path, []byte(`NOT JSON`)))
	require.NoError(t, storage.AppendRaw(path, []byte(`{"id":"b"}`)))

	good := 0
	corrupted := 0

	require.NoError(t, storage.ReadJSONL(path, func(raw []byte) error {
		var r sampleRec
		if err := json.Unmarshal(raw, &r); err != nil {
			corrupted++
			return nil
		}

		good++

		return nil
	}))
	require.Equal(t, 2, good)
	require.Equal(t, 1, corrupted)
}

func TestConcurrentAppendsSerialize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.jsonl")

	var wg sync.WaitGroup

	const n = 50
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()

			require.NoError(t, storage.AppendJSONL(path, sampleRec{ID: "x"}))
		}()
	}

	wg.Wait()

	total := 0

	require.NoError(t, storage.ReadJSONL(path, func(raw []byte) error { total++; return nil }))
	require.Equal(t, n, total)
}

func TestReadMissingFileReturnsNil(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.jsonl")
	called := false

	require.NoError(t, storage.ReadJSONL(path, func(raw []byte) error {
		called = true
		return nil
	}))
	require.False(t, called)
}
