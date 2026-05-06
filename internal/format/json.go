package format

import (
	"encoding/json"
	"fmt"
	"io"
)

// JSONIndent は v を 2-space インデントの JSON として書き、末尾に改行を付ける。
func JSONIndent(w io.Writer, v any) error {
	body, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("write json: %w", err)
	}

	if _, err := w.Write([]byte{'\n'}); err != nil {
		return fmt.Errorf("write json newline: %w", err)
	}

	return nil
}
