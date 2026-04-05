package database

import "testing"

func TestExtractFileBaseName(t *testing.T) {
	cases := []struct {
		name     string
		fileName string
		want     string
	}{
		{name: "unicode", fileName: "周杰伦-夜曲.mp3", want: "周杰伦-夜曲"},
		{name: "multiple dots", fileName: "jay.live.2004.mp3", want: "jay.live.2004"},
		{name: "no extension", fileName: "周杰伦", want: "周杰伦"},
		{name: "empty", fileName: "", want: ""},
		{name: "double suffix", fileName: "archive.tar.gz", want: "archive.tar"},
		{name: "path input", fileName: "a/b/JayChou.Live.mp3", want: "JayChou.Live"},
		{name: "windows path", fileName: "a\\b\\JayChou.Live.mp3", want: "JayChou.Live"},
		{name: "trim whitespace", fileName: "  Jay.mp3  ", want: "Jay"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractFileBaseName(tc.fileName); got != tc.want {
				t.Fatalf("extractFileBaseName(%q) = %q, want %q", tc.fileName, got, tc.want)
			}
		})
	}
}

func TestFileBaseNameContainsKeyword(t *testing.T) {
	cases := []struct {
		name     string
		fileName string
		keyword  string
		want     bool
	}{
		{name: "unicode match", fileName: "周杰伦-夜曲.mp3", keyword: "周杰伦", want: true},
		{name: "case insensitive", fileName: "JayChou.Live.mp3", keyword: "live", want: true},
		{name: "trim keyword whitespace", fileName: "JayChou.Live.mp3", keyword: "  live  ", want: true},
		{name: "no extension still matches", fileName: "周杰伦", keyword: "周杰伦", want: true},
		{name: "empty file name", fileName: "", keyword: "周杰伦", want: false},
		{name: "blank keyword", fileName: "周杰伦-夜曲.mp3", keyword: "   ", want: false},
		{name: "miss", fileName: "周杰伦-夜曲.mp3", keyword: "林俊杰", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := fileBaseNameContainsKeyword(tc.fileName, tc.keyword); got != tc.want {
				t.Fatalf("fileBaseNameContainsKeyword(%q, %q) = %v, want %v", tc.fileName, tc.keyword, got, tc.want)
			}
		})
	}
}

func TestExtractTimestamp16FromCursorKey(t *testing.T) {
	cases := []struct {
		name string
		key  string
		want string
	}{
		{name: "extension key", key: ".mp3:0000000400123456", want: "0000000400123456"},
		{name: "global meta key", key: "globalMeta:.mp3:0000000400123456", want: "0000000400123456"},
		{name: "plain value", key: "0000000400123456", want: "0000000400123456"},
		{name: "empty", key: "", want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractTimestamp16FromCursorKey(tc.key); got != tc.want {
				t.Fatalf("extractTimestamp16FromCursorKey(%q) = %q, want %q", tc.key, got, tc.want)
			}
		})
	}
}
