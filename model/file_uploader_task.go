package model

import "time"

// TaskStage indicates resumable stage for async chunk task.
type TaskStage string

const (
	TaskStageCreated          TaskStage = "created"           // Task initial state
	TaskStagePrepared         TaskStage = "prepared"          // Transactions prepared, not broadcast yet
	TaskStageMergeBroadcast   TaskStage = "merge_broadcast"   // Merge transaction broadcasted
	TaskStageFundingBroadcast TaskStage = "funding_broadcast" // Funding transaction broadcasted
	TaskStageChunkBroadcast   TaskStage = "chunk_broadcast"   // Chunk transactions broadcasted
	TaskStageIndexBroadcast   TaskStage = "index_broadcast"   // Index transaction broadcasted
	TaskStageCompleted        TaskStage = "completed"         // Finished
)

// FileUploaderTask represents an async chunk upload task
type FileUploaderTask struct {
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// Task identifier
	TaskId string `gorm:"uniqueIndex;type:varchar(255)" json:"task_id"` // Unique task ID

	// File info
	MetaId        string `gorm:"type:varchar(255)" json:"meta_id"`      // MetaID
	Address       string `gorm:"type:varchar(255)" json:"address"`      // Uploader address
	FileName      string `gorm:"type:varchar(255)" json:"file_name"`    // File name
	FileHash      string `gorm:"type:varchar(255)" json:"file_hash"`    // File SHA256 hash
	FileMd5       string `gorm:"type:varchar(255)" json:"file_md5"`     // File MD5
	FileSize      int64  `json:"file_size"`                             // File size
	ContentType   string `gorm:"type:varchar(100)" json:"content_type"` // MIME type
	Path          string `gorm:"type:varchar(255)" json:"path"`         // MetaID path
	Operation     string `gorm:"type:varchar(20)" json:"operation"`     // create/update
	ContentBase64 string `gorm:"type:longtext" json:"content_base64"`   // File content (base64)

	// Transaction info
	ChunkPreTxHex string `gorm:"type:text" json:"chunk_pre_tx_hex"` // Pre-built chunk tx
	IndexPreTxHex string `gorm:"type:text" json:"index_pre_tx_hex"` // Pre-built index tx
	MergeTxHex    string `gorm:"type:text" json:"merge_tx_hex"`     // Merge tx hex
	FeeRate       int64  `json:"fee_rate"`                          // Fee rate

	// Task status & progress
	Status          Status    `gorm:"type:varchar(20);default:'pending'" json:"status"` // pending/processing/success/failed
	Progress        int       `gorm:"type:int;default:0" json:"progress"`               // Percent (0-100)
	TotalChunks     int       `gorm:"type:int;default:0" json:"total_chunks"`           // Total chunks
	ProcessedChunks int       `gorm:"type:int;default:0" json:"processed_chunks"`       // Processed chunks
	CurrentStep     string    `gorm:"type:varchar(100)" json:"current_step"`            // Current step description
	Stage           TaskStage `gorm:"type:varchar(50);default:'created'" json:"stage"`  // resumable stage

	// Result info
	FileId         string `gorm:"type:varchar(255)" json:"file_id"`    // File ID (after success)
	ChunkFundingTx string `gorm:"type:text" json:"chunk_funding_tx"`   // Chunk funding tx hex
	ChunkTxIds     string `gorm:"type:text" json:"chunk_tx_ids"`       // Chunk tx IDs (JSON array)
	ChunkTxHexes   string `gorm:"type:longtext" json:"-"`              // Chunk tx hex list (JSON array, internal use)
	IndexTxId      string `gorm:"type:varchar(64)" json:"index_tx_id"` // Index tx ID
	ErrorMessage   string `gorm:"type:text" json:"error_message"`      // Error message

	// Timestamps
	CreatedAt  time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
	StartedAt  *time.Time `gorm:"type:timestamp" json:"started_at"`
	FinishedAt *time.Time `gorm:"type:timestamp" json:"finished_at"`
}

// TableName sets custom table name
func (FileUploaderTask) TableName() string {
	return "tb_file_uploader_task"
}
