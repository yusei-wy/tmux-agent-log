package hook

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type sampleIn struct {
	SessionID string `json:"session_id"`
	Extra     string `json:"extra"`
}

func TestReadInputParsesJSON(t *testing.T) {
	var s sampleIn
	require.NoError(t, ReadInput(bytes.NewBufferString(`{"session_id":"abc","extra":"x"}`), &s))
	require.Equal(t, "abc", s.SessionID)
	require.Equal(t, "x", s.Extra)
}

func TestReadInputIgnoresUnknownFields(t *testing.T) {
	var s sampleIn
	require.NoError(t, ReadInput(bytes.NewBufferString(`{"session_id":"abc","new_field":{"nested":true}}`), &s))
	require.Equal(t, "abc", s.SessionID)
}

func TestRunWithRecoverReturnsZeroOnPanic(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	require.Equal(t, 0, RunWithRecover(func() error { panic("boom") }))
}

func TestRunWithRecoverReturnsZeroOnError(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	require.Equal(t, 0, RunWithRecover(func() error { return errors.New("bad") }))
}
