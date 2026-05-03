package hook_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yusei-wy/tmux-agent-log/internal/config"
	"github.com/yusei-wy/tmux-agent-log/internal/hook"
	"github.com/yusei-wy/tmux-agent-log/internal/storage"
)

func TestToolPreAndPostAppendEvents(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	cwd := setupGitRepo(t)
	require.NoError(t, hook.RunSessionStart(bytes.NewBufferString(`{"session_id":"s1","cwd":"`+cwd+`"}`)))
	require.NoError(t, hook.RunTurnStart(bytes.NewBufferString(`{"session_id":"s1","cwd":"`+cwd+`","prompt":"p"}`)))

	sDir, _ := config.SessionDir(cwd, "s1")
	turns, _ := storage.ReadTurns(filepath.Join(sDir, "turns.jsonl"))
	turnID := turns[0].ID

	require.NoError(t, hook.RunToolPre(bytes.NewBufferString(
		`{"session_id":"s1","cwd":"`+cwd+`","turn_id":"`+turnID+`","tool_name":"Read","tool_input":{"file_path":"/a"}}`)))
	require.NoError(t, hook.RunToolPost(bytes.NewBufferString(
		`{"session_id":"s1","cwd":"`+cwd+`","turn_id":"`+turnID+`","tool_name":"Read","tool_response":{"success":true}}`)))

	events, _ := storage.ReadEvents(filepath.Join(sDir, "events.jsonl"), turnID)
	require.Len(t, events, 2)
	require.Equal(t, storage.EventPhasePre, events[0].Phase)
	require.Equal(t, storage.EventPhasePost, events[1].Phase)
	require.True(t, events[1].Success)
}
