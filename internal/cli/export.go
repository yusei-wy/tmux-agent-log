package cli

import (
	"errors"
	"fmt"
	"os"
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
				return err
			}
			meta, err := storage.ReadSessionMeta(sDir)
			if err != nil {
				return err
			}
			turns, err := storage.ReadTurns(filepath.Join(sDir, "turns.jsonl"))
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			title := meta.Goal
			if title == "" {
				title = "Session Export"
			}
			fmt.Fprintf(out, "# %s\n\n", title)
			fmt.Fprintf(out, "- session: `%s`\n", meta.ClaudeSessionID)
			fmt.Fprintf(out, "- cwd: `%s`\n", meta.Cwd)
			fmt.Fprintf(out, "- base: `%s`\n\n", meta.BaseSHA)
			fmt.Fprintln(out, "## Turns")
			for _, t := range turns {
				fmt.Fprintf(out, "\n### %s\n\n", t.ID)
				fmt.Fprintf(out, "- started_at: %s\n", format.Time(t.StartedAt))
				if t.UserPromptPreview != "" {
					fmt.Fprintf(out, "- prompt: %s\n", t.UserPromptPreview)
				}
				if t.DiffPath != "" {
					body, err := os.ReadFile(filepath.Join(sDir, t.DiffPath))
					if err == nil {
						fmt.Fprintln(out, "\n```diff")
						out.Write(body)
						fmt.Fprintln(out, "```")
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
