package model

import "time"

// FileChunk file chunk metadata model
type FileChunk struct {
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

	ChunkHash  string `gorm:"type:varchar(255)" json:"chunk_hash"` // Chunk hash
	ChunkSize  int64  `json:"chunk_size"`                          // Chunk size
	ChunkMd5   string `gorm:"type:varchar(255)" json:"chunk_md5"`  // Chunk MD5
	ChunkIndex int64  `json:"chunk_index"`                         // Chunk index
	FileHash   string `gorm:"type:varchar(255)" json:"file_hash"`  // File hash

	ContentHex string `gorm:"type:text" json:"content_hex"` // Content hexadecimal

	TxID        string `gorm:"uniqueIndex;type:varchar(64);not null" json:"tx_id"` // On-chain transaction ID
	PinId       string `gorm:"index;type:varchar(255);not null" json:"pin_id"`     // Pin ID
	Path        string `gorm:"index;type:varchar(255);not null" json:"path"`       // MetaID path
	ContentType string `gorm:"type:varchar(100)" json:"content_type"`              // Content type - metafile/chunk
	Size        int64  `json:"size"`                                               // File size
	StorageType string `gorm:"type:varchar(20)" json:"storage_type"`               // local/oss
	StoragePath string `gorm:"type:varchar(500)" json:"storage_path"`              // Storage path
	Operation   string `gorm:"type:varchar(20)" json:"operation"`                  // create/modify/revoke

	TxRaw  string `gorm:"type:text" json:"tx_raw"`        // Transaction raw data
	Status Status `gorm:"type:varchar(20)" json:"status"` // pending/success/failed

	BlockHeight int64     `gorm:"index" json:"block_height"`        // Block height
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"` // Creation time
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"` // Update time
	State       int64     `gorm:"type:int(11)" json:"state"`        // State 0:EXIST,1:DELETED
}

// TableName specify table name
func (FileChunk) TableName() string {
	return "tb_file_chunk"
}
