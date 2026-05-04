package storage

import (
	"errors"
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
		return err
	}
	return os.WriteFile(filepath.Join(sDir, TurnDiffRelPath(turnID)), content, 0o600)
}

// ファイル不在は (nil, nil) を返し、呼び出し側が「diff なし」として扱えるようにする。
func ReadTurnDiff(sDir, turnID string) ([]byte, error) {
	body, err := os.ReadFile(filepath.Join(sDir, TurnDiffRelPath(turnID)))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return body, nil
}
