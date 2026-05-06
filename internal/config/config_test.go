package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yusei-wy/tmux-agent-log/internal/config"
)

func TestLoad(t *testing.T) {
	cases := []struct {
		name             string
		body             string // 空文字列なら config.toml を作成しない
		wantSendEditor   string
		wantDisableOSC52 bool
	}{
		{
			name:           "missing file returns defaults",
			body:           "",
			wantSendEditor: config.DefaultSendEditorCommand,
		},
		{
			name:             "TOML overrides apply",
			body:             "send_editor_command = \"vim\"\ndisable_osc52_fallback = true\n",
			wantSendEditor:   "vim",
			wantDisableOSC52: true,
		},
		{
			name:           "empty SendEditorCommand falls back to default",
			body:           "send_editor_command = \"\"\n",
			wantSendEditor: config.DefaultSendEditorCommand,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", dir)

			if tc.body != "" {
				path := filepath.Join(dir, "tmux-agent-log", "config.toml")
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
				require.NoError(t, os.WriteFile(path, []byte(tc.body), 0o600))
			}

			cfg, err := config.Load()
			require.NoError(t, err)
			require.Equal(t, tc.wantSendEditor, cfg.SendEditorCommand)
			require.Equal(t, tc.wantDisableOSC52, cfg.DisableOSC52Fallback)
		})
	}
}
