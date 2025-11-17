package indexer_service

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"meta-file-system/conf"
	"meta-file-system/indexer"
	"meta-file-system/model"
	"meta-file-system/model/dao"
	"meta-file-system/storage"
)

// IndexerService indexer service
type IndexerService struct {
	scanner              *indexer.BlockScanner
	fileDAO              *dao.FileDAO
	indexerFileDAO       *dao.IndexerFileDAO
	indexerUserAvatarDAO *dao.IndexerUserAvatarDAO
	syncStatusDAO        *dao.IndexerSyncStatusDAO
	storage              storage.Storage
	chainType            indexer.ChainType
	parser               *indexer.MetaIDParser
}

// NewIndexerService create indexer service instance
func NewIndexerService(storage storage.Storage) (*IndexerService, error) {
	return NewIndexerServiceWithChain(storage, indexer.ChainTypeMVC)
}

// NewIndexerServiceWithChain create indexer service instance with specified chain type
func NewIndexerServiceWithChain(storage storage.Storage, chainType indexer.ChainType) (*IndexerService, error) {
	chainName := string(chainType)
	syncStatusDAO := dao.NewIndexerSyncStatusDAO()

	// Get current sync height from database
	var currentSyncHeight int64 = 0
	syncStatus, err := syncStatusDAO.GetByChainName(chainName)
	if err == nil && syncStatus != nil && syncStatus.CurrentSyncHeight > 0 {
		currentSyncHeight = syncStatus.CurrentSyncHeight
		log.Printf("Found existing sync status for %s chain, current sync height: %d", chainName, currentSyncHeight)
	}

	// Determine start height based on configuration
	configStartHeight := conf.Cfg.Indexer.StartHeight
	if configStartHeight == 0 {
		// Use chain-specific init height if not specified
		if chainType == indexer.ChainTypeMVC {
			configStartHeight = conf.Cfg.Indexer.MvcInitBlockHeight
		} else if chainType == indexer.ChainTypeBTC {
			configStartHeight = conf.Cfg.Indexer.BtcInitBlockHeight
		}
	}

	// Choose the higher value between config and current sync height
	startHeight := configStartHeight
	if currentSyncHeight > startHeight {
		startHeight = currentSyncHeight + 1 // Continue from next block
		log.Printf("Using current sync height + 1 as start height: %d", startHeight)
	} else if configStartHeight > 0 {
		log.Printf("Using configured start height: %d", startHeight)
	} else {
		// Default to 0 if no config and no sync status
		startHeight = 0
		log.Printf("No start height configured, starting from: %d", startHeight)
	}

	log.Printf("Indexer service will start from block height: %d (chain: %s)", startHeight, chainType)

	// Create block scanner with chain type
	scanner := indexer.NewBlockScannerWithChain(
		conf.Cfg.Chain.RpcUrl,
		conf.Cfg.Chain.RpcUser,
		conf.Cfg.Chain.RpcPass,
		startHeight,
		conf.Cfg.Indexer.ScanInterval,
		chainType,
	)

	// Enable ZMQ if configured
	if conf.Cfg.Indexer.ZmqEnabled && conf.Cfg.Indexer.ZmqAddress != "" {
		scanner.EnableZMQ(conf.Cfg.Indexer.ZmqAddress)
		log.Printf("ZMQ real-time monitoring enabled: %s", conf.Cfg.Indexer.ZmqAddress)
	} else {
		log.Println("ZMQ real-time monitoring disabled")
	}

	// Create parser
	parser := indexer.NewMetaIDParser("")
	parser.SetBlockScanner(scanner)

	service := &IndexerService{
		scanner:              scanner,
		fileDAO:              dao.NewFileDAO(),
		indexerFileDAO:       dao.NewIndexerFileDAO(),
		indexerUserAvatarDAO: dao.NewIndexerUserAvatarDAO(),
		syncStatusDAO:        dao.NewIndexerSyncStatusDAO(),
		storage:              storage,
		chainType:            chainType,
		parser:               parser,
	}

	// Initialize sync status in database
	if err := service.initializeSyncStatus(startHeight); err != nil {
		log.Printf("Failed to initialize sync status: %v", err)
	}

	return service, nil
}

