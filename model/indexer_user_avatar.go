package model

import "time"

// IndexerUserAvatar indexer user avatar model
type IndexerUserAvatar struct {
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// PIN information
	PinID string `gorm:"uniqueIndex;type:varchar(255)" json:"pin_id"` // PIN ID (unique identifier)
	TxID  string `gorm:"index;type:varchar(100)" json:"tx_id"`        // Transaction ID

	// MetaID information
	MetaId  string `gorm:"index;type:varchar(100)" json:"meta_id"` // Meta ID (SHA256 of address)
	Address string `gorm:"index;type:varchar(100)" json:"address"` // Address

	// Avatar information
	Avatar        string `gorm:"type:varchar(500)" json:"avatar"`        // Avatar storage path or URL
	ContentType   string `gorm:"type:varchar(100)" json:"content_type"`  // Content type (e.g., image/jpeg)
	FileSize      int64  `gorm:"type:bigint" json:"file_size"`           // File size (bytes)
	FileMd5       string `gorm:"type:varchar(64)" json:"file_md5"`       // File MD5 hash
	FileHash      string `gorm:"type:varchar(64)" json:"file_hash"`      // File Hash SHA256
	FileExtension string `gorm:"type:varchar(10)" json:"file_extension"` // File extension, e.g. .jpg, .png, .mp4, .mp3, .doc, .pdf, etc.
	FileType      string `gorm:"type:varchar(20)" json:"file_type"`      // File type (image/video/audio/document/other)

	// Chain information
	ChainName   string `gorm:"index;type:varchar(20)" json:"chain_name"` // Chain name: btc/mvc
	BlockHeight int64  `gorm:"index;type:bigint" json:"block_height"`    // Block height
	Timestamp   int64  `gorm:"index;type:bigint" json:"timestamp"`       // Timestamp (seconds since epoch)

	// Timestamps
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"` // Creation time
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"` // Update time
}

// TableName specify table name
func (IndexerUserAvatar) TableName() string {
	return "tb_indexer_user_avatar"
}
