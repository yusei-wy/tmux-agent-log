package format_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yusei-wy/tmux-agent-log/internal/format"
)

func TestTimeZero(t *testing.T) {
	require.Equal(t, "", format.Time(time.Time{}))
}

func TestTimeUTC(t *testing.T) {
	in := time.Date(2026, 4, 30, 12, 34, 56, 0, time.UTC)
	require.Equal(t, "2026-04-30 12:34:56", format.Time(in))
}

func TestTimeConvertsToUTC(t *testing.T) {
	jst := time.FixedZone("JST", 9*60*60)
	in := time.Date(2026, 4, 30, 21, 34, 56, 0, jst)
	require.Equal(t, "2026-04-30 12:34:56", format.Time(in))
}
