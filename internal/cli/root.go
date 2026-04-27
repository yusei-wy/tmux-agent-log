package cli

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "tmux-agent-log",
	Short: "tmux 内で動く Claude Code のための構造化履歴レイヤー",
	Long:  "Claude Code セッションを構造化 JSONL で記録し、turn ごとの diff を取り、行コメントを agent に送り返せる。",
}

// Execute は cmd/tmux-agent-log/main.go から呼ばれるエントリポイント。
func Execute() error {
	return rootCmd.Execute()
}
