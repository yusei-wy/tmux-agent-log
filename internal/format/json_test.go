package format_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yusei-wy/tmux-agent-log/internal/format"
)

func TestJSONIndent(t *testing.T) {
	cases := []struct {
		name    string
		in      any
		want    string
		wantErr bool
	}{
		{
			name: "struct is indented with 2 spaces and trailing newline",
			in: struct {
				Name string `json:"name"`
				N    int    `json:"n"`
			}{Name: "alice", N: 1},
			want: "{\n  \"name\": \"alice\",\n  \"n\": 1\n}\n",
		},
		{
			name: "map gets trailing newline",
			in:   map[string]int{"a": 1},
			want: "{\n  \"a\": 1\n}\n",
		},
		{
			name: "nil becomes literal null",
			in:   nil,
			want: "null\n",
		},
		{
			name:    "func value is unmarshalable and returns error",
			in:      func() {},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			err := format.JSONIndent(buf, tc.in)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, buf.String())
		})
	}
}
