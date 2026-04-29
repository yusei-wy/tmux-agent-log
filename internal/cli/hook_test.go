package cli

import (
	"bytes"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHookSubcommandSessionStart(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	cwd := t.TempDir()
	require.NoError(t, exec.Command("git", "-C", cwd, "init").Run())

	rootCmd.SetArgs([]string{"hook", "session-start"})
	rootCmd.SetIn(bytes.NewBufferString(`{"session_id":"abc","cwd":"` + cwd + `"}`))
	rootCmd.SetOut(new(bytes.Buffer))
	rootCmd.SetErr(new(bytes.Buffer))
	require.NoError(t, rootCmd.Execute())
}
