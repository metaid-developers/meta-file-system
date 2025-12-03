package model

// IndexerPinInfo PIN 信息模型，用于存储 PIN 的基本信息
type IndexerPinInfo struct {
	PinID       string `json:"pinId"`       // PIN ID
	FirstPinID  string `json:"firstPinId"`  // 第一个 PIN ID
	FirstPath   string `json:"firstPath"`   // 第一个 PIN 的路径
	Path        string `json:"path"`        // 路径
	Operation   string `json:"operation"`   // 操作类型 (create/modify/revoke)
	ContentType string `json:"contentType"` // 内容类型
	ChainName   string `json:"chainName"`   // 链名称
	BlockHeight int64  `json:"blockHeight"` // 区块高度
	Timestamp   int64  `json:"timestamp"`   // 时间戳
}
