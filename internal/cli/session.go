package cli

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/yusei-wy/tmux-agent-log/internal/config"
)

// findSessionDir は state ディレクトリ配下の全プロジェクトを横断して
// session ID にマッチする state ディレクトリを返す。
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
