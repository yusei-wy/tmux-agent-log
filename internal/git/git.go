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

func IsRepo(dir string) (bool, error) {
	out, err := Run(dir, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		var ge *Error
		if errors.As(err, &ge) {
			return false, nil
		}

		return false, err
	}

	return strings.TrimSpace(out) == "true", nil
}

func HeadSHA(dir string) (string, error) {
	out, err := Run(dir, "rev-parse", "HEAD")
	if err != nil {
		var ge *Error
		if errors.As(err, &ge) {
			if strings.Contains(ge.Stderr, "unknown revision") || strings.Contains(ge.Stderr, "ambiguous argument") {
				return "", nil
			}

			return "", err
		}

		return "", err
	}

	return strings.TrimSpace(out), nil
}

const emptyTreeHash = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

func DiffSince(dir, base string) (string, error) {
	if base == "" {
		base = emptyTreeHash
	}
	// NOTE: 未追跡ファイルを diff に載せるため intent-to-add で擬似的にステージする。
	// 大きなリポジトリで `git add -N -- .` の stat ウォーク代を避けるため、
	// untracked が存在するときだけ実行する。
	if untracked, err := Run(dir, "ls-files", "--others", "--exclude-standard"); err == nil && untracked != "" {
		_, _ = Run(dir, "add", "-N", "--", ".")
	}

	return Run(dir, "diff", "--no-color", "-U3", base, "--")
}
