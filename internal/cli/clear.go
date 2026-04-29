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
					return err
				}
				return os.RemoveAll(dir)
			}

			state, err := config.StateDir()
			if err != nil {
				return err
			}
			projects := filepath.Join(state, "projects")

			if all {
				if !confirmAll(cmd) {
					return errors.New("確認されなかった")
				}
				return os.RemoveAll(projects)
			}

			d, err := parseDuration(olderThan)
			if err != nil {
				return err
			}
			cutoff := time.Now().Add(-d)
			projEntries, err := os.ReadDir(projects)
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
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
					if st.ModTime().Before(cutoff) {
						if err := os.RemoveAll(full); err != nil {
							return err
						}
						fmt.Fprintln(cmd.OutOrStdout(), "removed", full)
					}
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

func parseDuration(s string) (time.Duration, error) {
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, fmt.Errorf("不正な日数: %q", s)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

func confirmAll(cmd *cobra.Command) bool {
	if os.Getenv("TMUX_AGENT_LOG_ASSUME_YES") == "1" {
		return true
	}
	st, _ := os.Stdin.Stat()
	if (st.Mode() & os.ModeCharDevice) == 0 {
		return false
	}
	fmt.Fprint(cmd.OutOrStdout(), "本当に全セッションを削除する? [y/N]: ")
	var ans string
	fmt.Fscanln(cmd.InOrStdin(), &ans)
	return strings.EqualFold(ans, "y")
}
