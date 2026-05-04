package tmux

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
)

func IsInsideTmux() bool {
	return os.Getenv("TMUX") != ""
}

func CurrentPane() string {
	return os.Getenv("TMUX_PANE")
}

func PaneExists(paneID string) (bool, error) {
	return paneExistsWithSocket("", paneID)
}

func paneExistsWithSocket(socket, paneID string) (bool, error) {
	if paneID == "" {
		return false, nil
	}
	args := []string{}
	if socket != "" {
		args = append(args, "-S", socket)
	}
	args = append(args, "list-panes", "-a", "-F", "#{pane_id}")

	cmd := exec.Command("tmux", args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return false, nil
		}
		return false, err
	}
	for _, line := range strings.Split(stdout.String(), "\n") {
		if strings.TrimSpace(line) == paneID {
			return true, nil
		}
	}
	return false, nil
}