// initializeSyncStatus initialize sync status in database
func (s *IndexerService) initializeSyncStatus(startHeight int64) error {
	chainName := string(s.chainType)

	// Try to get existing status
	existingStatus, err := s.syncStatusDAO.GetByChainName(chainName)
	if err == nil && existingStatus != nil {
		log.Printf("Sync status already exists for %s chain, current sync height: %d", chainName, existingStatus.CurrentSyncHeight)
		return nil
	}

	// Create initial status (only if not exists)
	initialHeight := int64(0)
	if startHeight > 0 {
		initialHeight = startHeight - 1 // Will be updated when first block is scanned
	}

	status := &model.IndexerSyncStatus{
		ChainName:         chainName,
		CurrentSyncHeight: initialHeight,
	}

	if err := s.syncStatusDAO.CreateOrUpdate(status); err != nil {
		return fmt.Errorf("failed to create sync status: %w", err)
	}

	log.Printf("Initialized sync status for %s chain with height: %d", chainName, initialHeight)
	return nil
}

// Start start indexer service
func (s *IndexerService) Start() {
	log.Println("Indexer service starting...")

	// Start block scanning with block complete callback
	s.scanner.Start(s.handleTransaction, s.onBlockComplete)
}

// GetScanner get block scanner instance
func (s *IndexerService) GetScanner() *indexer.BlockScanner {
	return s.scanner
}

// onBlockComplete called after each block is successfully scanned
func (s *IndexerService) onBlockComplete(height int64) error {
	chainName := string(s.chainType)

	// Update current sync height
	if err := s.syncStatusDAO.UpdateCurrentSyncHeight(chainName, height); err != nil {
		return fmt.Errorf("failed to update sync height: %w", err)
	}

	return nil
}

// handleTransaction handle transaction
// tx is interface{} to support both BTC (*btcwire.MsgTx) and MVC (*wire.MsgTx) transactions
func (s *IndexerService) handleTransaction(tx interface{}, metaDataTx *indexer.MetaIDDataTx, height, timestamp int64) error {
	if metaDataTx == nil || len(metaDataTx.MetaIDData) == 0 {
		return nil
	}

	// txID := metaDataTx.TxID
	// chainNameFromTx := metaDataTx.ChainName
	// pinId := metaDataTx.MetaIDData[0].PinID

	// log.Printf("Found MetaID pinId: %s,  transaction: %s at height %d (chain: %s), PIN count: %d",
	// 	pinId, txID, height, chainNameFromTx, len(metaDataTx.MetaIDData))

	// Process each PIN in the transaction
	for _, metaData := range metaDataTx.MetaIDData {
		// Check if this is a file PIN
		if isFilePath(metaData.Path) {
			log.Printf("Processing file PIN: %s (path: %s, operation: %s)",
				metaData.PinID, metaData.Path, metaData.Operation)

			// Check if already exists
			existingFile, err := s.indexerFileDAO.GetByPinID(metaData.PinID)
			if err == nil && existingFile != nil {
				log.Printf("File PIN already indexed: %s", metaData.PinID)

				// Update file content height
				if existingFile.BlockHeight < height && height > 0 {
					existingFile.BlockHeight = height
					if err := s.indexerFileDAO.Update(existingFile); err != nil {
						log.Printf("Failed to update file content height: %v", err)
					}
				}

				continue
			}

			// Process file content
			if err := s.processFileContent(metaData, height, timestamp); err != nil {
				log.Printf("Failed to process file content for PIN %s: %v", metaData.PinID, err)
				// Continue processing other PINs even if one fails
				continue
			}
		} else if isAvatarPath(metaData.Path) {
			// Check if this is an avatar PIN
			log.Printf("Processing avatar PIN: %s (path: %s, operation: %s)",
				metaData.PinID, metaData.Path, metaData.Operation)

			// Check if already exists
			existingAvatar, err := s.indexerUserAvatarDAO.GetByPinID(metaData.PinID)
			if err == nil && existingAvatar != nil {
				log.Printf("Avatar PIN already indexed: %s", metaData.PinID)

				// Update avatar content height
				if existingAvatar.BlockHeight < height && height > 0 {
					existingAvatar.BlockHeight = height
					if err := s.indexerUserAvatarDAO.Update(existingAvatar); err != nil {
						log.Printf("Failed to update avatar content height: %v", err)
					}
				}

				continue
			}

			// Process avatar content
			if err := s.processAvatarContent(metaData, height, timestamp); err != nil {
				log.Printf("Failed to process avatar content for PIN %s: %v", metaData.PinID, err)
				// Continue processing other PINs even if one fails
				continue
			}
		} else {
			// log.Printf("Skipping PIN: %s (path: %s)", metaData.PinID, metaData.Path)
		}
	}

	return nil
}

