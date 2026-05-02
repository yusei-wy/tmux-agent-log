package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/yusei-wy/tmux-agent-log/internal/format"
	"github.com/yusei-wy/tmux-agent-log/internal/git"
	"github.com/yusei-wy/tmux-agent-log/internal/storage"
)

func init() {
	rootCmd.AddCommand(showSessionCmd())
	rootCmd.AddCommand(showTurnCmd())
	rootCmd.AddCommand(showDiffCmd())
}

func showSessionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show-session <session-id>",
		Short: "session meta を整形 JSON で出力",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sDir, err := findSessionDir(args[0])
			if err != nil {
				return err
			}
			meta, err := storage.ReadSessionMeta(sDir)
			if err != nil {
				return err
			}
			return format.JSONIndent(cmd.OutOrStdout(), meta)
		},
	}
}

func showTurnCmd() *cobra.Command {
	var sessionID string
	var withDiff bool
	cmd := &cobra.Command{
		Use:   "show-turn <turn-id>",
		Short: "turn を整形 JSON で出力（--with-diff で patch 併記）",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if sessionID == "" {
				return errors.New("--session が必須")
			}
			sDir, err := findSessionDir(sessionID)
			if err != nil {
				return err
			}
			turns, err := storage.ReadTurns(filepath.Join(sDir, "turns.jsonl"))
			if err != nil {
				return err
			}
			var found *storage.Turn
			for i := range turns {
				if turns[i].ID == args[0] {
					found = &turns[i]
					break
				}
			}
			if found == nil {
				return fmt.Errorf("turn %q が見つからない", args[0])
			}
			if err := format.JSONIndent(cmd.OutOrStdout(), found); err != nil {
				return err
			}
			if withDiff && found.DiffPath != "" {
				body, err := os.ReadFile(filepath.Join(sDir, found.DiffPath))
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), "--- diff ---")
				cmd.OutOrStdout().Write(body)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&sessionID, "session", "", "セッション ID（必須）")
	cmd.Flags().BoolVar(&withDiff, "with-diff", false, "diff patch を併記")
	return cmd
}

func showDiffCmd() *cobra.Command {
	var base, turnID string
	cmd := &cobra.Command{
		Use:   "show-diff <session-id>",
		Short: "セッションの diff を出力",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sDir, err := findSessionDir(args[0])
			if err != nil {
				return err
			}
			meta, err := storage.ReadSessionMeta(sDir)
			if err != nil {
				return err
			}
			switch base {
			case "session":
				diff, err := git.DiffSince(meta.Cwd, meta.BaseSHA)
				if err != nil {
					return err
				}
				_, err = fmt.Fprint(cmd.OutOrStdout(), diff)
				return err
			case "turn":
				if turnID == "" {
					return errors.New("--base=turn のときは --turn が必須")
				}
				body, err := os.ReadFile(filepath.Join(sDir, "diffs", turnID+".patch"))
				if err != nil {
					if errors.Is(err, os.ErrNotExist) {
						return nil
					}
					return err
				}
				_, err = cmd.OutOrStdout().Write(body)
				return err
			case "main":
				diff, err := git.Run(meta.Cwd, "diff", "--no-color", "-U3", "main", "--")
				if err != nil {
					return err
				}
				_, err = fmt.Fprint(cmd.OutOrStdout(), diff)
				return err
			default:
				return fmt.Errorf("--base は session|turn|main のいずれか（got %q）", base)
			}
		},
	}
	cmd.Flags().StringVar(&base, "base", "session", "session | turn | main")
	cmd.Flags().StringVar(&turnID, "turn", "", "--base=turn のときの turn id")
	return cmd
}
