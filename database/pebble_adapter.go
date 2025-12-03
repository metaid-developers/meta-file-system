package database

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
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
	collectionLatestFileInfo  = "latest_file_info"  // key: {first_pin_id}, value: JSON(IndexerFile) - 最新文件信息
	collectionFilePinID       = "file_pin"          // key: {pin_id}, value: JSON(IndexerFile) - PinID 到 ID 的映射
	collectionFileAddress     = "file_addr"         // key: {address}:{first_pin_id}, value: JSON(IndexerFile) - 按地址索引
	collectionFileMetaID      = "file_meta"         // key: {meta_id}:{first_pin_id}, value: JSON(IndexerFile) - 按 MetaID 索引
	collectionFileHash        = "file_hash"         // key: {hash}:{pin_id}, value: JSON(IndexerFile) - 按 Hash 索引
	collectionFileInfoHistory = "file_info_history" // key: {first_pin_id}, value: JSON(List[{pin_id, path, operation, content_type, chain_name, block_height, timestamp}]) - 按地址索引

	// Avatar collections
	collectionAvatarPinID           = "avatar_pin"            // key: {pin_id}, value: JSON(IndexerUserAvatar) - PinID 到 ID 的映射
	collectionAvatarMetaID          = "avatar_meta"           // key: {meta_id}:{block_height}, value: JSON(IndexerUserAvatar) - 按 MetaID 索引
	collectionAvatarMetaIDTimestamp = "avatar_meta_timestamp" // key: {meta_id}:{timestamp}, value: JSON(IndexerUserAvatar) - 按 MetaID 和时间戳索引
	collectionAvatarAddr            = "avatar_addr"           // key: {address}:{block_height}, value: JSON(IndexerUserAvatar) - 按地址索引
	collectionAvatarHash            = "avatar_hash"           // key: {hash}:{pin_id}, value: JSON(IndexerUserAvatar) - 按 Hash 索引
	collectionLasestAvatarMetaID    = "avatar_lasest_meta_id" // key: {meta_id}, value: JSON(IndexerUserAvatar) - 按 MetaID 索引

	// FileChunk collections
	collectionFileChunkPinID       = "file_chunk_pin"    // key: {pin_id}, value: JSON(IndexerFileChunk) - PinID 到 chunk 的映射
	collectionFileChunkParentPinID = "file_chunk_parent" // key: {parent_pin_id}:{chunk_index}, value: JSON(IndexerFileChunk) - 按父 PIN ID 索引

	// UserInfo collections
	collectionMetaIdAddress               = "meta_id_address"                  // key: {meta_id} Or {address}, value: JSON({meta_id, address}) - 按 MetaID 或地址索引
	collectionLatestUserNameInfo          = "latest_user_name_info"            // key: {meta_id} Or {address}, value: JSON({name, pin_id, chain_name, block_height, timestamp}) - 按 MetaID 索引
	collectionLatestUserAvatarInfo        = "latest_user_avatar_info"          // key: {meta_id} Or {address}, value: JSON({avatar, pin_id, chain_name, block_height, timestamp}) - 按 MetaID 索引
	collectionLatestUserChatPublicKeyInfo = "latest_user_chat_public_key_info" // key: {meta_id} Or {address}, value: JSON({chat_public_key, pin_id, block_height, timestamp}) - 按 MetaID 或地址和区块高度索引
	collectionUserNameInfoHistory         = "user_name_info_history"           // key: {meta_id} Or {address}, value: JSON(List[{name, pin_id, chain_name, block_height, timestamp}]) - 按 MetaID 或地址和区块高度索引
	collectionUserAvatarInfoHistory       = "user_avatar_info_history"         // key: {meta_id} Or {address}, value: JSON(List[{avatar, pin_id, chain_name, block_height, timestamp}]) - 按 MetaID 或地址和区块高度索引
	collectionUserChatPublicKeyHistory    = "user_chat_public_key_history"     // key: {meta_id} Or {address}, value: JSON(List[{chat_public_key, pin_id, chain_name, block_height, timestamp}]) - 按 MetaID 或地址和区块高度索引
	collectionMetaIdTimestamp             = "meta_id_timestamp"                // key: {timestamp}:{meta_id}, value: JSON({meta_id, timestamp}) - 按 MetaID 和时间戳索引

	// PinInfo collections
	collectionPinInfo = "pin_info" // key: {pin_id}, value: JSON({path, operation, content_type, chain_name, block_height, timestamp}) - 按 PIN ID 索引

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
		collectionLatestFileInfo,
		collectionFilePinID,
		collectionFileAddress,
		collectionFileMetaID,
		collectionFileHash,
		collectionFileInfoHistory,
		collectionAvatarPinID,
		collectionAvatarMetaID,
		collectionAvatarMetaIDTimestamp,
		collectionAvatarAddr,
		collectionAvatarHash,
		collectionLasestAvatarMetaID,
		collectionFileChunkPinID,
		collectionFileChunkParentPinID,
		collectionMetaIdAddress,
		collectionMetaIdTimestamp,
		collectionLatestUserNameInfo,
		collectionLatestUserAvatarInfo,
		collectionLatestUserChatPublicKeyInfo,
		collectionUserNameInfoHistory,
		collectionUserAvatarInfoHistory,
		collectionUserChatPublicKeyHistory,
		collectionPinInfo,
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

