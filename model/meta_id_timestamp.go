package model

// MetaIdTimestamp MetaID 和时间戳的映射
type MetaIdTimestamp struct {
	MetaId    string `json:"metaId"`    // MetaID
	Timestamp int64  `json:"timestamp"` // 时间戳（记录最早的时间戳）
}
