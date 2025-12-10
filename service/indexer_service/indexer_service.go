package indexer_service

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"meta-file-system/conf"
	"meta-file-system/database"
	"meta-file-system/indexer"
	"meta-file-system/model"
	"meta-file-system/model/dao"
	"meta-file-system/service/common_service/metaid_protocols"
	"meta-file-system/storage"

	"github.com/bitcoinsv/bsvd/wire"
	btcwire "github.com/btcsuite/btcd/wire"
)

// RescanTaskStatus represents the status of a rescan task
type RescanTaskStatus string

const (
	RescanStatusIdle      RescanTaskStatus = "idle"
	RescanStatusRunning   RescanTaskStatus = "running"
	RescanStatusCompleted RescanTaskStatus = "completed"
	RescanStatusCancelled RescanTaskStatus = "cancelled"
	RescanStatusFailed    RescanTaskStatus = "failed"
)

// RescanTask represents a rescan task
type RescanTask struct {
	TaskID          string
	Chain           string
	Status          RescanTaskStatus
	StartHeight     int64
	EndHeight       int64
	CurrentHeight   int64
	ProcessedBlocks int64
	TotalBlocks     int64
	StartTime       time.Time
	ErrorMessage    string
	CancelFunc      context.CancelFunc
	mu              sync.RWMutex
}

