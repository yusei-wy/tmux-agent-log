package cli

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/yusei-wy/tmux-agent-log/internal/format"
	"github.com/yusei-wy/tmux-agent-log/internal/storage"
)

func init() {
	rootCmd.AddCommand(exportCmd())
}

func exportCmd() *cobra.Command {
	var sessionID, formatName string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "セッションを Markdown で出力",
		RunE: func(cmd *cobra.Command, args []string) error {
			if sessionID == "" {
				return errors.New("--session が必須")
			}

			if formatName != "md" {
				return fmt.Errorf("MVP では --format=md のみサポート（got %q）", formatName)
			}

			sDir, err := findSessionDir(sessionID)
			if err != nil {
				return fmt.Errorf("find session dir: %w", err)
			}

			meta, err := storage.ReadSessionMeta(sDir)
			if err != nil {
				return fmt.Errorf("read session meta: %w", err)
			}

			turns, err := storage.ReadTurns(filepath.Join(sDir, "turns.jsonl"))
			if err != nil {
				return fmt.Errorf("read turns: %w", err)
			}

			out := cmd.OutOrStdout()

			title := meta.Goal
			if title == "" {
				title = "Session Export"
			}

			_, _ = fmt.Fprintf(out, "# %s\n\n", title)
			_, _ = fmt.Fprintf(out, "- session: `%s`\n", meta.ClaudeSessionID)
			_, _ = fmt.Fprintf(out, "- cwd: `%s`\n", meta.Cwd)
			_, _ = fmt.Fprintf(out, "- base: `%s`\n\n", meta.BaseSHA)

			_, _ = fmt.Fprintln(out, "## Turns")
			for _, t := range turns {
				_, _ = fmt.Fprintf(out, "\n### %s\n\n", t.ID)

				_, _ = fmt.Fprintf(out, "- started_at: %s\n", format.Time(t.StartedAt))
				if t.UserPromptPreview != "" {
					_, _ = fmt.Fprintf(out, "- prompt: %s\n", t.UserPromptPreview)
				}

				if t.DiffPath != "" {
					body, err := storage.ReadTurnDiff(sDir, t.ID)
					if err != nil {
						return fmt.Errorf("read turn diff %s: %w", t.ID, err)
					}

					if body != nil {
						_, _ = fmt.Fprintln(out, "\n```diff")
						_, _ = out.Write(body)
						_, _ = fmt.Fprintln(out, "```")
					}
				}
			}

			return nil
		},
	}
	cmd.Flags().StringVar(&sessionID, "session", "", "セッション ID（必須）")
	cmd.Flags().StringVar(&formatName, "format", "md", "現状 md のみ")

	return cmd
}