// paginateFilesByTimestampDesc sorts files by timestamp desc (fallback PinID) then slices by cursor+size.
func paginateFilesByTimestampDesc(files []*model.IndexerFile, cursor int64, size int) ([]*model.IndexerFile, int64) {
	if len(files) == 0 || size <= 0 {
		return nil, cursor
	}

	if cursor < 0 {
		cursor = 0
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i].Timestamp == files[j].Timestamp {
			return files[i].PinID > files[j].PinID
		}
		return files[i].Timestamp > files[j].Timestamp
	})

	start := int(cursor)
	if start >= len(files) {
		return nil, cursor
	}

	end := start + size
	if end > len(files) {
		end = len(files)
	}

	paged := files[start:end]
	nextCursor := cursor + int64(len(paged))
	return paged, nextCursor
}

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

	// Store in LatestFileInfo collection (by first_pin_id)
	// key: first_pin_id, value: JSON(IndexerFile)
	if file.FirstPinID != "" {
		latestFileDB := p.collections[collectionLatestFileInfo]

		// Check if there's an existing file info
		existingData, closer, err := latestFileDB.Get([]byte(file.FirstPinID))
		if err != nil && err != pebble.ErrNotFound {
			return err
		}

		shouldUpdate := false
		if err == pebble.ErrNotFound {
			// No existing file, this is the first one
			shouldUpdate = true
		} else {
			// Compare timestamp with existing file
			defer closer.Close()
			var existingFile model.IndexerFile
			if err := json.Unmarshal(existingData, &existingFile); err != nil {
				return err
			}

			// Update if new file has a later timestamp
			if file.Timestamp > existingFile.Timestamp {
				shouldUpdate = true
			}
		}

		if shouldUpdate {
			if err := latestFileDB.Set([]byte(file.FirstPinID), data, pebble.Sync); err != nil {
				return err
			}
		}
	}

	// Store in Address index collection
	// key: address:first_pin_id, value: JSON(IndexerFile)
	firstPinID := file.FirstPinID
	if firstPinID == "" {
		firstPinID = file.PinID // Fallback to PinID if FirstPinID is not set
	}
	addressKey := file.CreatorAddress + ":" + firstPinID
	if err := p.collections[collectionFileAddress].Set([]byte(addressKey), data, pebble.Sync); err != nil {
		return err
	}

	// Store in MetaID index collection
	// key: meta_id:first_pin_id, value: JSON(IndexerFile)
	metaIDKey := file.CreatorMetaId + ":" + firstPinID
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

func (p *PebbleDatabase) ListIndexerFilesWithCursor(cursor int64, size int) ([]*model.IndexerFile, int64, error) {
	filePinDB := p.collections[collectionFilePinID]

	// Create iterator for PinID collection
	iter, err := filePinDB.NewIter(nil)
	if err != nil {
		return nil, 0, err
	}
	defer iter.Close()

	var files []*model.IndexerFile
	for iter.First(); iter.Valid(); iter.Next() {
		var file model.IndexerFile
		if err := json.Unmarshal(iter.Value(), &file); err != nil {
			continue
		}

		if file.Status == model.StatusSuccess {
			fileCopy := file
			files = append(files, &fileCopy)
		}
	}

	sorted, nextCursor := paginateFilesByTimestampDesc(files, cursor, size)
	return sorted, nextCursor, nil
}

