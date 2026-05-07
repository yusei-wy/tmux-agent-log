package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/yusei-wy/tmux-agent-log/internal/config"
	"github.com/yusei-wy/tmux-agent-log/internal/storage"
)

func init() {
	rootCmd.AddCommand(listSessionsCmd())
	rootCmd.AddCommand(listTurnsCmd())
}

func listSessionsCmd() *cobra.Command {
	var formatName string

	cmd := &cobra.Command{
		Use:   "list-sessions",
		Short: "全プロジェクトの全セッションを列挙する",
		RunE: func(cmd *cobra.Command, args []string) error {
			state, err := config.StateDir()
			if err != nil {
				return fmt.Errorf("resolve state dir: %w", err)
			}

			projects := filepath.Join(state, "projects")
			rows := [][]string{}

			projEntries, err := os.ReadDir(projects)
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("read projects dir %s: %w", projects, err)
			}

			for _, p := range projEntries {
				if !p.IsDir() {
					continue
				}

				sessDir := filepath.Join(projects, p.Name(), "sessions")

				sessEntries, err := os.ReadDir(sessDir)
				if err != nil {
					continue
				}

				for _, s := range sessEntries {
					if !s.IsDir() {
						continue
					}

					meta, err := storage.ReadSessionMeta(filepath.Join(sessDir, s.Name()))
					if err != nil {
						continue
					}

					rows = append(rows, []string{
						meta.ClaudeSessionID,
						p.Name(),
						meta.Goal,
						meta.Cwd,
						formatTime(meta.StartedAt),
					})
				}
			}

			sort.Slice(rows, func(i, j int) bool { return rows[i][4] > rows[j][4] })

			return writeFormatted(cmd.OutOrStdout(), formatName,
				[]string{"session_id", "project", "goal", "cwd", "started_at"}, rows)
		},
	}
	cmd.Flags().StringVar(&formatName, "format", "table", "tsv | jsonl | json | table")

	return cmd
}

func listTurnsCmd() *cobra.Command {
	var sessionID, formatName string

	cmd := &cobra.Command{
		Use:   "list-turns",
		Short: "セッションの turn 一覧",
		RunE: func(cmd *cobra.Command, args []string) error {
			if sessionID == "" {
				return errors.New("--session が必須")
			}

			sDir, err := findSessionDir(sessionID)
			if err != nil {
				return fmt.Errorf("find session dir: %w", err)
			}

			turns, err := storage.ReadTurns(filepath.Join(sDir, "turns.jsonl"))
			if err != nil {
				return fmt.Errorf("read turns: %w", err)
			}

			rows := make([][]string, 0, len(turns))
			for _, t := range turns {
				rows = append(rows, []string{
					t.ID,
					formatTime(t.StartedAt),
					formatTimePtr(t.EndedAt),
					string(t.Status),
					t.DiffPath,
					t.UserPromptPreview,
				})
			}

			return writeFormatted(cmd.OutOrStdout(), formatName,
				[]string{"id", "started_at", "ended_at", "status", "diff_path", "prompt_preview"}, rows)
		},
	}
	cmd.Flags().StringVar(&sessionID, "session", "", "セッション ID（必須）")
	cmd.Flags().StringVar(&formatName, "format", "table", "tsv | jsonl | json | table")

	return cmd
}
