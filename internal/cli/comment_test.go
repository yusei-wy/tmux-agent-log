package cli

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/yusei-wy/tmux-agent-log/internal/storage"
)

func TestParseLineRange(t *testing.T) {
	cases := []struct {
		in      string
		start   int
		end     int
		wantErr bool
	}{
		{"44", 44, 44, false},
		{"44-46", 44, 46, false},
		{"46-44", 0, 0, true},
		{"abc", 0, 0, true},
	}
	for _, c := range cases {
		s, e, err := parseLineRange(c.in)
		if c.wantErr {
			require.Error(t, err, c.in)
			continue
		}
		require.NoError(t, err, c.in)
		require.Equal(t, c.start, s)
		require.Equal(t, c.end, e)
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
