package git

import (
	"errors"
	"strings"
)

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
