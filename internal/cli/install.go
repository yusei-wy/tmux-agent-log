package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var hookEvents = []struct {
	Event string
	Sub   string
}{
	{"SessionStart", "session-start"},
	{"UserPromptSubmit", "turn-start"},
	{"PreToolUse", "tool-pre"},
	{"PostToolUse", "tool-post"},
	{"Stop", "turn-end"},
}

func init() {
	rootCmd.AddCommand(installHooksCmd())
	rootCmd.AddCommand(uninstallHooksCmd())
}

func installHooksCmd() *cobra.Command {
	var (
		dry   bool
		scope string
	)

	cmd := &cobra.Command{
		Use:   "install-hooks",
		Short: "settings.json に tmux-agent-log hook を追加",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveSettingsPath(scope)
			if err != nil {
				return fmt.Errorf("resolve claude settings path: %w", err)
			}

			bin := resolveBinName()
			if dry {
				for _, e := range hookEvents {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "+ %s: %s hook %s\n", e.Event, bin, e.Sub)
				}

				return nil
			}

			if err := installHooksTo(path, bin); err != nil {
				return fmt.Errorf("install hooks: %w", err)
			}

			return nil
		},
	}
	cmd.Flags().BoolVar(&dry, "dry", false, "書込まずに dry-run")
	cmd.Flags().StringVar(&scope, "scope", "user", "設定の書き込み先 (user: ~/.claude, project: ./.claude)")

	return cmd
}

func uninstallHooksCmd() *cobra.Command {
	var scope string

	cmd := &cobra.Command{
		Use:   "uninstall-hooks",
		Short: "settings.json から tmux-agent-log の hook エントリを削除",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveSettingsPath(scope)
			if err != nil {
				return fmt.Errorf("resolve claude settings path: %w", err)
			}

			if err := uninstallHooksFrom(path, resolveBinName()); err != nil {
				return fmt.Errorf("uninstall hooks: %w", err)
			}

			return nil
		},
	}
	cmd.Flags().StringVar(&scope, "scope", "user", "設定の書き込み先 (user: ~/.claude, project: ./.claude)")

	return cmd
}

func resolveBinName() string {
	if exe, err := os.Executable(); err == nil {
		return exe
	}

	return "tmux-agent-log"
}

func resolveSettingsPath(scope string) (string, error) {
	switch scope {
	case "user":
		home := os.Getenv("HOME")
		if home == "" {
			return "", errors.New("HOME が設定されていない")
		}

		return filepath.Join(home, ".claude", "settings.json"), nil
	case "project":
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get working directory: %w", err)
		}

		return filepath.Join(wd, ".claude", "settings.json"), nil
	default:
		return "", fmt.Errorf("unknown scope %q (user or project)", scope)
	}
}

func loadSettings(path string) (map[string]any, error) {
	//nolint:gosec // path は resolveSettingsPath() が組み立てた .claude/settings.json（user or project）。利用者自身の設定ファイルを読む設計上の意図。
	body, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return map[string]any{}, nil
		}

		return nil, fmt.Errorf("read claude settings %s: %w", path, err)
	}

	if len(strings.TrimSpace(string(body))) == 0 {
		return map[string]any{}, nil
	}

	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, fmt.Errorf("parse claude settings %s: %w", path, err)
	}

	if m == nil {
		m = map[string]any{}
	}

	return m, nil
}

func saveSettings(path string, m map[string]any) error {
	body, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal claude settings: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create claude settings dir: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o600); err != nil {
		return fmt.Errorf("write claude settings tmp %s: %w", tmp, err)
	}

	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename claude settings tmp to %s: %w", path, err)
	}

	return nil
}

func installHooksTo(path, bin string) error {
	settings, err := loadSettings(path)
	if err != nil {
		return err
	}

	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
	}

	for _, e := range hookEvents {
		cmdStr := bin + " hook " + e.Sub

		list, _ := hooks[e.Event].([]any)
		if alreadyInstalled(list, bin, e.Sub) {
			continue
		}

		list = append(list, map[string]any{
			"matcher": "*",
			"command": cmdStr,
		})
		hooks[e.Event] = list
	}

	settings["hooks"] = hooks

	return saveSettings(path, settings)
}

func uninstallHooksFrom(path, bin string) error {
	settings, err := loadSettings(path)
	if err != nil {
		return err
	}

	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		return nil
	}

	for _, e := range hookEvents {
		list, _ := hooks[e.Event].([]any)

		filtered := list[:0]
		for _, item := range list {
			m, ok := item.(map[string]any)
			if !ok {
				filtered = append(filtered, item)
				continue
			}

			cmd, _ := m["command"].(string)
			if strings.Contains(cmd, bin) {
				continue
			}

			filtered = append(filtered, item)
		}

		if len(filtered) == 0 {
			delete(hooks, e.Event)
		} else {
			hooks[e.Event] = filtered
		}
	}

	settings["hooks"] = hooks

	return saveSettings(path, settings)
}

func alreadyInstalled(list []any, bin, sub string) bool {
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}

		cmd, _ := m["command"].(string)
		if strings.Contains(cmd, bin) && strings.Contains(cmd, sub) {
			return true
		}
	}

	return false
}
