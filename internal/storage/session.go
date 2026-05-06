package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func MetaFile(sessionDir string) string {
	return filepath.Join(sessionDir, "meta.json")
}

func WriteSessionMeta(sessionDir string, m SessionMeta) error {
	if err := os.MkdirAll(sessionDir, 0o700); err != nil {
		return fmt.Errorf("create session dir: %w", err)
	}

	body, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session meta: %w", err)
	}

	if err := os.WriteFile(MetaFile(sessionDir), body, 0o600); err != nil {
		return fmt.Errorf("write session meta %s: %w", MetaFile(sessionDir), err)
	}

	return nil
}

func ReadSessionMeta(sessionDir string) (SessionMeta, error) {
	body, err := os.ReadFile(MetaFile(sessionDir))
	if err != nil {
		return SessionMeta{}, fmt.Errorf("read session meta %s: %w", MetaFile(sessionDir), err)
	}

	var m SessionMeta
	if err := json.Unmarshal(body, &m); err != nil {
		return SessionMeta{}, fmt.Errorf("parse session meta %s: %w", MetaFile(sessionDir), err)
	}

	return m, nil
}

func UpdateSessionGoal(sessionDir, goal string) error {
	m, err := ReadSessionMeta(sessionDir)
	if err != nil {
		return fmt.Errorf("update session goal: %w", err)
	}

	m.Goal = goal

	return WriteSessionMeta(sessionDir, m)
}
