package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func MetaFile(sessionDir string) string {
	return filepath.Join(sessionDir, "meta.json")
}

func WriteSessionMeta(sessionDir string, m SessionMeta) error {
	if err := os.MkdirAll(sessionDir, 0o700); err != nil {
		return err
	}
	body, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(MetaFile(sessionDir), body, 0o600)
}

func ReadSessionMeta(sessionDir string) (SessionMeta, error) {
	body, err := os.ReadFile(MetaFile(sessionDir))
	if err != nil {
		return SessionMeta{}, err
	}
	var m SessionMeta
	if err := json.Unmarshal(body, &m); err != nil {
		return SessionMeta{}, err
	}
	return m, nil
}

func UpdateSessionGoal(sessionDir, goal string) error {
	m, err := ReadSessionMeta(sessionDir)
	if err != nil {
		return err
	}
	m.Goal = goal
	return WriteSessionMeta(sessionDir, m)
}