func (p *PebbleDatabase) GetIndexerFilesByCreatorAddressWithCursor(address string, cursor int64, size int) ([]*model.IndexerFile, int64, error) {
	addressDB := p.collections[collectionFileAddress]
	prefix := address + ":"

	// Create iterator with prefix
	// key format: address:pin_id
	iter, err := addressDB.NewIter(&pebble.IterOptions{
		LowerBound: []byte(prefix),
		UpperBound: []byte(prefix + "~"),
	})
	if err != nil {
		return nil, 0, err
	}
	defer iter.Close()

	var files []*model.IndexerFile
	for iter.First(); iter.Valid(); iter.Next() {
		var file model.IndexerFile
		if err := json.Unmarshal(iter.Value(), &file); err != nil {
			continue
		}

		if file.Status == model.StatusSuccess {
			fileCopy := file
			files = append(files, &fileCopy)
		}
	}

	sorted, nextCursor := paginateFilesByTimestampDesc(files, cursor, size)
	return sorted, nextCursor, nil
}

func (p *PebbleDatabase) GetIndexerFilesByCreatorMetaIDWithCursor(metaID string, cursor int64, size int) ([]*model.IndexerFile, int64, error) {
	metaIDDB := p.collections[collectionFileMetaID]
	prefix := metaID + ":"

	// Create iterator with prefix
	// key format: meta_id:pin_id
	iter, err := metaIDDB.NewIter(&pebble.IterOptions{
		LowerBound: []byte(prefix),
		UpperBound: []byte(prefix + "~"),
	})
	if err != nil {
		return nil, 0, err
	}
	defer iter.Close()

	var files []*model.IndexerFile
	for iter.First(); iter.Valid(); iter.Next() {
		var file model.IndexerFile
		if err := json.Unmarshal(iter.Value(), &file); err != nil {
			continue
		}

		if file.Status == model.StatusSuccess {
			fileCopy := file
			files = append(files, &fileCopy)
		}
	}

	sorted, nextCursor := paginateFilesByTimestampDesc(files, cursor, size)
	return sorted, nextCursor, nil
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

// IndexerFileChunk operations

func (p *PebbleDatabase) CreateIndexerFileChunk(chunk *model.IndexerFileChunk) error {
	// Serialize chunk
	data, err := json.Marshal(chunk)
	if err != nil {
		return err
	}

	// Store in PinID collection (primary index)
	if err := p.collections[collectionFileChunkPinID].Set([]byte(chunk.PinID), data, pebble.Sync); err != nil {
		return err
	}

	// Store in ParentPinID collection if parent is set
	if chunk.ParentPinID != "" {
		parentKey := fmt.Sprintf("%s:%d", chunk.ParentPinID, chunk.ChunkIndex)
		if err := p.collections[collectionFileChunkParentPinID].Set([]byte(parentKey), data, pebble.Sync); err != nil {
			return err
		}
	}

	return nil
}

func (p *PebbleDatabase) GetIndexerFileChunkByPinID(pinID string) (*model.IndexerFileChunk, error) {
	// Get chunk data directly from PinID collection
	data, closer, err := p.collections[collectionFileChunkPinID].Get([]byte(pinID))
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	defer closer.Close()

	var chunk model.IndexerFileChunk
	if err := json.Unmarshal(data, &chunk); err != nil {
		return nil, err
	}

	return &chunk, nil
}

func (p *PebbleDatabase) GetIndexerFileChunksByParentPinID(parentPinID string) ([]*model.IndexerFileChunk, error) {
	var chunks []*model.IndexerFileChunk

	parentDB := p.collections[collectionFileChunkParentPinID]

	// Create iterator with prefix filter for parentPinID
	prefix := []byte(parentPinID + ":")
	iter, err := parentDB.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xFF),
	})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	// Iterate through all chunks with this parent
	for iter.First(); iter.Valid(); iter.Next() {
		var chunk model.IndexerFileChunk
		if err := json.Unmarshal(iter.Value(), &chunk); err != nil {
			continue
		}
		chunks = append(chunks, &chunk)
	}

	// Sort by chunk_index (they should already be in order due to key format)
	// But we can verify and sort if needed
	return chunks, nil
}

