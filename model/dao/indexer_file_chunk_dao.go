package dao

import (
	"meta-file-system/database"
	"meta-file-system/model"
)

// IndexerFileChunkDAO indexer file chunk data access object
type IndexerFileChunkDAO struct {
	db database.Database
}

// NewIndexerFileChunkDAO create indexer file chunk DAO instance
func NewIndexerFileChunkDAO() *IndexerFileChunkDAO {
	return &IndexerFileChunkDAO{
		db: database.DB,
	}
}

// Create create indexer file chunk record
func (dao *IndexerFileChunkDAO) Create(chunk *model.IndexerFileChunk) error {
	return dao.db.CreateIndexerFileChunk(chunk)
}

// GetByPinID get chunk by PIN ID
func (dao *IndexerFileChunkDAO) GetByPinID(pinID string) (*model.IndexerFileChunk, error) {
	chunk, err := dao.db.GetIndexerFileChunkByPinID(pinID)
	if err == database.ErrNotFound {
		return nil, nil
	}
	return chunk, err
}

// GetByParentPinID get all chunks by parent PIN ID (ordered by chunk_index)
func (dao *IndexerFileChunkDAO) GetByParentPinID(parentPinID string) ([]*model.IndexerFileChunk, error) {
	return dao.db.GetIndexerFileChunksByParentPinID(parentPinID)
}

// Update update chunk record
func (dao *IndexerFileChunkDAO) Update(chunk *model.IndexerFileChunk) error {
	return dao.db.UpdateIndexerFileChunk(chunk)
}
