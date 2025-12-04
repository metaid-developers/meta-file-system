package dao

import (
	"meta-file-system/database"
	"meta-file-system/model"
)

// IndexerFileDAO indexer file data access object
type IndexerFileDAO struct {
	db database.Database
}

// NewIndexerFileDAO create indexer file DAO instance
func NewIndexerFileDAO() *IndexerFileDAO {
	return &IndexerFileDAO{
		db: database.DB,
	}
}

// Create create indexer file record
func (dao *IndexerFileDAO) Create(file *model.IndexerFile) error {
	return dao.db.CreateIndexerFile(file)
}

// GetByPinID get file by PIN ID
func (dao *IndexerFileDAO) GetByPinID(pinID string) (*model.IndexerFile, error) {
	file, err := dao.db.GetIndexerFileByPinID(pinID)
	if err == database.ErrNotFound {
		return nil, nil
	}
	return file, err
}

// Update update file record
func (dao *IndexerFileDAO) Update(file *model.IndexerFile) error {
	return dao.db.UpdateIndexerFile(file)
}

// ListWithCursor get file list with cursor pagination
// cursor: number of records to skip (0 for first page)
// size: page size
// Returns: files, nextCursor, error
func (dao *IndexerFileDAO) ListWithCursor(cursor int64, size int) ([]*model.IndexerFile, int64, error) {
	return dao.db.ListIndexerFilesWithCursor(cursor, size)
}

// GetByCreatorAddressWithCursor get file list by creator address with cursor pagination
// cursor: number of records to skip (0 for first page)
// size: page size
// Returns: files, nextCursor, error
func (dao *IndexerFileDAO) GetByCreatorAddressWithCursor(address string, cursor int64, size int) ([]*model.IndexerFile, int64, error) {
	return dao.db.GetIndexerFilesByCreatorAddressWithCursor(address, cursor, size)
}

// GetByCreatorMetaIDWithCursor get file list by creator MetaID with cursor pagination
// cursor: number of records to skip (0 for first page)
// size: page size
// Returns: files, nextCursor, error
func (dao *IndexerFileDAO) GetByCreatorMetaIDWithCursor(metaID string, cursor int64, size int) ([]*model.IndexerFile, int64, error) {
	return dao.db.GetIndexerFilesByCreatorMetaIDWithCursor(metaID, cursor, size)
}

// GetFilesCount get total count of indexed files
func (dao *IndexerFileDAO) GetFilesCount() (int64, error) {
	return dao.db.GetIndexerFilesCount()
}

// GetLatestFileInfoByFirstPinID get latest file info by first PIN ID
func (dao *IndexerFileDAO) GetLatestFileInfoByFirstPinID(firstPinID string) (*model.IndexerFile, error) {
	return dao.db.GetLatestFileInfoByFirstPinID(firstPinID)
}