func (p *PebbleDatabase) UpdateIndexerFileChunk(chunk *model.IndexerFileChunk) error {
	// Simply recreate (overwrite)
	return p.CreateIndexerFileChunk(chunk)
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

// UserInfo operations

// CreateOrUpdateLatestUserNameInfo create or update latest user name info
func (p *PebbleDatabase) CreateOrUpdateLatestUserNameInfo(info *model.UserNameInfo, metaID string) error {
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}

	db := p.collections[collectionLatestUserNameInfo]

	// Check if there's an existing name info
	existingData, closer, err := db.Get([]byte(metaID))
	if err != nil && err != pebble.ErrNotFound {
		return err
	}

	shouldUpdate := false
	if err == pebble.ErrNotFound {
		// No existing info, this is the first one
		shouldUpdate = true
	} else {
		// Compare timestamp with existing info
		defer closer.Close()
		var existingInfo model.UserNameInfo
		if err := json.Unmarshal(existingData, &existingInfo); err != nil {
			return err
		}

		// Update if new info has a later timestamp
		if info.Timestamp > existingInfo.Timestamp {
			shouldUpdate = true
		}
	}

	if shouldUpdate {
		if err := db.Set([]byte(metaID), data, pebble.Sync); err != nil {
			return err
		}
		log.Printf("Latest user name updated for MetaID: %s (timestamp: %d)", metaID, info.Timestamp)
	}

	return nil
}

// GetLatestUserNameInfo get latest user name info by MetaID or Address
func (p *PebbleDatabase) GetLatestUserNameInfo(key string) (*model.UserNameInfo, error) {
	db := p.collections[collectionLatestUserNameInfo]

	data, closer, err := db.Get([]byte(key))
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	defer closer.Close()

	var info model.UserNameInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// AddUserNameInfoHistory add user name info to history
func (p *PebbleDatabase) AddUserNameInfoHistory(info *model.UserNameInfo, metaID string) error {
	db := p.collections[collectionUserNameInfoHistory]

	// Get existing history
	var history []model.UserNameInfo
	existingData, closer, err := db.Get([]byte(metaID))
	if err == nil {
		defer closer.Close()
		if err := json.Unmarshal(existingData, &history); err != nil {
			return err
		}
	} else if err != pebble.ErrNotFound {
		return err
	}

	// Check if this PinID already exists in history (deduplicate)
	exists := false
	for i, h := range history {
		if h.PinID == info.PinID {
			// Update existing entry
			history[i] = *info
			exists = true
			log.Printf("Updated existing user name history entry: PinID=%s, MetaID=%s", info.PinID, metaID)
			break
		}
	}

	// Append new info if not exists
	if !exists {
		history = append(history, *info)
		log.Printf("Added new user name history entry: PinID=%s, MetaID=%s", info.PinID, metaID)
	}

	// Sort by timestamp desc
	sort.Slice(history, func(i, j int) bool {
		return history[i].Timestamp > history[j].Timestamp
	})

	// Save history
	data, err := json.Marshal(history)
	if err != nil {
		return err
	}

	return db.Set([]byte(metaID), data, pebble.Sync)
}

// GetUserNameInfoHistory get user name info history by MetaID or Address
func (p *PebbleDatabase) GetUserNameInfoHistory(key string) ([]model.UserNameInfo, error) {
	db := p.collections[collectionUserNameInfoHistory]

	data, closer, err := db.Get([]byte(key))
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	defer closer.Close()

	var history []model.UserNameInfo
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, err
	}

	return history, nil
}

// CreateOrUpdateLatestUserAvatarInfo create or update latest user avatar info
func (p *PebbleDatabase) CreateOrUpdateLatestUserAvatarInfo(info *model.UserAvatarInfo, metaID string) error {
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}

	db := p.collections[collectionLatestUserAvatarInfo]

	// Check if there's an existing avatar info
	existingData, closer, err := db.Get([]byte(metaID))
	if err != nil && err != pebble.ErrNotFound {
		return err
	}

	shouldUpdate := false
	if err == pebble.ErrNotFound {
		// No existing info, this is the first one
		shouldUpdate = true
	} else {
		// Compare timestamp with existing info
		defer closer.Close()
		var existingInfo model.UserAvatarInfo
		if err := json.Unmarshal(existingData, &existingInfo); err != nil {
			return err
		}

		// Update if new info has a later timestamp
		if info.Timestamp > existingInfo.Timestamp {
			shouldUpdate = true
		}
	}

	if shouldUpdate {
		if err := db.Set([]byte(metaID), data, pebble.Sync); err != nil {
			return err
		}
		log.Printf("Latest user avatar updated for MetaID: %s (timestamp: %d)", metaID, info.Timestamp)
	}

	return nil
}

