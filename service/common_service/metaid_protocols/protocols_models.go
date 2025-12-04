package metaid_protocols

import (
	"fmt"
	"strings"
)

/*
*

	{
		"sha256": "",//总文件的hash
		"fileSize": 202400,
		"chunkNumber": 2,
		"chunkSize": 102400,
		"dataType": "text/plain",
		"name": "",
		"chunkList": [
			{
				"sha256": "",//分片的hash
				"pinId": "",//分片的pinId
			},
			{
				"sha256": "",//分片的hash
				"pinId": "",//分片的pinId
			}
		]
	}

*
*/
type MetaFileIndex struct {
	Sha256      string `json:"sha256"`
	FileSize    int64  `json:"fileSize"`
	ChunkNumber int    `json:"chunkNumber"`
	ChunkSize   int64  `json:"chunkSize"`
	DataType    string `json:"dataType"`
	Name        string `json:"name"`
	ChunkList   []struct {
		Sha256 string `json:"sha256"`
		PinId  string `json:"pinId"`
	} `json:"chunkList"`
}

// /file
const (
	MonitorFileChunk    = "_chunk"
	MonitorFileChunkOld = "chunk"
	MonitorFileIndex    = "index"

	MonitorMetaIdFileIndexContentType = "metafile/index"
	MonitorMetaIdFileChunkContentType = "metafile/chunk"

	MonitorMetaIdInfoNameContentType          = "name"
	MonitorMetaIdInfoAvatarContentType        = "avatar"
	MonitorMetaIdInfoChatPublicKeyContentType = "chatpubkey"
)

var (
	ProtocolList = []string{
		fmt.Sprintf("/file/%s", strings.ToLower(MonitorFileChunk)),
		fmt.Sprintf("/file/%s", strings.ToLower(MonitorFileChunkOld)),
		fmt.Sprintf("/file/%s", strings.ToLower(MonitorFileIndex)),
		"/file",

		fmt.Sprintf("/info/%s", strings.ToLower(MonitorMetaIdInfoNameContentType)),
		fmt.Sprintf("/info/%s", strings.ToLower(MonitorMetaIdInfoAvatarContentType)),
		fmt.Sprintf("/info/%s", strings.ToLower(MonitorMetaIdInfoChatPublicKeyContentType)),
	}
)

// IsProtocolPath checks if the given path is in the ProtocolList
func IsProtocolPath(path string) bool {
	if path == "" {
		return false
	}

	// Normalize path to lowercase for case-insensitive comparison
	normalizedPath := strings.ToLower(path)

	for _, protocol := range ProtocolList {
		if normalizedPath == protocol {
			return true
		}
		// Also check if path starts with the protocol (e.g., "/file/chunk/123" matches "/file/chunk")
		if strings.HasPrefix(normalizedPath, protocol+"/") || strings.HasPrefix(normalizedPath, protocol) {
			return true
		}
	}

	return false
}
