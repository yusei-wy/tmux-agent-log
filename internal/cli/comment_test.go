package cli_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yusei-wy/tmux-agent-log/internal/cli"
)

func TestParseCommentLineRange(t *testing.T) {
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
			s, e, err := cli.ParseCommentLineRange(tc.in)
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
