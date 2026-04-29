package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yusei-wy/tmux-agent-log/internal/git"
)

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

func TestDiffIncludesCommittedChanges(t *testing.T) {
	dir := setupRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello\n"), 0o644))
	require.NoError(t, exec.Command("git", "-C", dir, "add", "a.txt").Run())
	require.NoError(t, exec.Command("git", "-C", dir, "commit", "-m", "c1").Run())
	base, _ := git.HeadSHA(dir)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("world\n"), 0o644))
	require.NoError(t, exec.Command("git", "-C", dir, "add", "a.txt").Run())
	require.NoError(t, exec.Command("git", "-C", dir, "commit", "-m", "c2").Run())

	diff, err := git.DiffSince(dir, base)
	require.NoError(t, err)
	require.Contains(t, diff, "-hello")
	require.Contains(t, diff, "+world")
}

func TestDiffIncludesUnstagedChanges(t *testing.T) {
	dir := setupRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hi\n"), 0o644))
	require.NoError(t, exec.Command("git", "-C", dir, "add", "a.txt").Run())
	require.NoError(t, exec.Command("git", "-C", dir, "commit", "-m", "c1").Run())
	base, _ := git.HeadSHA(dir)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("changed\n"), 0o644))
	diff, err := git.DiffSince(dir, base)
	require.NoError(t, err)
	require.Contains(t, diff, "-hi")
	require.Contains(t, diff, "+changed")
}

func TestDiffSinceEmptyBaseUsesEmptyTree(t *testing.T) {
	dir := setupRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "new.txt"), []byte("hello\n"), 0o644))
	diff, err := git.DiffSince(dir, "")
	require.NoError(t, err)
	require.Contains(t, diff, "+hello")
}
