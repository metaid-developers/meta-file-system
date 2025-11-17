package database

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"

	"meta-file-system/model"

	"github.com/cockroachdb/pebble"
)

// PebbleDatabase PebbleDB database implementation with multiple collections
type PebbleDatabase struct {
	collections map[string]*pebble.DB // Map of collection name to PebbleDB instance

	fileIDCounter   atomic.Int64
	avatarIDCounter atomic.Int64
	statusIDCounter atomic.Int64
}

// PebbleConfig PebbleDB configuration
type PebbleConfig struct {
	DataDir string
}

// Collection names and their key-value formats
const (
	// File collections
	collectionFilePinID   = "file_pin"  // key: {pin_id}, value: JSON(IndexerFile) - PinID 到 ID 的映射
	collectionFileAddress = "file_addr" // key: {address}:{pin_id}, value: JSON(IndexerFile) - 按地址索引
	collectionFileMetaID  = "file_meta" // key: {meta_id}:{pin_id}, value: JSON(IndexerFile) - 按 MetaID 索引
	collectionFileHash    = "file_hash" // key: {hash}:{pin_id}, value: JSON(IndexerFile) - 按 Hash 索引

	// Avatar collections
	collectionAvatarPinID           = "avatar_pin"            // key: {pin_id}, value: JSON(IndexerUserAvatar) - PinID 到 ID 的映射
	collectionAvatarMetaID          = "avatar_meta"           // key: {meta_id}:{block_height}, value: JSON(IndexerUserAvatar) - 按 MetaID 索引
	collectionAvatarMetaIDTimestamp = "avatar_meta_timestamp" // key: {meta_id}:{timestamp}, value: JSON(IndexerUserAvatar) - 按 MetaID 和时间戳索引
	collectionAvatarAddr            = "avatar_addr"           // key: {address}:{block_height}, value: JSON(IndexerUserAvatar) - 按地址索引
	collectionAvatarHash            = "avatar_hash"           // key: {hash}:{pin_id}, value: JSON(IndexerUserAvatar) - 按 Hash 索引
	collectionLasestAvatarMetaID    = "avatar_lasest_meta_id" // key: {meta_id}, value: JSON(IndexerUserAvatar) - 按 MetaID 索引

	// System collections
	collectionSyncStatus = "sync_status" // key: {chain_name}, value: JSON(IndexerSyncStatus) - 同步状态
	collectionCounters   = "counters"    // key: file/avatar/status, value: {max_id} - ID 计数器
)

// Counter keys
const (
	keyFileCounter   = "file"
	keyAvatarCounter = "avatar"
	keyStatusCounter = "status"
)

// NewPebbleDatabase create PebbleDB database instance with multiple collections
func NewPebbleDatabase(config interface{}) (Database, error) {
	cfg, ok := config.(*PebbleConfig)
	if !ok {
		return nil, fmt.Errorf("invalid PebbleDB config type")
	}

	// Create data directory if not exists with full permissions
	if err := os.MkdirAll(cfg.DataDir, 0777); err != nil {
		return nil, fmt.Errorf("failed to create data directory %s: %w", cfg.DataDir, err)
	}

	log.Printf("PebbleDB data directory: %s", cfg.DataDir)

	// List of all collections
	collectionNames := []string{
		collectionFilePinID,
		collectionFileAddress,
		collectionFileMetaID,
		collectionFileHash,
		collectionAvatarPinID,
		collectionAvatarMetaID,
		collectionAvatarMetaIDTimestamp,
		collectionAvatarAddr,
		collectionAvatarHash,
		collectionLasestAvatarMetaID,
		collectionSyncStatus,
		collectionCounters,
	}

	// Open PebbleDB for each collection
	collections := make(map[string]*pebble.DB)
	for _, name := range collectionNames {
		// Create collection path: dataDir/collectionName
		collectionPath := filepath.Join(cfg.DataDir, "indexer_db", name)

		log.Printf("Opening collection: %s at %s", name, collectionPath)

		// PebbleDB will create the directory automatically, but we ensure parent exists
		// No need to create the collection directory manually
		db, err := pebble.Open(collectionPath, &pebble.Options{})
		if err != nil {
			// Close previously opened databases
			for _, openedDB := range collections {
				openedDB.Close()
			}
			return nil, fmt.Errorf("failed to open collection %s at %s: %w", name, collectionPath, err)
		}
		collections[name] = db
		log.Printf("Collection %s opened successfully", name)
	}

	pdb := &PebbleDatabase{
		collections: collections,
	}

	// Load counters
	if err := pdb.loadCounters(); err != nil {
		return nil, fmt.Errorf("failed to load counters: %w", err)
	}

	log.Printf("PebbleDB database connected successfully with %d collections", len(collections))
	return pdb, nil
}

