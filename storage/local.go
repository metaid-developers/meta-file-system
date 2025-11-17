package storage

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
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