// GetLatestUserAvatarInfo get latest user avatar info by MetaID or Address
func (p *PebbleDatabase) GetLatestUserAvatarInfo(key string) (*model.UserAvatarInfo, error) {
	db := p.collections[collectionLatestUserAvatarInfo]

	data, closer, err := db.Get([]byte(key))
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	defer closer.Close()

	var info model.UserAvatarInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// AddUserAvatarInfoHistory add user avatar info to history
func (p *PebbleDatabase) AddUserAvatarInfoHistory(info *model.UserAvatarInfo, metaID string) error {
	db := p.collections[collectionUserAvatarInfoHistory]

	// Get existing history
	var history []model.UserAvatarInfo
	existingData, closer, err := db.Get([]byte(metaID))
	if err == nil {
		defer closer.Close()
		if err := json.Unmarshal(existingData, &history); err != nil {
			return err
		}
	} else if err != pebble.ErrNotFound {
		return err
	}

	// Check if this PinID already exists in history (deduplicate)
	exists := false
	for i, h := range history {
		if h.PinID == info.PinID {
			// Update existing entry
			history[i] = *info
			exists = true
			log.Printf("Updated existing user avatar history entry: PinID=%s, MetaID=%s", info.PinID, metaID)
			break
		}
	}

	// Append new info if not exists
	if !exists {
		history = append(history, *info)
		log.Printf("Added new user avatar history entry: PinID=%s, MetaID=%s", info.PinID, metaID)
	}

	// Sort by timestamp desc
	sort.Slice(history, func(i, j int) bool {
		return history[i].Timestamp > history[j].Timestamp
	})

	// Save history
	data, err := json.Marshal(history)
	if err != nil {
		return err
	}

	return db.Set([]byte(metaID), data, pebble.Sync)
}

// GetUserAvatarInfoHistory get user avatar info history by MetaID or Address
func (p *PebbleDatabase) GetUserAvatarInfoHistory(key string) ([]model.UserAvatarInfo, error) {
	db := p.collections[collectionUserAvatarInfoHistory]

	data, closer, err := db.Get([]byte(key))
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	defer closer.Close()

	var history []model.UserAvatarInfo
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, err
	}

	return history, nil
}

// CreateOrUpdateLatestUserChatPublicKeyInfo create or update latest user chat public key info
func (p *PebbleDatabase) CreateOrUpdateLatestUserChatPublicKeyInfo(info *model.UserChatPublicKeyInfo, metaID string) error {
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}

	db := p.collections[collectionLatestUserChatPublicKeyInfo]

	// Check if there's an existing chat public key info
	existingData, closer, err := db.Get([]byte(metaID))
	if err != nil && err != pebble.ErrNotFound {
		return err
	}

	shouldUpdate := false
	if err == pebble.ErrNotFound {
		// No existing info, this is the first one
		shouldUpdate = true
	} else {
		// Compare timestamp with existing info
		defer closer.Close()
		var existingInfo model.UserChatPublicKeyInfo
		if err := json.Unmarshal(existingData, &existingInfo); err != nil {
			return err
		}

		// Update if new info has a later timestamp
		if info.Timestamp > existingInfo.Timestamp {
			shouldUpdate = true
		}
	}

	if shouldUpdate {
		if err := db.Set([]byte(metaID), data, pebble.Sync); err != nil {
			return err
		}
		log.Printf("Latest user chat public key updated for MetaID: %s (timestamp: %d)", metaID, info.Timestamp)
	}

	return nil
}

