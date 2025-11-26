package database

import (
	"fmt"
	"log"
	"time"

	"meta-file-system/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// MySQLDatabase MySQL database implementation
type MySQLDatabase struct {
	db *gorm.DB
}

// MySQLConfig MySQL configuration
type MySQLConfig struct {
	DSN          string
	MaxOpenConns int
	MaxIdleConns int
}

// NewMySQLDatabase create MySQL database instance
func NewMySQLDatabase(config interface{}) (Database, error) {
	cfg, ok := config.(*MySQLConfig)
	if !ok {
		return nil, fmt.Errorf("invalid MySQL config type")
	}

	// Connect database
	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect MySQL: %w", err)
	}

	// Get underlying sql.DB
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Set connection pool
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println("MySQL database connected successfully")

	return &MySQLDatabase{db: db}, nil
}

// IndexerFile operations

func (m *MySQLDatabase) CreateIndexerFile(file *model.IndexerFile) error {
	return m.db.Create(file).Error
}

func (m *MySQLDatabase) GetIndexerFileByPinID(pinID string) (*model.IndexerFile, error) {
	var file model.IndexerFile
	err := m.db.Where("pin_id = ?", pinID).First(&file).Error
	if err == gorm.ErrRecordNotFound {
		return nil, ErrNotFound
	}
	return &file, err
}

func (m *MySQLDatabase) UpdateIndexerFile(file *model.IndexerFile) error {
	return m.db.Save(file).Error
}

func (m *MySQLDatabase) ListIndexerFilesWithCursor(cursor int64, size int) ([]*model.IndexerFile, int64, error) {
	var files []*model.IndexerFile
	query := m.db.Where("status = ?", model.StatusSuccess)

	if cursor > 0 {
		query = query.Where("id < ?", cursor)
	}

	err := query.Order("id DESC").Limit(size).Find(&files).Error
	if err != nil {
		return nil, 0, err
	}

	// Calculate nextCursor: cursor + number of records returned
	nextCursor := cursor + int64(len(files))
	return files, nextCursor, nil
}

func (m *MySQLDatabase) GetIndexerFilesByCreatorAddressWithCursor(address string, cursor int64, size int) ([]*model.IndexerFile, int64, error) {
	var files []*model.IndexerFile
	query := m.db.Where("creator_address = ? AND status = ?", address, model.StatusSuccess)

	if cursor > 0 {
		query = query.Where("id < ?", cursor)
	}

	err := query.Order("id DESC").Limit(size).Find(&files).Error
	if err != nil {
		return nil, 0, err
	}

	// Calculate nextCursor: cursor + number of records returned
	nextCursor := cursor + int64(len(files))
	return files, nextCursor, nil
}

func (m *MySQLDatabase) GetIndexerFilesByCreatorMetaIDWithCursor(metaID string, cursor int64, size int) ([]*model.IndexerFile, int64, error) {
	var files []*model.IndexerFile
	query := m.db.Where("creator_meta_id = ? AND status = ?", metaID, model.StatusSuccess)

	if cursor > 0 {
		query = query.Where("id < ?", cursor)
	}

	err := query.Order("id DESC").Limit(size).Find(&files).Error
	if err != nil {
		return nil, 0, err
	}

	// Calculate nextCursor: cursor + number of records returned
	nextCursor := cursor + int64(len(files))
	return files, nextCursor, nil
}

func (m *MySQLDatabase) GetIndexerFilesCount() (int64, error) {
	var count int64
	err := m.db.Model(&model.IndexerFile{}).
		Where("status = ? AND state = 0", model.StatusSuccess).
		Count(&count).Error
	return count, err
}

// IndexerUserAvatar operations

func (m *MySQLDatabase) CreateIndexerUserAvatar(avatar *model.IndexerUserAvatar) error {
	return m.db.Create(avatar).Error
}

func (m *MySQLDatabase) GetIndexerUserAvatarByPinID(pinID string) (*model.IndexerUserAvatar, error) {
	var avatar model.IndexerUserAvatar
	err := m.db.Where("pin_id = ?", pinID).First(&avatar).Error
	if err == gorm.ErrRecordNotFound {
		return nil, ErrNotFound
	}
	return &avatar, err
}

