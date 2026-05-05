// Package errlog はフック実行時のエラーを構造化ログとして記録する。
package errlog

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/yusei-wy/tmux-agent-log/internal/config"
	"github.com/yusei-wy/tmux-agent-log/internal/storage"
)

func Record(component, event, sessionID, errMsg string) error {
	path, err := config.ErrorsPath()
	if err != nil {
		return fmt.Errorf("resolve errors path: %w", err)
	}

	entry := storage.ErrEntry{
		TS:          time.Now().UTC(),
		Component:   component,
		Event:       event,
		SessionID:   sessionID,
		ErrorString: errMsg,
	}

	return storage.AppendJSONL(path, entry)
}

func Read() ([]storage.ErrEntry, error) {
	path, err := config.ErrorsPath()
	if err != nil {
		return nil, fmt.Errorf("resolve errors path: %w", err)
	}

	var out []storage.ErrEntry

	err = storage.ReadJSONL(path, func(raw []byte) error {
		var e storage.ErrEntry
		if err := unmarshal(raw, &e); err != nil {
			return nil
		}

		out = append(out, e)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read errors log: %w", err)
	}

	return out, nil
}

func FileSize() (int64, error) {
	path, err := config.ErrorsPath()
	if err != nil {
		return 0, fmt.Errorf("resolve errors path: %w", err)
	}

	return storage.FileSize(path)
}

func Clear() error {
	path, err := config.ErrorsPath()
	if err != nil {
		return fmt.Errorf("resolve errors path: %w", err)
	}

	for _, p := range []string{path, path + ".lock"} {
		if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("remove %s: %w", p, err)
		}
	}

	return nil
}