// GetLatestUserChatPublicKeyInfo get latest user chat public key info by MetaID or Address
func (p *PebbleDatabase) GetLatestUserChatPublicKeyInfo(key string) (*model.UserChatPublicKeyInfo, error) {
	db := p.collections[collectionLatestUserChatPublicKeyInfo]

	data, closer, err := db.Get([]byte(key))
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	defer closer.Close()

	var info model.UserChatPublicKeyInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// AddUserChatPublicKeyHistory add user chat public key info to history
func (p *PebbleDatabase) AddUserChatPublicKeyHistory(info *model.UserChatPublicKeyInfo, metaID string) error {
	db := p.collections[collectionUserChatPublicKeyHistory]

	// Get existing history
	var history []model.UserChatPublicKeyInfo
	existingData, closer, err := db.Get([]byte(metaID))
	if err == nil {
		defer closer.Close()
		if err := json.Unmarshal(existingData, &history); err != nil {
			return err
		}
	} else if err != pebble.ErrNotFound {
		return err
	}

	// Check if this PinID already exists in history (deduplicate)
	exists := false
	for i, h := range history {
		if h.PinID == info.PinID {
			// Update existing entry
			history[i] = *info
			exists = true
			log.Printf("Updated existing user chat public key history entry: PinID=%s, MetaID=%s", info.PinID, metaID)
			break
		}
	}

	// Append new info if not exists
	if !exists {
		history = append(history, *info)
		log.Printf("Added new user chat public key history entry: PinID=%s, MetaID=%s", info.PinID, metaID)
	}

	// Sort by timestamp desc
	sort.Slice(history, func(i, j int) bool {
		return history[i].Timestamp > history[j].Timestamp
	})

	// Save history
	data, err := json.Marshal(history)
	if err != nil {
		return err
	}

	return db.Set([]byte(metaID), data, pebble.Sync)
}

// GetUserChatPublicKeyHistory get user chat public key info history by MetaID or Address
func (p *PebbleDatabase) GetUserChatPublicKeyHistory(key string) ([]model.UserChatPublicKeyInfo, error) {
	db := p.collections[collectionUserChatPublicKeyHistory]

	data, closer, err := db.Get([]byte(key))
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	defer closer.Close()

	var history []model.UserChatPublicKeyInfo
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, err
	}

	return history, nil
}

