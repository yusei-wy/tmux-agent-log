package cli

import (
	"io"

	"github.com/spf13/cobra"

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

func mkHook(name string, runner func(io.Reader) error) *cobra.Command {
	return &cobra.Command{
		Use: name,
		RunE: func(cmd *cobra.Command, args []string) error {
			hook.RunWithRecover(func() error { return runner(cmd.InOrStdin()) })
			return nil
		},
	}
}