// loadCounters load ID counters from counters collection
func (p *PebbleDatabase) loadCounters() error {
	counterDB := p.collections[collectionCounters]

	// Load file counter
	if val, closer, err := counterDB.Get([]byte(keyFileCounter)); err == nil {
		count, _ := strconv.ParseInt(string(val), 10, 64)
		p.fileIDCounter.Store(count)
		closer.Close()
	}

	// Load avatar counter
	if val, closer, err := counterDB.Get([]byte(keyAvatarCounter)); err == nil {
		count, _ := strconv.ParseInt(string(val), 10, 64)
		p.avatarIDCounter.Store(count)
		closer.Close()
	}

	// Load status counter
	if val, closer, err := counterDB.Get([]byte(keyStatusCounter)); err == nil {
		count, _ := strconv.ParseInt(string(val), 10, 64)
		p.statusIDCounter.Store(count)
		closer.Close()
	}

	return nil
}

// IndexerFile operations

func (p *PebbleDatabase) CreateIndexerFile(file *model.IndexerFile) error {
	// Serialize file
	data, err := json.Marshal(file)
	if err != nil {
		return err
	}

	// Store in PinID collection (primary index)
	// key: pin_id, value: JSON(IndexerFile)
	if err := p.collections[collectionFilePinID].Set([]byte(file.PinID), data, pebble.Sync); err != nil {
		return err
	}

	// Store in Address index collection
	// key: address:pin_id, value: JSON(IndexerFile)
	addressKey := file.CreatorAddress + ":" + file.PinID
	if err := p.collections[collectionFileAddress].Set([]byte(addressKey), data, pebble.Sync); err != nil {
		return err
	}

	// Store in MetaID index collection
	// key: meta_id:pin_id, value: JSON(IndexerFile)
	metaIDKey := file.CreatorMetaId + ":" + file.PinID
	if err := p.collections[collectionFileMetaID].Set([]byte(metaIDKey), data, pebble.Sync); err != nil {
		return err
	}

	// Store in Hash index collection
	// key: hash:pin_id, value: JSON(IndexerFile)
	hashKey := file.FileMd5 + ":" + file.PinID
	if err := p.collections[collectionFileHash].Set([]byte(hashKey), data, pebble.Sync); err != nil {
		return err
	}

	return nil
}

func (p *PebbleDatabase) GetIndexerFileByPinID(pinID string) (*model.IndexerFile, error) {
	// Get file data directly from PinID collection
	data, closer, err := p.collections[collectionFilePinID].Get([]byte(pinID))
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	defer closer.Close()

	var file model.IndexerFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}

	return &file, nil
}

func (p *PebbleDatabase) UpdateIndexerFile(file *model.IndexerFile) error {
	// Simply recreate (overwrite)
	return p.CreateIndexerFile(file)
}

func (p *PebbleDatabase) ListIndexerFilesWithCursor(cursor int64, size int) ([]*model.IndexerFile, error) {
	var files []*model.IndexerFile

	filePinDB := p.collections[collectionFilePinID]

	// Create iterator for PinID collection
	iter, err := filePinDB.NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	// Cursor is not used for PinID-based iteration (PinID is string, not sequential)
	// We iterate from last to first
	iter.Last()

	// Skip items until we reach the cursor PinID (if provided)
	cursorPinID := ""
	if cursor > 0 {
		// In this case, cursor would need to be a PinID or we skip cursor-based logic
		// For simplicity, we'll just iterate from the end
	}

	count := 0
	for iter.Valid() && count < size {
		var file model.IndexerFile
		if err := json.Unmarshal(iter.Value(), &file); err != nil {
			iter.Prev()
			continue
		}

		// Skip until we reach cursor (if cursorPinID is set)
		if cursorPinID != "" && file.PinID == cursorPinID {
			cursorPinID = "" // Found cursor, start collecting from next
			iter.Prev()
			continue
		}

		if file.Status == model.StatusSuccess {
			files = append(files, &file)
			count++
		}

		iter.Prev()
	}

	return files, nil
}

