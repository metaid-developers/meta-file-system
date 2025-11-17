package dao

import (
	"meta-file-system/database"
	"meta-file-system/model"
)

// IndexerSyncStatusDAO indexer sync status data access object
type IndexerSyncStatusDAO struct {
	db database.Database
}

// NewIndexerSyncStatusDAO create indexer sync status DAO instance
func NewIndexerSyncStatusDAO() *IndexerSyncStatusDAO {
	return &IndexerSyncStatusDAO{
		db: database.DB,
	}
}

// GetByChainName get sync status by chain name
func (dao *IndexerSyncStatusDAO) GetByChainName(chainName string) (*model.IndexerSyncStatus, error) {
	status, err := dao.db.GetIndexerSyncStatusByChainName(chainName)
	if err == database.ErrNotFound {
		return nil, nil
	}
	return status, err
}

// CreateOrUpdate create or update sync status
func (dao *IndexerSyncStatusDAO) CreateOrUpdate(status *model.IndexerSyncStatus) error {
	return dao.db.CreateOrUpdateIndexerSyncStatus(status)
}

// UpdateCurrentSyncHeight update current scanned height
func (dao *IndexerSyncStatusDAO) UpdateCurrentSyncHeight(chainName string, height int64) error {
	return dao.db.UpdateIndexerSyncStatusHeight(chainName, height)
}

// GetAll get all chain sync status
func (dao *IndexerSyncStatusDAO) GetAll() ([]*model.IndexerSyncStatus, error) {
	return dao.db.GetAllIndexerSyncStatus()
}
