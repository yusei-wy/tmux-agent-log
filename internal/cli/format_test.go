package cli_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/yusei-wy/tmux-agent-log/internal/cli"
)

func TestWriteJSONIndent(t *testing.T) {
	buf := &bytes.Buffer{}
	require.NoError(t, cli.WriteJSONIndent(buf, struct {
		Name string `json:"name"`
		N    int    `json:"n"`
	}{Name: "alice", N: 1}))
	require.Equal(t, "{\n  \"name\": \"alice\",\n  \"n\": 1\n}\n", buf.String())
}

func TestFormatTime(t *testing.T) {
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
			require.Equal(t, tc.want, cli.FormatTime(tc.in))
		})
	}
}

func TestWriteFormatted(t *testing.T) {
	cases := []struct {
		name         string
		format       string
		columns      []string
		rows         [][]string
		want         string
		wantContains []string
		wantErr      bool
	}{
		{
			name:    "tsv writes tab-separated rows without header",
			format:  "tsv",
			columns: []string{"id", "name"},
			rows:    [][]string{{"1", "alice"}, {"2", "bob"}},
			want:    "1\talice\n2\tbob\n",
		},
		{
			name:         "table writes header and aligned rows",
			format:       "table",
			columns:      []string{"id", "name"},
			rows:         [][]string{{"1", "alice"}, {"22", "bob"}},
			wantContains: []string{"id", "alice"},
		},
		{
			name:    "jsonl writes one json object per line",
			format:  "jsonl",
			columns: []string{"id", "name"},
			rows:    [][]string{{"1", "alice"}, {"2", "bob"}},
			want:    "{\"id\":\"1\",\"name\":\"alice\"}\n{\"id\":\"2\",\"name\":\"bob\"}\n",
		},
		{
			name:    "json writes single array",
			format:  "json",
			columns: []string{"id", "name"},
			rows:    [][]string{{"1", "alice"}, {"2", "bob"}},
			want:    "[{\"id\":\"1\",\"name\":\"alice\"},{\"id\":\"2\",\"name\":\"bob\"}]\n",
		},
		{
			name:    "unknown format returns error",
			format:  "xml",
			columns: []string{"a"},
			rows:    [][]string{{"b"}},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buf := &bytes.Buffer{}

			err := cli.WriteFormatted(buf, tc.format, tc.columns, tc.rows)
			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tc.want != "" {
				require.Equal(t, tc.want, buf.String())
			}

			for _, s := range tc.wantContains {
				require.Contains(t, buf.String(), s)
			}
		})
	}
}
