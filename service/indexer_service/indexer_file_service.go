package indexer_service

import (
	"errors"
	"fmt"

	"meta-file-system/model"
	"meta-file-system/model/dao"
	"meta-file-system/storage"

	"gorm.io/gorm"
)

// IndexerFileService indexer file service
type IndexerFileService struct {
	indexerFileDAO       *dao.IndexerFileDAO
	indexerUserAvatarDAO *dao.IndexerUserAvatarDAO
	storage              storage.Storage
}

// NewIndexerFileService create indexer file service instance
func NewIndexerFileService(storage storage.Storage) *IndexerFileService {
	return &IndexerFileService{
		indexerFileDAO:       dao.NewIndexerFileDAO(),
		indexerUserAvatarDAO: dao.NewIndexerUserAvatarDAO(),
		storage:              storage,
	}
}

// GetFileByPinID get file information by PIN ID
func (s *IndexerFileService) GetFileByPinID(pinID string) (*model.IndexerFile, error) {
	file, err := s.indexerFileDAO.GetByPinID(pinID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("file not found")
		}
		return nil, fmt.Errorf("failed to get file: %w", err)
	}
	return file, nil
}

// GetFilesByCreatorAddress get file list by creator address with cursor pagination
// cursor: last file ID (0 for first page)
// size: page size
// Returns: files, next_cursor, has_more, error
func (s *IndexerFileService) GetFilesByCreatorAddress(address string, cursor int64, size int) ([]*model.IndexerFile, int64, bool, error) {
	if size < 1 || size > 100 {
		size = 20
	}

	files, err := s.indexerFileDAO.GetByCreatorAddressWithCursor(address, cursor, size)
	if err != nil {
		return nil, 0, false, fmt.Errorf("failed to get files by creator address: %w", err)
	}

	// Determine next cursor and has_more
	var nextCursor int64
	hasMore := false

	if len(files) > 0 {
		// Next cursor is the ID of the last file
		nextCursor = files[len(files)-1].ID

		// Check if there are more records
		hasMore = len(files) == size
	}

	return files, nextCursor, hasMore, nil
}

// GetFilesByCreatorMetaID get file list by creator MetaID with cursor pagination
// cursor: last file ID (0 for first page)
// size: page size
// Returns: files, next_cursor, has_more, error
func (s *IndexerFileService) GetFilesByCreatorMetaID(metaID string, cursor int64, size int) ([]*model.IndexerFile, int64, bool, error) {
	if size < 1 || size > 100 {
		size = 20
	}

	files, err := s.indexerFileDAO.GetByCreatorMetaIDWithCursor(metaID, cursor, size)
	if err != nil {
		return nil, 0, false, fmt.Errorf("failed to get files by creator MetaID: %w", err)
	}

	// Determine next cursor and has_more
	var nextCursor int64
	hasMore := false

	if len(files) > 0 {
		// Next cursor is the ID of the last file
		nextCursor = files[len(files)-1].ID

		// Check if there are more records
		hasMore = len(files) == size
	}

	return files, nextCursor, hasMore, nil
}

// ListFiles get file list with cursor pagination
// cursor: last file ID (0 for first page)
// size: page size
// Returns: files, next_cursor, has_more, error
func (s *IndexerFileService) ListFiles(cursor int64, size int) ([]*model.IndexerFile, int64, bool, error) {
	if size < 1 || size > 100 {
		size = 20
	}

	files, err := s.indexerFileDAO.ListWithCursor(cursor, size)
	if err != nil {
		return nil, 0, false, fmt.Errorf("failed to list files: %w", err)
	}

	// Determine next cursor and has_more
	var nextCursor int64
	hasMore := false

	if len(files) > 0 {
		// Next cursor is the ID of the last file
		nextCursor = files[len(files)-1].ID

		// Check if there are more records
		hasMore = len(files) == size
	}

	return files, nextCursor, hasMore, nil
}

// GetFileContent get file content by PIN ID
func (s *IndexerFileService) GetFileContent(pinID string) ([]byte, string, string, error) {
	// Get file information
	file, err := s.GetFileByPinID(pinID)
	if err != nil {
		return nil, "", "", err
	}

	// Read file content from storage layer
	content, err := s.storage.Get(file.StoragePath)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to get file content: %w", err)
	}

	return content, file.ContentType, file.FileName, nil
}

// GetFilesCount get total count of indexed files
func (s *IndexerFileService) GetFilesCount() (int64, error) {
	return s.indexerFileDAO.GetFilesCount()
}

// ListAvatars get avatar list with cursor pagination
// cursor: last avatar ID (0 for first page)
// size: page size
// Returns: avatars, next_cursor, has_more, error
func (s *IndexerFileService) ListAvatars(cursor int64, size int) ([]*model.IndexerUserAvatar, int64, bool, error) {
	if size < 1 || size > 100 {
		size = 20
	}

	avatars, err := s.indexerUserAvatarDAO.ListWithCursor(cursor, size)
	if err != nil {
		return nil, 0, false, fmt.Errorf("failed to list avatars: %w", err)
	}

	// Determine next cursor and has_more
	var nextCursor int64
	hasMore := false

	if len(avatars) > 0 {
		// Next cursor is the ID of the last avatar
		nextCursor = avatars[len(avatars)-1].ID

		// Check if there are more records
		hasMore = len(avatars) == size
	}

	return avatars, nextCursor, hasMore, nil
}

// GetLatestAvatarByMetaID get latest avatar information by MetaID
func (s *IndexerFileService) GetLatestAvatarByMetaID(metaID string) (*model.IndexerUserAvatar, error) {
	avatar, err := s.indexerUserAvatarDAO.GetByMetaID(metaID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("avatar not found")
		}
		return nil, fmt.Errorf("failed to get avatar: %w", err)
	}
	return avatar, nil
}

// GetLatestAvatarByAddress get latest avatar information by address
func (s *IndexerFileService) GetLatestAvatarByAddress(address string) (*model.IndexerUserAvatar, error) {
	avatar, err := s.indexerUserAvatarDAO.GetByAddress(address)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("avatar not found")
		}
		return nil, fmt.Errorf("failed to get avatar: %w", err)
	}
	return avatar, nil
}

// GetAvatarContent get avatar content by PIN ID
func (s *IndexerFileService) GetAvatarContent(pinID string) ([]byte, string, string, error) {
	// Get avatar information
	avatar, err := s.indexerUserAvatarDAO.GetByPinID(pinID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", "", errors.New("avatar not found")
		}
		return nil, "", "", fmt.Errorf("failed to get avatar: %w", err)
	}

	// Read avatar content from storage layer
	content, err := s.storage.Get(avatar.Avatar)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to get avatar content: %w", err)
	}

	// Generate filename from PinID and extension
	fileName := avatar.PinID
	if avatar.FileExtension != "" {
		fileName = avatar.PinID + avatar.FileExtension
	}

	return content, avatar.ContentType, fileName, nil
}
