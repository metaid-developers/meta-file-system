package respond

import (
	"time"

	"meta-file-system/model"
)

// IndexerFileResponse file information response structure
type IndexerFileResponse struct {
	// ID             int64     `json:"id" example:"1"`
	PinID         string `json:"pin_id" example:"abc123def456i0"`
	TxID          string `json:"tx_id" example:"abc123def456789"`
	Path          string `json:"path" example:"/file/test.jpg"`
	Operation     string `json:"operation" example:"create"`
	Encryption    string `json:"encryption" example:"0"`
	ContentType   string `json:"content_type" example:"image/jpeg"`
	FileType      string `json:"file_type" example:"image"`
	FileExtension string `json:"file_extension" example:".jpg"`
	FileName      string `json:"file_name" example:"test.jpg"`
	FileSize      int64  `json:"file_size" example:"102400"`
	FileMd5       string `json:"file_md5" example:"d41d8cd98f00b204e9800998ecf8427e"`
	FileHash      string `json:"file_hash" example:"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"`
	// StorageType    string    `json:"storage_type" example:"oss"`
	StoragePath    string `json:"storage_path" example:"indexer/mvc/pinid123i0.jpg"`
	ChainName      string `json:"chain_name" example:"mvc"`
	BlockHeight    int64  `json:"block_height" example:"12345"`
	Timestamp      int64  `json:"timestamp" example:"1699999999"`
	CreatorMetaId  string `json:"creator_meta_id" example:"abc123def456..."`
	CreatorAddress string `json:"creator_address" example:"1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa"`
	OwnerMetaId    string `json:"owner_meta_id" example:"abc123def456..."`
	OwnerAddress   string `json:"owner_address" example:"1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa"`
	// Status         string    `json:"status" example:"success"`
	// CreatedAt      time.Time `json:"created_at" example:"2024-01-01T00:00:00Z"`
	// UpdatedAt      time.Time `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

// IndexerAvatarResponse avatar information response structure
type IndexerAvatarResponse struct {
	// ID            int64     `json:"id" example:"1"`
	PinID         string    `json:"pin_id" example:"xyz789i0"`
	TxID          string    `json:"tx_id" example:"xyz789"`
	MetaId        string    `json:"meta_id" example:"abc123def456..."`
	Address       string    `json:"address" example:"1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa"`
	Avatar        string    `json:"avatar" example:"indexer/avatar/mvc/xyz789/xyz789i0.jpg"`
	ContentType   string    `json:"content_type" example:"image/jpeg"`
	FileSize      int64     `json:"file_size" example:"102400"`
	FileMd5       string    `json:"file_md5" example:"d41d8cd98f00b204e9800998ecf8427e"`
	FileHash      string    `json:"file_hash" example:"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"`
	FileExtension string    `json:"file_extension" example:".jpg"`
	FileType      string    `json:"file_type" example:"image"`
	ChainName     string    `json:"chain_name" example:"mvc"`
	BlockHeight   int64     `json:"block_height" example:"12345"`
	Timestamp     int64     `json:"timestamp" example:"1699999999"`
	CreatedAt     time.Time `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt     time.Time `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

// RescanRequest request structure for block rescan
type RescanRequest struct {
	Chain       string `json:"chain" binding:"required" example:"mvc"`
	StartHeight int64  `json:"start_height" binding:"required,gt=0" example:"100000"`
	EndHeight   int64  `json:"end_height" binding:"required,gtefield=StartHeight" example:"100100"`
}

// RescanResponse response structure for block rescan
type RescanResponse struct {
	Message     string `json:"message" example:"Block rescan task started successfully"`
	Chain       string `json:"chain" example:"mvc"`
	StartHeight int64  `json:"start_height" example:"100000"`
	EndHeight   int64  `json:"end_height" example:"100100"`
	TaskID      string `json:"task_id" example:"rescan_mvc_100000_100100_1699999999"`
}

// RescanStatusResponse response structure for rescan status query
type RescanStatusResponse struct {
	TaskID            string  `json:"task_id" example:"rescan_mvc_100000_100100_1699999999"`
	Chain             string  `json:"chain" example:"mvc"`
	Status            string  `json:"status" example:"running"` // idle, running, completed, cancelled, failed
	StartHeight       int64   `json:"start_height" example:"100000"`
	EndHeight         int64   `json:"end_height" example:"100100"`
	CurrentHeight     int64   `json:"current_height" example:"100050"`
	ProcessedBlocks   int64   `json:"processed_blocks" example:"50"`
	TotalBlocks       int64   `json:"total_blocks" example:"101"`
	Progress          float64 `json:"progress" example:"49.50"` // percentage
	Speed             float64 `json:"speed" example:"12.34"`    // blocks per second
	StartTime         int64   `json:"start_time" example:"1699999999"`
	ElapsedTime       int64   `json:"elapsed_time" example:"4050"`        // milliseconds
	EstimatedTimeLeft int64   `json:"estimated_time_left" example:"4100"` // milliseconds
	ErrorMessage      string  `json:"error_message,omitempty" example:""`
}

// RescanStopResponse response structure for stop rescan
type RescanStopResponse struct {
	Message string `json:"message" example:"Rescan task stopped successfully"`
	TaskID  string `json:"task_id" example:"rescan_mvc_100000_100100_1699999999"`
	Status  string `json:"status" example:"cancelled"`
}

// IndexerPinInfoResponse PIN information response structure
type IndexerPinInfoResponse struct {
	PinID       string `json:"pin_id" example:"abc123def456i0"`
	FirstPinID  string `json:"first_pin_id" example:"xyz789i0"`
	FirstPath   string `json:"first_path" example:"/protocols/simplebuzz/info/name"`
	Path        string `json:"path" example:"@xyz789i0"`
	Operation   string `json:"operation" example:"modify"`
	ContentType string `json:"content_type" example:"text/plain"`
	ChainName   string `json:"chain_name" example:"mvc"`
	BlockHeight int64  `json:"block_height" example:"12345"`
	Timestamp   int64  `json:"timestamp" example:"1699999999"`
}

// IndexerSyncStatusResponse sync status response structure
type IndexerSyncStatusResponse struct {
	// ID                int64     `json:"id" example:"1"`
	ChainName         string    `json:"chain_name" example:"mvc"`
	CurrentSyncHeight int64     `json:"current_sync_height" example:"12345"`
	LatestBlockHeight int64     `json:"latest_block_height" example:"12350"`
	CreatedAt         time.Time `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt         time.Time `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

// IndexerFileListResponse file list response structure
type IndexerFileListResponse struct {
	Files      []IndexerFileResponse `json:"files"`
	NextCursor int64                 `json:"next_cursor" example:"100"`
	HasMore    bool                  `json:"has_more" example:"true"`
}

// IndexerAvatarListResponse avatar list response structure
type IndexerAvatarListResponse struct {
	Avatars    []IndexerAvatarResponse `json:"avatars"`
	NextCursor int64                   `json:"next_cursor" example:"100"`
	HasMore    bool                    `json:"has_more" example:"true"`
}

// IndexerStatsResponse statistics response structure
type IndexerStatsResponse struct {
	TotalFiles int64            `json:"total_files" example:"12345"`
	ChainStats map[string]int64 `json:"chain_stats,omitempty"` // Per-chain file counts
}

// UserInfoListResponse user info list response structure
type UserInfoListResponse struct {
	Users      []*model.IndexerUserInfo `json:"users"`
	NextCursor int64                    `json:"next_cursor" example:"100"`
	HasMore    bool                     `json:"has_more" example:"true"`
	Total      int64                    `json:"total" example:"1000"` // Total number of users
}

// ToIndexerFileResponse convert model to response
func ToIndexerFileResponse(file *model.IndexerFile) IndexerFileResponse {
	if file == nil {
		return IndexerFileResponse{}
	}
	return IndexerFileResponse{
		// ID:             file.ID,
		PinID:         file.PinID,
		TxID:          file.TxID,
		Path:          file.Path,
		Operation:     file.Operation,
		Encryption:    file.Encryption,
		ContentType:   file.ContentType,
		FileType:      file.FileType,
		FileExtension: file.FileExtension,
		FileName:      file.FileName,
		FileSize:      file.FileSize,
		FileMd5:       file.FileMd5,
		FileHash:      file.FileHash,
		// StorageType:    file.StorageType,
		StoragePath:    file.StoragePath,
		ChainName:      file.ChainName,
		BlockHeight:    file.BlockHeight,
		Timestamp:      file.Timestamp,
		CreatorMetaId:  file.CreatorMetaId,
		CreatorAddress: file.CreatorAddress,
		OwnerMetaId:    file.OwnerMetaId,
		OwnerAddress:   file.OwnerAddress,
		// Status:         string(file.Status),
		// CreatedAt:      file.CreatedAt,
		// UpdatedAt:      file.UpdatedAt,
	}
}

// ToIndexerFileListResponse convert file list to response
func ToIndexerFileListResponse(files []*model.IndexerFile, nextCursor int64, hasMore bool) IndexerFileListResponse {
	var fileResponses []IndexerFileResponse
	for _, file := range files {
		fileResponses = append(fileResponses, ToIndexerFileResponse(file))
	}
	return IndexerFileListResponse{
		Files:      fileResponses,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}
}

// ToIndexerAvatarResponse convert model to response
func ToIndexerAvatarResponse(avatar *model.IndexerUserAvatar) IndexerAvatarResponse {
	if avatar == nil {
		return IndexerAvatarResponse{}
	}
	return IndexerAvatarResponse{
		// ID:            avatar.ID,
		PinID:         avatar.PinID,
		TxID:          avatar.TxID,
		MetaId:        avatar.MetaId,
		Address:       avatar.Address,
		Avatar:        avatar.Avatar,
		ContentType:   avatar.ContentType,
		FileSize:      avatar.FileSize,
		FileMd5:       avatar.FileMd5,
		FileHash:      avatar.FileHash,
		FileExtension: avatar.FileExtension,
		FileType:      avatar.FileType,
		ChainName:     avatar.ChainName,
		BlockHeight:   avatar.BlockHeight,
		Timestamp:     avatar.Timestamp,
		CreatedAt:     avatar.CreatedAt,
		UpdatedAt:     avatar.UpdatedAt,
	}
}

// ToIndexerAvatarListResponse convert avatar list to response
func ToIndexerAvatarListResponse(avatars []*model.IndexerUserAvatar, nextCursor int64, hasMore bool) IndexerAvatarListResponse {
	var avatarResponses []IndexerAvatarResponse
	for _, avatar := range avatars {
		avatarResponses = append(avatarResponses, ToIndexerAvatarResponse(avatar))
	}
	return IndexerAvatarListResponse{
		Avatars:    avatarResponses,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}
}

// ToIndexerSyncStatusResponse convert model to response
func ToIndexerSyncStatusResponse(status *model.IndexerSyncStatus, latestBlockHeight int64) IndexerSyncStatusResponse {
	if status == nil {
		return IndexerSyncStatusResponse{}
	}
	return IndexerSyncStatusResponse{
		// ID:                status.ID,
		ChainName:         status.ChainName,
		CurrentSyncHeight: status.CurrentSyncHeight,
		LatestBlockHeight: latestBlockHeight,
		CreatedAt:         status.CreatedAt,
		UpdatedAt:         status.UpdatedAt,
	}
}

// ToIndexerPinInfoResponse convert model to response
func ToIndexerPinInfoResponse(pinInfo *model.IndexerPinInfo) IndexerPinInfoResponse {
	if pinInfo == nil {
		return IndexerPinInfoResponse{}
	}
	return IndexerPinInfoResponse{
		PinID:       pinInfo.PinID,
		FirstPinID:  pinInfo.FirstPinID,
		FirstPath:   pinInfo.FirstPath,
		Path:        pinInfo.Path,
		Operation:   pinInfo.Operation,
		ContentType: pinInfo.ContentType,
		ChainName:   pinInfo.ChainName,
		BlockHeight: pinInfo.BlockHeight,
		Timestamp:   pinInfo.Timestamp,
	}
}

// ToIndexerStatsResponse convert stats to response
func ToIndexerStatsResponse(totalFiles int64) IndexerStatsResponse {
	return IndexerStatsResponse{
		TotalFiles: totalFiles,
	}
}

// ToIndexerStatsResponseWithChains convert stats with chain breakdown to response
func ToIndexerStatsResponseWithChains(totalFiles int64, chainStats map[string]int64) IndexerStatsResponse {
	return IndexerStatsResponse{
		TotalFiles: totalFiles,
		ChainStats: chainStats,
	}
}

// IndexerMultiChainSyncStatusResponse multi-chain sync status response
type IndexerMultiChainSyncStatusResponse struct {
	Chains []IndexerSyncStatusResponse `json:"chains"`
}

// ToIndexerMultiChainSyncStatusResponse convert multiple statuses to response
func ToIndexerMultiChainSyncStatusResponse(statuses []*model.IndexerSyncStatus, latestHeights map[string]int64) IndexerMultiChainSyncStatusResponse {
	chains := make([]IndexerSyncStatusResponse, 0, len(statuses))
	for _, status := range statuses {
		latestHeight := int64(0)
		if h, ok := latestHeights[status.ChainName]; ok {
			latestHeight = h
		}
		chains = append(chains, ToIndexerSyncStatusResponse(status, latestHeight))
	}
	return IndexerMultiChainSyncStatusResponse{
		Chains: chains,
	}
}

// ToUserInfoListResponse convert to user info list response
func ToUserInfoListResponse(users []*model.IndexerUserInfo, nextCursor int64, hasMore bool, total int64) UserInfoListResponse {
	return UserInfoListResponse{
		Users:      users,
		NextCursor: nextCursor,
		HasMore:    hasMore,
		Total:      total,
	}
}

// MetaIDUserInfo MetaID user info response (compatible with external API format)
type MetaIDUserInfo struct {
	Metaid       string `json:"metaid" example:"abc123def456..."`
	Name         string `json:"name" example:"John Doe"`
	Address      string `json:"address" example:"1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa"`
	Avatar       string `json:"avatar" example:"https://oss.example.com/avatar.jpg"`
	AvatarId     string `json:"avatarId" example:"xyz789i0"`
	Chatpubkey   string `json:"chatpubkey" example:"02abc123..."`
	ChatpubkeyId string `json:"chatpubkeyId" example:"def456i0"`
}

// ToMetaIDUserInfo convert IndexerUserInfo to MetaIDUserInfo
func ToMetaIDUserInfo(userInfo *model.IndexerUserInfo) *MetaIDUserInfo {
	return &MetaIDUserInfo{
		Metaid:  userInfo.MetaId,
		Name:    userInfo.Name,
		Address: userInfo.Address,
		// Avatar:       userInfo.Avatar,
		Avatar:       "/content/" + userInfo.AvatarPinId,
		AvatarId:     userInfo.AvatarPinId,
		Chatpubkey:   userInfo.ChatPublicKey,
		ChatpubkeyId: userInfo.ChatPublicKeyPinId,
	}
}
