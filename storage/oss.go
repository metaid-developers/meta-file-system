package storage

import (
	"bytes"
	"fmt"
	"io/ioutil"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// OSSStorage Alibaba Cloud OSS storage
type OSSStorage struct {
	bucket *oss.Bucket
}

// NewOSSStorage create OSS storage instance
func NewOSSStorage(endpoint, accessKey, secretKey, bucketName string) (*OSSStorage, error) {
	if endpoint == "" || accessKey == "" || secretKey == "" || bucketName == "" {
		return nil, ErrInvalid
	}

	// Create OSS client instance
	client, err := oss.New(endpoint, accessKey, secretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create oss client: %w", err)
	}

	// Get storage bucket
	bucket, err := client.Bucket(bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket: %w", err)
	}

	return &OSSStorage{
		bucket: bucket,
	}, nil
}

// Save save file to OSS
func (s *OSSStorage) Save(key string, data []byte) error {
	err := s.bucket.PutObject(key, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to upload to oss: %w", err)
	}
	return nil
}

// Get get file from OSS
func (s *OSSStorage) Get(key string) ([]byte, error) {
	body, err := s.bucket.GetObject(key)
	if err != nil {
		if ossErr, ok := err.(oss.ServiceError); ok && ossErr.StatusCode == 404 {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get from oss: %w", err)
	}
	defer body.Close()

	data, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("failed to read oss object: %w", err)
	}

	return data, nil
}

// Delete delete file from OSS
func (s *OSSStorage) Delete(key string) error {
	err := s.bucket.DeleteObject(key)
	if err != nil {
		return fmt.Errorf("failed to delete from oss: %w", err)
	}
	return nil
}

// Exists check if file exists in OSS
func (s *OSSStorage) Exists(key string) bool {
	exists, err := s.bucket.IsObjectExist(key)
	if err != nil {
		return false
	}
	return exists
}
