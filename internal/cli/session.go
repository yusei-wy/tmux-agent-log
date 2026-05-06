package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/yusei-wy/tmux-agent-log/internal/config"
)

func findSessionDir(sessionID string) (string, error) {
	state, err := config.StateDir()
	if err != nil {
		return "", fmt.Errorf("resolve state dir: %w", err)
	}

	projects := filepath.Join(state, "projects")

	entries, err := os.ReadDir(projects)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", os.ErrNotExist
		}

		return "", fmt.Errorf("read projects dir %s: %w", projects, err)
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
