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
	MonitorFileChunk = "_chunk"
	MonitorFileIndex = "index"

	MonitorMetaIdFileIndexContentType = "metafile/index"
	MonitorMetaIdFileChunkContentType = "metafile/chunk"
)

var (
	ProtocolList = []string{
		fmt.Sprintf("/file/%s", strings.ToLower(MonitorFileChunk)),
		fmt.Sprintf("/file/%s", strings.ToLower(MonitorFileIndex)),
	}
)