// isFilePath check if path is a file path
func isFilePath(path string) bool {
	// Check if path starts with /file or contains /file
	return strings.HasPrefix(path, "/file") || strings.Contains(path, "/file")
}

// isAvatarPath check if path is an avatar path
func isAvatarPath(path string) bool {
	// Check if path starts with /info/avatar or contains /info/avatar
	return strings.HasPrefix(path, "/info/avatar") || strings.Contains(path, "/info/avatar")
}

// processFileContent process and save file content
func (s *IndexerService) processFileContent(metaData *indexer.MetaIDData, height, timestamp int64) error {
	// Get real creator address from CreatorInputLocation if available
	creatorAddress := metaData.CreatorAddress
	if metaData.CreatorInputLocation != "" {
		realAddress, err := s.parser.FindCreatorAddressFromCreatorInputLocation(metaData.CreatorInputLocation, s.chainType)
		if err != nil {
			log.Printf("Failed to get creator address from location %s: %v, using fallback address",
				metaData.CreatorInputLocation, err)
		} else {
			creatorAddress = realAddress
			log.Printf("Found real creator address: %s (from location: %s)", realAddress, metaData.CreatorInputLocation)
		}
	}

	// Extract file name from path
	fileName := extractFileName(metaData.Path)

	// Detect real content type from file content
	realContentType := detectRealContentType(metaData.Content, metaData.ContentType)

	// Extract file extension (using real content type and path)
	fileExtension := extractFileExtension(metaData.Path, realContentType, metaData.Content)

	// Calculate file hashes
	fileMd5 := calculateMD5(metaData.Content)
	fileHash := calculateSHA256(metaData.Content)

	// Detect file type from real content type
	fileType := detectFileType(realContentType)

	// Determine storage path: indexer/{chain}/{txid}/{pinid}{extension}
	// Use pinID as filename to ensure uniqueness, with file extension
	storagePath := fmt.Sprintf("indexer/%s/%s%s",
		metaData.ChainName,
		metaData.PinID,
		fileExtension)

	// Save file to storage
	storageType := "local"
	if conf.Cfg.Storage.Type == "oss" {
		storageType = "oss"
	}

	if err := s.storage.Save(storagePath, metaData.Content); err != nil {
		return fmt.Errorf("failed to save file to storage: %w", err)
	}

	log.Printf("File saved to storage: %s (size: %d bytes)", storagePath, len(metaData.Content))

	// Calculate Creator MetaID (SHA256 of address)
	creatorMetaID := calculateMetaID(creatorAddress)

	// Create database record
	indexerFile := &model.IndexerFile{
		PinID:          metaData.PinID,
		TxID:           metaData.TxID,
		Vout:           metaData.Vout,
		Path:           metaData.Path,
		Operation:      metaData.Operation,
		ParentPath:     metaData.ParentPath,
		Encryption:     metaData.Encryption,
		Version:        metaData.Version,
		ContentType:    metaData.ContentType,
		FileType:       fileType,
		FileExtension:  fileExtension,
		FileName:       fileName,
		FileSize:       int64(len(metaData.Content)),
		FileMd5:        fileMd5,
		FileHash:       fileHash,
		StorageType:    storageType,
		StoragePath:    storagePath,
		ChainName:      metaData.ChainName,
		BlockHeight:    height,
		Timestamp:      timestamp,
		CreatorMetaId:  creatorMetaID,
		CreatorAddress: creatorAddress, // Use real creator address
		OwnerAddress:   metaData.OwnerAddress,
		OwnerMetaId:    calculateMetaID(metaData.OwnerAddress),
		Status:         model.StatusSuccess,
		State:          0,
	}

	// Save to database
	if err := s.indexerFileDAO.Create(indexerFile); err != nil {
		return fmt.Errorf("failed to save file to database: %w", err)
	}

	log.Printf("File indexed successfully: PIN=%s, Path=%s, Type=%s, Ext=%s, Size=%d",
		metaData.PinID, metaData.Path, fileType, fileExtension, len(metaData.Content))

	return nil
}

