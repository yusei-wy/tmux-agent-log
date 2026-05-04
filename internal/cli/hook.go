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
	hookCmd.AddCommand(mkHook("session-start", hook.RunSessionStart))
	hookCmd.AddCommand(mkHook("turn-start", hook.RunTurnStart))
	hookCmd.AddCommand(mkHook("tool-pre", hook.RunToolPre))
	hookCmd.AddCommand(mkHook("tool-post", hook.RunToolPost))
	hookCmd.AddCommand(mkHook("turn-end", hook.RunTurnEnd))
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

func mkHook(name string, runner func(io.Reader) error) *cobra.Command {
	return &cobra.Command{
		Use: name,
		RunE: func(cmd *cobra.Command, args []string) error {
			runWithRecover(func() error { return runner(cmd.InOrStdin()) })
			return nil
		},
	}
}
