package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/yusei-wy/tmux-agent-log/internal/errlog"
	"github.com/yusei-wy/tmux-agent-log/internal/hook"
)

func init() {
	hookCmd := &cobra.Command{
		Use:   "hook",
		Short: "Claude Code hook エンドポイント（agent から呼ばれる、人間は直接使わない）",
	}
	hookCmd.AddCommand(makeHookCmd("session-start", "セッション開始を記録", hook.RunSessionStart))
	hookCmd.AddCommand(makeHookCmd("turn-start", "turn 開始を記録", hook.RunTurnStart))
	hookCmd.AddCommand(makeHookCmd("tool-pre", "ツール実行前の状態を記録", hook.RunToolPre))
	hookCmd.AddCommand(makeHookCmd("tool-post", "ツール実行後の状態を記録", hook.RunToolPost))
	hookCmd.AddCommand(makeHookCmd("turn-end", "turn 終了を記録し diff を保存", hook.RunTurnEnd))
	rootCmd.AddCommand(hookCmd)
}

// 補助ツールなので panic が発生しても握りつぶす
func runWithRecover(fn func() error) int {
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("%v", r)
			_ = errlog.Record("hook", "panic", "", msg)
			_, _ = fmt.Fprintln(os.Stderr, "tmux-agent-log: hook panic:", msg)
		}
	}()

	if err := fn(); err != nil {
		_ = errlog.Record("hook", "error", "", err.Error())
		_, _ = fmt.Fprintln(os.Stderr, "tmux-agent-log: hook error:", err)
	}

	return 0
}

func makeHookCmd(name, short string, runner func(io.Reader) error) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			runWithRecover(func() error {
				return runner(cmd.InOrStdin())
			})

			return nil
		},
	}
}
