package storage

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// LocalStorage local file system storage
type LocalStorage struct {
	basePath string
}

// NewLocalStorage create local storage instance
func NewLocalStorage(basePath string) (*LocalStorage, error) {
	if basePath == "" {
		basePath = "./data/files"
	}

	// Ensure directory exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base path: %w", err)
	}

	return &LocalStorage{
		basePath: basePath,
	}, nil
}

// Save save file
func (s *LocalStorage) Save(key string, data []byte) error {
	filePath := filepath.Join(s.basePath, key)

	// Ensure parent directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Get get file
func (s *LocalStorage) Get(key string) ([]byte, error) {
	filePath := filepath.Join(s.basePath, key)

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

// Delete delete file
func (s *LocalStorage) Delete(key string) error {
	filePath := filepath.Join(s.basePath, key)

	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// Exists check if file exists
func (s *LocalStorage) Exists(key string) bool {
	filePath := filepath.Join(s.basePath, key)

	_, err := os.Stat(filePath)
	return err == nil
}

// InitiateMultipartUpload initiate multipart upload (local storage implementation)
func (s *LocalStorage) InitiateMultipartUpload(key string) (string, error) {
	// For local storage, we use a simple approach: create a temp directory
	uploadId := fmt.Sprintf("upload_%d", time.Now().UnixNano())
	uploadDir := filepath.Join(s.basePath, ".uploads", uploadId)

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Store upload metadata
	metaPath := filepath.Join(uploadDir, "meta.json")
	meta := fmt.Sprintf(`{"key":"%s","uploadId":"%s"}`, key, uploadId)
	if err := ioutil.WriteFile(metaPath, []byte(meta), 0644); err != nil {
		return "", fmt.Errorf("failed to write upload metadata: %w", err)
	}

	return uploadId, nil
}

// UploadPart upload a part (local storage implementation)
func (s *LocalStorage) UploadPart(key, uploadId string, partNumber int, data []byte) (string, error) {
	uploadDir := filepath.Join(s.basePath, ".uploads", uploadId)
	partPath := filepath.Join(uploadDir, fmt.Sprintf("part_%d", partNumber))

	if err := ioutil.WriteFile(partPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write part %d: %w", partNumber, err)
	}

	// Calculate simple etag (MD5 would be better, but for simplicity we use part number)
	etag := fmt.Sprintf("%d_%d", partNumber, len(data))
	return etag, nil
}

// CompleteMultipartUpload complete multipart upload (local storage implementation)
func (s *LocalStorage) CompleteMultipartUpload(key, uploadId string, parts []PartInfo) error {
	uploadDir := filepath.Join(s.basePath, ".uploads", uploadId)
	filePath := filepath.Join(s.basePath, key)

	// Ensure parent directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Sort parts by part number
	sort.Slice(parts, func(i, j int) bool {
		return parts[i].PartNumber < parts[j].PartNumber
	})

	// Combine all parts
	outFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	for _, part := range parts {
		partPath := filepath.Join(uploadDir, fmt.Sprintf("part_%d", part.PartNumber))
		partData, err := ioutil.ReadFile(partPath)
		if err != nil {
			return fmt.Errorf("failed to read part %d: %w", part.PartNumber, err)
		}

		if _, err := outFile.Write(partData); err != nil {
			return fmt.Errorf("failed to write part %d: %w", part.PartNumber, err)
		}
	}

	// Clean up upload directory
	os.RemoveAll(uploadDir)

	return nil
}

// AbortMultipartUpload abort multipart upload (local storage implementation)
func (s *LocalStorage) AbortMultipartUpload(key, uploadId string) error {
	uploadDir := filepath.Join(s.basePath, ".uploads", uploadId)
	return os.RemoveAll(uploadDir)
}

// ListParts list uploaded parts (local storage implementation)
func (s *LocalStorage) ListParts(key, uploadId string) ([]PartInfo, error) {
	uploadDir := filepath.Join(s.basePath, ".uploads", uploadId)

	files, err := ioutil.ReadDir(uploadDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read upload directory: %w", err)
	}

	parts := make([]PartInfo, 0)
	for _, file := range files {
		if file.IsDir() || file.Name() == "meta.json" {
			continue
		}

		var partNumber int
		if _, err := fmt.Sscanf(file.Name(), "part_%d", &partNumber); err != nil {
			continue
		}

		parts = append(parts, PartInfo{
			PartNumber: partNumber,
			ETag:       fmt.Sprintf("%d_%d", partNumber, file.Size()),
			Size:       file.Size(),
		})
	}

	return parts, nil
}

// GetMultipartUpload get complete file from multipart upload (local storage implementation)
func (s *LocalStorage) GetMultipartUpload(key, uploadId string) ([]byte, error) {
	// After completion, file is at the key location
	return s.Get(key)
}
