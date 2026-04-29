package cli

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/yusei-wy/tmux-agent-log/internal/config"
)

func findSessionDir(sessionID string) (string, error) {
	state, err := config.StateDir()
	if err != nil {
		return "", err
	}
	projects := filepath.Join(state, "projects")
	entries, err := os.ReadDir(projects)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", os.ErrNotExist
		}
		return "", err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		candidate := filepath.Join(projects, e.Name(), "sessions", sessionID)
		if st, err := os.Stat(candidate); err == nil && st.IsDir() {
			return candidate, nil
		}
	}
	return "", os.ErrNotExist
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02 15:04:05")
}