func (p *PebbleDatabase) GetIndexerFilesByCreatorAddressWithCursor(address string, cursor int64, size int) ([]*model.IndexerFile, error) {
	var files []*model.IndexerFile

	addressDB := p.collections[collectionFileAddress]
	prefix := address + ":"

	// Create iterator with prefix
	// key format: address:pin_id
	iter, err := addressDB.NewIter(&pebble.IterOptions{
		LowerBound: []byte(prefix),
		UpperBound: []byte(prefix + "~"),
	})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	// Start from last (most recent)
	iter.Last()

	// Cursor is based on PinID (not sequential ID)
	cursorPinID := ""
	if cursor > 0 {
		// For PinID-based keys, cursor would be a PinID string
		// For now, we'll skip cursor logic and iterate from end
	}

	count := 0
	for iter.Valid() && count < size {
		var file model.IndexerFile
		if err := json.Unmarshal(iter.Value(), &file); err != nil {
			iter.Prev()
			continue
		}

		// Skip until cursor is reached
		if cursorPinID != "" && file.PinID == cursorPinID {
			cursorPinID = ""
			iter.Prev()
			continue
		}

		if file.Status == model.StatusSuccess {
			files = append(files, &file)
			count++
		}

		iter.Prev()
	}

	return files, nil
}

func (p *PebbleDatabase) GetIndexerFilesByCreatorMetaIDWithCursor(metaID string, cursor int64, size int) ([]*model.IndexerFile, error) {
	var files []*model.IndexerFile

	metaIDDB := p.collections[collectionFileMetaID]
	prefix := metaID + ":"

	// Create iterator with prefix
	// key format: meta_id:pin_id
	iter, err := metaIDDB.NewIter(&pebble.IterOptions{
		LowerBound: []byte(prefix),
		UpperBound: []byte(prefix + "~"),
	})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	// Start from last (most recent)
	iter.Last()

	// Cursor is based on PinID (not sequential ID)
	cursorPinID := ""
	if cursor > 0 {
		// For PinID-based keys, cursor would be a PinID string
		// For now, we'll skip cursor logic and iterate from end
	}

	count := 0
	for iter.Valid() && count < size {
		var file model.IndexerFile
		if err := json.Unmarshal(iter.Value(), &file); err != nil {
			iter.Prev()
			continue
		}

		// Skip until cursor is reached
		if cursorPinID != "" && file.PinID == cursorPinID {
			cursorPinID = ""
			iter.Prev()
			continue
		}

		if file.Status == model.StatusSuccess {
			files = append(files, &file)
			count++
		}

		iter.Prev()
	}

	return files, nil
}

func (p *PebbleDatabase) GetIndexerFilesCount() (int64, error) {
	var count int64

	filePinDB := p.collections[collectionFilePinID]

	// Iterate through all files and count
	iter, err := filePinDB.NewIter(nil)
	if err != nil {
		return 0, err
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		var file model.IndexerFile
		if err := json.Unmarshal(iter.Value(), &file); err != nil {
			continue
		}

		// Only count successful files
		if file.Status == model.StatusSuccess {
			count++
		}
	}

	return count, nil
}

// IndexerUserAvatar operations