// GetLatestFileInfoByFirstPinID get latest file info by first PIN ID
func (p *PebbleDatabase) GetLatestFileInfoByFirstPinID(firstPinID string) (*model.IndexerFile, error) {
	db := p.collections[collectionLatestFileInfo]

	data, closer, err := db.Get([]byte(firstPinID))
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

// AddFileInfoHistory add file info to history
func (p *PebbleDatabase) AddFileInfoHistory(history *model.FileInfoHistory, firstPinID string) error {
	db := p.collections[collectionFileInfoHistory]

	// key: first_pin_id
	key := firstPinID

	// Get existing history
	var historyList []model.FileInfoHistory
	existingData, closer, err := db.Get([]byte(key))
	if err == nil {
		defer closer.Close()
		if err := json.Unmarshal(existingData, &historyList); err != nil {
			return err
		}
	} else if err != pebble.ErrNotFound {
		return err
	}

	// Check if this PinID already exists in history (deduplicate)
	exists := false
	for i, h := range historyList {
		if h.PinID == history.PinID {
			// Update existing entry
			historyList[i] = *history
			exists = true
			log.Printf("Updated existing file history entry: PinID=%s, FirstPinID=%s", history.PinID, firstPinID)
			break
		}
	}

	// Append new history if not exists
	if !exists {
		historyList = append(historyList, *history)
		log.Printf("Added new file history entry: PinID=%s, FirstPinID=%s", history.PinID, firstPinID)
	}

	// Sort by timestamp desc
	sort.Slice(historyList, func(i, j int) bool {
		return historyList[i].Timestamp > historyList[j].Timestamp
	})

	// Save history
	data, err := json.Marshal(historyList)
	if err != nil {
		return err
	}

	return db.Set([]byte(key), data, pebble.Sync)
}

// GetFileInfoHistory get file info history by first PIN ID
func (p *PebbleDatabase) GetFileInfoHistory(firstPinID string) ([]model.FileInfoHistory, error) {
	db := p.collections[collectionFileInfoHistory]

	// key: first_pin_id
	key := firstPinID

	data, closer, err := db.Get([]byte(key))
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	defer closer.Close()

	var historyList []model.FileInfoHistory
	if err := json.Unmarshal(data, &historyList); err != nil {
		return nil, err
	}

	return historyList, nil
}

// MetaIdTimestamp operations

// SaveMetaIdTimestamp save MetaID with timestamp (only keeps earliest timestamp per MetaID)
func (p *PebbleDatabase) SaveMetaIdTimestamp(metaID string, timestamp int64) error {
	if metaID == "" || timestamp <= 0 {
		return fmt.Errorf("metaID and timestamp must be valid")
	}

	db := p.collections[collectionMetaIdTimestamp]

	// Check if MetaID already has a timestamp recorded
	// We need to scan for existing entries with this MetaID
	prefix := []byte(fmt.Sprintf("%d:", 0))
	upperBound := []byte(fmt.Sprintf("%d:", 9999999999999)) // Max timestamp

	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: upperBound,
	})
	if err != nil {
		return err
	}
	defer iter.Close()

	// Look for existing entry with this MetaID
	var existingTimestamp int64 = 0
	var existingKey []byte
	for iter.First(); iter.Valid(); iter.Next() {
		var entry model.MetaIdTimestamp
		if err := json.Unmarshal(iter.Value(), &entry); err != nil {
			continue
		}
		if entry.MetaId == metaID {
			existingTimestamp = entry.Timestamp
			existingKey = append([]byte(nil), iter.Key()...) // Copy key
			break
		}
	}

	// If MetaID already exists with an earlier timestamp, skip
	if existingTimestamp > 0 && existingTimestamp <= timestamp {
		log.Printf("MetaID %s already has earlier or same timestamp: %d (new: %d), skipping", metaID, existingTimestamp, timestamp)
		return nil
	}

	// If we have an existing entry with a later timestamp, delete it
	if existingTimestamp > 0 && existingTimestamp > timestamp {
		if err := db.Delete(existingKey, pebble.Sync); err != nil {
			log.Printf("Failed to delete old MetaID timestamp entry: %v", err)
		}
		log.Printf("Deleted old MetaID timestamp entry: MetaID=%s, OldTimestamp=%d", metaID, existingTimestamp)
	}

	// Save new entry
	entry := &model.MetaIdTimestamp{
		MetaId:    metaID,
		Timestamp: timestamp,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	// key: timestamp:meta_id
	key := fmt.Sprintf("%d:%s", timestamp, metaID)
	if err := db.Set([]byte(key), data, pebble.Sync); err != nil {
		return err
	}

	log.Printf("MetaID timestamp saved: MetaID=%s, Timestamp=%d", metaID, timestamp)
	return nil
}

// ListMetaIdsByTimestamp list MetaIDs ordered by timestamp (descending) with cursor pagination
// cursor: last timestamp (0 for first page)
// size: page size
// Returns: list of MetaIdTimestamp, nextCursor (timestamp), hasMore, error
func (p *PebbleDatabase) ListMetaIdsByTimestamp(cursor int64, size int) ([]model.MetaIdTimestamp, int64, bool, error) {
	if size <= 0 {
		size = 20
	}

	db := p.collections[collectionMetaIdTimestamp]

	// Create iterator
	iter, err := db.NewIter(nil)
	if err != nil {
		return nil, 0, false, err
	}
	defer iter.Close()

	var results []model.MetaIdTimestamp
	count := 0

	// If cursor is provided, seek to cursor position
	if cursor > 0 {
		// Seek to the cursor position (timestamp)
		seekKey := []byte(fmt.Sprintf("%d:", cursor))
		iter.SeekLT(seekKey) // Seek to less than cursor (for descending order)
	} else {
		// Start from the last (highest timestamp)
		iter.Last()
	}

	// Iterate in reverse order (descending timestamp)
	for iter.Valid() && count < size {
		var entry model.MetaIdTimestamp
		if err := json.Unmarshal(iter.Value(), &entry); err != nil {
			iter.Prev()
			continue
		}

		results = append(results, entry)
		count++
		iter.Prev()
	}

	// Determine next cursor and hasMore
	var nextCursor int64 = 0
	hasMore := false

	if len(results) > 0 {
		// Next cursor is the timestamp of the last result
		nextCursor = results[len(results)-1].Timestamp
		// Check if there are more records
		hasMore = iter.Valid()
	}

	return results, nextCursor, hasMore, nil
}

