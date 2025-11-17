package storage

import (
	"errors"
	"meta-file-system/conf"
)

// Storage unified storage interface
type Storage interface {
	Save(key string, data []byte) error
	Get(key string) ([]byte, error)
	Delete(key string) error
	Exists(key string) bool
}

var (
	ErrNotFound = errors.New("file not found")
	ErrInvalid  = errors.New("invalid storage configuration")
)

// NewStorage create storage instance by configuration
func NewStorage() (Storage, error) {
	storageType := conf.Cfg.Storage.Type

	switch storageType {
	case "local":
		return NewLocalStorage(conf.Cfg.Storage.Local.BasePath)
	case "oss":
		return NewOSSStorage(conf.Cfg.Storage.OSS.Endpoint, conf.Cfg.Storage.OSS.AccessKey,
			conf.Cfg.Storage.OSS.SecretKey, conf.Cfg.Storage.OSS.Bucket)
	default:
		// Default to local storage
		return NewLocalStorage(conf.Cfg.Storage.Local.BasePath)
	}
}
