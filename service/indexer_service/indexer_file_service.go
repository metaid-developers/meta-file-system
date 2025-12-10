package indexer_service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"meta-file-system/conf"
	"meta-file-system/database"
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

// GetPinInfoByPinID get PIN information by PIN ID from collectionPinInfo
func (s *IndexerFileService) GetPinInfoByPinID(pinID string) (*model.IndexerPinInfo, error) {
	if pinID == "" {
		return nil, errors.New("pinID is empty")
	}

	pinInfo, err := database.DB.GetPinInfoByPinID(pinID)
	if err != nil {
		if err == database.ErrNotFound {
			return nil, fmt.Errorf("PIN info not found for PIN ID: %s", pinID)
		}
		return nil, fmt.Errorf("failed to get PIN info: %w", err)
	}

	return pinInfo, nil
}

// GetLatestFileByFirstPinID get latest file information by first PIN ID
func (s *IndexerFileService) GetLatestFileByFirstPinID(firstPinID string) (*model.IndexerFile, error) {
	if firstPinID == "" {
		return nil, errors.New("firstPinID is empty")
	}

	file, err := s.indexerFileDAO.GetLatestFileInfoByFirstPinID(firstPinID)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest file by firstPinID: %w", err)
	}

	return file, nil
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
	if file == nil {
		return nil, errors.New("file not found")
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

// GetLatestFileContentByFirstPinID get latest file content by first PIN ID
func (s *IndexerFileService) GetLatestFileContentByFirstPinID(firstPinID string) ([]byte, string, string, error) {
	// Get latest file info by firstPinID
	file, err := s.GetLatestFileByFirstPinID(firstPinID)
	if err != nil {
		return nil, "", "", err
	}

	// Use the latest file's pinID to get content
	return s.GetFileContent(file.PinID)
}

// GetFileContent get file content by PIN ID
func (s *IndexerFileService) GetFileContent(pinID string) ([]byte, string, string, error) {
	// Get file information
	file, err := s.GetFileByPinID(pinID)
	if err != nil {
		return nil, "", "", err
	}
	if file == nil {
		return nil, "", "", errors.New("file not found")
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

// GetFilesCountByChains get file count for each chain
func (s *IndexerFileService) GetFilesCountByChains() (map[string]int64, error) {
	// Get all sync statuses to know which chains exist
	statuses, err := database.DB.GetAllIndexerSyncStatus()
	if err != nil {
		return nil, fmt.Errorf("failed to get sync statuses: %w", err)
	}

	chainStats := make(map[string]int64)
	for _, status := range statuses {
		count, err := database.DB.GetIndexerFilesCountByChain(status.ChainName)
		if err != nil {
			// Log error but continue with other chains
			chainStats[status.ChainName] = 0
		} else {
			chainStats[status.ChainName] = count
		}
	}

	return chainStats, nil
}

// ============================================================
// Old Avatar methods - DEPRECATED (commented out)
// Use new UserInfo methods instead
// ============================================================

// // ListAvatars get avatar list with cursor pagination
// // cursor: last avatar ID (0 for first page)
// // size: page size
// // Returns: avatars, next_cursor, has_more, error
// func (s *IndexerFileService) ListAvatars(cursor int64, size int) ([]*model.IndexerUserAvatar, int64, bool, error) {
// 	if size < 1 || size > 100 {
// 		size = 20
// 	}
//
// 	avatars, err := s.indexerUserAvatarDAO.ListWithCursor(cursor, size)
// 	if err != nil {
// 		return nil, 0, false, fmt.Errorf("failed to list avatars: %w", err)
// 	}
//
// 	// Determine next cursor and has_more
// 	var nextCursor int64
// 	hasMore := false
//
// 	if len(avatars) > 0 {
// 		// Next cursor is the ID of the last avatar
// 		nextCursor = avatars[len(avatars)-1].ID
//
// 		// Check if there are more records
// 		hasMore = len(avatars) == size
// 	}
//
// 	return avatars, nextCursor, hasMore, nil
// }

// // GetLatestAvatarByMetaID get latest avatar information by MetaID
// func (s *IndexerFileService) GetLatestAvatarByMetaID(metaID string) (*model.IndexerUserAvatar, error) {
// 	avatar, err := s.indexerUserAvatarDAO.GetByMetaID(metaID)
// 	if err != nil {
// 		if errors.Is(err, gorm.ErrRecordNotFound) {
// 			return nil, errors.New("avatar not found")
// 		}
// 		return nil, fmt.Errorf("failed to get avatar: %w", err)
// 	}
// 	return avatar, nil
// }

// // GetLatestAvatarByAddress get latest avatar information by address
// func (s *IndexerFileService) GetLatestAvatarByAddress(address string) (*model.IndexerUserAvatar, error) {
// 	avatar, err := s.indexerUserAvatarDAO.GetByAddress(address)
// 	if err != nil {
// 		if errors.Is(err, gorm.ErrRecordNotFound) {
// 			return nil, errors.New("avatar not found")
// 		}
// 		return nil, fmt.Errorf("failed to get avatar: %w", err)
// 	}
// 	return avatar, nil
// }

// // GetAvatarContent get avatar content by PIN ID
// func (s *IndexerFileService) GetAvatarContent(pinID string) ([]byte, string, string, error) {
// 	// Get avatar information
// 	avatar, err := s.indexerUserAvatarDAO.GetByPinID(pinID)
// 	if err != nil {
// 		if errors.Is(err, gorm.ErrRecordNotFound) {
// 			return nil, "", "", errors.New("avatar not found")
// 		}
// 		return nil, "", "", fmt.Errorf("failed to get avatar: %w", err)
// 	}
//
// 	// Read avatar content from storage layer
// 	content, err := s.storage.Get(avatar.Avatar)
// 	if err != nil {
// 		return nil, "", "", fmt.Errorf("failed to get avatar content: %w", err)
// 	}
//
// 	// Generate filename from PinID and extension
// 	fileName := avatar.PinID
// 	if avatar.FileExtension != "" {
// 		fileName = avatar.PinID + avatar.FileExtension
// 	}
//
// 	return content, avatar.ContentType, fileName, nil
// }

// ============================================================
// New UserInfo methods
// ============================================================

// GetUserInfoByMetaID get user information by MetaID
func (s *IndexerFileService) GetUserInfoByMetaID(metaID string) (*model.IndexerUserInfo, error) {
	// Try to get from cache first
	cacheKey := "user:metaid:" + metaID
	var cachedUser model.IndexerUserInfo
	if err := database.GetCache(cacheKey, &cachedUser); err == nil {
		// Cache hit
		return &cachedUser, nil
	}

	// Cache miss, query from database
	// Get latest user name
	nameInfo, err := database.DB.GetLatestUserNameInfo(metaID)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		return nil, fmt.Errorf("failed to get user name info: %w", err)
	}

	// Get latest user avatar
	avatarInfo, err := database.DB.GetLatestUserAvatarInfo(metaID)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		return nil, fmt.Errorf("failed to get user avatar info: %w", err)
	}

	// Get latest user chat public key
	chatPubKeyInfo, err := database.DB.GetLatestUserChatPublicKeyInfo(metaID)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		return nil, fmt.Errorf("failed to get user chat public key info: %w", err)
	}

	//get meta id address
	address, err := database.DB.GetAddressByMetaID(metaID)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		return nil, fmt.Errorf("failed to get meta id address: %w", err)
	}

	// Build user info
	userInfo := &model.IndexerUserInfo{
		MetaId:  metaID,
		Address: address,
	}

	if nameInfo != nil {
		userInfo.Name = nameInfo.Name
		userInfo.NamePinId = nameInfo.PinID
		userInfo.ChainName = nameInfo.ChainName
		userInfo.BlockHeight = nameInfo.BlockHeight
		userInfo.Timestamp = nameInfo.Timestamp
	}

	if avatarInfo != nil {
		userInfo.Avatar = avatarInfo.AvatarUrl
		userInfo.AvatarPinId = avatarInfo.PinID
		// Use avatar's timestamp if it's later
		if avatarInfo.Timestamp < userInfo.Timestamp {
			userInfo.Timestamp = avatarInfo.Timestamp
			userInfo.BlockHeight = avatarInfo.BlockHeight
			userInfo.ChainName = avatarInfo.ChainName
		}
	}

	if chatPubKeyInfo != nil {
		userInfo.ChatPublicKey = chatPubKeyInfo.ChatPublicKey
		userInfo.ChatPublicKeyPinId = chatPubKeyInfo.PinID
		// Use chat public key's timestamp if it's later
		if chatPubKeyInfo.Timestamp < userInfo.Timestamp {
			userInfo.Timestamp = chatPubKeyInfo.Timestamp
			userInfo.BlockHeight = chatPubKeyInfo.BlockHeight
			userInfo.ChainName = chatPubKeyInfo.ChainName
		}
	}

	// Set cache
	if err := database.SetCache(cacheKey, userInfo); err != nil {
		log.Printf("⚠️  Failed to set cache for MetaID %s: %v", metaID, err)
	}

	return userInfo, nil
}

