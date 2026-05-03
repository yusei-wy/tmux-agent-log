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

func TestReadInput(t *testing.T) {
	cases := []struct {
		name          string
		input         string
		wantSessionID string
		wantExtra     string
	}{
		{
			name:          "parses known fields",
			input:         `{"session_id":"abc","extra":"x"}`,
			wantSessionID: "abc",
			wantExtra:     "x",
		},
		{
			name:          "ignores unknown nested fields",
			input:         `{"session_id":"abc","new_field":{"nested":true}}`,
			wantSessionID: "abc",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var s sampleIn
			require.NoError(t, ReadInput(bytes.NewBufferString(tc.input), &s))
			require.Equal(t, tc.wantSessionID, s.SessionID)
			require.Equal(t, tc.wantExtra, s.Extra)
		})
	}
}

func TestRunWithRecoverReturnsZeroOnPanic(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	require.Equal(t, 0, RunWithRecover(func() error { panic("boom") }))
}

func TestRunWithRecoverReturnsZeroOnError(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	require.Equal(t, 0, RunWithRecover(func() error { return errors.New("bad") }))
}
