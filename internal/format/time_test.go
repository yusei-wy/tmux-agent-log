package format_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/yusei-wy/tmux-agent-log/internal/format"
)

func TestTime(t *testing.T) {
	jst := time.FixedZone("JST", 9*60*60)

	cases := []struct {
		name string
		in   time.Time
		want string
	}{
		{
			name: "zero value returns empty string",
			in:   time.Time{},
			want: "",
		},
		{
			name: "utc input is formatted as-is",
			in:   time.Date(2026, 4, 30, 12, 34, 56, 0, time.UTC),
			want: "2026-04-30 12:34:56",
		},
		{
			name: "non-utc input is converted to utc",
			in:   time.Date(2026, 4, 30, 21, 34, 56, 0, jst),
			want: "2026-04-30 12:34:56",
		},
		{
			name: "sub-second precision is dropped",
			in:   time.Date(2026, 4, 30, 12, 34, 56, 999_999_999, time.UTC),
			want: "2026-04-30 12:34:56",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, format.Time(tc.in))
		})
	}
}