// GetUserInfoByAddress get user information by address
func (s *IndexerFileService) GetUserInfoByAddress(address string) (*model.IndexerUserInfo, error) {
	// Try to get from cache first
	cacheKey := "user:address:" + address
	var cachedUser model.IndexerUserInfo
	if err := database.GetCache(cacheKey, &cachedUser); err == nil {
		// Cache hit
		return &cachedUser, nil
	}

	// Cache miss, query from database
	// Calculate MetaID from address (SHA256)
	metaID := calculateMetaIDFromAddress(address)

	userInfo, err := s.GetUserInfoByMetaID(metaID)
	if err != nil {
		return nil, err
	}

	userInfo.Address = address

	// Set cache
	if err := database.SetCache(cacheKey, userInfo); err != nil {
		log.Printf("⚠️  Failed to set cache for address %s: %v", address, err)
	}

	return userInfo, nil
}

// GetUserInfoHistoryByKey get all user info history (name, avatar, chat public key) by MetaID or Address
func (s *IndexerFileService) GetUserInfoHistoryByKey(key string) (*model.UserInfoHistory, error) {
	if key == "" {
		return nil, errors.New("key is required")
	}

	history, err := database.DB.GetUserInfoHistoryByKey(key)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, fmt.Errorf("user info history not found for key: %s", key)
		}
		return nil, fmt.Errorf("failed to get user info history: %w", err)
	}

	return history, nil
}

