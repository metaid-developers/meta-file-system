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
	TotalFiles int64 `json:"total_files" example:"12345"`
}

// UserInfoListResponse user info list response structure
type UserInfoListResponse struct {
	Users      []*model.IndexerUserInfo `json:"users"`
	NextCursor int64                    `json:"next_cursor" example:"100"`
	HasMore    bool                     `json:"has_more" example:"true"`
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

// ToIndexerStatsResponse convert stats to response
func ToIndexerStatsResponse(totalFiles int64) IndexerStatsResponse {
	return IndexerStatsResponse{
		TotalFiles: totalFiles,
	}
}

// ToUserInfoListResponse convert to user info list response
func ToUserInfoListResponse(users []*model.IndexerUserInfo, nextCursor int64, hasMore bool) UserInfoListResponse {
	return UserInfoListResponse{
		Users:      users,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}
}
