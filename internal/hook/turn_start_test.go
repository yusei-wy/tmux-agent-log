package hook

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yusei-wy/tmux-agent-log/internal/config"
	"github.com/yusei-wy/tmux-agent-log/internal/storage"
)

func TestTurnStartAppendsOpenRecord(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	cwd := setupGitRepo(t)

	require.NoError(t, RunSessionStart(bytes.NewBufferString(`{"session_id":"abc","cwd":"`+cwd+`"}`)))
	require.NoError(t, RunTurnStart(bytes.NewBufferString(`{"session_id":"abc","cwd":"`+cwd+`","prompt":"do the thing"}`)))

	sDir, _ := config.SessionDir(cwd, "abc")
	turns, err := storage.ReadTurns(filepath.Join(sDir, "turns.jsonl"))
	require.NoError(t, err)
	require.Len(t, turns, 1)
	require.Equal(t, "open", turns[0].Status)
	require.Equal(t, "do the thing", turns[0].UserPromptPreview)
	require.NotEmpty(t, turns[0].HeadSHAPre)
}

func TestPreviewFirstLinesTruncatesAndLimitsLines(t *testing.T) {
	require.Equal(t, "line1\nline2", previewFirstLines("line1\nline2\nline3", 100))
	require.Equal(t, "abc…", previewFirstLines("abcdef", 3))
}
