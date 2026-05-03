package git_test

import (
	"os/exec"
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
