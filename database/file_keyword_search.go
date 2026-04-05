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

ext := filepath.Ext(trimmed)
	if ext == "" {
		return trimmed
	}

	return strings.TrimSuffix(trimmed, ext)
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

func extractTimestamp16FromCursorKey(key string) string {
	if key == "" {
		return ""
	}

	slash := strings.LastIndex(key, ":")
	if slash < 0 {
		return key
	}

	return key[slash+1:]
}