// SearchUserInfo fuzzy search user info by keyword and keytype
// keytype: "metaid" (fuzzy match metaid) or "name" (fuzzy match name)
// limit: maximum number of results to return
// Returns list of matching users
func (s *IndexerFileService) SearchUserInfo(keyword string, keytype string, limit int) ([]*model.IndexerUserInfo, error) {
	if keyword == "" {
		return nil, errors.New("keyword is required")
	}

	if keytype == "" {
		keytype = "metaid" // Default to metaid
	}

	if limit < 1 {
		limit = 10 // Default limit
	}

	// Fuzzy search from cache
	if keytype == "metaid" {
		return s.fuzzySearchByMetaID(keyword, limit)
	} else if keytype == "name" {
		return s.fuzzySearchByName(keyword, limit)
	} else {
		return nil, fmt.Errorf("invalid keytype: %s (expected: metaid or name)", keytype)
	}
}

// fuzzySearchByMetaID fuzzy search users by MetaID from cache
func (s *IndexerFileService) fuzzySearchByMetaID(keyword string, limit int) ([]*model.IndexerUserInfo, error) {
	// Get all user name mappings from cache (also contains metaID)
	userMap, err := database.GetAllHashFields("user:name:index")
	if err != nil || len(userMap) == 0 {
		// Cache miss or empty, rebuild cache
		if err := s.rebuildUserNameCache(); err != nil {
			log.Printf("⚠️  Failed to rebuild user name cache: %v", err)
			return nil, errors.New("search is not available (cache not ready)")
		}

		// Retry
		userMap, err = database.GetAllHashFields("user:name:index")
		if err != nil || len(userMap) == 0 {
			return nil, errors.New("no users found")
		}
	}

	// Fuzzy match: find all metaIDs that contain the keyword (case-insensitive)
	keywordLower := strings.ToLower(keyword)
	var matchedMetaIDs []string

	for metaID := range userMap {
		// Stop if we have enough matches
		if len(matchedMetaIDs) >= limit {
			break
		}

		metaIDLower := strings.ToLower(metaID)
		if strings.Contains(metaIDLower, keywordLower) {
			matchedMetaIDs = append(matchedMetaIDs, metaID)
		}
	}

	if len(matchedMetaIDs) == 0 {
		return []*model.IndexerUserInfo{}, nil // Return empty list
	}

	// Get full user info for matched MetaIDs (up to limit)
	var users []*model.IndexerUserInfo
	for _, metaID := range matchedMetaIDs {
		if len(users) >= limit {
			break
		}

		userInfo, err := s.GetUserInfoByMetaID(metaID)
		if err != nil {
			log.Printf("⚠️  Failed to get user info for MetaID %s: %v", metaID, err)
			continue
		}
		users = append(users, userInfo)
	}

	return users, nil
}

