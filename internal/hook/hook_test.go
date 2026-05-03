package hook

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunWithRecoverReturnsZeroOnPanic(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	require.Equal(t, 0, RunWithRecover(func() error { panic("boom") }))
}

func TestRunWithRecoverReturnsZeroOnError(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	require.Equal(t, 0, RunWithRecover(func() error { return errors.New("bad") }))
}
