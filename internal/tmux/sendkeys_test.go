package tmux

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSendKeysToValidPane(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}
	sock := t.TempDir() + "/tmux.sock"
	_ = exec.Command("tmux", "-S", sock, "kill-server").Run()
	require.NoError(t, exec.Command("tmux", "-S", sock, "new-session", "-d", "-s", "t", "cat").Run())
	defer exec.Command("tmux", "-S", sock, "kill-server").Run()

	out, _ := exec.Command("tmux", "-S", sock, "list-panes", "-t", "t", "-F", "#{pane_id}").Output()
	paneID := strings.TrimSpace(string(out))

	res := SendToPaneWithSocket(sock, paneID, "hello world")
	require.Equal(t, SendResultOK, res.Kind)

	time.Sleep(100 * time.Millisecond)
	captured, _ := exec.Command("tmux", "-S", sock, "capture-pane", "-t", paneID, "-p").Output()
	require.Contains(t, string(captured), "hello world")
}

func TestSendKeysFallsBackOnMissingPane(t *testing.T) {
	clip := &bytes.Buffer{}
	res := sendToPaneWithWriters("", "%9999", "hello", clip, nil)
	require.Equal(t, SendResultFallbackClipboard, res.Kind)
	require.True(t, strings.HasPrefix(clip.String(), "\x1b]52;c;"))
}
