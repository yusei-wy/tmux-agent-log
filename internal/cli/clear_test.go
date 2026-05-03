package cli_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/yusei-wy/tmux-agent-log/internal/cli"
)

func TestParseDuration(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    time.Duration
		wantErr bool
	}{
		{name: "days suffix", in: "7d", want: 7 * 24 * time.Hour},
		{name: "stdlib duration", in: "24h", want: 24 * time.Hour},
		{name: "garbage input returns error", in: "abc", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := cli.ParseDuration(tc.in)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
