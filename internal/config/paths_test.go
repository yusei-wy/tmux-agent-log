package config_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yusei-wy/tmux-agent-log/internal/config"
)

func TestStateDir(t *testing.T) {
	cases := []struct {
		name string
		xdg  string
		home string
		want string
	}{
		{
			name: "XDG_STATE_HOME takes precedence",
			xdg:  "/tmp/xdg-state",
			want: "/tmp/xdg-state/tmux-agent-log",
		},
		{
			name: "fallback to HOME/.local/state when XDG unset",
			xdg:  "",
			home: "/tmp/myhome",
			want: "/tmp/myhome/.local/state/tmux-agent-log",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("XDG_STATE_HOME", tc.xdg)
			if tc.home != "" {
				t.Setenv("HOME", tc.home)
			}
			dir, err := config.StateDir()
			require.NoError(t, err)
			require.Equal(t, tc.want, dir)
		})
	}
}

func TestProjectSlug(t *testing.T) {
	slug := config.ProjectSlug("/Users/alias/src/myproject")
	require.Contains(t, slug, "myproject-")
	require.Len(t, slug, len("myproject-")+8)
}

// 同じ basename でも cwd が異なれば slug は衝突しない（hash 設計の主目的）。
func TestProjectSlugAvoidsBaseNameCollision(t *testing.T) {
	a := config.ProjectSlug("/Users/alice/src/myproject")
	b := config.ProjectSlug("/Users/bob/src/myproject")
	require.NotEqual(t, a, b)
	require.Contains(t, a, "myproject-")
	require.Contains(t, b, "myproject-")
}

func TestSessionDir(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/tmp/xdg-state")
	got, err := config.SessionDir("/Users/alias/src/myproject", "abc-123")
	require.NoError(t, err)
	want := filepath.Join("/tmp/xdg-state/tmux-agent-log/projects", config.ProjectSlug("/Users/alias/src/myproject"), "sessions", "abc-123")
	require.Equal(t, want, got)
}
