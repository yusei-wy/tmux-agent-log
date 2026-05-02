package format_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yusei-wy/tmux-agent-log/internal/format"
)

func TestJSONIndentStruct(t *testing.T) {
	buf := &bytes.Buffer{}
	require.NoError(t, format.JSONIndent(buf, struct {
		Name string `json:"name"`
		N    int    `json:"n"`
	}{Name: "alice", N: 1}))
	require.Equal(t, "{\n  \"name\": \"alice\",\n  \"n\": 1\n}\n", buf.String())
}

func TestJSONIndentTrailingNewline(t *testing.T) {
	buf := &bytes.Buffer{}
	require.NoError(t, format.JSONIndent(buf, map[string]int{"a": 1}))
	require.True(t, bytes.HasSuffix(buf.Bytes(), []byte{'\n'}))
}

func TestJSONIndentMarshalError(t *testing.T) {
	buf := &bytes.Buffer{}
	// 関数値は encoding/json でエンコードできず、MarshalIndent がエラーを返す
	require.Error(t, format.JSONIndent(buf, func() {}))
}