// MetaIdAddress operations

// SaveMetaIdAddress save or update MetaID-Address mapping (supports bidirectional lookup)
func (p *PebbleDatabase) SaveMetaIdAddress(metaID, address string) error {
	if metaID == "" || address == "" {
		return fmt.Errorf("metaID and address cannot be empty")
	}

	db := p.collections[collectionMetaIdAddress]

	mapping := &model.MetaIdAddress{
		MetaId:  metaID,
		Address: address,
	}

	data, err := json.Marshal(mapping)
	if err != nil {
		return err
	}

	// Store by MetaID as key
	if err := db.Set([]byte(metaID), data, pebble.Sync); err != nil {
		return err
	}

	// Store by Address as key (for reverse lookup)
	if err := db.Set([]byte(address), data, pebble.Sync); err != nil {
		return err
	}

	log.Printf("MetaID-Address mapping saved: MetaID=%s, Address=%s", metaID, address)
	return nil
}

// GetAddressByMetaID get address by MetaID
func (p *PebbleDatabase) GetAddressByMetaID(metaID string) (string, error) {
	db := p.collections[collectionMetaIdAddress]

	data, closer, err := db.Get([]byte(metaID))
	if err != nil {
		if err == pebble.ErrNotFound {
			return "", ErrNotFound
		}
		return "", err
	}
	defer closer.Close()

	var mapping model.MetaIdAddress
	if err := json.Unmarshal(data, &mapping); err != nil {
		return "", err
	}

	return mapping.Address, nil
}

// GetMetaIDByAddress get MetaID by address
func (p *PebbleDatabase) GetMetaIDByAddress(address string) (string, error) {
	db := p.collections[collectionMetaIdAddress]

	data, closer, err := db.Get([]byte(address))
	if err != nil {
		if err == pebble.ErrNotFound {
			return "", ErrNotFound
		}
		return "", err
	}
	defer closer.Close()

	var mapping model.MetaIdAddress
	if err := json.Unmarshal(data, &mapping); err != nil {
		return "", err
	}

	return mapping.MetaId, nil
}

// PinInfo operations

// CreateOrUpdatePinInfo create or update PIN info
func (p *PebbleDatabase) CreateOrUpdatePinInfo(pinInfo *model.IndexerPinInfo) error {
	data, err := json.Marshal(pinInfo)
	if err != nil {
		return err
	}

	db := p.collections[collectionPinInfo]

	// Check if there's an existing PIN info
	existingData, closer, err := db.Get([]byte(pinInfo.PinID))
	if err != nil && err != pebble.ErrNotFound {
		return err
	}

	shouldUpdate := false
	if err == pebble.ErrNotFound {
		// No existing info, this is the first one
		shouldUpdate = true
	} else {
		// Compare timestamp with existing info
		defer closer.Close()
		var existingInfo model.IndexerPinInfo
		if err := json.Unmarshal(existingData, &existingInfo); err != nil {
			return err
		}

		// Update if new info has a later timestamp or same timestamp but different operation
		if pinInfo.Timestamp > existingInfo.Timestamp ||
			(pinInfo.Timestamp == existingInfo.Timestamp && pinInfo.Operation != existingInfo.Operation) {
			shouldUpdate = true
		}
	}

	if shouldUpdate {
		if err := db.Set([]byte(pinInfo.PinID), data, pebble.Sync); err != nil {
			return err
		}
		log.Printf("PIN info updated: PinID=%s, Path=%s, Operation=%s, Timestamp=%d",
			pinInfo.PinID, pinInfo.Path, pinInfo.Operation, pinInfo.Timestamp)
	}

	return nil
}

// GetPinInfoByPinID get PIN info by PIN ID
func (p *PebbleDatabase) GetPinInfoByPinID(pinID string) (*model.IndexerPinInfo, error) {
	db := p.collections[collectionPinInfo]

	data, closer, err := db.Get([]byte(pinID))
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	defer closer.Close()

	var pinInfo model.IndexerPinInfo
	if err := json.Unmarshal(data, &pinInfo); err != nil {
		return nil, err
	}

	return &pinInfo, nil
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
