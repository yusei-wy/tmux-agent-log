package hook

import "strings"

// truncate は s が maxLen を超える場合だけ末尾に "…" を付けて切り詰める。
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "…"
}

// previewFirstLines は先頭 2 行までを保持しつつ maxLen で切り詰めた preview を返す。
// turn-start の user prompt を JSONL に短く残す用途。
func previewFirstLines(s string, maxLen int) string {
	parts := strings.SplitN(s, "\n", 3)
	if len(parts) > 2 {
		parts = parts[:2]
	}
	return truncate(strings.Join(parts, "\n"), maxLen)
}
