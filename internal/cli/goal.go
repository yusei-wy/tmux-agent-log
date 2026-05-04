package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yusei-wy/tmux-agent-log/internal/storage"
)

func init() {
	rootCmd.AddCommand(goalCmd())
}

func goalCmd() *cobra.Command {
	var sessionID string
	cmd := &cobra.Command{
		Use:   "goal [title]",
		Short: "セッションの goal を取得 / 設定",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if sessionID == "" {
				return errors.New("--session が必須")
			}
			sDir, err := findSessionDir(sessionID)
			if err != nil {
				return fmt.Errorf("find session dir: %w", err)
			}
			if len(args) == 0 {
				meta, err := storage.ReadSessionMeta(sDir)
				if err != nil {
					return fmt.Errorf("read session meta: %w", err)
				}
				out := meta.Goal
				if out == "" {
					out = "(no goal)"
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), out)
				return nil
			}
			if err := storage.UpdateSessionGoal(sDir, args[0]); err != nil {
				return fmt.Errorf("update session goal: %w", err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&sessionID, "session", "", "セッション ID（必須）")
	return cmd
}
