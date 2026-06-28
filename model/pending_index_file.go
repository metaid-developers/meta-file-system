package model

import "time"

// PendingIndexFile records an on-chain index pin whose multi-chunk merge was
// deferred because not all of its chunk pins were indexed yet (the index pin
// was scanned before some chunk pins).
//
// It is created by IndexerService.processIndexContent's chunk-miss branch
// and retried by IndexerService.retryPendingIndexMerges (wired into
// onBlockComplete) once all chunks have arrived; on a successful merge the
// record is deleted. Survives restarts since it lives in PebbleDB.
//
// MetaData holds the marshalled indexer.MetaIDData (creator/owner/etc.) and
// IndexJSON the marshalled metafile index (sha256/fileSize/chunkList), both as
// raw JSON strings so package model does not import package indexer.
type PendingIndexFile struct {
	PinID       string `gorm:"uniqueIndex;type:varchar(255)" json:"pin_id"` // index pin ID (key)
	FirstPinID  string `gorm:"index;type:varchar(255)" json:"first_pin_id"`  // first pin ID for the file
	FirstPath   string `json:"first_path"`                                   // first pin path
	TxID        string `gorm:"index;type:varchar(64)" json:"tx_id"`          // transaction id of the index pin
	ChainName   string `gorm:"index;type:varchar(20)" json:"chain_name"`     // btc/mvc/doge
	BlockHeight int64  `gorm:"index" json:"block_height"`                    // block the index pin landed in
	Timestamp   int64  `json:"timestamp"`                                    // index pin timestamp
	MetaData    string `json:"meta_data"`                                    // marshalled indexer.MetaIDData
	IndexJSON   string `json:"index_json"`                                   // marshalled metafile index (chunkList)
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName specify table name (MySQL; indexer uses Pebble in production).
func (PendingIndexFile) TableName() string {
	return "tb_pending_index_file"
}