// IndexerService indexer service
type IndexerService struct {
	scanner              *indexer.BlockScanner
	fileDAO              *dao.FileDAO
	indexerFileDAO       *dao.IndexerFileDAO
	indexerFileChunkDAO  *dao.IndexerFileChunkDAO
	indexerUserAvatarDAO *dao.IndexerUserAvatarDAO
	syncStatusDAO        *dao.IndexerSyncStatusDAO
	storage              storage.Storage
	chainType            indexer.ChainType
	parser               *indexer.MetaIDParser

	// Multi-chain support
	coordinator  *indexer.MultiChainCoordinator
	isMultiChain bool

	// Rescan task management
	currentRescanTask *RescanTask
	rescanMu          sync.Mutex
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
		indexerFileChunkDAO:  dao.NewIndexerFileChunkDAO(),
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

// NewMultiChainIndexerService create multi-chain indexer service instance
func NewMultiChainIndexerService(storage storage.Storage, chainConfigs []conf.ChainInstanceConfig) (*IndexerService, error) {
	if len(chainConfigs) == 0 {
		return nil, fmt.Errorf("no chain configurations provided")
	}

	log.Printf("Creating multi-chain indexer service with %d chains", len(chainConfigs))

	// Create coordinator
	coordinator := indexer.NewMultiChainCoordinator(conf.Cfg.Indexer.TimeOrderingEnabled)

	// Create service instance
	service := &IndexerService{
		fileDAO:              dao.NewFileDAO(),
		indexerFileDAO:       dao.NewIndexerFileDAO(),
		indexerFileChunkDAO:  dao.NewIndexerFileChunkDAO(),
		indexerUserAvatarDAO: dao.NewIndexerUserAvatarDAO(),
		syncStatusDAO:        dao.NewIndexerSyncStatusDAO(),
		storage:              storage,
		coordinator:          coordinator,
		isMultiChain:         true,
		parser:               indexer.NewMetaIDParser(""),
	}

	// Create scanner for each chain
	for _, chainConfig := range chainConfigs {
		if err := service.addChainScanner(chainConfig); err != nil {
			return nil, fmt.Errorf("failed to add chain %s: %w", chainConfig.Name, err)
		}
	}

	// Set block event handler
	coordinator.SetHandler(service.handleBlockEvent)

	log.Println("Multi-chain indexer service created successfully")
	return service, nil
}

// addChainScanner adds a chain scanner to the coordinator
func (s *IndexerService) addChainScanner(chainConfig conf.ChainInstanceConfig) error {
	// Determine chain type
	var chainType indexer.ChainType
	switch strings.ToLower(chainConfig.Name) {
	case "btc":
		chainType = indexer.ChainTypeBTC
	case "mvc":
		chainType = indexer.ChainTypeMVC
	default:
		return fmt.Errorf("unsupported chain type: %s", chainConfig.Name)
	}

	chainName := string(chainType)
	syncStatusDAO := dao.NewIndexerSyncStatusDAO()

	// Get current sync height from database
	var currentSyncHeight int64 = 0
	syncStatus, err := syncStatusDAO.GetByChainName(chainName)
	if err == nil && syncStatus != nil && syncStatus.CurrentSyncHeight > 0 {
		currentSyncHeight = syncStatus.CurrentSyncHeight
		log.Printf("[%s] Found existing sync status, current sync height: %d", chainName, currentSyncHeight)
	}

	// Determine start height
	startHeight := chainConfig.StartHeight
	if currentSyncHeight > startHeight {
		startHeight = currentSyncHeight + 1
		log.Printf("[%s] Using current sync height + 1 as start height: %d", chainName, startHeight)
	} else if chainConfig.StartHeight > 0 {
		log.Printf("[%s] Using configured start height: %d", chainName, startHeight)
	} else {
		startHeight = 0
		log.Printf("[%s] No start height configured, starting from: %d", chainName, startHeight)
	}

	// Create block scanner
	scanner := indexer.NewBlockScannerWithChain(
		chainConfig.RpcUrl,
		chainConfig.RpcUser,
		chainConfig.RpcPass,
		startHeight,
		conf.Cfg.Indexer.ScanInterval,
		chainType,
	)

	// Enable ZMQ if configured
	if chainConfig.ZmqEnabled && chainConfig.ZmqAddress != "" {
		scanner.EnableZMQ(chainConfig.ZmqAddress)
		log.Printf("[%s] ZMQ real-time monitoring enabled: %s", chainName, chainConfig.ZmqAddress)

		// Set ZMQ transaction handler for this chain
		scanner.SetZMQTransactionHandler(func(tx interface{}, metaDataTx *indexer.MetaIDDataTx) error {
			// Call the same handler but with height = 0 (mempool transaction)
			// and current timestamp for ZMQ transactions
			return s.handleTransaction(tx, metaDataTx, 0, time.Now().UnixMilli())
		})
		log.Printf("[%s] ZMQ transaction handler configured", chainName)
	}

	// Add to coordinator
	if err := s.coordinator.AddChain(chainName, scanner); err != nil {
		return err
	}

	// Initialize sync status in database
	if err := s.initializeSyncStatusForChain(chainName, startHeight); err != nil {
		log.Printf("[%s] Failed to initialize sync status: %v", chainName, err)
	}

	log.Printf("[%s] Chain scanner added successfully", chainName)
	return nil
}

// initializeSyncStatusForChain initialize sync status for a specific chain
func (s *IndexerService) initializeSyncStatusForChain(chainName string, startHeight int64) error {
	// Try to get existing status
	existingStatus, err := s.syncStatusDAO.GetByChainName(chainName)
	if err == nil && existingStatus != nil {
		log.Printf("[%s] Sync status already exists, current sync height: %d", chainName, existingStatus.CurrentSyncHeight)
		return nil
	}

	// Create initial status
	initialHeight := int64(0)
	if startHeight > 0 {
		initialHeight = startHeight - 1
	}

	status := &model.IndexerSyncStatus{
		ChainName:         chainName,
		CurrentSyncHeight: initialHeight,
		CreatedAt:         time.Now(),
	}

	if err := s.syncStatusDAO.CreateOrUpdate(status); err != nil {
		return fmt.Errorf("failed to create sync status: %w", err)
	}

	log.Printf("[%s] Initialized sync status with height: %d", chainName, initialHeight)
	return nil
}

// handleBlockEvent handles a block event from the multi-chain coordinator
func (s *IndexerService) handleBlockEvent(event *indexer.BlockEvent) error {
	log.Printf("[%s] Processing block at height %d (timestamp: %d)",
		event.ChainName, event.Height, event.Timestamp)

	// Determine chain type
	var chainType indexer.ChainType
	switch strings.ToLower(event.ChainName) {
	case "btc":
		chainType = indexer.ChainTypeBTC
	case "mvc":
		chainType = indexer.ChainTypeMVC
	default:
		return fmt.Errorf("unsupported chain type: %s", event.ChainName)
	}

	// Parse MetaID transactions from the block
	parser := indexer.NewMetaIDParser("")

	// Process transactions based on chain type
	if chainType == indexer.ChainTypeBTC {
		btcBlock, ok := event.Block.(*btcwire.MsgBlock)
		if !ok {
			return fmt.Errorf("invalid BTC block type")
		}
		//判断event.Timestamp的位数是否13位，不是的话，则认为是10位，则需要乘以1000
		if len(strconv.FormatInt(event.Timestamp, 10)) != 13 {
			event.Timestamp = event.Timestamp * 1000
		}

		// Process each transaction
		for _, tx := range btcBlock.Transactions {
			metaDataTx, err := parser.ParseAllPINs(tx, chainType)
			if err != nil || metaDataTx == nil {
				continue
			}

			// Handle the transaction
			if err := s.handleTransaction(tx, metaDataTx, event.Height, event.Timestamp); err != nil {
				log.Printf("[%s] Failed to handle transaction %s: %v", event.ChainName, metaDataTx.TxID, err)
			}
		}
	} else {
		mvcBlock, ok := event.Block.(*wire.MsgBlock)
		if !ok {
			return fmt.Errorf("invalid MVC block type")
		}

		//判断event.Timestamp的位数是否13位，不是的话，则认为是10位，则需要乘以1000
		if len(strconv.FormatInt(event.Timestamp, 10)) != 13 {
			event.Timestamp = event.Timestamp * 1000
		}

		// Process each transaction
		for _, tx := range mvcBlock.Transactions {
			metaDataTx, err := parser.ParseAllPINs(tx, chainType)
			if err != nil || metaDataTx == nil {
				continue
			}

			// Handle the transaction
			if err := s.handleTransaction(tx, metaDataTx, event.Height, event.Timestamp); err != nil {
				log.Printf("[%s] Failed to handle transaction %s: %v", event.ChainName, metaDataTx.TxID, err)
			}
		}
	}

	// Update sync status
	if err := s.syncStatusDAO.UpdateCurrentSyncHeight(event.ChainName, event.Height); err != nil {
		return fmt.Errorf("failed to update sync height: %w", err)
	}

	return nil
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
		CreatedAt:         time.Now(),
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

	if s.isMultiChain {
		// Multi-chain mode
		log.Println("Starting in multi-chain mode...")
		if err := s.coordinator.Start(); err != nil {
			log.Fatalf("Failed to start multi-chain coordinator: %v", err)
		}
	} else {
		// Single-chain mode
		log.Println("Starting in single-chain mode...")
		s.scanner.Start(s.handleTransaction, s.onBlockComplete)
	}
}

// GetScanner get block scanner instance (for single-chain mode)
func (s *IndexerService) GetScanner() *indexer.BlockScanner {
	return s.scanner
}

// GetCoordinator get multi-chain coordinator instance (for multi-chain mode)
func (s *IndexerService) GetCoordinator() *indexer.MultiChainCoordinator {
	return s.coordinator
}

// IsMultiChain returns whether the service is running in multi-chain mode
func (s *IndexerService) IsMultiChain() bool {
	return s.isMultiChain
}

// Stop stops the indexer service
func (s *IndexerService) Stop() {
	log.Println("Stopping indexer service...")

	if s.isMultiChain && s.coordinator != nil {
		s.coordinator.Stop()
	} else if s.scanner != nil {
		s.scanner.Stop()
	}

	log.Println("Indexer service stopped")
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
		// Track firstPinID for modify operations
		var firstPinID string
		var firstPath string

		// Handle based on operation type
		if metaData.Operation == "create" {
			// Create operation: use original path and save PIN info for future reference
			firstPinID = metaData.PinID // For create, firstPinID = PinID
			firstPath = metaData.Path   // For create, firstPath = Path

			if !metaid_protocols.IsProtocolPath(firstPath) {
				continue
			}

			pinInfo := &model.IndexerPinInfo{
				PinID:       metaData.PinID,
				FirstPinID:  firstPinID,
				FirstPath:   firstPath,
				Path:        metaData.Path,
				Operation:   metaData.Operation,
				ContentType: metaData.ContentType,
				ChainName:   metaData.ChainName,
				BlockHeight: height,
				Timestamp:   timestamp,
			}
			if err := database.DB.CreateOrUpdatePinInfo(pinInfo); err != nil {
				log.Printf("Failed to save PIN info for %s: %v", metaData.PinID, err)
			}
		} else if metaData.Operation == "modify" || metaData.Operation == "revoke" {
			// Modify/Revoke operation: resolve path and firstPinID if it's a reference (@pinId or host:@pinId)
			resolvedPath, resolvedFirstPinID, resolvedFirstPath, isValidOperation := s.resolvePathAndFirstPinID(metaData.Path)
			if !isValidOperation {
				log.Printf("Invalid operation: %s, path: %s", metaData.Operation, metaData.Path)
				continue
			}
			if resolvedPath != metaData.Path {
				log.Printf("Resolved path reference for %s: %s -> %s (firstPinID: %s, firstPath: %s)",
					metaData.Operation, metaData.Path, resolvedPath, resolvedFirstPinID, resolvedFirstPath)
				// metaData.Path = resolvedPath
				firstPinID = resolvedFirstPinID
				firstPath = resolvedFirstPath
			} else {
				// Path not a reference, need to find firstPinID by path
				// This is a fallback - ideally modify/revoke should use @pinId reference
				firstPinID = metaData.PinID
				firstPath = metaData.Path
				log.Printf("Warning: %s operation without @pinId reference, using PinID as firstPinID: %s", metaData.Operation, firstPinID)
			}

			if !metaid_protocols.IsProtocolPath(firstPath) {
				continue
			}

			// Save PIN info for modify/revoke operations
			pinInfo := &model.IndexerPinInfo{
				PinID:       metaData.PinID,
				FirstPinID:  firstPinID,
				FirstPath:   firstPath,
				Path:        metaData.Path,
				Operation:   metaData.Operation,
				ContentType: metaData.ContentType,
				ChainName:   metaData.ChainName,
				BlockHeight: height,
				Timestamp:   timestamp,
			}
			if err := database.DB.CreateOrUpdatePinInfo(pinInfo); err != nil {
				log.Printf("Failed to save PIN info for %s: %v", metaData.PinID, err)
			}
		} else {
			// For other operations, use PinID as firstPinID
			firstPinID = metaData.PinID
			firstPath = metaData.Path
		}

		// Store firstPinID in metadata for use in processing functions
		// We'll pass it through a context or store it temporarily
		// For now, we'll use a simple approach by modifying the processing functions
		// Check if this is a chunk or index PIN (for large file splitting)
		log.Printf("Processing PIN: %s (path: %s, operation: %s, content type: %s)",
			metaData.PinID, metaData.Path, metaData.Operation, metaData.ContentType)
		if isChunkPath(metaData.Path) && isChunkContentType(metaData.ContentType) {
			log.Printf("Processing chunk PIN: %s (path: %s, operation: %s)",
				metaData.PinID, metaData.Path, metaData.Operation)

			// Check if already exists
			existingChunk, err := s.indexerFileChunkDAO.GetByPinID(metaData.PinID)
			if err == nil && existingChunk != nil {
				log.Printf("Chunk PIN already indexed: %s", metaData.PinID)
				continue
			}

			// Process chunk content
			if err := s.processChunkContent(metaData, firstPinID, height, timestamp); err != nil {
				log.Printf("Failed to process chunk content for PIN %s: %v", metaData.PinID, err)
				continue
			}
		} else if isIndexPath(firstPath) && isIndexContentType(metaData.ContentType) {
			log.Printf("Processing index PIN: %s (firstPath: %s, path: %s, operation: %s)",
				metaData.PinID, firstPath, metaData.Path, metaData.Operation)

			// Check if already exists
			existingFile, err := s.indexerFileDAO.GetByPinID(metaData.PinID)
			if err == nil && existingFile != nil {
				log.Printf("Index PIN already indexed: %s", metaData.PinID)
				continue
			}

			// Process index content
			if err := s.processIndexContent(metaData, firstPinID, firstPath, height, timestamp); err != nil {
				log.Printf("Failed to process index content for PIN %s: %v", metaData.PinID, err)
				continue
			}
		} else if isFilePath(firstPath) {
			// Check if this is a file PIN
			log.Printf("Processing file PIN: %s (firstPath: %s, path: %s, operation: %s)",
				metaData.PinID, firstPath, metaData.Path, metaData.Operation)

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
			if err := s.processFileContent(metaData, firstPinID, firstPath, height, timestamp); err != nil {
				log.Printf("Failed to process file content for PIN %s: %v", metaData.PinID, err)
				// Continue processing other PINs even if one fails
				continue
			}
		} else if isUserNamePath(firstPath) {
			// Check if this is a user name PIN
			log.Printf("Processing user name PIN: %s (firstPath: %s, path: %s, operation: %s)",
				metaData.PinID, firstPath, metaData.Path, metaData.Operation)

			// Process user name content
			if err := s.processUserNameContent(metaData, firstPinID, firstPath, height, timestamp); err != nil {
				log.Printf("Failed to process user name content for PIN %s: %v", metaData.PinID, err)
				continue
			}
		} else if isUserAvatarInfoPath(firstPath) {
			// Check if this is a user avatar info PIN (different from avatar file)
			log.Printf("Processing user avatar info PIN: %s (firstPath: %s, path: %s, operation: %s)",
				metaData.PinID, firstPath, metaData.Path, metaData.Operation)

			// Process user avatar info content
			if err := s.processUserAvatarInfoContent(metaData, firstPinID, firstPath, height, timestamp); err != nil {
				log.Printf("Failed to process user avatar info content for PIN %s: %v", metaData.PinID, err)
				continue
			}
		} else if isUserChatPublicKeyPath(firstPath) {
			// Check if this is a user chat public key PIN
			log.Printf("Processing user chat public key PIN: %s (firstPath: %s, path: %s, operation: %s)",
				metaData.PinID, firstPath, metaData.Path, metaData.Operation)

			// Process user chat public key content
			if err := s.processUserChatPublicKeyContent(metaData, firstPinID, firstPath, height, timestamp); err != nil {
				log.Printf("Failed to process user chat public key content for PIN %s: %v", metaData.PinID, err)
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
	// But exclude chunk and index paths
	if isChunkPath(path) || isIndexPath(path) {
		return false
	}
	return strings.HasPrefix(path, "/file") || strings.Contains(path, "/file")
}

// isAvatarPath check if path is an avatar path (avatar file, not info)
// func isAvatarPath(path string) bool {
// 	// Check if path starts with /info/avatar or contains /info/avatar
// 	// But exclude /info/avatar info path (which is for avatar info text)
// 	if isUserAvatarInfoPath(path) {
// 		return false
// 	}
// 	return strings.HasPrefix(path, "/info/avatar") || strings.Contains(path, "/info/avatar")
// }

// isUserNamePath check if path is a user name path
func isUserNamePath(path string) bool {
	// Check if path starts with /info/name or contains /info/name
	return strings.HasPrefix(path, "/info/name") || strings.Contains(path, "/info/name")
}

// isUserAvatarInfoPath check if path is a user avatar info path (text info, not file)
func isUserAvatarInfoPath(path string) bool {
	// This is for avatar info text, not avatar file
	// Usually path like /info/avatar with text content
	return strings.HasPrefix(path, "/info/avatar") || strings.Contains(path, "/info/avatar")
}

// isUserChatPublicKeyPath check if path is a user chat public key path
func isUserChatPublicKeyPath(path string) bool {
	// Check if path starts with /info/chatpubkey or contains /info/chatpubkey
	return strings.HasPrefix(strings.ToLower(path), "/info/chatpubkey") || strings.Contains(strings.ToLower(path), "/info/chatpubkey") ||
		strings.HasPrefix(strings.ToLower(path), strings.ToLower("/info/chatPublicKey")) || strings.Contains(strings.ToLower(path), strings.ToLower("/info/chatPublicKey"))
}

// isChunkPath check if path is a chunk path
func isChunkPath(path string) bool {
	// Check if path contains /file/_chunk
	return strings.Contains(path, "/file/_chunk") || strings.Contains(path, "/file/"+metaid_protocols.MonitorFileChunk) ||
		strings.Contains(path, "/file/chunk") || strings.Contains(path, "/file/"+metaid_protocols.MonitorFileChunkOld)
}

// isIndexPath check if path is an index path
func isIndexPath(path string) bool {
	// Check if path contains /file/index
	return strings.Contains(path, "/file/index") || strings.Contains(path, "/file/"+metaid_protocols.MonitorFileIndex)
}

// isChunkContentType check if content type is metafile/chunk
func isChunkContentType(contentType string) bool {
	// Check if content type is metafile/chunk (with or without parameters)
	normalized := strings.ToLower(strings.TrimSpace(contentType))
	return strings.HasPrefix(normalized, "metafile/chunk")
}

// isIndexContentType check if content type is metafile/index
func isIndexContentType(contentType string) bool {
	// Check if content type is metafile/index (with or without parameters)
	normalized := strings.ToLower(strings.TrimSpace(contentType))
	return strings.HasPrefix(normalized, "metafile/index")
}

// isGzipCompressed check if content is gzip compressed
func isGzipCompressed(content []byte) bool {
	// Gzip magic number: 1f 8b
	if len(content) < 2 {
		return false
	}
	return content[0] == 0x1f && content[1] == 0x8b
}

// decompressGzip decompress gzip compressed content
func decompressGzip(content []byte) ([]byte, error) {
	reader := bytes.NewReader(content)
	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	decompressed, err := io.ReadAll(gzReader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress gzip content: %w", err)
	}

	return decompressed, nil
}

// processFileContent process and save file content (unified for create and modify)
func (s *IndexerService) processFileContent(metaData *indexer.MetaIDData, firstPinID, firstPath string, height, timestamp int64) error {
	if metaData.Operation == "create" {
		return s.processFileContentCreate(metaData, firstPinID, firstPath, height, timestamp)
	} else if metaData.Operation == "modify" {
		return s.processFileContentModify(metaData, firstPinID, firstPath, height, timestamp)
	}
	// For other operations (like revoke), use create logic as fallback
	return s.processFileContentCreate(metaData, firstPinID, firstPath, height, timestamp)
}

// processFileContentCreate process and save file content for create operation
func (s *IndexerService) processFileContentCreate(metaData *indexer.MetaIDData, firstPinID, firstPath string, height, timestamp int64) error {
	// Get real creator address from CreatorInputLocation if available
	creatorAddress := metaData.CreatorAddress
	if metaData.CreatorInputLocation != "" {
		realAddress, err := s.parser.FindCreatorAddressFromCreatorInputLocation(metaData.CreatorInputLocation, metaData.CreatorInputTxVinLocation, s.chainType)
		if err != nil {
			log.Printf("Failed to get creator address from location %s: %v, using fallback address",
				metaData.CreatorInputLocation, err)
		} else {
			creatorAddress = realAddress
			log.Printf("Found real creator address: %s (from location: %s)", realAddress, metaData.CreatorInputLocation)
		}
	}

	// Check if content is gzip compressed and decompress if needed
	fileContent := metaData.Content
	isCompressed := isGzipCompressed(metaData.Content)
	if isCompressed {
		log.Printf("Detected gzip compressed content for PIN: %s, decompressing...", metaData.PinID)
		decompressed, err := decompressGzip(metaData.Content)
		if err != nil {
			log.Printf("Failed to decompress gzip content for PIN %s: %v, using original content", metaData.PinID, err)
			// Continue with original content if decompression fails
		} else {
			fileContent = decompressed
			log.Printf("Successfully decompressed gzip content for PIN: %s (original size: %d, decompressed size: %d)",
				metaData.PinID, len(metaData.Content), len(fileContent))
		}
	}

	// Extract file name from path
	fileName := extractFileName(metaData.Path)

	// Detect real content type from file content (use decompressed content if available)
	realContentType := detectRealContentType(fileContent, metaData.ContentType)

	// Extract file extension (using real content type and path)
	fileExtension := extractFileExtension(metaData.Path, realContentType, fileContent)

	// Calculate file hashes (use decompressed content if available)
	fileMd5 := calculateMD5(fileContent)
	fileHash := calculateSHA256(fileContent)

	// Detect file type from real content type
	fileType := detectFileType(realContentType)

	// Determine storage path: indexer/{chain}/{txid}/{pinid}{extension}
	// Use pinID as filename to ensure uniqueness, with file extension
	storagePath := fmt.Sprintf("indexer/%s/%s%s",
		metaData.ChainName,
		metaData.PinID,
		fileExtension)

	// Save file to storage (save decompressed content if available)
	storageType := "local"
	if conf.Cfg.Storage.Type == "oss" {
		storageType = "oss"
	}

	if err := s.storage.Save(storagePath, fileContent); err != nil {
		return fmt.Errorf("failed to save file to storage: %w", err)
	}

	log.Printf("File saved to storage: %s (size: %d bytes, compressed: %v)", storagePath, len(fileContent), isCompressed)

	// Calculate Creator MetaID (SHA256 of address)
	creatorMetaID := calculateMetaID(creatorAddress)

	// Create database record
	indexerFile := &model.IndexerFile{
		FirstPinID:       firstPinID,
		FirstPath:        firstPath,
		PinID:            metaData.PinID,
		TxID:             metaData.TxID,
		Vout:             metaData.Vout,
		Path:             metaData.Path,
		Operation:        metaData.Operation,
		ParentPath:       metaData.ParentPath,
		Encryption:       metaData.Encryption,
		Version:          metaData.Version,
		ContentType:      metaData.ContentType,
		ChunkType:        model.ChunkTypeSingle,
		FileType:         fileType,
		FileExtension:    fileExtension,
		FileName:         fileName,
		FileSize:         int64(len(fileContent)),
		FileMd5:          fileMd5,
		FileHash:         fileHash,
		IsGzipCompressed: isCompressed,
		StorageType:      storageType,
		StoragePath:      storagePath,
		ChainName:        metaData.ChainName,
		BlockHeight:      height,
		Timestamp:        timestamp,
		CreatorMetaId:    creatorMetaID,
		CreatorAddress:   creatorAddress, // Use real creator address
		OwnerAddress:     metaData.OwnerAddress,
		OwnerMetaId:      calculateMetaID(metaData.OwnerAddress),
		Status:           model.StatusSuccess,
		State:            0,
	}

	// Save to database
	if err := s.indexerFileDAO.Create(indexerFile); err != nil {
		return fmt.Errorf("failed to save file to database: %w", err)
	}

	// Add to file info history
	fileHistory := &model.FileInfoHistory{
		FirstPinID:  firstPinID,
		FirstPath:   firstPath,
		PinID:       metaData.PinID,
		Path:        metaData.Path,
		Operation:   metaData.Operation,
		ContentType: metaData.ContentType,
		ChainName:   metaData.ChainName,
		BlockHeight: height,
		Timestamp:   timestamp,
	}
	if err := database.DB.AddFileInfoHistory(fileHistory, firstPinID); err != nil {
		log.Printf("Failed to add file info to history: %v", err)
	}

	log.Printf("File indexed successfully (create): PIN=%s, Path=%s, Type=%s, Ext=%s, Size=%d",
		metaData.PinID, metaData.Path, fileType, fileExtension, len(fileContent))

	return nil
}

// processFileContentModify process and save file content for modify operation
func (s *IndexerService) processFileContentModify(metaData *indexer.MetaIDData, firstPinID, firstPath string, height, timestamp int64) error {
	// Get real creator address
	creatorAddress := metaData.CreatorAddress
	if metaData.CreatorInputLocation != "" {
		realAddress, err := s.parser.FindCreatorAddressFromCreatorInputLocation(metaData.CreatorInputLocation, metaData.CreatorInputTxVinLocation, s.chainType)
		if err == nil {
			creatorAddress = realAddress
		}
	}

	// Process file content
	fileContent := metaData.Content
	isCompressed := isGzipCompressed(metaData.Content)
	if isCompressed {
		decompressed, err := decompressGzip(metaData.Content)
		if err == nil {
			fileContent = decompressed
		}
	}

	fileName := extractFileName(metaData.Path)
	realContentType := detectRealContentType(fileContent, metaData.ContentType)
	fileExtension := extractFileExtension(metaData.Path, realContentType, fileContent)
	fileMd5 := calculateMD5(fileContent)
	fileHash := calculateSHA256(fileContent)
	fileType := detectFileType(realContentType)

	storagePath := fmt.Sprintf("indexer/%s/%s%s",
		metaData.ChainName,
		metaData.PinID,
		fileExtension)

	storageType := "local"
	if conf.Cfg.Storage.Type == "oss" {
		storageType = "oss"
	}

	if err := s.storage.Save(storagePath, fileContent); err != nil {
		return fmt.Errorf("failed to save file to storage: %w", err)
	}

	creatorMetaID := calculateMetaID(creatorAddress)

	// Use firstPinID from parameter (resolved from @pinId reference)
	if firstPinID == "" {
		firstPinID = metaData.PinID // Fallback
	}

	// Create database record
	indexerFile := &model.IndexerFile{
		FirstPinID:       firstPinID, // Reference to first create PIN
		FirstPath:        firstPath,
		PinID:            metaData.PinID,
		TxID:             metaData.TxID,
		Vout:             metaData.Vout,
		Path:             metaData.Path,
		Operation:        metaData.Operation,
		ParentPath:       metaData.ParentPath,
		Encryption:       metaData.Encryption,
		Version:          metaData.Version,
		ContentType:      metaData.ContentType,
		ChunkType:        model.ChunkTypeSingle,
		FileType:         fileType,
		FileExtension:    fileExtension,
		FileName:         fileName,
		FileSize:         int64(len(fileContent)),
		FileMd5:          fileMd5,
		FileHash:         fileHash,
		IsGzipCompressed: isCompressed,
		StorageType:      storageType,
		StoragePath:      storagePath,
		ChainName:        metaData.ChainName,
		BlockHeight:      height,
		Timestamp:        timestamp,
		CreatorMetaId:    creatorMetaID,
		CreatorAddress:   creatorAddress,
		OwnerAddress:     metaData.OwnerAddress,
		OwnerMetaId:      calculateMetaID(metaData.OwnerAddress),
		Status:           model.StatusSuccess,
		State:            0,
	}

	if err := s.indexerFileDAO.Create(indexerFile); err != nil {
		return fmt.Errorf("failed to save file to database: %w", err)
	}

	// Add to file info history
	fileHistory := &model.FileInfoHistory{
		FirstPinID:  firstPinID,
		FirstPath:   firstPath,
		PinID:       metaData.PinID,
		Path:        metaData.Path,
		Operation:   metaData.Operation,
		ContentType: metaData.ContentType,
		ChainName:   metaData.ChainName,
		BlockHeight: height,
		Timestamp:   timestamp,
	}
	if err := database.DB.AddFileInfoHistory(fileHistory, firstPinID); err != nil {
		log.Printf("Failed to add file info to history: %v", err)
	}

	log.Printf("File indexed successfully (modify): PIN=%s, FirstPIN=%s, Path=%s, Type=%s, Size=%d",
		metaData.PinID, firstPinID, metaData.Path, fileType, len(fileContent))

	return nil
}

// // processAvatarContent process and save avatar content
// func (s *IndexerService) processAvatarContent(metaData *indexer.MetaIDData, height, timestamp int64) error {
// 	// Get real creator address from CreatorInputLocation if available
// 	creatorAddress := metaData.CreatorAddress
// 	if metaData.CreatorInputLocation != "" {
// 		realAddress, err := s.parser.FindCreatorAddressFromCreatorInputLocation(metaData.CreatorInputLocation, metaData.CreatorInputTxVinLocation, s.chainType)
// 		if err != nil {
// 			log.Printf("Failed to get creator address from location %s: %v, using fallback address",
// 				metaData.CreatorInputLocation, err)
// 		} else {
// 			creatorAddress = realAddress
// 			log.Printf("Found real creator address for avatar: %s (from location: %s)", realAddress, metaData.CreatorInputLocation)
// 		}
// 	}

// 	// Detect real content type from file content
// 	realContentType := detectRealContentType(metaData.Content, metaData.ContentType)

// 	// Extract file extension from real content type
// 	fileExtension := extractAvatarFileExtension(realContentType, metaData.Content)

// 	// Calculate file hashes
// 	fileMd5 := calculateMD5(metaData.Content)
// 	fileHash := calculateSHA256(metaData.Content)

// 	// Detect file type from real content type
// 	fileType := detectFileType(realContentType)

// 	// Determine storage path: indexer/avatar/{chain}/{txid}/{pinid}{extension}
// 	// Use pinID as filename to ensure uniqueness, with file extension
// 	storagePath := fmt.Sprintf("indexer/avatar/%s/%s/%s%s",
// 		metaData.ChainName,
// 		metaData.TxID,
// 		metaData.PinID,
// 		fileExtension)

// 	// Save file to storage
// 	if err := s.storage.Save(storagePath, metaData.Content); err != nil {
// 		return fmt.Errorf("failed to save avatar to storage: %w", err)
// 	}

// 	log.Printf("Avatar saved to storage: %s (size: %d bytes)", storagePath, len(metaData.Content))

// 	// Calculate Creator MetaID (SHA256 of address)
// 	creatorMetaID := calculateMetaID(creatorAddress)

// 	// Create database record
// 	indexerUserAvatar := &model.IndexerUserAvatar{
// 		PinID:         metaData.PinID,
// 		TxID:          metaData.TxID,
// 		MetaId:        creatorMetaID,
// 		Address:       creatorAddress, // Use real creator address
// 		Avatar:        storagePath,
// 		ContentType:   metaData.ContentType,
// 		FileSize:      int64(len(metaData.Content)),
// 		FileMd5:       fileMd5,
// 		FileHash:      fileHash,
// 		FileExtension: fileExtension,
// 		FileType:      fileType,
// 		ChainName:     metaData.ChainName,
// 		BlockHeight:   height,
// 		Timestamp:     timestamp,
// 	}

// 	// Save to database
// 	if err := s.indexerUserAvatarDAO.Create(indexerUserAvatar); err != nil {
// 		return fmt.Errorf("failed to save avatar to database: %w", err)
// 	}

// 	log.Printf("Avatar indexed successfully: PIN=%s, Path=%s, Type=%s, Ext=%s, Size=%d, MetaID=%s, Address=%s",
// 		metaData.PinID, metaData.Path, fileType, fileExtension, len(metaData.Content), creatorMetaID, creatorAddress)

// 	return nil
// }

// processUserNameContent process and save user name content
func (s *IndexerService) processUserNameContent(metaData *indexer.MetaIDData, firstPinID, firstPath string, height, timestamp int64) error {
	// Get real creator address from CreatorInputLocation if available
	creatorAddress := metaData.CreatorAddress
	if metaData.CreatorInputLocation != "" {
		realAddress, err := s.parser.FindCreatorAddressFromCreatorInputLocation(metaData.CreatorInputLocation, metaData.CreatorInputTxVinLocation, s.chainType)
		if err != nil {
			log.Printf("Failed to get creator address from location %s: %v, using fallback address",
				metaData.CreatorInputLocation, err)
		} else {
			creatorAddress = realAddress
			log.Printf("Found real creator address for user name: %s (from location: %s)", realAddress, metaData.CreatorInputLocation)
		}
	}

	// Calculate Creator MetaID (SHA256 of address)
	creatorMetaID := calculateMetaID(creatorAddress)

	// Save MetaID-Address mapping for bidirectional lookup
	if err := database.DB.SaveMetaIdAddress(creatorMetaID, creatorAddress); err != nil {
		log.Printf("Failed to save MetaID-Address mapping: %v", err)
	}

	// Save MetaID-Timestamp mapping (only earliest timestamp)
	if err := database.DB.SaveMetaIdTimestamp(creatorMetaID, timestamp); err != nil {
		log.Printf("Failed to save MetaID-Timestamp mapping: %v", err)
	}

	// Extract user name from content (assume content is text)
	userName := string(metaData.Content)

	// Create user name info
	userNameInfo := &model.UserNameInfo{
		FirstPinID:  firstPinID,
		FirstPath:   firstPath,
		Name:        userName,
		PinID:       metaData.PinID,
		ChainName:   metaData.ChainName,
		BlockHeight: height,
		Timestamp:   timestamp,
	}

	// Save to database - latest info
	if err := database.DB.CreateOrUpdateLatestUserNameInfo(userNameInfo, creatorMetaID); err != nil {
		return fmt.Errorf("failed to save user name info to database: %w", err)
	}

	// Save to database - history
	if err := database.DB.AddUserNameInfoHistory(userNameInfo, creatorMetaID); err != nil {
		log.Printf("Failed to add user name info to history: %v", err)
	}

	log.Printf("User name indexed successfully: PIN=%s, Name=%s, MetaID=%s, Address=%s",
		metaData.PinID, userName, creatorMetaID, creatorAddress)

	return nil
}

// processUserAvatarInfoContent process and save user avatar info content
func (s *IndexerService) processUserAvatarInfoContent(metaData *indexer.MetaIDData, firstPinID, firstPath string, height, timestamp int64) error {
	// Get real creator address from CreatorInputLocation if available
	creatorAddress := metaData.CreatorAddress
	if metaData.CreatorInputLocation != "" {
		realAddress, err := s.parser.FindCreatorAddressFromCreatorInputLocation(metaData.CreatorInputLocation, metaData.CreatorInputTxVinLocation, s.chainType)
		if err != nil {
			log.Printf("Failed to get creator address from location %s: %v, using fallback address",
				metaData.CreatorInputLocation, err)
		} else {
			creatorAddress = realAddress
			log.Printf("Found real creator address for user avatar info: %s (from location: %s)", realAddress, metaData.CreatorInputLocation)
		}
	}

	// Calculate Creator MetaID (SHA256 of address)
	creatorMetaID := calculateMetaID(creatorAddress)

	// Save MetaID-Address mapping for bidirectional lookup
	if err := database.DB.SaveMetaIdAddress(creatorMetaID, creatorAddress); err != nil {
		log.Printf("Failed to save MetaID-Address mapping: %v", err)
	}

	// Save MetaID-Timestamp mapping (only earliest timestamp)
	if err := database.DB.SaveMetaIdTimestamp(creatorMetaID, timestamp); err != nil {
		log.Printf("Failed to save MetaID-Timestamp mapping: %v", err)
	}

	// Detect real content type from file content
	realContentType := detectRealContentType(metaData.Content, metaData.ContentType)

	// Extract file extension from real content type
	fileExtension := contentTypeToExtension(realContentType)

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

	// Build avatar URL based on storage type
	var avatarUrl string
	if conf.Cfg.Storage.Type == "oss" && conf.Cfg.Storage.OSS.Domain != "" {
		// OSS storage: use domain + storage path
		avatarUrl = fmt.Sprintf("%s/%s", conf.Cfg.Storage.OSS.Domain, storagePath)
	} else {
		// Local storage: use indexer API endpoint
		avatarUrl = fmt.Sprintf("/api/v1/avatars/content/%s", metaData.PinID)
	}

	// Create user avatar info
	userAvatarInfo := &model.UserAvatarInfo{
		FirstPinID:    firstPinID,
		FirstPath:     firstPath,
		Avatar:        storagePath, // Storage path
		AvatarUrl:     avatarUrl,   // URL for accessing avatar
		PinID:         metaData.PinID,
		ChainName:     metaData.ChainName,
		BlockHeight:   height,
		Timestamp:     timestamp,
		ContentType:   metaData.ContentType,
		FileSize:      int64(len(metaData.Content)),
		FileMd5:       fileMd5,
		FileHash:      fileHash,
		FileExtension: fileExtension,
		FileType:      fileType,
	}

	// Save to database - latest info
	if err := database.DB.CreateOrUpdateLatestUserAvatarInfo(userAvatarInfo, creatorMetaID); err != nil {
		return fmt.Errorf("failed to save user avatar info to database: %w", err)
	}

	// Save to database - history
	if err := database.DB.AddUserAvatarInfoHistory(userAvatarInfo, creatorMetaID); err != nil {
		log.Printf("Failed to add user avatar info to history: %v", err)
	}

	log.Printf("User avatar info indexed successfully: PIN=%s, Avatar=%s, URL=%s, Type=%s, Ext=%s, Size=%d, MetaID=%s, Address=%s",
		metaData.PinID, storagePath, avatarUrl, fileType, fileExtension, len(metaData.Content), creatorMetaID, creatorAddress)

	return nil
}

// processUserChatPublicKeyContent process and save user chat public key content
func (s *IndexerService) processUserChatPublicKeyContent(metaData *indexer.MetaIDData, firstPinID, firstPath string, height, timestamp int64) error {
	// Get real creator address from CreatorInputLocation if available
	creatorAddress := metaData.CreatorAddress
	if metaData.CreatorInputLocation != "" {
		realAddress, err := s.parser.FindCreatorAddressFromCreatorInputLocation(metaData.CreatorInputLocation, metaData.CreatorInputTxVinLocation, s.chainType)
		if err != nil {
			log.Printf("Failed to get creator address from location %s: %v, using fallback address",
				metaData.CreatorInputLocation, err)
		} else {
			creatorAddress = realAddress
			log.Printf("Found real creator address for user chat public key: %s (from location: %s)", realAddress, metaData.CreatorInputLocation)
		}
	}

	// Calculate Creator MetaID (SHA256 of address)
	creatorMetaID := calculateMetaID(creatorAddress)

	// Save MetaID-Address mapping for bidirectional lookup
	if err := database.DB.SaveMetaIdAddress(creatorMetaID, creatorAddress); err != nil {
		log.Printf("Failed to save MetaID-Address mapping: %v", err)
	}

	// Save MetaID-Timestamp mapping (only earliest timestamp)
	if err := database.DB.SaveMetaIdTimestamp(creatorMetaID, timestamp); err != nil {
		log.Printf("Failed to save MetaID-Timestamp mapping: %v", err)
	}

	// Extract chat public key from content (assume content is text)
	chatPublicKey := string(metaData.Content)

	// Create user chat public key info
	userChatPublicKeyInfo := &model.UserChatPublicKeyInfo{
		FirstPinID:    firstPinID,
		FirstPath:     firstPath,
		ChatPublicKey: chatPublicKey,
		PinID:         metaData.PinID,
		ChainName:     metaData.ChainName,
		BlockHeight:   height,
		Timestamp:     timestamp,
	}

	// Save to database - latest info
	if err := database.DB.CreateOrUpdateLatestUserChatPublicKeyInfo(userChatPublicKeyInfo, creatorMetaID); err != nil {
		return fmt.Errorf("failed to save user chat public key info to database: %w", err)
	}

	// Save to database - history
	if err := database.DB.AddUserChatPublicKeyHistory(userChatPublicKeyInfo, creatorMetaID); err != nil {
		log.Printf("Failed to add user chat public key info to history: %v", err)
	}

	log.Printf("User chat public key indexed successfully: PIN=%s, ChatPublicKey=%s, MetaID=%s, Address=%s",
		metaData.PinID, chatPublicKey, creatorMetaID, creatorAddress)

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

// // extractAvatarFileExtension extract file extension from content type and content for avatar
// func extractAvatarFileExtension(contentType string, content []byte) string {
// 	return contentTypeToExtension(contentType)
// }

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

// resolvePathAndFirstPinID resolve path and firstPinID if it's a reference (@pinId or host:@pinId)
// Returns: (resolvedPath, firstPinID, firstPath)
func (s *IndexerService) resolvePathAndFirstPinID(path string) (string, string, string, bool) {
	// Check if path is a reference to another PIN
	// Format: @pinId or host:@pinId
	var refPinID string
	isValidOperation := true

	if strings.HasPrefix(path, "@") {
		// Format: @pinId
		refPinID = strings.TrimPrefix(path, "@")
	} else if strings.Contains(path, ":@") {
		// Format: host:@pinId
		parts := strings.SplitN(path, ":@", 2)
		if len(parts) == 2 {
			refPinID = parts[1]
		}
	}
	//  else {
	// 	isValidOperation = false
	// }

	// If no reference found, return original path and empty values
	if refPinID == "" {
		return path, "", "", isValidOperation
	}

	// Look up the referenced PIN info
	pinInfo, err := database.DB.GetPinInfoByPinID(refPinID)
	if err != nil {
		isValidOperation = false
		log.Printf("Failed to resolve PIN reference %s: %v, using original path", refPinID, err)
		return path, "", "", isValidOperation
	}

	// Get firstPinID and firstPath from the referenced PIN
	// If the referenced PIN is a create operation, use its PinID as firstPinID
	// If the referenced PIN is a modify/revoke operation, use its FirstPinID
	firstPinID := pinInfo.FirstPinID
	firstPath := pinInfo.FirstPath

	if firstPinID == "" {
		// Fallback: if FirstPinID is not set, use the referenced PinID
		firstPinID = refPinID
		firstPath = pinInfo.Path
	}

	log.Printf("Resolved PIN reference: @%s -> %s (firstPinID: %s, firstPath: %s, operation: %s)",
		refPinID, pinInfo.Path, firstPinID, firstPath, pinInfo.Operation)
	return pinInfo.Path, firstPinID, firstPath, isValidOperation
}

// processChunkContent process and save chunk content
func (s *IndexerService) processChunkContent(metaData *indexer.MetaIDData, firstPinID string, height, timestamp int64) error {
	// Check if content is gzip compressed and decompress if needed
	chunkContent := metaData.Content
	isCompressed := isGzipCompressed(metaData.Content)
	if isCompressed {
		log.Printf("Detected gzip compressed chunk content for PIN: %s, decompressing...", metaData.PinID)
		decompressed, err := decompressGzip(metaData.Content)
		if err != nil {
			log.Printf("Failed to decompress gzip chunk content for PIN %s: %v, using original content", metaData.PinID, err)
			// Continue with original content if decompression fails
		} else {
			chunkContent = decompressed
			log.Printf("Successfully decompressed gzip chunk content for PIN: %s (original size: %d, decompressed size: %d)",
				metaData.PinID, len(metaData.Content), len(chunkContent))
		}
	}

	// Calculate chunk hashes (use decompressed content if available)
	chunkMd5 := calculateMD5(chunkContent)
	chunkHash := calculateSHA256(chunkContent)

	// Determine storage path: indexer/chunk/{chain}/{txid}/{pinid}
	storagePath := fmt.Sprintf("indexer/chunk/%s/%s/%s",
		metaData.ChainName,
		metaData.TxID,
		metaData.PinID)

	// Save chunk to storage (save decompressed content if available)
	storageType := "local"
	if conf.Cfg.Storage.Type == "oss" {
		storageType = "oss"
	}

	if err := s.storage.Save(storagePath, chunkContent); err != nil {
		return fmt.Errorf("failed to save chunk to storage: %w", err)
	}

	log.Printf("Chunk saved to storage: %s (size: %d bytes, compressed: %v)", storagePath, len(chunkContent), isCompressed)

	// Extract chunk index from path or metadata (if available)
	// For now, we'll set it to 0 and it should be updated when index is processed
	chunkIndex := 0

	// Create database record
	indexerFileChunk := &model.IndexerFileChunk{
		PinID:            metaData.PinID,
		TxID:             metaData.TxID,
		Vout:             metaData.Vout,
		Path:             metaData.Path,
		Operation:        metaData.Operation,
		ContentType:      metaData.ContentType,
		ChunkIndex:       chunkIndex,
		ChunkSize:        int64(len(chunkContent)),
		ChunkMd5:         chunkMd5,
		IsGzipCompressed: isCompressed,
		ParentPinID:      "", // Will be set when index is processed
		StorageType:      storageType,
		StoragePath:      storagePath,
		ChainName:        metaData.ChainName,
		BlockHeight:      height,
		Status:           model.StatusSuccess,
		State:            0,
	}

	// Save to database
	if err := s.indexerFileChunkDAO.Create(indexerFileChunk); err != nil {
		return fmt.Errorf("failed to save chunk to database: %w", err)
	}

	log.Printf("Chunk indexed successfully: PIN=%s, Path=%s, Size=%d, Hash=%s, Compressed=%v",
		metaData.PinID, metaData.Path, len(chunkContent), chunkHash, isCompressed)

	return nil
}

// processIndexContent process and save index content, then merge chunks if all are available
func (s *IndexerService) processIndexContent(metaData *indexer.MetaIDData, firstPinID, firstPath string, height, timestamp int64) error {
	// Get real creator address from CreatorInputLocation if available
	creatorAddress := metaData.CreatorAddress
	if metaData.CreatorInputLocation != "" {
		realAddress, err := s.parser.FindCreatorAddressFromCreatorInputLocation(metaData.CreatorInputLocation, metaData.CreatorInputTxVinLocation, s.chainType)
		if err != nil {
			log.Printf("Failed to get creator address from location %s: %v, using fallback address",
				metaData.CreatorInputLocation, err)
		} else {
			creatorAddress = realAddress
			log.Printf("Found real creator address for index: %s (from location: %s)", realAddress, metaData.CreatorInputLocation)
		}
	}

	// Parse index JSON content
	// First parse to a flexible structure to handle numeric fields that might be floats
	var rawIndex map[string]interface{}
	if err := json.Unmarshal(metaData.Content, &rawIndex); err != nil {
		return fmt.Errorf("failed to parse index JSON: %w", err)
	}

	// Convert float values to int for numeric fields (chunkSize, fileSize)
	if chunkSize, ok := rawIndex["chunkSize"]; ok {
		switch v := chunkSize.(type) {
		case float64:
			rawIndex["chunkSize"] = int64(v)
		}
	}
	if fileSize, ok := rawIndex["fileSize"]; ok {
		switch v := fileSize.(type) {
		case float64:
			rawIndex["fileSize"] = int64(v)
		}
	}

	// Re-marshal and unmarshal to the proper struct
	correctedJSON, err := json.Marshal(rawIndex)
	if err != nil {
		return fmt.Errorf("failed to re-marshal corrected JSON: %w", err)
	}

	var metaFileIndex metaid_protocols.MetaFileIndex
	if err := json.Unmarshal(correctedJSON, &metaFileIndex); err != nil {
		return fmt.Errorf("failed to parse index JSON: %w", err)
	}

	log.Printf("Parsed index: sha256=%s, fileSize=%d, chunkNumber=%d, chunkSize=%d, dataType=%s, name=%s",
		metaFileIndex.Sha256, metaFileIndex.FileSize, metaFileIndex.ChunkNumber,
		metaFileIndex.ChunkSize, metaFileIndex.DataType, metaFileIndex.Name)

	// Check if all chunks are available
	allChunksAvailable := true
	var chunks []*model.IndexerFileChunk
	for _, chunkInfo := range metaFileIndex.ChunkList {
		chunk, err := s.indexerFileChunkDAO.GetByPinID(chunkInfo.PinId)
		if err != nil || chunk == nil {
			log.Printf("Chunk not found: PIN=%s, SHA256=%s", chunkInfo.PinId, chunkInfo.Sha256)
			allChunksAvailable = false
			break
		}

		// Verify chunk hash matches
		if chunk.ChunkMd5 != "" {
			// We can verify using SHA256 if available, but for now just check existence
			// The actual verification should be done when merging
		}

		chunks = append(chunks, chunk)
	}

	// Update parent_pin_id for all chunks
	indexPinID := metaData.PinID
	for i, chunk := range chunks {
		if chunk.ParentPinID != indexPinID {
			chunk.ParentPinID = indexPinID
			chunk.ChunkIndex = i // Set chunk index based on order in chunkList
			if err := s.indexerFileChunkDAO.Update(chunk); err != nil {
				log.Printf("Failed to update chunk parent PIN ID: %v", err)
			}
		}
	}

	// If all chunks are available, merge and save complete file
	if allChunksAvailable && len(chunks) > 0 {
		log.Printf("All chunks available, merging file: index PIN=%s", indexPinID)

		// Check if all chunks are gzip compressed
		allChunksCompressed := true
		for _, chunk := range chunks {
			if !chunk.IsGzipCompressed {
				allChunksCompressed = false
				break
			}
		}

		// Merge chunks in order
		var mergedContent []byte
		for _, chunk := range chunks {
			// Load chunk content from storage
			chunkContent, err := s.storage.Get(chunk.StoragePath)
			if err != nil {
				return fmt.Errorf("failed to load chunk from storage: %w", err)
			}
			mergedContent = append(mergedContent, chunkContent...)
		}

		// Verify merged file hash
		mergedHash := calculateSHA256(mergedContent)
		if mergedHash != metaFileIndex.Sha256 {
			log.Printf("Warning: Merged file hash mismatch. Expected: %s, Got: %s", metaFileIndex.Sha256, mergedHash)
			// Continue anyway, but log the warning
		}

		// Verify file size
		if int64(len(mergedContent)) != metaFileIndex.FileSize {
			log.Printf("Warning: Merged file size mismatch. Expected: %d, Got: %d", metaFileIndex.FileSize, len(mergedContent))
		}

		// Detect real content type
		realContentType := detectRealContentType(mergedContent, metaFileIndex.DataType)

		// Extract file extension
		fileExtension := contentTypeToExtension(realContentType)
		if fileExtension == "" && metaFileIndex.Name != "" {
			// Try to get extension from name
			fileExtension = filepath.Ext(metaFileIndex.Name)
		}

		// Calculate file hashes
		fileMd5 := calculateMD5(mergedContent)
		fileHash := calculateSHA256(mergedContent)

		// Detect file type
		fileType := detectFileType(realContentType)

		// Determine storage path: indexer/{chain}/{indexPinID}{extension}
		storagePath := fmt.Sprintf("indexer/%s/%s%s",
			metaData.ChainName,
			indexPinID,
			fileExtension)

		// Save merged file to storage
		storageType := "local"
		if conf.Cfg.Storage.Type == "oss" {
			storageType = "oss"
		}

		if err := s.storage.Save(storagePath, mergedContent); err != nil {
			return fmt.Errorf("failed to save merged file to storage: %w", err)
		}

		log.Printf("Merged file saved to storage: %s (size: %d bytes)", storagePath, len(mergedContent))

		// Calculate Creator MetaID
		creatorMetaID := calculateMetaID(creatorAddress)

		data, err := json.Marshal(metaFileIndex)
		if err != nil {
			return fmt.Errorf("failed to marshal metaFileIndex: %w", err)
		}

		// Determine firstPinID based on operation
		fileFirstPinID := firstPinID
		if fileFirstPinID == "" {
			fileFirstPinID = indexPinID // Fallback to indexPinID
		}
		if metaData.Operation == "create" {
			fileFirstPinID = indexPinID // For create, firstPinID = PinID
		}

		// Create database record for merged file
		indexerFile := &model.IndexerFile{
			FirstPinID:       fileFirstPinID,
			FirstPath:        firstPath,
			PinID:            indexPinID,
			TxID:             metaData.TxID,
			Vout:             metaData.Vout,
			Path:             metaData.Path,
			Operation:        metaData.Operation,
			ParentPath:       metaData.ParentPath,
			Encryption:       metaData.Encryption,
			Version:          metaData.Version,
			ContentType:      metaFileIndex.DataType,
			Data:             string(data),
			ChunkType:        model.ChunkTypeMulti,
			FileType:         fileType,
			FileExtension:    fileExtension,
			FileName:         metaFileIndex.Name,
			FileSize:         metaFileIndex.FileSize,
			FileMd5:          fileMd5,
			FileHash:         fileHash,
			IsGzipCompressed: allChunksCompressed,
			StorageType:      storageType,
			StoragePath:      storagePath,
			ChainName:        metaData.ChainName,
			BlockHeight:      height,
			Timestamp:        timestamp,
			CreatorMetaId:    creatorMetaID,
			CreatorAddress:   creatorAddress,
			OwnerAddress:     metaData.OwnerAddress,
			OwnerMetaId:      calculateMetaID(metaData.OwnerAddress),
			Status:           model.StatusSuccess,
			State:            0,
		}

		// Save to database
		if err := s.indexerFileDAO.Create(indexerFile); err != nil {
			return fmt.Errorf("failed to save merged file to database: %w", err)
		}

		// Add to file info history
		fileHistory := &model.FileInfoHistory{
			FirstPinID:  fileFirstPinID,
			FirstPath:   firstPath,
			PinID:       indexPinID,
			Path:        metaData.Path,
			Operation:   metaData.Operation,
			ContentType: metaData.ContentType,
			ChainName:   metaData.ChainName,
			BlockHeight: height,
			Timestamp:   timestamp,
		}
		if err := database.DB.AddFileInfoHistory(fileHistory, fileFirstPinID); err != nil {
			log.Printf("Failed to add file info to history: %v", err)
		}

		log.Printf("Merged file indexed successfully (%s): PIN=%s, FirstPIN=%s, Name=%s, Type=%s, Size=%d",
			metaData.Operation, indexPinID, fileFirstPinID, metaFileIndex.Name, fileType, metaFileIndex.FileSize)
	} else {
		log.Printf("Not all chunks available yet for index PIN=%s. Chunks found: %d/%d",
			indexPinID, len(chunks), metaFileIndex.ChunkNumber)
		// We still save the index information, but the file will be merged later when all chunks are available
		// For now, we just log that chunks are missing
	}

	return nil
}

// RescanBlocksAsync asynchronously rescans blocks within a specified range
func (s *IndexerService) RescanBlocksAsync(chain string, startHeight, endHeight int64) (string, error) {
	// Check if a task is already running
	s.rescanMu.Lock()
	if s.currentRescanTask != nil && s.currentRescanTask.Status == RescanStatusRunning {
		s.rescanMu.Unlock()
		return "", fmt.Errorf("another rescan task is already running: %s", s.currentRescanTask.TaskID)
	}

	// Validate parameters
	if startHeight <= 0 {
		s.rescanMu.Unlock()
		return "", fmt.Errorf("start height must be greater than 0")
	}
	if endHeight < startHeight {
		s.rescanMu.Unlock()
		return "", fmt.Errorf("end height must be greater than or equal to start height")
	}

	// Validate chain
	var chainType indexer.ChainType
	switch strings.ToLower(chain) {
	case "btc":
		chainType = indexer.ChainTypeBTC
	case "mvc":
		chainType = indexer.ChainTypeMVC
	default:
		s.rescanMu.Unlock()
		return "", fmt.Errorf("unsupported chain: %s, only 'btc' and 'mvc' are supported", chain)
	}

	chainName := string(chainType)

	// Check if we're in multi-chain mode
	var scanner *indexer.BlockScanner
	if s.isMultiChain {
		// Get scanner from coordinator
		if s.coordinator == nil {
			s.rescanMu.Unlock()
			return "", fmt.Errorf("coordinator not initialized")
		}
		scanner = s.coordinator.GetScanner(chainName)
		if scanner == nil {
			s.rescanMu.Unlock()
			return "", fmt.Errorf("scanner not found for chain: %s", chainName)
		}
	} else {
		// Single chain mode
		if s.scanner == nil {
			s.rescanMu.Unlock()
			return "", fmt.Errorf("scanner not initialized")
		}
		if string(s.chainType) != chainName {
			s.rescanMu.Unlock()
			return "", fmt.Errorf("current scanner is for chain %s, cannot rescan chain %s", s.chainType, chainName)
		}
		scanner = s.scanner
	}

	// Generate task ID
	taskID := fmt.Sprintf("rescan_%s_%d_%d_%d", chainName, startHeight, endHeight, time.Now().Unix())

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Create task
	totalBlocks := endHeight - startHeight + 1
	task := &RescanTask{
		TaskID:          taskID,
		Chain:           chainName,
		Status:          RescanStatusRunning,
		StartHeight:     startHeight,
		EndHeight:       endHeight,
		CurrentHeight:   startHeight,
		ProcessedBlocks: 0,
		TotalBlocks:     totalBlocks,
		StartTime:       time.Now(),
		CancelFunc:      cancel,
	}

	s.currentRescanTask = task
	s.rescanMu.Unlock()

	// Create handler for processing transactions during rescan
	handler := s.handleTransaction

	// Start rescan in goroutine
	go func() {
		log.Printf("[Rescan %s] Starting rescan task: %s (height %d to %d)", chainName, taskID, startHeight, endHeight)

		defer func() {
			// Clean up on completion
			s.rescanMu.Lock()
			if s.currentRescanTask != nil && s.currentRescanTask.TaskID == taskID {
				if s.currentRescanTask.Status == RescanStatusRunning {
					s.currentRescanTask.Status = RescanStatusCompleted
				}
			}
			s.rescanMu.Unlock()
		}()

		for height := startHeight; height <= endHeight; height++ {
			// Check for cancellation
			select {
			case <-ctx.Done():
				task.mu.Lock()
				task.Status = RescanStatusCancelled
				task.mu.Unlock()
				log.Printf("[Rescan %s] Task cancelled: %s at height %d", chainName, taskID, height)
				return
			default:
			}

			// Scan block
			_, err := scanner.ScanBlock(height, handler)
			if err != nil {
				log.Printf("[Rescan %s] Failed to scan block %d: %v", chainName, height, err)
				// Update error but continue
				task.mu.Lock()
				if task.ErrorMessage == "" {
					task.ErrorMessage = fmt.Sprintf("Failed to scan block %d: %v", height, err)
				}
				task.mu.Unlock()
				continue
			}

			// Update task progress
			task.mu.Lock()
			task.ProcessedBlocks++
			task.CurrentHeight = height
			task.mu.Unlock()

			// Log progress every 100 blocks or at the end
			if task.ProcessedBlocks%100 == 0 || height == endHeight {
				task.mu.RLock()
				elapsed := time.Since(task.StartTime)
				blocksPerSecond := float64(task.ProcessedBlocks) / elapsed.Seconds()
				progress := float64(task.ProcessedBlocks) / float64(totalBlocks) * 100

				log.Printf("[Rescan %s] Progress: %.2f%% (%d/%d blocks), Speed: %.2f blocks/sec",
					chainName, progress, task.ProcessedBlocks, totalBlocks, blocksPerSecond)
				task.mu.RUnlock()
			}
		}

		elapsed := time.Since(task.StartTime)
		log.Printf("[Rescan %s] Completed task %s: rescanned %d blocks in %v (%.2f blocks/sec)",
			chainName, taskID, task.ProcessedBlocks, elapsed, float64(task.ProcessedBlocks)/elapsed.Seconds())
	}()

	log.Printf("[Rescan %s] Rescan task queued: %s (height %d to %d)", chainName, taskID, startHeight, endHeight)
	return taskID, nil
}

// GetRescanStatus returns the current rescan task status
func (s *IndexerService) GetRescanStatus() *RescanTask {
	s.rescanMu.Lock()
	defer s.rescanMu.Unlock()

	if s.currentRescanTask == nil {
		// Return an idle task
		return &RescanTask{
			Status: RescanStatusIdle,
		}
	}

	// Return a copy of the current task to avoid race conditions
	s.currentRescanTask.mu.RLock()
	defer s.currentRescanTask.mu.RUnlock()

	taskCopy := &RescanTask{
		TaskID:          s.currentRescanTask.TaskID,
		Chain:           s.currentRescanTask.Chain,
		Status:          s.currentRescanTask.Status,
		StartHeight:     s.currentRescanTask.StartHeight,
		EndHeight:       s.currentRescanTask.EndHeight,
		CurrentHeight:   s.currentRescanTask.CurrentHeight,
		ProcessedBlocks: s.currentRescanTask.ProcessedBlocks,
		TotalBlocks:     s.currentRescanTask.TotalBlocks,
		StartTime:       s.currentRescanTask.StartTime,
		ErrorMessage:    s.currentRescanTask.ErrorMessage,
	}

	return taskCopy
}

// StopRescan stops the current rescan task
func (s *IndexerService) StopRescan() error {
	s.rescanMu.Lock()
	defer s.rescanMu.Unlock()

	if s.currentRescanTask == nil {
		return fmt.Errorf("no rescan task is running")
	}

	if s.currentRescanTask.Status != RescanStatusRunning {
		return fmt.Errorf("rescan task is not running (status: %s)", s.currentRescanTask.Status)
	}

	// Cancel the task
	if s.currentRescanTask.CancelFunc != nil {
		s.currentRescanTask.CancelFunc()
	}

	log.Printf("[Rescan] Stopping task: %s", s.currentRescanTask.TaskID)
	return nil
}