func (p *PebbleDatabase) CreateIndexerUserAvatar(avatar *model.IndexerUserAvatar) error {
	data, err := json.Marshal(avatar)
	if err != nil {
		return err
	}

	blockHeightKey := strconv.FormatInt(avatar.BlockHeight, 10)
	timestampKey := strconv.FormatInt(avatar.Timestamp, 10)

	// Store in PinID collection (primary index)
	// key: pin_id, value: JSON(IndexerUserAvatar)
	if err := p.collections[collectionAvatarPinID].Set([]byte(avatar.PinID), data, pebble.Sync); err != nil {
		return err
	}

	// Store in MetaID index collection by block height
	// key: meta_id:block_height, value: JSON(IndexerUserAvatar)
	metaIDKey := avatar.MetaId + ":" + blockHeightKey
	if err := p.collections[collectionAvatarMetaID].Set([]byte(metaIDKey), data, pebble.Sync); err != nil {
		return err
	}

	// Store in MetaID index collection by timestamp
	// key: meta_id:timestamp, value: JSON(IndexerUserAvatar)
	metaIDTimestampKey := avatar.MetaId + ":" + timestampKey
	if err := p.collections[collectionAvatarMetaIDTimestamp].Set([]byte(metaIDTimestampKey), data, pebble.Sync); err != nil {
		return err
	}

	// Store in Address index collection
	// key: address:block_height, value: JSON(IndexerUserAvatar)
	addressKey := avatar.Address + ":" + blockHeightKey
	if err := p.collections[collectionAvatarAddr].Set([]byte(addressKey), data, pebble.Sync); err != nil {
		return err
	}

	// Store in Hash index collection
	// key: hash:pin_id, value: JSON(IndexerUserAvatar)
	hashKey := avatar.FileMd5 + ":" + avatar.PinID
	if err := p.collections[collectionAvatarHash].Set([]byte(hashKey), data, pebble.Sync); err != nil {
		return err
	}

	// Update latest avatar for this MetaID (compare timestamp)
	// key: meta_id, value: JSON(IndexerUserAvatar)
	latestAvatarDB := p.collections[collectionLasestAvatarMetaID]

	// Check if there's an existing latest avatar for this MetaID
	existingData, closer, err := latestAvatarDB.Get([]byte(avatar.MetaId))
	if err != nil && err != pebble.ErrNotFound {
		return err
	}

	shouldUpdate := false
	if err == pebble.ErrNotFound {
		// No existing avatar, this is the first one
		shouldUpdate = true
	} else {
		// Compare timestamp with existing avatar
		defer closer.Close()
		var existingAvatar model.IndexerUserAvatar
		if err := json.Unmarshal(existingData, &existingAvatar); err != nil {
			return err
		}

		// Update if new avatar has a later timestamp
		if avatar.Timestamp > existingAvatar.Timestamp {
			shouldUpdate = true
			log.Printf("Updating latest avatar for MetaID %s: old timestamp=%d, new timestamp=%d",
				avatar.MetaId, existingAvatar.Timestamp, avatar.Timestamp)
		}
	}

	if shouldUpdate {
		if err := latestAvatarDB.Set([]byte(avatar.MetaId), data, pebble.Sync); err != nil {
			return err
		}
		log.Printf("Latest avatar updated for MetaID: %s (timestamp: %d)", avatar.MetaId, avatar.Timestamp)
	}

	return nil
}

func (p *PebbleDatabase) GetIndexerUserAvatarByPinID(pinID string) (*model.IndexerUserAvatar, error) {
	// Get avatar data directly from PinID collection
	data, closer, err := p.collections[collectionAvatarPinID].Get([]byte(pinID))
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	defer closer.Close()

	var avatar model.IndexerUserAvatar
	if err := json.Unmarshal(data, &avatar); err != nil {
		return nil, err
	}

	return &avatar, nil
}

func (p *PebbleDatabase) GetIndexerUserAvatarByMetaID(metaID string) (*model.IndexerUserAvatar, error) {
	// Try to get from latest avatar collection first
	latestAvatarDB := p.collections[collectionLasestAvatarMetaID]
	data, closer, err := latestAvatarDB.Get([]byte(metaID))
	if err == nil {
		defer closer.Close()
		var avatar model.IndexerUserAvatar
		if err := json.Unmarshal(data, &avatar); err != nil {
			return nil, err
		}
		return &avatar, nil
	}

	// If not found in latest collection or error, fallback to timestamp-based query
	if err != pebble.ErrNotFound {
		log.Printf("Error getting latest avatar for MetaID %s: %v, falling back to timestamp query", metaID, err)
	}

	// Fallback: query from timestamp collection and get the latest one
	timestampDB := p.collections[collectionAvatarMetaIDTimestamp]
	prefix := metaID + ":"

	iter, err := timestampDB.NewIter(&pebble.IterOptions{
		LowerBound: []byte(prefix),
		UpperBound: []byte(prefix + "~"),
	})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	// Seek to last (highest timestamp)
	if !iter.Last() {
		return nil, ErrNotFound
	}

	var avatar model.IndexerUserAvatar
	if err := json.Unmarshal(iter.Value(), &avatar); err != nil {
		return nil, err
	}

	return &avatar, nil
}

