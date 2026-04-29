package config_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yusei-wy/tmux-agent-log/internal/config"
)

func TestStateDirDefaultToXDG(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/tmp/xdg-state")
	dir, err := config.StateDir()
	require.NoError(t, err)
	require.Equal(t, "/tmp/xdg-state/tmux-agent-log", dir)
}

func TestStateDirFallbackToHome(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", "/tmp/myhome")
	dir, err := config.StateDir()
	require.NoError(t, err)
	require.Equal(t, "/tmp/myhome/.local/state/tmux-agent-log", dir)
}

func TestProjectSlug(t *testing.T) {
	slug := config.ProjectSlug("/Users/alias/src/myproject")
	require.Contains(t, slug, "myproject-")
	require.Len(t, slug, len("myproject-")+8)
}

func TestSessionDir(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/tmp/xdg-state")
	got, err := config.SessionDir("/Users/alias/src/myproject", "abc-123")
	require.NoError(t, err)
	want := filepath.Join("/tmp/xdg-state/tmux-agent-log/projects", config.ProjectSlug("/Users/alias/src/myproject"), "sessions", "abc-123")
	require.Equal(t, want, got)
}
