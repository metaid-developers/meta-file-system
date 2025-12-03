package model

import "time"

// IndexerFileChunk indexer file chunk metadata model (for multi-chunk files)
type IndexerFileChunk struct {
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// MetaID related fields
	PinID       string `gorm:"uniqueIndex;type:varchar(255);not null" json:"pin_id"` // PIN ID (txid + i + vout)
	TxID        string `gorm:"index;type:varchar(64);not null" json:"tx_id"`         // Transaction ID
	Vout        uint32 `gorm:"type:int" json:"vout"`                                 // Output index
	Path        string `gorm:"index;type:varchar(500);not null" json:"path"`         // MetaID path
	Operation   string `gorm:"type:varchar(20)" json:"operation"`                    // create/modify/revoke
	ContentType string `gorm:"type:varchar(100)" json:"content_type"`                // Content type - metafile/chunk

	// Chunk related fields
	ChunkIndex       int    `gorm:"type:int" json:"chunk_index"`                         // Chunk index (0-based)
	ChunkSize        int64  `json:"chunk_size"`                                          // Chunk size
	ChunkMd5         string `gorm:"type:varchar(64)" json:"chunk_md5"`                   // Chunk MD5
	ParentFirstPinID string `gorm:"index;type:varchar(255)" json:"parent_first_pin_id"`  // Parent file First PIN ID
	ParentPinID      string `gorm:"index;type:varchar(255)" json:"parent_pin_id"`        // Parent file PIN ID
	IsGzipCompressed bool   `gorm:"type:tinyint(1);default:0" json:"is_gzip_compressed"` // Whether the original content was gzip compressed

	// Storage related fields
	StorageType string `gorm:"type:varchar(20)" json:"storage_type"`  // local/oss
	StoragePath string `gorm:"type:varchar(500)" json:"storage_path"` // Storage path

	// Blockchain related fields
	ChainName   string `gorm:"type:varchar(20);not null" json:"chain_name"` // btc/mvc
	BlockHeight int64  `gorm:"index" json:"block_height"`                   // Block height

	// Status fields
	Status Status `gorm:"type:varchar(20);default:'success'" json:"status"` // success/failed

	// Timestamps
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`    // Creation time
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`    // Update time
	State     int64     `gorm:"type:int(11);default:0" json:"state"` // State 0:EXIST,2:DELETED
}

// TableName specify table name
func (IndexerFileChunk) TableName() string {
	return "tb_indexer_file_chunk"
}
