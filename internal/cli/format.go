package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"
)

func writeFormatted(w io.Writer, fmtName string, columns []string, rows [][]string) error {
	switch fmtName {
	case "tsv":
		return writeTSV(w, rows)
	case "table":
		return writeTable(w, columns, rows)
	case "jsonl":
		return writeJSONL(w, columns, rows)
	case "json":
		return writeJSON(w, columns, rows)
	default:
		return fmt.Errorf("unknown format: %q", fmtName)
	}
}

func writeTSV(w io.Writer, rows [][]string) error {
	for _, row := range rows {
		if _, err := fmt.Fprintln(w, strings.Join(row, "\t")); err != nil {
			return fmt.Errorf("write tsv row: %w", err)
		}
	}

	return nil
}

func writeTable(w io.Writer, columns []string, rows [][]string) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, strings.Join(columns, "\t")); err != nil {
		return fmt.Errorf("write table header: %w", err)
	}

	for _, row := range rows {
		if _, err := fmt.Fprintln(tw, strings.Join(row, "\t")); err != nil {
			return fmt.Errorf("write table row: %w", err)
		}
	}

	if err := tw.Flush(); err != nil {
		return fmt.Errorf("flush tabwriter: %w", err)
	}

	return nil
}

func writeJSONL(w io.Writer, columns []string, rows [][]string) error {
	for _, row := range rows {
		line, err := buildOrderedJSON(columns, row)
		if err != nil {
			return fmt.Errorf("build jsonl row: %w", err)
		}

		if _, err := w.Write(line); err != nil {
			return fmt.Errorf("write jsonl row: %w", err)
		}

		if _, err := w.Write([]byte{'\n'}); err != nil {
			return fmt.Errorf("write jsonl newline: %w", err)
		}
	}

	return nil
}

func writeJSON(w io.Writer, columns []string, rows [][]string) error {
	if _, err := w.Write([]byte{'['}); err != nil {
		return fmt.Errorf("write json open bracket: %w", err)
	}

	for i, row := range rows {
		if i > 0 {
			if _, err := w.Write([]byte{','}); err != nil {
				return fmt.Errorf("write json comma: %w", err)
			}
		}

		line, err := buildOrderedJSON(columns, row)
		if err != nil {
			return fmt.Errorf("build json row: %w", err)
		}

		if _, err := w.Write(line); err != nil {
			return fmt.Errorf("write json row: %w", err)
		}
	}

	if _, err := w.Write([]byte("]\n")); err != nil {
		return fmt.Errorf("write json close bracket: %w", err)
	}

	return nil
}

func buildOrderedJSON(columns []string, row []string) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')

	for i, col := range columns {
		if i > 0 {
			buf.WriteByte(',')
		}

		k, err := json.Marshal(col)
		if err != nil {
			return nil, fmt.Errorf("marshal column %q: %w", col, err)
		}

		buf.Write(k)
		buf.WriteByte(':')

		val := ""
		if i < len(row) {
			val = row[i]
		}

		v, err := json.Marshal(val)
		if err != nil {
			return nil, fmt.Errorf("marshal value for %q: %w", col, err)
		}

		buf.Write(v)
	}

	buf.WriteByte('}')

	return buf.Bytes(), nil
}

// writeJSONIndent は v を 2-space インデントの JSON として書き、末尾に改行を付ける。
func writeJSONIndent(w io.Writer, v any) error {
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

// formatTime は表示用に時刻を UTC の "YYYY-MM-DD HH:MM:SS" で整形する。zero 値は空文字。
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.UTC().Format("2006-01-02 15:04:05")
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}

	return formatTime(*t)
}
