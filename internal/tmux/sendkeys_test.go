package tmux_test

import (
	"bytes"
	"encoding/base64"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/yusei-wy/tmux-agent-log/internal/tmux"
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

	res := tmux.SendToPaneWithWriters(sock, paneID, "hello world", os.Stdout, os.Stderr)

	require.Equal(t, tmux.SendResultOK, res.Kind)

	time.Sleep(100 * time.Millisecond)
	captured, _ := exec.Command("tmux", "-S", sock, "capture-pane", "-t", paneID, "-p").Output()
	require.Contains(t, string(captured), "hello world")
}

// TestSendKeysFallsBackOnMissingPane は OSC 52 fallback の出力形式を契約として固定する：
// ESC ]52;c; <base64-standard> BEL の構造で、payload は元 text と round-trip 一致する。
func TestSendKeysFallsBackOnMissingPane(t *testing.T) {
	clip := &bytes.Buffer{}
	res := tmux.SendToPaneWithWriters("", "%9999", "hello world", clip, nil)
	require.Equal(t, tmux.SendResultFallbackClipboard, res.Kind)

	seq := clip.String()
	require.True(t, strings.HasPrefix(seq, "\x1b]52;c;"), "missing OSC 52 prefix")
	require.True(t, strings.HasSuffix(seq, "\x07"), "missing BEL terminator")

	payload := strings.TrimSuffix(strings.TrimPrefix(seq, "\x1b]52;c;"), "\x07")
	decoded, err := base64.StdEncoding.DecodeString(payload)
	require.NoError(t, err)
	require.Equal(t, "hello world", string(decoded))
}
