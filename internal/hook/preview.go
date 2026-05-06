package hook

import "strings"

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}

	return string(runes[:maxLen]) + "…"
}

func promptPreview(s string, maxLines, maxLen int) string {
	parts := strings.SplitN(s, "\n", maxLines+1)
	if len(parts) > maxLines {
		parts = parts[:maxLines]
	}

	return truncate(strings.Join(parts, "\n"), maxLen)
}