// processAvatarContent process and save avatar content
func (s *IndexerService) processAvatarContent(metaData *indexer.MetaIDData, height, timestamp int64) error {
	// Get real creator address from CreatorInputLocation if available
	creatorAddress := metaData.CreatorAddress
	if metaData.CreatorInputLocation != "" {
		realAddress, err := s.parser.FindCreatorAddressFromCreatorInputLocation(metaData.CreatorInputLocation, s.chainType)
		if err != nil {
			log.Printf("Failed to get creator address from location %s: %v, using fallback address",
				metaData.CreatorInputLocation, err)
		} else {
			creatorAddress = realAddress
			log.Printf("Found real creator address for avatar: %s (from location: %s)", realAddress, metaData.CreatorInputLocation)
		}
	}

	// Detect real content type from file content
	realContentType := detectRealContentType(metaData.Content, metaData.ContentType)

	// Extract file extension from real content type
	fileExtension := extractAvatarFileExtension(realContentType, metaData.Content)

	// Calculate file hashes
	fileMd5 := calculateMD5(metaData.Content)
	fileHash := calculateSHA256(metaData.Content)

	// Detect file type from real content type
	fileType := detectFileType(realContentType)

	// Determine storage path: indexer/avatar/{chain}/{txid}/{pinid}{extension}
	// Use pinID as filename to ensure uniqueness, with file extension
	storagePath := fmt.Sprintf("indexer/avatar/%s/%s/%s%s",
		metaData.ChainName,
		metaData.TxID,
		metaData.PinID,
		fileExtension)

	// Save file to storage
	if err := s.storage.Save(storagePath, metaData.Content); err != nil {
		return fmt.Errorf("failed to save avatar to storage: %w", err)
	}

	log.Printf("Avatar saved to storage: %s (size: %d bytes)", storagePath, len(metaData.Content))

	// Calculate Creator MetaID (SHA256 of address)
	creatorMetaID := calculateMetaID(creatorAddress)

	// Create database record
	indexerUserAvatar := &model.IndexerUserAvatar{
		PinID:         metaData.PinID,
		TxID:          metaData.TxID,
		MetaId:        creatorMetaID,
		Address:       creatorAddress, // Use real creator address
		Avatar:        storagePath,
		ContentType:   metaData.ContentType,
		FileSize:      int64(len(metaData.Content)),
		FileMd5:       fileMd5,
		FileHash:      fileHash,
		FileExtension: fileExtension,
		FileType:      fileType,
		ChainName:     metaData.ChainName,
		BlockHeight:   height,
		Timestamp:     timestamp,
	}

	// Save to database
	if err := s.indexerUserAvatarDAO.Create(indexerUserAvatar); err != nil {
		return fmt.Errorf("failed to save avatar to database: %w", err)
	}

	log.Printf("Avatar indexed successfully: PIN=%s, Path=%s, Type=%s, Ext=%s, Size=%d, MetaID=%s, Address=%s",
		metaData.PinID, metaData.Path, fileType, fileExtension, len(metaData.Content), creatorMetaID, creatorAddress)

	return nil
}

// extractFileName extract file name from path (may return empty string)
func extractFileName(path string) string {
	// Remove host prefix if exists (e.g., "host:/file/test.jpg" -> "/file/test.jpg")
	if idx := strings.Index(path, ":"); idx != -1 {
		path = path[idx+1:]
	}

	// Get base name
	fileName := filepath.Base(path)

	// If path is just "/file" or similar, fileName will be "file" which is not a real filename
	// Check if it looks like a filename (has extension or is not a common path segment)
	if fileName == "" || fileName == "/" || fileName == "." || fileName == "file" {
		return "" // No filename in path
	}

	return fileName
}

// detectRealContentType detect real content type from file content
func detectRealContentType(content []byte, declaredContentType string) string {
	// Use http.DetectContentType to detect real content type from file content
	// This function reads the first 512 bytes to determine the content type
	detectedType := http.DetectContentType(content)

	// Log if detected type differs from declared type
	if detectedType != declaredContentType {
		log.Printf("Content type mismatch - Declared: %s, Detected: %s", declaredContentType, detectedType)
	}

	// Prefer detected type over declared type for better accuracy
	// But for some specific types that http.DetectContentType can't detect well,
	// we trust the declared type
	if detectedType == "application/octet-stream" && declaredContentType != "" {
		// If detection returns generic binary type but we have a declared type, use declared
		return declaredContentType
	}

	return detectedType
}