func (m *MySQLDatabase) GetIndexerUserAvatarByMetaID(metaID string) (*model.IndexerUserAvatar, error) {
	var avatar model.IndexerUserAvatar
	err := m.db.Where("meta_id = ?", metaID).
		Order("block_height DESC, created_at DESC").
		First(&avatar).Error
	if err == gorm.ErrRecordNotFound {
		return nil, ErrNotFound
	}
	return &avatar, err
}

func (m *MySQLDatabase) GetIndexerUserAvatarByAddress(address string) (*model.IndexerUserAvatar, error) {
	var avatar model.IndexerUserAvatar
	err := m.db.Where("address = ?", address).
		Order("block_height DESC, created_at DESC").
		First(&avatar).Error
	if err == gorm.ErrRecordNotFound {
		return nil, ErrNotFound
	}
	return &avatar, err
}

func (m *MySQLDatabase) UpdateIndexerUserAvatar(avatar *model.IndexerUserAvatar) error {
	return m.db.Save(avatar).Error
}

func (m *MySQLDatabase) ListIndexerUserAvatarsWithCursor(cursor int64, size int) ([]*model.IndexerUserAvatar, error) {
	var avatars []*model.IndexerUserAvatar
	query := m.db

	if cursor > 0 {
		query = query.Where("id < ?", cursor)
	}

	err := query.Order("id DESC").Limit(size).Find(&avatars).Error
	return avatars, err
}

// IndexerFileChunk operations

func (m *MySQLDatabase) CreateIndexerFileChunk(chunk *model.IndexerFileChunk) error {
	return m.db.Create(chunk).Error
}

func (m *MySQLDatabase) GetIndexerFileChunkByPinID(pinID string) (*model.IndexerFileChunk, error) {
	var chunk model.IndexerFileChunk
	err := m.db.Where("pin_id = ?", pinID).First(&chunk).Error
	if err == gorm.ErrRecordNotFound {
		return nil, ErrNotFound
	}
	return &chunk, err
}

func (m *MySQLDatabase) GetIndexerFileChunksByParentPinID(parentPinID string) ([]*model.IndexerFileChunk, error) {
	var chunks []*model.IndexerFileChunk
	err := m.db.Where("parent_pin_id = ?", parentPinID).
		Order("chunk_index ASC").
		Find(&chunks).Error
	return chunks, err
}

func (m *MySQLDatabase) UpdateIndexerFileChunk(chunk *model.IndexerFileChunk) error {
	return m.db.Save(chunk).Error
}

// IndexerSyncStatus operations

func (m *MySQLDatabase) CreateOrUpdateIndexerSyncStatus(status *model.IndexerSyncStatus) error {
	var existing model.IndexerSyncStatus
	err := m.db.Where("chain_name = ?", status.ChainName).First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		return m.db.Create(status).Error
	} else if err != nil {
		return err
	}

	status.ID = existing.ID
	return m.db.Save(status).Error
}

func (m *MySQLDatabase) GetIndexerSyncStatusByChainName(chainName string) (*model.IndexerSyncStatus, error) {
	var status model.IndexerSyncStatus
	err := m.db.Where("chain_name = ?", chainName).First(&status).Error
	if err == gorm.ErrRecordNotFound {
		return nil, ErrNotFound
	}
	return &status, err
}

func (m *MySQLDatabase) UpdateIndexerSyncStatusHeight(chainName string, height int64) error {
	return m.db.Model(&model.IndexerSyncStatus{}).
		Where("chain_name = ?", chainName).
		Update("current_sync_height", height).Error
}

func (m *MySQLDatabase) GetAllIndexerSyncStatus() ([]*model.IndexerSyncStatus, error) {
	var statuses []*model.IndexerSyncStatus
	err := m.db.Find(&statuses).Error
	return statuses, err
}

// Close close database connection
func (m *MySQLDatabase) Close() error {
	sqlDB, err := m.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// GetGormDB get underlying GORM database instance
func (m *MySQLDatabase) GetGormDB() *gorm.DB {
	return m.db
}
