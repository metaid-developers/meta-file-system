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
	ListIndexerFilesWithCursor(cursor int64, size int) ([]*model.IndexerFile, int64, error)
	GetIndexerFilesByCreatorAddressWithCursor(address string, cursor int64, size int) ([]*model.IndexerFile, int64, error)
	GetIndexerFilesByCreatorMetaIDWithCursor(metaID string, cursor int64, size int) ([]*model.IndexerFile, int64, error)
	GetIndexerFilesByCreatorGlobalMetaIDWithCursor(globalMetaID string, cursor int64, size int) ([]*model.IndexerFile, int64, error)
	GetIndexerFilesByExtensionWithCursor(extension string, cursor string, size int) ([]*model.IndexerFile, string, error)
	GetIndexerFilesByGlobalMetaIDAndExtensionWithCursor(globalMetaID string, extension string, cursor string, size int) ([]*model.IndexerFile, string, error)
	GetIndexerFilesCount() (int64, error)
	GetIndexerFilesCountByChain(chainName string) (int64, error)
	GetLatestFileInfoByFirstPinID(firstPinID string) (*model.IndexerFile, error)
	AddFileInfoHistory(history *model.FileInfoHistory, firstPinID string) error
	GetFileInfoHistory(firstPinID string) ([]model.FileInfoHistory, error)

	// Schema version (for migrate)
	GetIndexerSchemaVersion() (int, error)
	SetIndexerSchemaVersion(version int) error
	// Migrate helpers: iterate latest file info; write file to global_meta + extension indexes only
	IterateLatestFileInfo(fn func(*model.IndexerFile) error) error
	WriteFileToExtensionAndGlobalMetaIndexes(file *model.IndexerFile) error

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

	// UserInfo operations
	// User Name
	CreateOrUpdateLatestUserNameInfo(info *model.UserNameInfo, metaID string) error
	GetLatestUserNameInfo(key string) (*model.UserNameInfo, error)
	AddUserNameInfoHistory(info *model.UserNameInfo, metaID string) error
	GetUserNameInfoHistory(key string) ([]model.UserNameInfo, error)
	// User Avatar
	CreateOrUpdateLatestUserAvatarInfo(info *model.UserAvatarInfo, metaID string) error
	GetLatestUserAvatarInfo(key string) (*model.UserAvatarInfo, error)
	GetUserAvatarInfoByPinID(pinID string) (*model.UserAvatarInfo, error)
	AddUserAvatarInfoHistory(info *model.UserAvatarInfo, metaID string) error
	GetUserAvatarInfoHistory(key string) ([]model.UserAvatarInfo, error)
	// User Chat Public Key
	CreateOrUpdateLatestUserChatPublicKeyInfo(info *model.UserChatPublicKeyInfo, metaID string) error
	GetLatestUserChatPublicKeyInfo(key string) (*model.UserChatPublicKeyInfo, error)
	AddUserChatPublicKeyHistory(info *model.UserChatPublicKeyInfo, metaID string) error
	GetUserChatPublicKeyHistory(key string) ([]model.UserChatPublicKeyInfo, error)
	// User Name by GlobalMetaId
	CreateOrUpdateLatestUserNameInfoByGlobalMetaId(info *model.UserNameInfo, globalMetaId string) error
	GetLatestUserNameInfoByGlobalMetaId(globalMetaId string) (*model.UserNameInfo, error)
	AddUserNameInfoHistoryByGlobalMetaId(info *model.UserNameInfo, globalMetaId string) error
	GetUserNameInfoHistoryByGlobalMetaId(globalMetaId string) ([]model.UserNameInfo, error)
	// User Avatar by GlobalMetaId
	CreateOrUpdateLatestUserAvatarInfoByGlobalMetaId(info *model.UserAvatarInfo, globalMetaId string) error
	GetLatestUserAvatarInfoByGlobalMetaId(globalMetaId string) (*model.UserAvatarInfo, error)
	AddUserAvatarInfoHistoryByGlobalMetaId(info *model.UserAvatarInfo, globalMetaId string) error
	GetUserAvatarInfoHistoryByGlobalMetaId(globalMetaId string) ([]model.UserAvatarInfo, error)
	// User Chat Public Key by GlobalMetaId
	CreateOrUpdateLatestUserChatPublicKeyInfoByGlobalMetaId(info *model.UserChatPublicKeyInfo, globalMetaId string) error
	GetLatestUserChatPublicKeyInfoByGlobalMetaId(globalMetaId string) (*model.UserChatPublicKeyInfo, error)
	AddUserChatPublicKeyHistoryByGlobalMetaId(info *model.UserChatPublicKeyInfo, globalMetaId string) error
	GetUserChatPublicKeyHistoryByGlobalMetaId(globalMetaId string) ([]model.UserChatPublicKeyInfo, error)
	// Get all user info history by MetaID or Address
	GetUserInfoHistoryByKey(key string) (*model.UserInfoHistory, error)

	// PinInfo operations
	CreateOrUpdatePinInfo(pinInfo *model.IndexerPinInfo) error
	GetPinInfoByPinID(pinID string) (*model.IndexerPinInfo, error)

	// MetaIdAddress operations
	SaveMetaIdAddress(metaID, address string) error
	GetAddressByMetaID(metaID string) (string, error)
	GetMetaIDByAddress(address string) (string, error)

	// GlobalMetaIdAddress operations
	SaveGlobalMetaIdAddress(globalMetaId, metaID, address string) error
	GetGlobalMetaIdAddress(globalMetaId string) (*model.GlobalMetaIdAddress, error)

	// MetaIdTimestamp operations
	SaveMetaIdTimestamp(metaID string, timestamp int64) error
	ListMetaIdsByTimestamp(cursor int64, size int) ([]model.MetaIdTimestamp, int64, bool, error)
	GetMetaIDCount() (int64, error)

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