// extractFileExtension extract file extension from path, content type, or file content
func extractFileExtension(path string, contentType string, content []byte) string {
	// Remove host prefix if exists
	if idx := strings.Index(path, ":"); idx != -1 {
		path = path[idx+1:]
	}

	// Try to get extension from path first
	ext := filepath.Ext(path)
	if ext != "" && ext != "." {
		return ext
	}

	// If no extension in path, derive from content type
	return contentTypeToExtension(contentType)
}

// extractAvatarFileExtension extract file extension from content type and content for avatar
func extractAvatarFileExtension(contentType string, content []byte) string {
	return contentTypeToExtension(contentType)
}

// contentTypeToExtension map content type to file extension
func contentTypeToExtension(contentType string) string {
	// Remove parameters from content type (e.g., "image/jpeg;binary" -> "image/jpeg")
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = contentType[:idx]
	}
	contentType = strings.TrimSpace(strings.ToLower(contentType))

	// Map content type to file extension
	extensionMap := map[string]string{
		// Images
		"image/jpeg":    ".jpg",
		"image/jpg":     ".jpg",
		"image/png":     ".png",
		"image/gif":     ".gif",
		"image/webp":    ".webp",
		"image/svg+xml": ".svg",
		"image/bmp":     ".bmp",
		"image/tiff":    ".tiff",
		"image/ico":     ".ico",

		// Videos
		"video/mp4":       ".mp4",
		"video/mpeg":      ".mpeg",
		"video/webm":      ".webm",
		"video/ogg":       ".ogv",
		"video/quicktime": ".mov",
		"video/x-msvideo": ".avi",

		// Audio
		"audio/mpeg": ".mp3",
		"audio/mp3":  ".mp3",
		"audio/wav":  ".wav",
		"audio/ogg":  ".ogg",
		"audio/webm": ".weba",
		"audio/aac":  ".aac",
		"audio/flac": ".flac",

		// Documents
		"application/pdf":    ".pdf",
		"application/msword": ".doc",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": ".docx",
		"application/vnd.ms-excel": ".xls",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         ".xlsx",
		"application/vnd.ms-powerpoint":                                             ".ppt",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": ".pptx",

		// Text
		"text/plain":             ".txt",
		"text/html":              ".html",
		"text/css":               ".css",
		"text/javascript":        ".js",
		"application/javascript": ".js",
		"application/json":       ".json",
		"text/xml":               ".xml",
		"application/xml":        ".xml",
		"text/csv":               ".csv",
		"text/markdown":          ".md",

		// Archives
		"application/zip":              ".zip",
		"application/x-rar-compressed": ".rar",
		"application/x-7z-compressed":  ".7z",
		"application/x-tar":            ".tar",
		"application/gzip":             ".gz",
	}

	if ext, ok := extensionMap[contentType]; ok {
		return ext
	}

	// Default: no extension or use generic .bin
	return ""
}

// detectFileType detect file type category from content type
func detectFileType(contentType string) string {
	// Remove parameters from content type
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = contentType[:idx]
	}
	contentType = strings.TrimSpace(strings.ToLower(contentType))

	// Detect file type category
	switch {
	case strings.HasPrefix(contentType, "image/"):
		return "image"
	case strings.HasPrefix(contentType, "video/"):
		return "video"
	case strings.HasPrefix(contentType, "audio/"):
		return "audio"
	case strings.HasPrefix(contentType, "text/"):
		return "text"
	case strings.Contains(contentType, "pdf"):
		return "document"
	case strings.Contains(contentType, "word") || strings.Contains(contentType, "excel") ||
		strings.Contains(contentType, "powerpoint") || strings.Contains(contentType, "document"):
		return "document"
	case strings.Contains(contentType, "zip") || strings.Contains(contentType, "rar") ||
		strings.Contains(contentType, "tar") || strings.Contains(contentType, "gzip") ||
		strings.Contains(contentType, "compressed"):
		return "archive"
	case strings.Contains(contentType, "json") || strings.Contains(contentType, "xml"):
		return "data"
	default:
		return "other"
	}
}

// calculateMD5 calculate MD5 hash of content
func calculateMD5(content []byte) string {
	hash := md5.Sum(content)
	return hex.EncodeToString(hash[:])
}

// calculateSHA256 calculate SHA256 hash of content
func calculateSHA256(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

// calculateMetaID calculate MetaID from address (SHA256 hash)
func calculateMetaID(address string) string {
	if address == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(address))
	return hex.EncodeToString(hash[:])
}
