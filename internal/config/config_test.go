package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yusei-wy/tmux-agent-log/internal/config"
)

func TestLoadReturnsDefaultsWhenFileMissing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, err := config.Load()
	require.NoError(t, err)
	require.Equal(t, config.DefaultSendEditorCommand, cfg.SendEditorCommand)
	require.False(t, cfg.DisableOSC52Fallback)
}

func TestLoadReadsTOMLOverrides(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path := filepath.Join(dir, "tmux-agent-log", "config.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	body := "send_editor_command = \"vim\"\ndisable_osc52_fallback = true\n"
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Equal(t, "vim", cfg.SendEditorCommand)
	require.True(t, cfg.DisableOSC52Fallback)
}

func TestLoadFallsBackToDefaultWhenSendEditorCommandIsEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path := filepath.Join(dir, "tmux-agent-log", "config.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("send_editor_command = \"\"\n"), 0o600))

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Equal(t, config.DefaultSendEditorCommand, cfg.SendEditorCommand)
}