func (p *PebbleDatabase) GetIndexerUserAvatarByAddress(address string) (*model.IndexerUserAvatar, error) {
	addressDB := p.collections[collectionAvatarAddr]
	prefix := address + ":"

	iter, err := addressDB.NewIter(&pebble.IterOptions{
		LowerBound: []byte(prefix),
		UpperBound: []byte(prefix + "~"),
	})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	if !iter.Last() {
		return nil, ErrNotFound
	}

	var avatar model.IndexerUserAvatar
	if err := json.Unmarshal(iter.Value(), &avatar); err != nil {
		return nil, err
	}

	return &avatar, nil
}

func (p *PebbleDatabase) UpdateIndexerUserAvatar(avatar *model.IndexerUserAvatar) error {
	return p.CreateIndexerUserAvatar(avatar)
}

func (p *PebbleDatabase) ListIndexerUserAvatarsWithCursor(cursor int64, size int) ([]*model.IndexerUserAvatar, error) {
	var avatars []*model.IndexerUserAvatar

	avatarPinDB := p.collections[collectionAvatarPinID]

	// Create iterator for PinID collection
	iter, err := avatarPinDB.NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	// Start from last
	iter.Last()

	// Cursor is based on PinID (not sequential ID)
	cursorPinID := ""
	if cursor > 0 {
		// For now, we'll skip cursor logic and iterate from end
	}

	count := 0
	for iter.Valid() && count < size {
		var avatar model.IndexerUserAvatar
		if err := json.Unmarshal(iter.Value(), &avatar); err != nil {
			iter.Prev()
			continue
		}

		// Skip until cursor is reached
		if cursorPinID != "" && avatar.PinID == cursorPinID {
			cursorPinID = ""
			iter.Prev()
			continue
		}

		avatars = append(avatars, &avatar)
		count++
		iter.Prev()
	}

	return avatars, nil
}

// IndexerSyncStatus operations

func (p *PebbleDatabase) CreateOrUpdateIndexerSyncStatus(status *model.IndexerSyncStatus) error {
	if status.ID == 0 {
		status.ID = p.statusIDCounter.Add(1)
		// Save counter
		p.collections[collectionCounters].Set(
			[]byte(keyStatusCounter),
			[]byte(strconv.FormatInt(status.ID, 10)),
			pebble.Sync,
		)
	}

	data, err := json.Marshal(status)
	if err != nil {
		return err
	}

	syncStatusDB := p.collections[collectionSyncStatus]

	// Store by chain name (primary key for sync status)
	return syncStatusDB.Set([]byte(status.ChainName), data, pebble.Sync)
}

func (p *PebbleDatabase) GetIndexerSyncStatusByChainName(chainName string) (*model.IndexerSyncStatus, error) {
	syncStatusDB := p.collections[collectionSyncStatus]

	data, closer, err := syncStatusDB.Get([]byte(chainName))
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	defer closer.Close()

	var status model.IndexerSyncStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

func (p *PebbleDatabase) UpdateIndexerSyncStatusHeight(chainName string, height int64) error {
	status, err := p.GetIndexerSyncStatusByChainName(chainName)
	if err != nil {
		return err
	}

	status.CurrentSyncHeight = height
	return p.CreateOrUpdateIndexerSyncStatus(status)
}

func (p *PebbleDatabase) GetAllIndexerSyncStatus() ([]*model.IndexerSyncStatus, error) {
	var statuses []*model.IndexerSyncStatus

	syncStatusDB := p.collections[collectionSyncStatus]

	iter, err := syncStatusDB.NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		var status model.IndexerSyncStatus
		if err := json.Unmarshal(iter.Value(), &status); err != nil {
			continue
		}
		statuses = append(statuses, &status)
	}

	return statuses, nil
}

// Close close all database connections
func (p *PebbleDatabase) Close() error {
	var lastErr error
	for name, db := range p.collections {
		if err := db.Close(); err != nil {
			log.Printf("Failed to close collection %s: %v", name, err)
			lastErr = err
		}
	}
	return lastErr
}
