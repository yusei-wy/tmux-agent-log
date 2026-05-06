package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yusei-wy/tmux-agent-log/internal/config"
	"github.com/yusei-wy/tmux-agent-log/internal/errlog"
)

func init() {
	cmd := &cobra.Command{
		Use:   "errors",
		Short: "errors.jsonl を読み書きする",
	}
	cmd.AddCommand(errorsListCmd())
	cmd.AddCommand(errorsClearCmd())
	rootCmd.AddCommand(cmd)
}

func errorsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "errors を 1 行 JSON で出力",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := config.Load()
			if size, err := errlog.FileSize(); err == nil && cfg.MaxLogSize > 0 && size > cfg.MaxLogSize {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
					"warning: errors.jsonl is %d MB — consider `tal errors clear` to free space\n",
					size/(1<<20),
				)
			}

			entries, err := errlog.Read()
			if err != nil {
				return fmt.Errorf("read errors log: %w", err)
			}

			out := cmd.OutOrStdout()
			for _, e := range entries {
				body, err := json.Marshal(e)
				if err != nil {
					return fmt.Errorf("marshal error entry: %w", err)
				}

				_, _ = out.Write(body)
				_, _ = out.Write([]byte{'\n'})
			}

			return nil
		},
	}
}

func errorsClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "errors.jsonl を削除",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := errlog.Clear(); err != nil {
				return fmt.Errorf("clear errors log: %w", err)
			}

			return nil
		},
	}
}
