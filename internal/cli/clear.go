package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/yusei-wy/tmux-agent-log/internal/config"
)

func init() {
	rootCmd.AddCommand(clearCmd())
}

func clearCmd() *cobra.Command {
	var sessionID, olderThan string
	var all bool
	cmd := &cobra.Command{
		Use:   "clear",
		Short: "セッション or 全プロジェクトを削除",
		RunE: func(cmd *cobra.Command, args []string) error {
			set := 0
			if sessionID != "" {
				set++
			}
			if all {
				set++
			}
			if olderThan != "" {
				set++
			}
			if set != 1 {
				return errors.New("--session / --all / --older-than のうち 1 つを指定する")
			}

			if sessionID != "" {
				dir, err := findSessionDir(sessionID)
				if err != nil {
					return fmt.Errorf("find session dir: %w", err)
				}
				if err := os.RemoveAll(dir); err != nil {
					return fmt.Errorf("remove session dir %s: %w", dir, err)
				}
				return nil
			}

			state, err := config.StateDir()
			if err != nil {
				return fmt.Errorf("resolve state dir: %w", err)
			}
			projects := filepath.Join(state, "projects")

			if all {
				if !confirmClearAll(cmd) {
					return errors.New("確認されなかった")
				}
				if err := os.RemoveAll(projects); err != nil {
					return fmt.Errorf("remove projects dir %s: %w", projects, err)
				}
				return nil
			}

			d, err := parseDurationWithDays(olderThan)
			if err != nil {
				return fmt.Errorf("parse --older-than: %w", err)
			}
			cutoff := time.Now().Add(-d)
			projEntries, err := os.ReadDir(projects)
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return fmt.Errorf("read projects dir %s: %w", projects, err)
			}
			for _, p := range projEntries {
				sessDir := filepath.Join(projects, p.Name(), "sessions")
				sessEntries, err := os.ReadDir(sessDir)
				if err != nil {
					continue
				}
				for _, s := range sessEntries {
					full := filepath.Join(sessDir, s.Name())
					st, err := os.Stat(full)
					if err != nil {
						continue
					}
					if !st.ModTime().Before(cutoff) {
						continue
					}
					if err := os.RemoveAll(full); err != nil {
						return fmt.Errorf("remove session dir %s: %w", full, err)
					}
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "removed", full)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&sessionID, "session", "", "削除するセッション ID")
	cmd.Flags().BoolVar(&all, "all", false, "全プロジェクトを削除")
	cmd.Flags().StringVar(&olderThan, "older-than", "", "この期間より古いセッションを削除（例: 24h, 7d）")
	return cmd
}

func parseDurationWithDays(s string) (time.Duration, error) {
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, fmt.Errorf("不正な日数: %q", s)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

func confirmClearAll(cmd *cobra.Command) bool {
	if os.Getenv("TMUX_AGENT_LOG_ASSUME_YES") == "1" {
		return true
	}
	st, _ := os.Stdin.Stat()
	if (st.Mode() & os.ModeCharDevice) == 0 {
		return false
	}
	_, _ = fmt.Fprint(cmd.OutOrStdout(), "本当に全セッションを削除する? [y/N]: ")
	var ans string
	_, _ = fmt.Fscanln(cmd.InOrStdin(), &ans)
	return strings.EqualFold(ans, "y")
}