// fuzzySearchByName fuzzy search users by name from cache
func (s *IndexerFileService) fuzzySearchByName(keyword string, limit int) ([]*model.IndexerUserInfo, error) {
	// Get all user name mappings from cache
	userMap, err := database.GetAllHashFields("user:name:index")
	if err != nil || len(userMap) == 0 {
		// Cache miss or empty, rebuild cache
		if err := s.rebuildUserNameCache(); err != nil {
			log.Printf("⚠️  Failed to rebuild user name cache: %v", err)
			return nil, errors.New("search is not available (cache not ready)")
		}

		// Retry
		userMap, err = database.GetAllHashFields("user:name:index")
		if err != nil || len(userMap) == 0 {
			return nil, errors.New("no users found")
		}
	}

	// Fuzzy match: find all names that contain the keyword (case-insensitive)
	keywordLower := strings.ToLower(keyword)
	var matchedMetaIDs []string

	for metaID, nameJSON := range userMap {
		// Stop if we have enough matches
		if len(matchedMetaIDs) >= limit {
			break
		}

		var name string
		if err := json.Unmarshal([]byte(nameJSON), &name); err != nil {
			continue
		}

		nameLower := strings.ToLower(name)
		if strings.Contains(nameLower, keywordLower) {
			matchedMetaIDs = append(matchedMetaIDs, metaID)
		}
	}

	if len(matchedMetaIDs) == 0 {
		return []*model.IndexerUserInfo{}, nil // Return empty list
	}

	// Get full user info for matched MetaIDs (up to limit)
	var users []*model.IndexerUserInfo
	for _, metaID := range matchedMetaIDs {
		if len(users) >= limit {
			break
		}

		userInfo, err := s.GetUserInfoByMetaID(metaID)
		if err != nil {
			log.Printf("⚠️  Failed to get user info for MetaID %s: %v", metaID, err)
			continue
		}
		users = append(users, userInfo)
	}

	return users, nil
}

// rebuildUserNameCache rebuild user name cache from database
func (s *IndexerFileService) rebuildUserNameCache() error {
	log.Println("🔄 Rebuilding user name cache...")

	if !database.IsRedisEnabled() {
		return errors.New("redis is not enabled")
	}

	// Get all users
	metaIdTimestamps, _, _, err := database.DB.ListMetaIdsByTimestamp(0, 10000)
	if err != nil {
		return fmt.Errorf("failed to list MetaIDs: %w", err)
	}

	count := 0
	for _, metaIdTs := range metaIdTimestamps {
		nameInfo, err := database.DB.GetLatestUserNameInfo(metaIdTs.MetaId)
		if err != nil || nameInfo == nil {
			continue
		}

		// Store in Redis hash: user:name:index
		// field: metaID, value: name
		if err := database.SetHashField("user:name:index", metaIdTs.MetaId, nameInfo.Name); err != nil {
			log.Printf("⚠️  Failed to cache user name for MetaID %s: %v", metaIdTs.MetaId, err)
			continue
		}

		count++
		if count%100 == 0 {
			log.Printf("Progress: Cached %d user names...", count)
		}
	}

	log.Printf("✅ User name cache rebuilt: %d users cached", count)
	return nil
}

// calculateMetaIDFromAddress calculate MetaID from address (SHA256 hash)
func calculateMetaIDFromAddress(address string) string {
	if address == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(address))
	return hex.EncodeToString(hash[:])
}

