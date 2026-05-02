package format

import "time"

// Time は表示用に時刻を UTC の "YYYY-MM-DD HH:MM:SS" で整形する。zero 値は空文字。
func Time(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02 15:04:05")
}
