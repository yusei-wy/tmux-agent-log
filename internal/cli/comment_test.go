package cli

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/yusei-wy/tmux-agent-log/internal/storage"
)

func TestParseLineRange(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		start   int
		end     int
		wantErr bool
	}{
		{name: "single line", in: "44", start: 44, end: 44},
		{name: "range", in: "44-46", start: 44, end: 46},
		{name: "end before start returns error", in: "46-44", wantErr: true},
		{name: "non-numeric returns error", in: "abc", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, e, err := parseLineRange(tc.in)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.start, s)
			require.Equal(t, tc.end, e)
		})
	}
}

func TestRenderSendPromptContainsAllEntries(t *testing.T) {
	cs := []storage.Comment{
		{File: "a.go", LineStart: 10, LineEnd: 12, Text: "foo", CreatedAt: time.Unix(1, 0).UTC()},
		{File: "b.go", LineStart: 30, LineEnd: 30, Text: "bar", CreatedAt: time.Unix(2, 0).UTC()},
	}
	out := renderSendPrompt(cs)
	require.Contains(t, out, "a.go:10-12")
	require.Contains(t, out, "foo")
	require.Contains(t, out, "b.go:30-30")
	require.Contains(t, out, "bar")
}
