package dao

import (
	"fmt"
	"time"

	"meta-file-system/database"
	"meta-file-system/model"
)

// MultipartUploadDAO data access layer for multipart upload records
type MultipartUploadDAO struct{}

// NewMultipartUploadDAO creates a new DAO instance
func NewMultipartUploadDAO() *MultipartUploadDAO {
	return &MultipartUploadDAO{}
}

// Create inserts a new multipart upload record
func (dao *MultipartUploadDAO) Create(upload *model.MultipartUpload) error {
	return database.UploaderDB.Create(upload).Error
}

// GetByUploadID fetches a record by upload ID
func (dao *MultipartUploadDAO) GetByUploadID(uploadID string) (*model.MultipartUpload, error) {
	var upload model.MultipartUpload
	err := database.UploaderDB.Where("upload_id = ?", uploadID).First(&upload).Error
	if err != nil {
		return nil, err
	}
	return &upload, nil
}

// GetByKey fetches a record by storage key
func (dao *MultipartUploadDAO) GetByKey(key string) (*model.MultipartUpload, error) {
	var upload model.MultipartUpload
	err := database.UploaderDB.Where("key = ?", key).First(&upload).Error
	if err != nil {
		return nil, err
	}
	return &upload, nil
}

// Update persists upload changes
func (dao *MultipartUploadDAO) Update(upload *model.MultipartUpload) error {
	if upload == nil {
		return fmt.Errorf("upload is nil")
	}
	return database.UploaderDB.Model(&model.MultipartUpload{}).
		Where("id = ?", upload.ID).
		Select("*").
		Updates(upload).Error
}

// UpdateStatus updates upload status
func (dao *MultipartUploadDAO) UpdateStatus(uploadID string, status model.MultipartUploadStatus) error {
	return database.UploaderDB.Model(&model.MultipartUpload{}).
		Where("upload_id = ?", uploadID).
		Update("status", status).Error
}

// UpdatePartCount updates part count
func (dao *MultipartUploadDAO) UpdatePartCount(uploadID string, partCount int) error {
	return database.UploaderDB.Model(&model.MultipartUpload{}).
		Where("upload_id = ?", uploadID).
		Update("part_count", partCount).Error
}

// ListExpiredUploads returns expired uploads for cleanup
func (dao *MultipartUploadDAO) ListExpiredUploads(beforeTime time.Time, limit int) ([]*model.MultipartUpload, error) {
	var uploads []*model.MultipartUpload
	err := database.UploaderDB.Where("expires_at < ? AND status != ?", beforeTime, model.MultipartUploadStatusExpired).
		Order("expires_at ASC").
		Limit(limit).
		Find(&uploads).Error
	return uploads, err
}

// ListByStatus returns uploads by status
func (dao *MultipartUploadDAO) ListByStatus(status model.MultipartUploadStatus, limit int) ([]*model.MultipartUpload, error) {
	var uploads []*model.MultipartUpload
	err := database.UploaderDB.Where("status = ?", status).
		Order("created_at DESC").
		Limit(limit).
		Find(&uploads).Error
	return uploads, err
}

// Delete deletes a multipart upload record
func (dao *MultipartUploadDAO) Delete(id int64) error {
	return database.UploaderDB.Delete(&model.MultipartUpload{}, id).Error
}

// DeleteByUploadID deletes a record by upload ID
func (dao *MultipartUploadDAO) DeleteByUploadID(uploadID string) error {
	return database.UploaderDB.Where("upload_id = ?", uploadID).Delete(&model.MultipartUpload{}).Error
}

// DeleteExpiredUploads deletes expired uploads (batch cleanup)
func (dao *MultipartUploadDAO) DeleteExpiredUploads(beforeTime time.Time) (int64, error) {
	result := database.UploaderDB.Where("expires_at < ?", beforeTime).
		Delete(&model.MultipartUpload{})
	return result.RowsAffected, result.Error
}

// CountByStatus returns count of uploads by status
func (dao *MultipartUploadDAO) CountByStatus(status model.MultipartUploadStatus) (int64, error) {
	var count int64
	err := database.UploaderDB.Model(&model.MultipartUpload{}).
		Where("status = ?", status).
		Count(&count).Error
	return count, err
}
