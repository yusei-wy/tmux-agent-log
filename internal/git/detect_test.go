package git_test

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yusei-wy/tmux-agent-log/internal/git"
)

func TestIsRepo(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T) string
		want  bool
	}{
		{
			name:  "non-repo directory returns false",
			setup: func(t *testing.T) string { return t.TempDir() },
			want:  false,
		},
		{
			name: "initialized repo returns true",
			setup: func(t *testing.T) string {
				d := t.TempDir()
				require.NoError(t, exec.Command("git", "-C", d, "init").Run())
				return d
			},
			want: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ok, err := git.IsRepo(tc.setup(t))
			require.NoError(t, err)
			require.Equal(t, tc.want, ok)
		})
	}
}

func TestHeadSHA(t *testing.T) {
	cases := []struct {
		name    string
		setup   func(t *testing.T) string
		wantLen int
	}{
		{
			name: "fresh repo returns empty SHA",
			setup: func(t *testing.T) string {
				d := t.TempDir()
				require.NoError(t, exec.Command("git", "-C", d, "init").Run())
				return d
			},
			wantLen: 0,
		},
		{
			name: "repo with commit returns 40-char SHA",
			setup: func(t *testing.T) string {
				d := t.TempDir()
				for _, args := range [][]string{
					{"init"},
					{"config", "user.email", "a@b"},
					{"config", "user.name", "a"},
					{"commit", "--allow-empty", "-m", "x"},
				} {
					require.NoError(t, exec.Command("git", append([]string{"-C", d}, args...)...).Run())
				}
				return d
			},
			wantLen: 40,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sha, err := git.HeadSHA(tc.setup(t))
			require.NoError(t, err)
			require.Len(t, sha, tc.wantLen)
		})
	}
}
