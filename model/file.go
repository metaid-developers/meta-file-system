package model

import "time"

type ChunkType string

const (
	ChunkTypeSingle ChunkType = "single"
	ChunkTypeMulti  ChunkType = "multi"
)

type Status string

const (
	StatusPending Status = "pending"
	StatusSuccess Status = "success"
	StatusFailed  Status = "failed"
)

// File file metadata model
type File struct {
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

	FileId string `gorm:"type:varchar(255)" json:"file_id"` // metaid_filehash

	FileName        string    `gorm:"type:varchar(255)" json:"file_name"`         // File name
	FileHash        string    `gorm:"type:varchar(255)" json:"file_hash"`         // File hash
	FileSize        int64     `json:"file_size"`                                  // File size
	FileType        string    `gorm:"type:varchar(20)" json:"file_type"`          // image/video/audio/document/other
	FileMd5         string    `gorm:"type:varchar(255)" json:"file_md5"`          // File MD5
	FileContentType string    `gorm:"type:varchar(100)" json:"file_content_type"` // File content type
	ChunkType       ChunkType `gorm:"type:varchar(20)" json:"chunk_type"`         // single/multi

	ContentHex string `gorm:"type:text" json:"content_hex"` // Content hexadecimal

	MetaId  string `gorm:"type:varchar(255)" json:"meta_id"` // MetaID
	Address string `gorm:"type:varchar(255)" json:"address"` //

	TxID        string `gorm:"uniqueIndex;type:varchar(64);not null" json:"tx_id"` // On-chain transaction ID
	PinId       string `gorm:"index;type:varchar(255);not null" json:"pin_id"`     // Pin ID
	Path        string `gorm:"index;type:varchar(255);not null" json:"path"`       // MetaID path
	ContentType string `gorm:"type:varchar(100)" json:"content_type"`              // Content type   - metafile/index
	StorageType string `gorm:"type:varchar(20)" json:"storage_type"`               // local/oss
	StoragePath string `gorm:"type:varchar(500)" json:"storage_path"`              // Storage path
	Operation   string `gorm:"type:varchar(20)" json:"operation"`                  // create/modify/revoke

	PreTxRaw string `gorm:"type:text" json:"pre_tx_raw"`    // Pre-transaction raw data
	TxRaw    string `gorm:"type:text" json:"tx_raw"`        // Transaction raw data
	Status   Status `gorm:"type:varchar(20)" json:"status"` // pending/success/failed

	BlockHeight int64     `gorm:"index" json:"block_height"`        // Block height
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"` // Creation time
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"` // Update time
	State       int64     `gorm:"type:int(11)" json:"state"`        // State 0:EXIST,2:DELETED
}

// TableName specify table name
func (File) TableName() string {
	return "tb_file"
}
