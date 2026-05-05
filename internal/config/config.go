// Package config はユーザー設定の読み込みとデフォルト値を提供する。
package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const DefaultSendEditorCommand = "${EDITOR:-nvim}"

// DefaultMaxLogSize はログファイルサイズ警告閾値のデフォルト (5 MB)。
const DefaultMaxLogSize int64 = 5 << 20

type Config struct {
	SendEditorCommand    string `toml:"send_editor_command"`
	DisableOSC52Fallback bool   `toml:"disable_osc52_fallback"`
	StateDirOverride     string `toml:"state_dir"`
	MaxLogSize           int64  `toml:"max_log_size"`
}

func Load() (Config, error) {
	cfg := Config{
		SendEditorCommand: DefaultSendEditorCommand,
		MaxLogSize:        DefaultMaxLogSize,
	}

	dir, err := ConfigDir()
	if err != nil {
		return cfg, nil
	}

	path := filepath.Join(dir, "config.toml")

	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		if errors.Is(err, fs.ErrNotExist) || os.IsNotExist(err) {
			return cfg, nil
		}

		return Config{}, err
	}

	if cfg.SendEditorCommand == "" {
		cfg.SendEditorCommand = DefaultSendEditorCommand
	}

	return cfg, nil
}
