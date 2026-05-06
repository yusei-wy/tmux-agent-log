package hook_test

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
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
