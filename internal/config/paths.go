package config

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const appName = "tmux-agent-log"

func StateDir() (string, error) {
	if v := os.Getenv("XDG_STATE_HOME"); v != "" {
		return filepath.Join(v, appName), nil
	}

	home := os.Getenv("HOME")
	if home == "" {
		return "", errors.New("both XDG_STATE_HOME and HOME are unset")
	}

	return filepath.Join(home, ".local", "state", appName), nil
}

func ConfigDir() (string, error) {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return filepath.Join(v, appName), nil
	}

	home := os.Getenv("HOME")
	if home == "" {
		return "", errors.New("both XDG_CONFIG_HOME and HOME are unset")
	}

	return filepath.Join(home, ".config", appName), nil
}

func ProjectSlug(cwd string) string {
	base := strings.ReplaceAll(filepath.Base(cwd), "/", "_")
	sum := sha256.Sum256([]byte(cwd))
	return base + "-" + hex.EncodeToString(sum[:4])
}

func SessionDir(cwd, claudeSessionID string) (string, error) {
	state, err := StateDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(state, "projects", ProjectSlug(cwd), "sessions", claudeSessionID), nil
}

func ErrorsPath() (string, error) {
	state, err := StateDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(state, "errors.jsonl"), nil
}
