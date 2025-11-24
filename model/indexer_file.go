package model

import "time"

// IndexerFile indexer file metadata model (for indexer service)
type IndexerFile struct {
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// MetaID related fields
	PinID       string    `gorm:"uniqueIndex;type:varchar(255);not null" json:"pin_id"` // PIN ID (txid + i + vout)
	TxID        string    `gorm:"index;type:varchar(64);not null" json:"tx_id"`         // Transaction ID
	Vout        uint32    `gorm:"type:int" json:"vout"`                                 // Output index
	Path        string    `gorm:"index;type:varchar(500);not null" json:"path"`         // MetaID path
	Operation   string    `gorm:"type:varchar(20)" json:"operation"`                    // create/modify/revoke
	ParentPath  string    `gorm:"type:varchar(500)" json:"parent_path"`                 // Parent path
	Encryption  string    `gorm:"type:varchar(50)" json:"encryption"`                   // Encryption method
	Version     string    `gorm:"type:varchar(50)" json:"version"`                      // Version
	ContentType string    `gorm:"type:varchar(100)" json:"content_type"`                // Content type - metafile/index
	Data        string    `gorm:"type:text" json:"data"`                                // Data - for metafile/index
	ChunkType   ChunkType `gorm:"type:varchar(20)" json:"chunk_type"`                   // single/multi

	// File related fields
	FileType         string `gorm:"type:varchar(20)" json:"file_type"`          // File type (image/video/audio/document/other)
	FileExtension    string `gorm:"type:varchar(10)" json:"file_extension"`      // File extension, e.g. .jpg, .png, .mp4, .mp3, .doc, .pdf, etc.
	FileName         string `gorm:"type:varchar(255)" json:"file_name"`          // File name (extracted from path)
	FileSize         int64  `json:"file_size"`                                   // File size
	FileMd5          string `gorm:"type:varchar(64)" json:"file_md5"`            // File MD5
	FileHash         string `gorm:"type:varchar(64)" json:"file_hash"`           // File Hash SHA256
	IsGzipCompressed bool   `gorm:"type:tinyint(1);default:0" json:"is_gzip_compressed"` // Whether the original content was gzip compressed

	// Storage related fields
	StorageType string `gorm:"type:varchar(20)" json:"storage_type"`  // local/oss
	StoragePath string `gorm:"type:varchar(500)" json:"storage_path"` // Storage path

	// Blockchain related fields
	ChainName      string `gorm:"type:varchar(20);not null" json:"chain_name"`    // btc/mvc
	BlockHeight    int64  `gorm:"index" json:"block_height"`                      // Block height
	Timestamp      int64  `gorm:"index" json:"timestamp"`                         // Timestamp (seconds since epoch)
	CreatorMetaId  string `gorm:"index;type:varchar(64)" json:"creator_meta_id"`  // Creator MetaID (SHA256 hash)
	CreatorAddress string `gorm:"index;type:varchar(100)" json:"creator_address"` // Creator address
	OwnerAddress   string `gorm:"index;type:varchar(100)" json:"owner_address"`   // Owner address (current)
	OwnerMetaId    string `gorm:"index;type:varchar(64)" json:"owner_meta_id"`    // Owner MetaID (SHA256 hash)

	// Status fields
	Status Status `gorm:"type:varchar(20);default:'success'" json:"status"` // success/failed

	// Timestamps
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`    // Creation time
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`    // Update time
	State     int64     `gorm:"type:int(11);default:0" json:"state"` // State 0:EXIST,2:DELETED
}

// TableName specify table name
func (IndexerFile) TableName() string {
	return "tb_indexer_file"
}
