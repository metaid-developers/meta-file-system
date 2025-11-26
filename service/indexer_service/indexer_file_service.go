package indexer_service

import (
	"errors"
	"fmt"
	"strings"

	"meta-file-system/conf"
	"meta-file-system/model"
	"meta-file-system/model/dao"
	"meta-file-system/storage"

	"gorm.io/gorm"
)

// OSS process parameters
const (
	OssProcess640           = "?x-oss-process=image/auto-orient,1/interlace,1/resize,m_lfit,w_640/quality,q_90"
	OssProcess128           = "?x-oss-process=image/auto-orient,1/resize,m_fill,w_128,h_128/quality,q_90"
	OssProcessVideoFirstImg = "?x-oss-process=video/snapshot,t_1,m_fast"
	OssProcess235           = "?x-oss-process=image/auto-orient,1/quality,q_80/resize,m_lfit,w_235"
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
// cursor: number of records to skip (0 for first page)
// size: page size
// Returns: files, next_cursor, has_more, error
func (s *IndexerFileService) GetFilesByCreatorAddress(address string, cursor int64, size int) ([]*model.IndexerFile, int64, bool, error) {
	if size < 1 || size > 100 {
		size = 20
	}

	files, nextCursor, err := s.indexerFileDAO.GetByCreatorAddressWithCursor(address, cursor, size)
	if err != nil {
		return nil, 0, false, fmt.Errorf("failed to get files by creator address: %w", err)
	}

	// Determine has_more: if we got exactly size records, there might be more
	hasMore := len(files) == size

	return files, nextCursor, hasMore, nil
}

// GetFilesByCreatorMetaID get file list by creator MetaID with cursor pagination
// cursor: number of records to skip (0 for first page)
// size: page size
// Returns: files, next_cursor, has_more, error
func (s *IndexerFileService) GetFilesByCreatorMetaID(metaID string, cursor int64, size int) ([]*model.IndexerFile, int64, bool, error) {
	if size < 1 || size > 100 {
		size = 20
	}

	files, nextCursor, err := s.indexerFileDAO.GetByCreatorMetaIDWithCursor(metaID, cursor, size)
	if err != nil {
		return nil, 0, false, fmt.Errorf("failed to get files by creator MetaID: %w", err)
	}

	// Determine has_more: if we got exactly size records, there might be more
	hasMore := len(files) == size

	return files, nextCursor, hasMore, nil
}

// ListFiles get file list with cursor pagination
// cursor: number of records to skip (0 for first page)
// size: page size
// Returns: files, next_cursor, has_more, error
func (s *IndexerFileService) ListFiles(cursor int64, size int) ([]*model.IndexerFile, int64, bool, error) {
	if size < 1 || size > 100 {
		size = 20
	}

	files, nextCursor, err := s.indexerFileDAO.ListWithCursor(cursor, size)
	if err != nil {
		return nil, 0, false, fmt.Errorf("failed to list files: %w", err)
	}

	// Determine has_more: if we got exactly size records, there might be more
	hasMore := len(files) == size

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

// GetFastFileOSSURL get OSS URL for fast file content redirect
// processType: "preview" for image preview (640), "thumbnail" for thumbnail (235), "video" for video first frame, "" for original
// Returns: OSS URL, ContentType, FileName, FileType, error
func (s *IndexerFileService) GetFastFileOSSURL(pinID string, processType string) (string, string, string, string, error) {
	// Get file information
	file, err := s.GetFileByPinID(pinID)
	if err != nil {
		return "", "", "", "", err
	}

	// Check if storage type is OSS
	if file.StorageType != "oss" {
		return "", "", "", "", errors.New("file is not stored in OSS")
	}

	// Check if domain is configured
	if conf.Cfg.Storage.OSS.Domain == "" {
		return "", "", "", "", errors.New("OSS domain is not configured")
	}

	// Build base URL
	baseURL := strings.TrimSuffix(conf.Cfg.Storage.OSS.Domain, "/")
	storagePath := strings.TrimPrefix(file.StoragePath, "/")
	url := fmt.Sprintf("%s/%s", baseURL, storagePath)

	// Determine Content-Type
	contentType := file.ContentType
	if contentType == "" {
		// Fallback to default based on file type
		switch file.FileType {
		case "image":
			contentType = "image/jpeg"
		case "video":
			contentType = "video/mp4"
		case "audio":
			contentType = "audio/mpeg"
		case "text":
			contentType = "text/plain"
		case "document":
			contentType = "application/pdf"
		default:
			contentType = "application/octet-stream"
		}
	}

	// Determine filename
	fileName := file.FileName
	if fileName == "" {
		fileName = pinID
		if file.FileExtension != "" {
			fileName = pinID + file.FileExtension
		}
	}

	// Add process parameters based on file type and process type
	if processType == "" {
		// Original file, no processing
		return url, contentType, fileName, file.FileType, nil
	}

	// Determine process parameter based on file type and process type
	var processParam string
	switch processType {
	case "preview":
		// Image preview: 640px width
		if file.FileType == "image" {
			processParam = OssProcess640
		} else {
			return "", "", "", "", errors.New("preview only supports image files")
		}
	case "thumbnail":
		// Thumbnail: 235px width
		if file.FileType == "image" {
			processParam = OssProcess235
		} else {
			return "", "", "", "", errors.New("thumbnail only supports image files")
		}
	case "video":
		// Video first frame
		if file.FileType == "video" {
			processParam = OssProcessVideoFirstImg
		} else {
			return "", "", "", "", errors.New("video process only supports video files")
		}
	default:
		return "", "", "", "", fmt.Errorf("unknown process type: %s", processType)
	}

	return url + processParam, contentType, fileName, file.FileType, nil
}

// GetFastAvatarOSSURL get OSS URL for fast avatar content redirect
// processType: "preview" for 640px, "thumbnail" for 128x128, "" for original
func (s *IndexerFileService) GetFastAvatarOSSURL(pinID string, processType string) (string, error) {
	// Get avatar information
	avatar, err := s.indexerUserAvatarDAO.GetByPinID(pinID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("avatar not found")
		}
		return "", fmt.Errorf("failed to get avatar: %w", err)
	}

	// Check if OSS domain is configured
	if conf.Cfg.Storage.OSS.Domain == "" {
		return "", errors.New("OSS domain is not configured")
	}

	// Check if storage type is OSS
	if conf.Cfg.Storage.Type != "oss" {
		return "", errors.New("storage type is not OSS")
	}

	// Build base URL
	baseURL := strings.TrimSuffix(conf.Cfg.Storage.OSS.Domain, "/")
	storagePath := strings.TrimPrefix(avatar.Avatar, "/")
	url := fmt.Sprintf("%s/%s", baseURL, storagePath)

	// Add process parameters
	if processType == "" {
		// Original avatar, no processing
		return url, nil
	}

	var processParam string
	switch processType {
	case "preview":
		// Avatar preview: 640px width
		if avatar.FileType == "image" {
			processParam = OssProcess640
		} else {
			return "", errors.New("preview only supports image avatars")
		}
	case "thumbnail":
		// Avatar thumbnail: 128x128
		if avatar.FileType == "image" {
			processParam = OssProcess128
		} else {
			return "", errors.New("thumbnail only supports image avatars")
		}
	default:
		return "", fmt.Errorf("unknown process type: %s", processType)
	}

	return url + processParam, nil
}

// GetFastAvatarOSSURLByMetaID get OSS URL for avatar by MetaID
func (s *IndexerFileService) GetFastAvatarOSSURLByMetaID(metaID string, processType string) (string, error) {
	avatar, err := s.GetLatestAvatarByMetaID(metaID)
	if err != nil {
		return "", err
	}
	return s.GetFastAvatarOSSURL(avatar.PinID, processType)
}

// GetFastAvatarOSSURLByAddress get OSS URL for avatar by address
func (s *IndexerFileService) GetFastAvatarOSSURLByAddress(address string, processType string) (string, error) {
	avatar, err := s.GetLatestAvatarByAddress(address)
	if err != nil {
		return "", err
	}
	return s.GetFastAvatarOSSURL(avatar.PinID, processType)
}
