package format_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yusei-wy/tmux-agent-log/internal/format"
)

// TestJSONIndent は format.JSONIndent が 2-space インデント + 末尾改行という
// 自前 policy で出力することをロックする（library 動作そのものは検証しない）。
func TestJSONIndent(t *testing.T) {
	buf := &bytes.Buffer{}
	require.NoError(t, format.JSONIndent(buf, struct {
		Name string `json:"name"`
		N    int    `json:"n"`
	}{Name: "alice", N: 1}))
	require.Equal(t, "{\n  \"name\": \"alice\",\n  \"n\": 1\n}\n", buf.String())
}
