package storage

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"sort"

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

// InitiateMultipartUpload initiate multipart upload
func (s *OSSStorage) InitiateMultipartUpload(key string) (string, error) {
	imur, err := s.bucket.InitiateMultipartUpload(key)
	if err != nil {
		return "", fmt.Errorf("failed to initiate multipart upload: %w", err)
	}
	return imur.UploadID, nil
}

// UploadPart upload a part
func (s *OSSStorage) UploadPart(key, uploadId string, partNumber int, data []byte) (string, error) {
	imur := oss.InitiateMultipartUploadResult{
		Key:      key,
		UploadID: uploadId,
	}

	part, err := s.bucket.UploadPart(imur, bytes.NewReader(data), int64(len(data)), partNumber)
	if err != nil {
		return "", fmt.Errorf("failed to upload part %d: %w", partNumber, err)
	}

	return part.ETag, nil
}

// CompleteMultipartUpload complete multipart upload
func (s *OSSStorage) CompleteMultipartUpload(key, uploadId string, parts []PartInfo) error {
	imur := oss.InitiateMultipartUploadResult{
		Key:      key,
		UploadID: uploadId,
	}

	// Convert PartInfo to oss.UploadPart
	ossParts := make([]oss.UploadPart, 0, len(parts))
	for _, p := range parts {
		ossParts = append(ossParts, oss.UploadPart{
			PartNumber: p.PartNumber,
			ETag:       p.ETag,
		})
	}

	// Sort parts by part number
	sort.Slice(ossParts, func(i, j int) bool {
		return ossParts[i].PartNumber < ossParts[j].PartNumber
	})

	_, err := s.bucket.CompleteMultipartUpload(imur, ossParts)
	if err != nil {
		return fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	return nil
}

// AbortMultipartUpload abort multipart upload
func (s *OSSStorage) AbortMultipartUpload(key, uploadId string) error {
	imur := oss.InitiateMultipartUploadResult{
		Key:      key,
		UploadID: uploadId,
	}

	err := s.bucket.AbortMultipartUpload(imur)
	if err != nil {
		return fmt.Errorf("failed to abort multipart upload: %w", err)
	}

	return nil
}

// ListParts list uploaded parts
func (s *OSSStorage) ListParts(key, uploadId string) ([]PartInfo, error) {
	imur := oss.InitiateMultipartUploadResult{
		Key:      key,
		UploadID: uploadId,
	}

	partsResult, err := s.bucket.ListUploadedParts(imur)
	if err != nil {
		return nil, fmt.Errorf("failed to list parts: %w", err)
	}

	result := make([]PartInfo, 0, len(partsResult.UploadedParts))
	for _, p := range partsResult.UploadedParts {
		result = append(result, PartInfo{
			PartNumber: p.PartNumber,
			ETag:       p.ETag,
			Size:       int64(p.Size),
		})
	}

	return result, nil
}

// GetMultipartUpload get complete file from multipart upload (after completion)
func (s *OSSStorage) GetMultipartUpload(key, uploadId string) ([]byte, error) {
	// After multipart upload is completed, the file is stored at the key
	// So we can just use Get method
	return s.Get(key)
}
