package format_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yusei-wy/tmux-agent-log/internal/format"
)

func TestTSV(t *testing.T) {
	buf := &bytes.Buffer{}
	require.NoError(t, format.Write(buf, "tsv", []string{"id", "name"}, [][]string{{"1", "alice"}, {"2", "bob"}}))
	require.Equal(t, "1\talice\n2\tbob\n", buf.String())
}

func TestTable(t *testing.T) {
	buf := &bytes.Buffer{}
	require.NoError(t, format.Write(buf, "table", []string{"id", "name"}, [][]string{{"1", "alice"}, {"22", "bob"}}))
	require.Contains(t, buf.String(), "id")
	require.Contains(t, buf.String(), "alice")
}

func TestJSONL(t *testing.T) {
	buf := &bytes.Buffer{}
	require.NoError(t, format.Write(buf, "jsonl", []string{"id", "name"}, [][]string{{"1", "alice"}, {"2", "bob"}}))
	require.Equal(t, "{\"id\":\"1\",\"name\":\"alice\"}\n{\"id\":\"2\",\"name\":\"bob\"}\n", buf.String())
}

func TestUnknownFormatErrors(t *testing.T) {
	err := format.Write(&bytes.Buffer{}, "xml", []string{"a"}, [][]string{{"b"}})
	require.Error(t, err)
}
