package hook

import (
	"bytes"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yusei-wy/tmux-agent-log/internal/config"
	"github.com/yusei-wy/tmux-agent-log/internal/storage"
)

func setupGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, c := range [][]string{
		{"init"},
		{"config", "user.email", "a@b"},
		{"config", "user.name", "a"},
		{"commit", "--allow-empty", "-m", "base"},
	} {
		require.NoError(t, exec.Command("git", append([]string{"-C", dir}, c...)...).Run())
	}
	return dir
}

func TestSessionStart(t *testing.T) {
	cases := []struct {
		name           string
		useGit         bool
		sessionID      string
		wantTracked    bool
		wantBaseSHALen int
	}{
		{
			name:           "git repo is tracked with 40-char base SHA",
			useGit:         true,
			sessionID:      "abc",
			wantTracked:    true,
			wantBaseSHALen: 40,
		},
		{
			name:           "non-git directory has tracked false and empty base SHA",
			useGit:         false,
			sessionID:      "def",
			wantTracked:    false,
			wantBaseSHALen: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("XDG_STATE_HOME", t.TempDir())
			var cwd string
			if tc.useGit {
				cwd = setupGitRepo(t)
			} else {
				cwd = t.TempDir()
			}
			require.NoError(t, RunSessionStart(bytes.NewBufferString(
				`{"session_id":"`+tc.sessionID+`","cwd":"`+cwd+`","transcript_path":"/tmp/t.jsonl"}`)))

			sDir, _ := config.SessionDir(cwd, tc.sessionID)
			meta, err := storage.ReadSessionMeta(sDir)
			require.NoError(t, err)
			require.Equal(t, tc.sessionID, meta.ClaudeSessionID)
			require.Equal(t, cwd, meta.Cwd)
			require.Equal(t, tc.wantTracked, meta.GitTracked)
			require.Len(t, meta.BaseSHA, tc.wantBaseSHALen)
			require.NotZero(t, meta.StartedAt)
		})
	}
}
