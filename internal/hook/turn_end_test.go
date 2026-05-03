package hook

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yusei-wy/tmux-agent-log/internal/config"
	"github.com/yusei-wy/tmux-agent-log/internal/storage"
)

func TestTurnEnd(t *testing.T) {
	cases := []struct {
		name             string
		modifyAfterStart func(t *testing.T, cwd string)
		wantDiffContains string // 空なら DiffPath が空であることを検証する
	}{
		{
			name: "modifications close turn with diff path and patch contents",
			modifyAfterStart: func(t *testing.T, cwd string) {
				require.NoError(t, os.WriteFile(filepath.Join(cwd, "hello.txt"), []byte("hi\n"), 0o644))
				require.NoError(t, exec.Command("git", "-C", cwd, "add", "hello.txt").Run())
				require.NoError(t, exec.Command("git", "-C", cwd, "commit", "-m", "t1").Run())
			},
			wantDiffContains: "+hi",
		},
		{
			name:             "no modifications close turn with empty diff path",
			modifyAfterStart: func(t *testing.T, cwd string) {},
			wantDiffContains: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("XDG_STATE_HOME", t.TempDir())
			cwd := setupGitRepo(t)
			sessionID := "s"
			require.NoError(t, RunSessionStart(bytes.NewBufferString(`{"session_id":"`+sessionID+`","cwd":"`+cwd+`"}`)))
			require.NoError(t, RunTurnStart(bytes.NewBufferString(`{"session_id":"`+sessionID+`","cwd":"`+cwd+`","prompt":"p"}`)))

			tc.modifyAfterStart(t, cwd)

			require.NoError(t, RunTurnEnd(bytes.NewBufferString(`{"session_id":"`+sessionID+`","cwd":"`+cwd+`"}`)))

			sDir, _ := config.SessionDir(cwd, sessionID)
			turns, err := storage.ReadTurns(filepath.Join(sDir, "turns.jsonl"))
			require.NoError(t, err)
			require.Equal(t, storage.TurnStatusDone, turns[0].Status)

			if tc.wantDiffContains == "" {
				require.Equal(t, "", turns[0].DiffPath)
				return
			}
			require.NotEmpty(t, turns[0].HeadSHA)
			require.NotEmpty(t, turns[0].DiffPath)
			raw, err := os.ReadFile(filepath.Join(sDir, turns[0].DiffPath))
			require.NoError(t, err)
			require.Contains(t, string(raw), tc.wantDiffContains)
		})
	}
}