// GetLatestFastFileOSSURLByFirstPinID get latest OSS URL for fast file content redirect by first PIN ID
// processType: "preview" for image preview (640), "thumbnail" for thumbnail (235), "video" for video first frame, "" for original
// Returns: OSS URL, ContentType, FileName, FileType, error
func (s *IndexerFileService) GetLatestFastFileOSSURLByFirstPinID(firstPinID string, processType string) (string, string, string, string, error) {
	// Get latest file info by firstPinID
	file, err := s.GetLatestFileByFirstPinID(firstPinID)
	if err != nil {
		return "", "", "", "", err
	}

	// Use the latest file's pinID to get OSS URL
	return s.GetFastFileOSSURL(file.PinID, processType)
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

	if file == nil {
		return "", "", "", "", errors.New("file not found")
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

// // GetFastAvatarOSSURL get OSS URL for fast avatar content redirect
// // processType: "preview" for 640px, "thumbnail" for 128x128, "" for original
// func (s *IndexerFileService) GetFastAvatarOSSURL(pinID string, processType string) (string, error) {
// 	// Get avatar information
// 	avatar, err := s.indexerUserAvatarDAO.GetByPinID(pinID)
// 	if err != nil {
// 		if errors.Is(err, gorm.ErrRecordNotFound) {
// 			return "", errors.New("avatar not found")
// 		}
// 		return "", fmt.Errorf("failed to get avatar: %w", err)
// 	}
//
// 	// Check if OSS domain is configured
// 	if conf.Cfg.Storage.OSS.Domain == "" {
// 		return "", errors.New("OSS domain is not configured")
// 	}
//
// 	// Check if storage type is OSS
// 	if conf.Cfg.Storage.Type != "oss" {
// 		return "", errors.New("storage type is not OSS")
// 	}
//
// 	// Build base URL
// 	baseURL := strings.TrimSuffix(conf.Cfg.Storage.OSS.Domain, "/")
// 	storagePath := strings.TrimPrefix(avatar.Avatar, "/")
// 	url := fmt.Sprintf("%s/%s", baseURL, storagePath)
//
// 	// Add process parameters
// 	if processType == "" {
// 		// Original avatar, no processing
// 		return url, nil
// 	}
//
// 	var processParam string
// 	switch processType {
// 	case "preview":
// 		// Avatar preview: 640px width
// 		if avatar.FileType == "image" {
// 			processParam = OssProcess640
// 		} else {
// 			return "", errors.New("preview only supports image avatars")
// 		}
// 	case "thumbnail":
// 		// Avatar thumbnail: 128x128
// 		if avatar.FileType == "image" {
// 			processParam = OssProcess128
// 		} else {
// 			return "", errors.New("thumbnail only supports image avatars")
// 		}
// 	default:
// 		return "", fmt.Errorf("unknown process type: %s", processType)
// 	}
//
// 	return url + processParam, nil
// }

// // GetFastAvatarOSSURLByMetaID get OSS URL for avatar by MetaID
// func (s *IndexerFileService) GetFastAvatarOSSURLByMetaID(metaID string, processType string) (string, error) {
// 	avatar, err := s.GetLatestAvatarByMetaID(metaID)
// 	if err != nil {
// 		return "", err
// 	}
// 	return s.GetFastAvatarOSSURL(avatar.PinID, processType)
// }

// // GetFastAvatarOSSURLByAddress get OSS URL for avatar by address
// func (s *IndexerFileService) GetFastAvatarOSSURLByAddress(address string, processType string) (string, error) {
// 	avatar, err := s.GetLatestAvatarByAddress(address)
// 	if err != nil {
// 		return "", err
// 	}
// 	return s.GetFastAvatarOSSURL(avatar.PinID, processType)
// }

// ============================================================
// New UserInfo List methods
// ============================================================

// GetUserInfoList get user info list with pagination
// This method uses MetaIdTimestamp collection to get users ordered by earliest activity timestamp
func (s *IndexerFileService) GetUserInfoList(cursor int64, size int) ([]*model.IndexerUserInfo, int64, bool, int64, error) {
	if size < 1 || size > 100 {
		size = 20
	}

	// Get total count of users
	total, err := database.DB.GetMetaIDCount()
	if err != nil {
		log.Printf("⚠️  Failed to get MetaID count: %v", err)
		total = 0 // Continue even if count fails
	}

	// Get MetaID list ordered by timestamp (descending)
	metaIdTimestamps, nextCursor, hasMore, err := database.DB.ListMetaIdsByTimestamp(cursor, size)
	if err != nil {
		return nil, 0, false, 0, fmt.Errorf("failed to list MetaIDs by timestamp: %w", err)
	}

	// Build user info for each MetaID
	var users []*model.IndexerUserInfo
	for _, metaIdTs := range metaIdTimestamps {
		userInfo, err := s.GetUserInfoByMetaID(metaIdTs.MetaId)
		if err != nil {
			// Skip users that can't be retrieved
			continue
		}
		users = append(users, userInfo)
	}

	return users, nextCursor, hasMore, total, nil
}

// GetAvatarOSSURLByMetaID get avatar OSS URL or content by MetaID
// Returns: (ossURL, contentType, fileName, fileType, isOSS, error)
func (s *IndexerFileService) GetAvatarOSSURLByMetaID(metaID string) (string, string, string, string, bool, error) {
	// Get avatar info by MetaID
	avatarInfo, err := database.DB.GetLatestUserAvatarInfo(metaID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return "", "", "", "", false, errors.New("avatar not found")
		}
		return "", "", "", "", false, fmt.Errorf("failed to get avatar info: %w", err)
	}

	// Check if avatar has OSS URL
	if avatarInfo.AvatarUrl == "" {
		return "", "", "", "", false, errors.New("avatar URL not available")
	}

	// Determine filename
	fileName := avatarInfo.PinID
	if avatarInfo.FileExtension != "" {
		fileName = avatarInfo.PinID + avatarInfo.FileExtension
	}

	// Check if it's an OSS URL (starts with http:// or https://)
	isOSSURL := strings.HasPrefix(avatarInfo.AvatarUrl, "http://") ||
		strings.HasPrefix(avatarInfo.AvatarUrl, "https://")

	return avatarInfo.AvatarUrl, avatarInfo.ContentType, fileName, avatarInfo.FileType, isOSSURL, nil
}

// GetAvatarContentByMetaID get avatar content by MetaID (from storage)
// Returns: (content, contentType, fileName, error)
func (s *IndexerFileService) GetAvatarContentByMetaID(metaID string) ([]byte, string, string, error) {
	// Get avatar info by MetaID
	avatarInfo, err := database.DB.GetLatestUserAvatarInfo(metaID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, "", "", errors.New("avatar not found")
		}
		return nil, "", "", fmt.Errorf("failed to get avatar info: %w", err)
	}

	// Read avatar content from storage
	content, err := s.storage.Get(avatarInfo.Avatar)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to get avatar content from storage: %w", err)
	}

	// Determine filename
	fileName := avatarInfo.PinID
	if avatarInfo.FileExtension != "" {
		fileName = avatarInfo.PinID + avatarInfo.FileExtension
	}

	contentType := avatarInfo.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return content, contentType, fileName, nil
}

