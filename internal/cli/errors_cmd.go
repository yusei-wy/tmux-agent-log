package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/yusei-wy/tmux-agent-log/internal/errlog"
)

func init() {
	cmd := &cobra.Command{
		Use:   "errors",
		Short: "errors.jsonl を読み書きする",
	}
	cmd.AddCommand(errorsListCmd())
	cmd.AddCommand(errorsClearCmd())
	rootCmd.AddCommand(cmd)
}

func errorsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "errors を 1 行 JSON で出力",
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := errlog.Read()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			for _, e := range entries {
				body, err := json.Marshal(e)
				if err != nil {
					return err
				}
				out.Write(body)
				out.Write([]byte{'\n'})
			}
			return nil
		},
	}
}

func errorsClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "errors.jsonl を削除",
		RunE: func(cmd *cobra.Command, args []string) error {
			return errlog.Clear()
		},
	}
}
