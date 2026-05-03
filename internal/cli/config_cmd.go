package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/yusei-wy/tmux-agent-log/internal/config"
	"github.com/yusei-wy/tmux-agent-log/internal/format"
)

func init() {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "config を読む / パス確認 / 編集",
	}
	cmd.AddCommand(configShowCmd())
	cmd.AddCommand(configPathCmd())
	cmd.AddCommand(configEditCmd())
	rootCmd.AddCommand(cmd)
}

func configShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "現在の config を整形 JSON で出力",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return format.JSONIndent(cmd.OutOrStdout(), cfg)
		},
	}
}

func configPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "config.toml の絶対パスを出力",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := configFilePath()
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), path)
			return nil
		},
	}
}

func configEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "$EDITOR で config.toml を開く",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := configFilePath()
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				return err
			}
			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "vi"
			}
			//nolint:gosec // EDITOR は利用者自身の環境変数。ユーザーが指定したエディタで設定ファイルを開くのは設計上の意図。
			c := exec.Command(editor, path)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		},
	}
}

func configFilePath() (string, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}
