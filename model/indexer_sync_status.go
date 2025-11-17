package model

import "time"

// IndexerSyncStatus indexer synchronization status model
type IndexerSyncStatus struct {
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// Chain information
	ChainName string `gorm:"uniqueIndex;type:varchar(20);not null" json:"chain_name"` // btc/mvc

	// Sync status
	CurrentSyncHeight int64 `gorm:"type:bigint;not null;default:0" json:"current_sync_height"` // Current scanned block height

	// Timestamps
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"` // Creation time
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"` // Update time
}

// TableName specify table name
func (IndexerSyncStatus) TableName() string {
	return "tb_indexer_sync_status"
}
