package dao

import (
	"meta-file-system/database"
	"meta-file-system/model"
)

// FileDAO file data access object (for Uploader service, always uses MySQL)
type FileDAO struct{}

// NewFileDAO create file DAO instance
func NewFileDAO() *FileDAO {
	return &FileDAO{}
}

// Create create file record
func (dao *FileDAO) Create(file *model.File) error {
	// Uploader always uses MySQL (UploaderDB)
	return database.UploaderDB.Create(file).Error
}

// GetByFileID get file by file ID
func (dao *FileDAO) GetByFileID(fileID string) (*model.File, error) {
	var file model.File
	err := database.UploaderDB.Where("file_id = ?", fileID).First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

// GetByTxID get file by transaction ID
func (dao *FileDAO) GetByTxID(txID string) (*model.File, error) {
	var file model.File
	err := database.UploaderDB.Where("tx_id = ?", txID).First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

// GetByPath get file by path
func (dao *FileDAO) GetByPath(path string) (*model.File, error) {
	var file model.File
	err := database.UploaderDB.Where("path = ?", path).First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

// List query file list
func (dao *FileDAO) List(offset, limit int) ([]*model.File, error) {
	var files []*model.File
	err := database.UploaderDB.Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&files).Error
	if err != nil {
		return nil, err
	}
	return files, nil
}

// ListByBlockHeight query files by block height
func (dao *FileDAO) ListByBlockHeight(height int64) ([]*model.File, error) {
	var files []*model.File
	err := database.UploaderDB.Where("block_height = ?", height).Find(&files).Error
	if err != nil {
		return nil, err
	}
	return files, nil
}

// Update update file record
func (dao *FileDAO) Update(file *model.File) error {
	return database.UploaderDB.Save(file).Error
}

// Delete delete file record
func (dao *FileDAO) Delete(id int64) error {
	return database.UploaderDB.Delete(&model.File{}, id).Error
}

// Count count total files
func (dao *FileDAO) Count() (int64, error) {
	var count int64
	err := database.UploaderDB.Model(&model.File{}).Count(&count).Error
	return count, err
}

// GetMaxBlockHeight get max block height
func (dao *FileDAO) GetMaxBlockHeight() (int64, error) {
	var maxHeight int64
	err := database.UploaderDB.Model(&model.File{}).Select("COALESCE(MAX(block_height), 0)").Scan(&maxHeight).Error
	return maxHeight, err
}

// GetByID get file by primary key ID
func (dao *FileDAO) GetByID(id int64) (*model.File, error) {
	var file model.File
	err := database.UploaderDB.Where("id = ?", id).First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}
