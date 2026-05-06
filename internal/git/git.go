// Package git は git コマンドの実行と差分取得を薄くラップする。
package git

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type Error struct {
	Args     []string
	Stderr   string
	ExitCode int
}

func (e *Error) Error() string {
	return fmt.Sprintf("git %s exited %d: %s", strings.Join(e.Args, " "), e.ExitCode, strings.TrimSpace(e.Stderr))
}

func Run(dir string, args ...string) (string, error) {
	full := append([]string{"-C", dir}, args...)
	cmd := exec.Command("git", full...)

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout

	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		exitCode := -1

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}

		return "", &Error{Args: args, Stderr: stderr.String(), ExitCode: exitCode}
	}

	return strings.TrimRight(stdout.String(), "\n"), nil
}
