package format_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yusei-wy/tmux-agent-log/internal/format"
)

func TestWrite(t *testing.T) {
	cases := []struct {
		name         string
		format       string
		columns      []string
		rows         [][]string
		want         string   // exact bytes; "" でスキップ
		wantContains []string // table format のように exact bytes が脆い場合の部分一致
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
			err := format.Write(buf, tc.format, tc.columns, tc.rows)
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
