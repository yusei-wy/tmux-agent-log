package hook

import "strings"

// truncate は s が maxLen を超える場合だけ末尾に "…" を付けて切り詰める。
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	return s[:maxLen] + "…"
}

// promptPreview は先頭 maxLines 行を保持しつつ maxLen 文字で切り詰めた preview を返す。
func promptPreview(s string, maxLines, maxLen int) string {
	parts := strings.SplitN(s, "\n", maxLines+1)
	if len(parts) > maxLines {
		parts = parts[:maxLines]
	}

	return truncate(strings.Join(parts, "\n"), maxLen)
}
