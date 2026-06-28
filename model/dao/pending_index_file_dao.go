package dao

import (
	"meta-file-system/database"
	"meta-file-system/model"
)

// PendingIndexFileDAO data access object for deferred multi-chunk index merges.
type PendingIndexFileDAO struct {
	db database.Database
}

// NewPendingIndexFileDAO create pending index file DAO instance.
func NewPendingIndexFileDAO() *PendingIndexFileDAO {
	return &PendingIndexFileDAO{
		db: database.DB,
	}
}

// Create persists a deferred index merge record (overwrites on conflict).
func (dao *PendingIndexFileDAO) Create(p *model.PendingIndexFile) error {
	return dao.db.CreatePendingIndexFile(p)
}

// GetByPinID returns the pending record for an index pinId, or (nil, nil) when
// no pending record exists.
func (dao *PendingIndexFileDAO) GetByPinID(pinID string) (*model.PendingIndexFile, error) {
	p, err := dao.db.GetPendingIndexFileByPinID(pinID)
	if err == database.ErrNotFound {
		return nil, nil
	}
	return p, err
}

// ListByChain returns all pending records for a chain.
func (dao *PendingIndexFileDAO) ListByChain(chainName string) ([]*model.PendingIndexFile, error) {
	return dao.db.ListPendingIndexFilesByChain(chainName)
}

// Delete removes a pending record (after a successful merge).
func (dao *PendingIndexFileDAO) Delete(pinID string) error {
	return dao.db.DeletePendingIndexFile(pinID)
}
