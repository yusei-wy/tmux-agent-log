package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yusei-wy/tmux-agent-log/internal/cli"
)

func TestInstallHooksCreatesSettingsFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	target := filepath.Join(home, ".claude", "settings.json")
	require.NoError(t, cli.InstallHooksTo(target, "tmux-agent-log"))

	raw, err := os.ReadFile(target)
	require.NoError(t, err)

	var settings map[string]any
	require.NoError(t, json.Unmarshal(raw, &settings))

	hooks := settings["hooks"].(map[string]any)
	for _, k := range []string{"SessionStart", "UserPromptSubmit", "PreToolUse", "PostToolUse", "Stop"} {
		_, ok := hooks[k]
		require.True(t, ok, k)
	}
}

func TestInstallHooksIsIdempotent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	target := filepath.Join(home, ".claude", "settings.json")
	require.NoError(t, cli.InstallHooksTo(target, "tmux-agent-log"))
	require.NoError(t, cli.InstallHooksTo(target, "tmux-agent-log"))

	raw, _ := os.ReadFile(target)

	var settings map[string]any
	require.NoError(t, json.Unmarshal(raw, &settings))
	hooks := settings["hooks"].(map[string]any)
	require.Len(t, hooks["SessionStart"].([]any), 1)
}

func TestUninstallRemovesOurHooks(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	target := filepath.Join(home, ".claude", "settings.json")
	require.NoError(t, cli.InstallHooksTo(target, "tmux-agent-log"))
	require.NoError(t, cli.UninstallHooksFrom(target, "tmux-agent-log"))

	raw, _ := os.ReadFile(target)

	var settings map[string]any
	require.NoError(t, json.Unmarshal(raw, &settings))

	hooks, _ := settings["hooks"].(map[string]any)
	for _, k := range []string{"SessionStart", "UserPromptSubmit", "PreToolUse", "PostToolUse", "Stop"} {
		arr, _ := hooks[k].([]any)
		for _, e := range arr {
			m := e.(map[string]any)
			require.NotContains(t, m["command"].(string), "tmux-agent-log")
		}
	}
}
