package database

import (
	"meta-file-system/model"
)

// Database interface for different database implementations
type Database interface {
	// IndexerFile operations
	CreateIndexerFile(file *model.IndexerFile) error
	GetIndexerFileByPinID(pinID string) (*model.IndexerFile, error)
	UpdateIndexerFile(file *model.IndexerFile) error
	ListIndexerFilesWithCursor(cursor int64, size int) ([]*model.IndexerFile, error)
	GetIndexerFilesByCreatorAddressWithCursor(address string, cursor int64, size int) ([]*model.IndexerFile, error)
	GetIndexerFilesByCreatorMetaIDWithCursor(metaID string, cursor int64, size int) ([]*model.IndexerFile, error)
	GetIndexerFilesCount() (int64, error)

	// IndexerUserAvatar operations
	CreateIndexerUserAvatar(avatar *model.IndexerUserAvatar) error
	GetIndexerUserAvatarByPinID(pinID string) (*model.IndexerUserAvatar, error)
	GetIndexerUserAvatarByMetaID(metaID string) (*model.IndexerUserAvatar, error)
	GetIndexerUserAvatarByAddress(address string) (*model.IndexerUserAvatar, error)
	UpdateIndexerUserAvatar(avatar *model.IndexerUserAvatar) error
	ListIndexerUserAvatarsWithCursor(cursor int64, size int) ([]*model.IndexerUserAvatar, error)

	// IndexerFileChunk operations
	CreateIndexerFileChunk(chunk *model.IndexerFileChunk) error
	GetIndexerFileChunkByPinID(pinID string) (*model.IndexerFileChunk, error)
	GetIndexerFileChunksByParentPinID(parentPinID string) ([]*model.IndexerFileChunk, error)
	UpdateIndexerFileChunk(chunk *model.IndexerFileChunk) error

	// IndexerSyncStatus operations
	CreateOrUpdateIndexerSyncStatus(status *model.IndexerSyncStatus) error
	GetIndexerSyncStatusByChainName(chainName string) (*model.IndexerSyncStatus, error)
	UpdateIndexerSyncStatusHeight(chainName string, height int64) error
	GetAllIndexerSyncStatus() ([]*model.IndexerSyncStatus, error)

	// General operations
	Close() error
}

// DBType database type
type DBType string

const (
	DBTypeMySQL  DBType = "mysql"
	DBTypePebble DBType = "pebble"
)

// Global database instance
var DB Database

// currentDBType stores the current database type
var currentDBType DBType

// InitDatabase initialize database with specified type
func InitDatabase(dbType DBType, config interface{}) error {
	var err error

	switch dbType {
	case DBTypeMySQL:
		DB, err = NewMySQLDatabase(config)
		currentDBType = DBTypeMySQL
	case DBTypePebble:
		DB, err = NewPebbleDatabase(config)
		currentDBType = DBTypePebble
	default:
		return ErrUnsupportedDBType
	}

	return err
}

// GetGormDB get GORM database instance (only for MySQL)
func GetGormDB() interface{} {
	if currentDBType == DBTypeMySQL {
		if mysqlDB, ok := DB.(*MySQLDatabase); ok {
			return mysqlDB.GetGormDB()
		}
	}
	return nil
}

// GetDBType get current database type
func GetDBType() DBType {
	return currentDBType
}
