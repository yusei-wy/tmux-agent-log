package format

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

func Write(w io.Writer, fmtName string, columns []string, rows [][]string) error {
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
			return err
		}
	}
	return nil
}

func writeTable(w io.Writer, columns []string, rows [][]string) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, strings.Join(columns, "\t")); err != nil {
		return err
	}
	for _, row := range rows {
		if _, err := fmt.Fprintln(tw, strings.Join(row, "\t")); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func writeJSONL(w io.Writer, columns []string, rows [][]string) error {
	for _, row := range rows {
		line, err := buildOrderedJSON(columns, row)
		if err != nil {
			return err
		}
		if _, err := w.Write(line); err != nil {
			return err
		}
		if _, err := w.Write([]byte{'\n'}); err != nil {
			return err
		}
	}
	return nil
}

func writeJSON(w io.Writer, columns []string, rows [][]string) error {
	if _, err := w.Write([]byte{'['}); err != nil {
		return err
	}
	for i, row := range rows {
		if i > 0 {
			if _, err := w.Write([]byte{','}); err != nil {
				return err
			}
		}
		line, err := buildOrderedJSON(columns, row)
		if err != nil {
			return err
		}
		if _, err := w.Write(line); err != nil {
			return err
		}
	}
	if _, err := w.Write([]byte("]\n")); err != nil {
		return err
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
			return nil, err
		}
		buf.Write(k)
		buf.WriteByte(':')
		val := ""
		if i < len(row) {
			val = row[i]
		}
		v, err := json.Marshal(val)
		if err != nil {
			return nil, err
		}
		buf.Write(v)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}