// GetAvatarContentByPinID get specific avatar content by avatar PIN ID
// Returns: (content, contentType, fileName, error)
func (s *IndexerFileService) GetAvatarContentByPinID(avatarPinID string) ([]byte, string, string, error) {
	// Get avatar info by PinID
	avatarInfo, err := database.DB.GetUserAvatarInfoByPinID(avatarPinID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, "", "", errors.New("avatar not found")
		}
		return nil, "", "", fmt.Errorf("failed to get avatar info: %w", err)
	}

	// Read avatar content from storage
	content, err := s.storage.Get(avatarInfo.Avatar)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to get avatar content from storage: %w", err)
	}

	// Determine filename
	fileName := avatarInfo.PinID
	if avatarInfo.FileExtension != "" {
		fileName = avatarInfo.PinID + avatarInfo.FileExtension
	}

	contentType := avatarInfo.ContentType
	if contentType == "" {
		contentType = "image/jpeg" // Default for avatars
	}

	return content, contentType, fileName, nil
}

// GetFastAvatarOSSURLByPinID get OSS URL for fast avatar content redirect by avatar PIN ID
// processType: "preview" for preview (640), "thumbnail" for thumbnail (128), "" for original
// Returns: OSS URL, ContentType, FileName, FileType, error
func (s *IndexerFileService) GetFastAvatarOSSURLByPinID(avatarPinID string, processType string) (string, string, string, string, error) {
	// Get avatar info by PinID
	avatarInfo, err := database.DB.GetUserAvatarInfoByPinID(avatarPinID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return "", "", "", "", errors.New("avatar not found")
		}
		return "", "", "", "", fmt.Errorf("failed to get avatar info: %w", err)
	}

	// Check if avatar has OSS URL
	if avatarInfo.AvatarUrl == "" {
		return "", "", "", "", errors.New("avatar URL not available, please use direct content endpoint")
	}

	// Determine filename
	fileName := avatarInfo.PinID
	if avatarInfo.FileExtension != "" {
		fileName = avatarInfo.PinID + avatarInfo.FileExtension
	}

	// Determine content type
	contentType := avatarInfo.ContentType
	if contentType == "" {
		contentType = "image/jpeg" // Default for avatars
	}

	// Build OSS URL with processing parameters
	ossURL := avatarInfo.AvatarUrl

	// Apply OSS processing based on processType
	if processType == "preview" {
		// Preview size (640px width)
		ossURL += OssProcess640
	} else if processType == "thumbnail" {
		// Thumbnail size (128x128)
		ossURL += OssProcess128
	}
	// For empty processType, return original URL

	return ossURL, contentType, fileName, "image", nil
}
