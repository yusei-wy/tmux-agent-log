package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yusei-wy/tmux-agent-log/internal/git"
)

func TestIsRepoReturnsFalseForNonRepo(t *testing.T) {
	ok, err := git.IsRepo(t.TempDir())
	require.NoError(t, err)
	require.False(t, ok)
}

func TestIsRepoReturnsTrueForRepo(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, exec.Command("git", "-C", dir, "init").Run())
	ok, err := git.IsRepo(dir)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestHeadSHAOfFreshRepoIsEmpty(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, exec.Command("git", "-C", dir, "init").Run())
	sha, err := git.HeadSHA(dir)
	require.NoError(t, err)
	require.Equal(t, "", sha)
}

func TestHeadSHAAfterCommit(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "a@b"},
		{"config", "user.name", "a"},
		{"commit", "--allow-empty", "-m", "x"},
	} {
		require.NoError(t, exec.Command("git", append([]string{"-C", dir}, args...)...).Run())
	}

	sha, err := git.HeadSHA(dir)
	require.NoError(t, err)
	require.Len(t, sha, 40)
}

func setupRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	for _, c := range [][]string{
		{"init"},
		{"config", "user.email", "a@b"},
		{"config", "user.name", "a"},
	} {
		require.NoError(t, exec.Command("git", append([]string{"-C", dir}, c...)...).Run())
	}

	return dir
}

func TestDiffSince(t *testing.T) {
	cases := []struct {
		name         string
		setup        func(t *testing.T, dir string) string
		wantContains []string
	}{
		{
			name: "includes committed changes between base and HEAD",
			setup: func(t *testing.T, dir string) string {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello\n"), 0o644))
				require.NoError(t, exec.Command("git", "-C", dir, "add", "a.txt").Run())
				require.NoError(t, exec.Command("git", "-C", dir, "commit", "-m", "c1").Run())
				base, err := git.HeadSHA(dir)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("world\n"), 0o644))
				require.NoError(t, exec.Command("git", "-C", dir, "add", "a.txt").Run())
				require.NoError(t, exec.Command("git", "-C", dir, "commit", "-m", "c2").Run())

				return base
			},
			wantContains: []string{"-hello", "+world"},
		},
		{
			name: "includes unstaged working tree changes",
			setup: func(t *testing.T, dir string) string {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hi\n"), 0o644))
				require.NoError(t, exec.Command("git", "-C", dir, "add", "a.txt").Run())
				require.NoError(t, exec.Command("git", "-C", dir, "commit", "-m", "c1").Run())
				base, err := git.HeadSHA(dir)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("changed\n"), 0o644))

				return base
			},
			wantContains: []string{"-hi", "+changed"},
		},
		{
			name: "empty base diffs against the empty tree",
			setup: func(t *testing.T, dir string) string {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "new.txt"), []byte("hello\n"), 0o644))
				return ""
			},
			wantContains: []string{"+hello"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := setupRepo(t)
			base := tc.setup(t, dir)
			diff, err := git.DiffSince(dir, base)
			require.NoError(t, err)

			for _, s := range tc.wantContains {
				require.Contains(t, diff, s)
			}
		})
	}
}
