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

	// Multipart upload methods for large files
	InitiateMultipartUpload(key string) (string, error)                           // Returns uploadId
	UploadPart(key, uploadId string, partNumber int, data []byte) (string, error) // Returns etag
	CompleteMultipartUpload(key, uploadId string, parts []PartInfo) error         // Complete upload
	AbortMultipartUpload(key, uploadId string) error                              // Abort upload
	ListParts(key, uploadId string) ([]PartInfo, error)                           // List uploaded parts for resume
	GetMultipartUpload(key, uploadId string) ([]byte, error)                      // Get complete file from multipart upload
}

// PartInfo part information for multipart upload
type PartInfo struct {
	PartNumber int    `json:"partNumber"`
	ETag       string `json:"etag"`
	Size       int64  `json:"size"`
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
	case "s3":
		return NewS3Storage(conf.Cfg.Storage.S3.Region, conf.Cfg.Storage.S3.Endpoint,
			conf.Cfg.Storage.S3.AccessKey, conf.Cfg.Storage.S3.SecretKey, conf.Cfg.Storage.S3.Bucket)
	case "minio":
		return NewMinIOStorage(conf.Cfg.Storage.MinIO.Endpoint, conf.Cfg.Storage.MinIO.AccessKey,
			conf.Cfg.Storage.MinIO.SecretKey, conf.Cfg.Storage.MinIO.Bucket)
	default:
		// Default to local storage
		return NewLocalStorage(conf.Cfg.Storage.Local.BasePath)
	}
}
