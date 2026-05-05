package storage

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const turnDiffsDir = "diffs"

func TurnDiffRelPath(turnID string) string {
	return filepath.Join(turnDiffsDir, turnID+".patch")
}

func WriteTurnDiff(sDir, turnID string, content []byte) error {
	if err := os.MkdirAll(filepath.Join(sDir, turnDiffsDir), 0o700); err != nil {
		return fmt.Errorf("create turn diffs dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(sDir, TurnDiffRelPath(turnID)), content, 0o600); err != nil {
		return fmt.Errorf("write turn diff: %w", err)
	}
	return nil
}

// ReadTurnDiff はファイル不在時に (nil, nil) を返す。呼び出し側は「diff なし」として扱える。
func ReadTurnDiff(sDir, turnID string) ([]byte, error) {
	path := filepath.Join(sDir, TurnDiffRelPath(turnID))
	body, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read turn diff %s: %w", path, err)
	}
	return body, nil
}
