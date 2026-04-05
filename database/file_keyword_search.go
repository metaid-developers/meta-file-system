package database

import (
	"path/filepath"
	"strings"
)

func extractFileBaseName(fileName string) string {
	trimmed := strings.TrimSpace(fileName)
	if trimmed == "" {
		return ""
	}

	trimmed = strings.TrimSpace(stripPathSegments(trimmed))

	ext := filepath.Ext(trimmed)
	if ext == "" {
		return trimmed
	}

	return strings.TrimSuffix(trimmed, ext)
}

func stripPathSegments(name string) string {
	idx := strings.LastIndex(name, "/")
	if backslash := strings.LastIndex(name, "\\"); backslash > idx {
		idx = backslash
	}
	if idx >= 0 {
		return name[idx+1:]
	}
	return name
}

func fileBaseNameContainsKeyword(fileName, keyword string) bool {
	baseName := extractFileBaseName(fileName)
	if baseName == "" {
		return false
	}

	trimmedKeyword := strings.TrimSpace(keyword)
	if trimmedKeyword == "" {
		return false
	}

	lowerBase := strings.ToLower(baseName)
	lowerKeyword := strings.ToLower(trimmedKeyword)
	return strings.Contains(lowerBase, lowerKeyword)
}

// extractTimestamp16FromCursorKey returns everything after the final colon without validating the timestamp format.
func extractTimestamp16FromCursorKey(key string) string {
	if key == "" {
		return ""
	}

	colonIdx := strings.LastIndex(key, ":")
	if colonIdx < 0 {
		return key
	}

	return key[colonIdx+1:]
}
