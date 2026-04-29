package tmux

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsInsideTmuxDetectsEnv(t *testing.T) {
	t.Setenv("TMUX", "/tmp/tmux-1000/default,123,0")
	require.True(t, IsInsideTmux())
}

func TestIsInsideTmuxFalseWhenUnset(t *testing.T) {
	t.Setenv("TMUX", "")
	require.False(t, IsInsideTmux())
}

func TestPaneExists(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}
	sock := t.TempDir() + "/tmux.sock"
	_ = exec.Command("tmux", "-S", sock, "kill-server").Run()
	require.NoError(t, exec.Command("tmux", "-S", sock, "new-session", "-d", "-s", "t", "sleep", "5").Run())
	defer exec.Command("tmux", "-S", sock, "kill-server").Run()

	out, err := exec.Command("tmux", "-S", sock, "list-panes", "-t", "t", "-F", "#{pane_id}").Output()
	require.NoError(t, err)
	paneID := strings.TrimSpace(string(out))

	ok, err := paneExistsWithSocket(sock, paneID)
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = paneExistsWithSocket(sock, "%9999")
	require.NoError(t, err)
	require.False(t, ok)
}
