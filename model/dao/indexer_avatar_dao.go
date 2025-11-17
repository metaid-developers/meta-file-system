package dao

import (
	"meta-file-system/database"
	"meta-file-system/model"
)

// IndexerUserAvatarDAO indexer user avatar data access object
type IndexerUserAvatarDAO struct {
	db database.Database
}

// NewIndexerUserAvatarDAO create indexer user avatar DAO instance
func NewIndexerUserAvatarDAO() *IndexerUserAvatarDAO {
	return &IndexerUserAvatarDAO{
		db: database.DB,
	}
}

// Create create avatar record
func (dao *IndexerUserAvatarDAO) Create(avatar *model.IndexerUserAvatar) error {
	return dao.db.CreateIndexerUserAvatar(avatar)
}

// GetByPinID get avatar by PIN ID
func (dao *IndexerUserAvatarDAO) GetByPinID(pinID string) (*model.IndexerUserAvatar, error) {
	avatar, err := dao.db.GetIndexerUserAvatarByPinID(pinID)
	if err == database.ErrNotFound {
		return nil, nil
	}
	return avatar, err
}

// GetByMetaID get latest avatar by MetaID
func (dao *IndexerUserAvatarDAO) GetByMetaID(metaID string) (*model.IndexerUserAvatar, error) {
	avatar, err := dao.db.GetIndexerUserAvatarByMetaID(metaID)
	if err == database.ErrNotFound {
		return nil, nil
	}
	return avatar, err
}

// GetByAddress get latest avatar by address
func (dao *IndexerUserAvatarDAO) GetByAddress(address string) (*model.IndexerUserAvatar, error) {
	avatar, err := dao.db.GetIndexerUserAvatarByAddress(address)
	if err == database.ErrNotFound {
		return nil, nil
	}
	return avatar, err
}

// Update update avatar record
func (dao *IndexerUserAvatarDAO) Update(avatar *model.IndexerUserAvatar) error {
	return dao.db.UpdateIndexerUserAvatar(avatar)
}

// ListWithCursor list avatars with cursor pagination
func (dao *IndexerUserAvatarDAO) ListWithCursor(cursor int64, size int) ([]*model.IndexerUserAvatar, error) {
	return dao.db.ListIndexerUserAvatarsWithCursor(cursor, size)
}
