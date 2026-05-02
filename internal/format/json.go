package format

import (
	"encoding/json"
	"io"
)

// JSONIndent は v を 2-space インデントの JSON として書き、末尾に改行を付ける。
func JSONIndent(w io.Writer, v any) error {
	body, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if _, err := w.Write(body); err != nil {
		return err
	}
	_, err = w.Write([]byte{'\n'})
	return err
}
