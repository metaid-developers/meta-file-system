package model

import "time"

// MultipartUploadStatus multipart upload status
type MultipartUploadStatus string

const (
	MultipartUploadStatusInitiated MultipartUploadStatus = "initiated" // Upload initiated
	MultipartUploadStatusUploading MultipartUploadStatus = "uploading" // Parts being uploaded
	MultipartUploadStatusCompleted MultipartUploadStatus = "completed" // Upload completed
	MultipartUploadStatusAborted   MultipartUploadStatus = "aborted"   // Upload aborted
	MultipartUploadStatusExpired   MultipartUploadStatus = "expired"   // Upload expired (for cleanup)
)

// MultipartUpload represents a multipart upload session
type MultipartUpload struct {
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// Upload identifier
	UploadId string `gorm:"uniqueIndex;type:varchar(255)" json:"upload_id"` // Upload ID from storage
	Key      string `gorm:"type:varchar(500);index" json:"key"`             // Storage key

	// File information
	FileName  string `gorm:"type:varchar(255)" json:"file_name"`   // File name
	FileSize  int64  `json:"file_size"`                            // Total file size
	MetaId    string `gorm:"type:varchar(255)" json:"meta_id"`     // MetaID (optional)
	Address   string `gorm:"type:varchar(255)" json:"address"`     // User address (optional)
	PartCount int    `gorm:"type:int;default:0" json:"part_count"` // Total number of parts

	// Upload status
	Status MultipartUploadStatus `gorm:"type:varchar(20);default:'initiated'" json:"status"` // initiated/uploading/completed/aborted/expired

	// Timestamps
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
	ExpiresAt *time.Time `gorm:"type:timestamp" json:"expires_at"` // Expiration time for cleanup
}

// TableName sets custom table name
func (MultipartUpload) TableName() string {
	return "tb_multipart_upload"
}
