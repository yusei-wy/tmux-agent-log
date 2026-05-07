package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yusei-wy/tmux-agent-log/internal/cli"
)

func TestFindSessionDir(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", stateDir)

	sessDir := filepath.Join(stateDir, "tmux-agent-log", "projects", "myproj-abc12345", "sessions", "sess-001")
	require.NoError(t, os.MkdirAll(sessDir, 0o700))

	cases := []struct {
		name      string
		sessionID string
		wantPath  string
		wantErr   bool
	}{
		{name: "found", sessionID: "sess-001", wantPath: sessDir},
		{name: "not found", sessionID: "nonexistent", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := cli.FindSessionDir(tc.sessionID)
			if tc.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.wantPath, got)
		})
	}
}

func TestFindSessionDirEmptyState(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	_, err := cli.FindSessionDir("any-id")
	require.Error(t, err)
}
