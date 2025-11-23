package dao

import (
	"meta-file-system/database"
	"meta-file-system/model"
)

// FileChunkDAO file chunk data access object
type FileChunkDAO struct{}

// NewFileChunkDAO create file chunk DAO instance
func NewFileChunkDAO() *FileChunkDAO {
	return &FileChunkDAO{}
}

// Create create file chunk record
func (dao *FileChunkDAO) Create(chunk *model.FileChunk) error {
	return database.UploaderDB.Create(chunk).Error
}

// GetByPinID get chunk by PIN ID
func (dao *FileChunkDAO) GetByPinID(pinID string) (*model.FileChunk, error) {
	var chunk model.FileChunk
	err := database.UploaderDB.Where("pin_id = ?", pinID).First(&chunk).Error
	if err != nil {
		return nil, err
	}
	return &chunk, nil
}

// GetByFileHash get chunks by file hash
func (dao *FileChunkDAO) GetByFileHash(fileHash string) ([]*model.FileChunk, error) {
	var chunks []*model.FileChunk
	err := database.UploaderDB.Where("file_hash = ?", fileHash).
		Order("chunk_index ASC").
		Find(&chunks).Error
	return chunks, err
}

// Update update file chunk record
func (dao *FileChunkDAO) Update(chunk *model.FileChunk) error {
	return database.UploaderDB.Save(chunk).Error
}
