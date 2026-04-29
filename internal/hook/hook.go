package hook

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/yusei-wy/tmux-agent-log/internal/errlog"
)

func ReadInput(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}

func RunWithRecover(fn func() error) int {
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("%v", r)
			_ = errlog.Record("hook", "panic", "", msg)
			fmt.Fprintln(os.Stderr, "tmux-agent-log: hook panic:", msg)
		}
	}()

	if err := fn(); err != nil {
		_ = errlog.Record("hook", "error", "", err.Error())
		fmt.Fprintln(os.Stderr, "tmux-agent-log: hook error:", err)
	}
	return 0
}
