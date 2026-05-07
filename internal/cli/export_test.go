package cli

import (
	"bytes"
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	ParseCommentLineRange = parseCommentLineRange
	ParseDurationWithDays = parseDurationWithDays
	InstallHooksTo        = installHooksTo
	UninstallHooksFrom    = uninstallHooksFrom
	ResolveSettingsPath   = resolveSettingsPath
	WriteFormatted        = writeFormatted
	WriteJSONIndent       = writeJSONIndent
	FormatTime            = formatTime
	FormatTimePtr         = formatTimePtr
	FindSessionDir        = findSessionDir
)

// RunCLI は rootCmd に args を渡して実行し stdout 出力を返す。
func RunCLI(args ...string) (string, error) {
	resetCmdFlags(rootCmd)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()

	rootCmd.SetOut(nil)
	rootCmd.SetErr(nil)
	rootCmd.SetArgs(nil)

	return buf.String(), err
}

func resetCmdFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})

	for _, sub := range cmd.Commands() {
		resetCmdFlags(sub)
	}
}
