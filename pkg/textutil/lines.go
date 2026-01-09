package textutil

import "strings"

func DetectLineEnding(content string) string {
	if strings.Contains(content, "\r\n") {
		return "\r\n"
	}
	return "\n"
}

func SplitLines(content string) []string {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	return strings.Split(normalized, "\n")
}

func JoinLines(lines []string, lineEnding string) string {
	return strings.Join(lines, lineEnding)
}
