package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/yusei-wy/tmux-agent-log/internal/storage"
	"github.com/yusei-wy/tmux-agent-log/internal/tmux"
)

func init() {
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "diff に対する行コメントを管理する",
	}
	cmd.AddCommand(commentAddCmd())
	cmd.AddCommand(commentListCmd())
	cmd.AddCommand(commentDeleteCmd())
	cmd.AddCommand(commentSendCmd())
	rootCmd.AddCommand(cmd)
}

func commentsPath(sessionID string) (string, error) {
	sDir, err := findSessionDir(sessionID)
	if err != nil {
		return "", err
	}
	return filepath.Join(sDir, "comments.jsonl"), nil
}

func commentAddCmd() *cobra.Command {
	var sessionID, file, line, text string
	cmd := &cobra.Command{
		Use:   "add",
		Short: "コメントを追加",
		RunE: func(cmd *cobra.Command, args []string) error {
			if sessionID == "" || file == "" || line == "" || text == "" {
				return errors.New("--session/--file/--line/--text すべて必須")
			}
			start, end, err := parseLineRange(line)
			if err != nil {
				return err
			}
			path, err := commentsPath(sessionID)
			if err != nil {
				return err
			}
			c := storage.Comment{
				ID:        "cmt-" + uuid.NewString(),
				File:      file,
				LineStart: start,
				LineEnd:   end,
				Text:      text,
				CreatedAt: time.Now().UTC(),
			}
			if err := storage.AppendComment(path, c); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), c.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&sessionID, "session", "", "セッション ID")
	cmd.Flags().StringVar(&file, "file", "", "ファイルパス")
	cmd.Flags().StringVar(&line, "line", "", "行番号 / 範囲（例: 44 / 44-46）")
	cmd.Flags().StringVar(&text, "text", "", "コメント本文")
	return cmd
}

func commentListCmd() *cobra.Command {
	var sessionID string
	var unsent bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "コメントを列挙",
		RunE: func(cmd *cobra.Command, args []string) error {
			if sessionID == "" {
				return errors.New("--session が必須")
			}
			path, err := commentsPath(sessionID)
			if err != nil {
				return err
			}
			var comments []storage.Comment
			if unsent {
				comments, err = storage.UnsentComments(path)
			} else {
				comments, err = storage.ReadComments(path)
			}
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			for _, c := range comments {
				flag := ""
				if !c.SentAt.IsZero() {
					flag = " [sent]"
				}
				fmt.Fprintf(out, "%s  %s:%d-%d%s\n  %s\n", c.ID, c.File, c.LineStart, c.LineEnd, flag, c.Text)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&sessionID, "session", "", "セッション ID")
	cmd.Flags().BoolVar(&unsent, "unsent", false, "未送信のみ")
	return cmd
}

func commentDeleteCmd() *cobra.Command {
	var sessionID string
	cmd := &cobra.Command{
		Use:   "delete <comment-id>",
		Short: "コメントを論理削除",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if sessionID == "" {
				return errors.New("--session が必須")
			}
			path, err := commentsPath(sessionID)
			if err != nil {
				return err
			}
			return storage.DeleteComment(path, args[0])
		},
	}
	cmd.Flags().StringVar(&sessionID, "session", "", "セッション ID")
	return cmd
}

func commentSendCmd() *cobra.Command {
	var sessionID string
	var preview bool
	cmd := &cobra.Command{
		Use:   "send",
		Short: "未送信コメントを Claude に送る",
		RunE: func(cmd *cobra.Command, args []string) error {
			if sessionID == "" {
				return errors.New("--session が必須")
			}
			sDir, err := findSessionDir(sessionID)
			if err != nil {
				return err
			}
			meta, err := storage.ReadSessionMeta(sDir)
			if err != nil {
				return err
			}
			path := filepath.Join(sDir, "comments.jsonl")
			unsent, err := storage.UnsentComments(path)
			if err != nil {
				return err
			}
			if len(unsent) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "(no unsent comments)")
				return nil
			}
			prompt := renderSendPrompt(unsent)
			if preview {
				fmt.Fprint(cmd.OutOrStdout(), prompt)
				return nil
			}
			res := tmux.SendToPane(meta.TmuxPane, prompt)
			ids := make([]string, 0, len(unsent))
			for _, c := range unsent {
				ids = append(ids, c.ID)
			}
			switch res.Kind {
			case tmux.SendResultOK:
				return storage.MarkCommentsSent(path, ids, time.Now().UTC())
			case tmux.SendResultFallbackClipboard:
				fmt.Fprintln(os.Stderr, "tmux pane が見つからないためクリップボード経由で送信しました")
				return storage.MarkCommentsSent(path, ids, time.Now().UTC())
			case tmux.SendResultFailed:
				return fmt.Errorf("send failed: %w", res.Err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&sessionID, "session", "", "セッション ID")
	cmd.Flags().BoolVar(&preview, "preview", false, "送信せずにプロンプトのみ表示")
	return cmd
}

func parseLineRange(s string) (int, int, error) {
	if !strings.Contains(s, "-") {
		n, err := strconv.Atoi(s)
		if err != nil {
			return 0, 0, fmt.Errorf("不正な line: %q", s)
		}
		return n, n, nil
	}
	parts := strings.SplitN(s, "-", 2)
	start, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("不正な line: %q", s)
	}
	end, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("不正な line: %q", s)
	}
	if end < start {
		return 0, 0, fmt.Errorf("end < start: %q", s)
	}
	return start, end, nil
}

func renderSendPrompt(cs []storage.Comment) string {
	var b strings.Builder
	b.WriteString("以下のレビューコメントを反映してください:\n\n")
	for _, c := range cs {
		fmt.Fprintf(&b, "- %s:%d-%d\n  %s\n\n", c.File, c.LineStart, c.LineEnd, c.Text)
	}
	b.WriteString("(反映後、関連テストを実行して結果を報告してください)")
	return b.String()
}
