package dao

import (
	"meta-file-system/database"
	"meta-file-system/model"

	"gorm.io/gorm"
)

// FileAssistentDAO file assistent data access object
type FileAssistentDAO struct{}

// NewFileAssistentDAO create file assistent DAO instance
func NewFileAssistentDAO() *FileAssistentDAO {
	return &FileAssistentDAO{}
}

// Create create file assistent record
func (dao *FileAssistentDAO) Create(assistent *model.FileAssistent) error {
	return database.UploaderDB.Create(assistent).Error
}

// GetByMetaID get assistent by MetaID
func (dao *FileAssistentDAO) GetByMetaID(metaID string) (*model.FileAssistent, error) {
	var assistent model.FileAssistent
	err := database.UploaderDB.Where("meta_id = ? AND status = ?", metaID, model.StatusSuccess).
		Order("created_at DESC").
		First(&assistent).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &assistent, nil
}

// GetByAddress get assistent by user address
func (dao *FileAssistentDAO) GetByAddress(address string) (*model.FileAssistent, error) {
	var assistent model.FileAssistent
	err := database.UploaderDB.Where("address = ? AND status = ?", address, model.StatusSuccess).
		Order("created_at DESC").
		First(&assistent).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &assistent, nil
}

// Update update file assistent record
func (dao *FileAssistentDAO) Update(assistent *model.FileAssistent) error {
	return database.UploaderDB.Save(assistent).Error
}
