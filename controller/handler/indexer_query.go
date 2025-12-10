package handler

import (
	"fmt"
	"strconv"
	"time"

	"meta-file-system/controller/respond"
	"meta-file-system/service/indexer_service"

	"github.com/gin-gonic/gin"
)

// IndexerQueryHandler indexer query handler
type IndexerQueryHandler struct {
	indexerFileService *indexer_service.IndexerFileService
	syncStatusService  *indexer_service.SyncStatusService
	indexerService     *indexer_service.IndexerService
}

// NewIndexerQueryHandler create indexer query handler instance
func NewIndexerQueryHandler(indexerFileService *indexer_service.IndexerFileService, syncStatusService *indexer_service.SyncStatusService) *IndexerQueryHandler {
	return &IndexerQueryHandler{
		indexerFileService: indexerFileService,
		syncStatusService:  syncStatusService,
		indexerService:     nil,
	}
}

// SetIndexerService sets the indexer service (for rescan operations)
func (h *IndexerQueryHandler) SetIndexerService(indexerService *indexer_service.IndexerService) {
	h.indexerService = indexerService
}

// GetLatestByFirstPinID get latest file information by first PIN ID
// @Summary      Get latest file by first PIN ID
// @Description  Query latest file details by first PIN ID
// @Tags         Indexer File Query
// @Accept       json
// @Produce      json
// @Param        firstPinId  path      string  true  "First PIN ID"
// @Success      200         {object}  respond.Response{data=respond.IndexerFileResponse}
// @Failure      404         {object}  respond.Response
// @Router       /files/latest/{firstPinId} [get]
func (h *IndexerQueryHandler) GetLatestByFirstPinID(c *gin.Context) {
	firstPinID := c.Param("firstPinId")
	if firstPinID == "" {
		respond.InvalidParam(c, "firstPinId is required")
		return
	}

	file, err := h.indexerFileService.GetLatestFileByFirstPinID(firstPinID)
	if err != nil {
		respond.NotFound(c, err.Error())
		return
	}

	respond.Success(c, respond.ToIndexerFileResponse(file))
}

// GetByPinID get file information by PIN ID
// @Summary      Get file by PIN ID
// @Description  Query file details by PIN ID
// @Tags         Indexer File Query
// @Accept       json
// @Produce      json
// @Param        pinId  path      string  true  "PIN ID"
// @Success      200    {object}  respond.Response{data=respond.IndexerFileResponse}
// @Failure      404    {object}  respond.Response
// @Router       /files/{pinId} [get]
func (h *IndexerQueryHandler) GetByPinID(c *gin.Context) {
	pinID := c.Param("pinId")
	if pinID == "" {
		respond.InvalidParam(c, "pinId is required")
		return
	}

	file, err := h.indexerFileService.GetFileByPinID(pinID)
	if err != nil {
		respond.NotFound(c, err.Error())
		return
	}

	respond.Success(c, respond.ToIndexerFileResponse(file))
}

// GetPinInfoByPinID get PIN information by PIN ID from collectionPinInfo
// @Summary      Get PIN info by PIN ID
// @Description  Query PIN details from collectionPinInfo by PIN ID
// @Tags         Indexer PIN Query
// @Accept       json
// @Produce      json
// @Param        pinId  path      string  true  "PIN ID"
// @Success      200    {object}  respond.Response{data=respond.IndexerPinInfoResponse}
// @Failure      404    {object}  respond.Response
// @Router       /pins/{pinId} [get]
func (h *IndexerQueryHandler) GetPinInfoByPinID(c *gin.Context) {
	pinID := c.Param("pinId")
	if pinID == "" {
		respond.InvalidParam(c, "pinId is required")
		return
	}

	pinInfo, err := h.indexerFileService.GetPinInfoByPinID(pinID)
	if err != nil {
		respond.NotFound(c, err.Error())
		return
	}

	respond.Success(c, respond.ToIndexerPinInfoResponse(pinInfo))
}

// GetByCreatorAddress get file list by creator address
// @Summary      Get files by creator address
// @Description  Query file list by creator address with cursor pagination
// @Tags         Indexer File Query
// @Accept       json
// @Produce      json
// @Param        address  path   string  true   "Creator address"
// @Param        cursor   query  int     false  "Cursor" default(0)
// @Param        size     query  int     false  "Page size"             default(20)
// @Success      200      {object}  respond.Response{data=respond.IndexerFileListResponse}
// @Failure      500      {object}  respond.Response
// @Router       /files/creator/{address} [get]
func (h *IndexerQueryHandler) GetByCreatorAddress(c *gin.Context) {
	address := c.Param("address")
	if address == "" {
		respond.InvalidParam(c, "address is required")
		return
	}

	// Get cursor and size parameters
	cursorStr := c.DefaultQuery("cursor", "0")
	sizeStr := c.DefaultQuery("size", "20")

	cursor, _ := strconv.ParseInt(cursorStr, 10, 64)
	size, _ := strconv.Atoi(sizeStr)

	// Query file list
	files, nextCursor, hasMore, err := h.indexerFileService.GetFilesByCreatorAddress(address, cursor, size)
	if err != nil {
		respond.ServerError(c, err.Error())
		return
	}

	respond.Success(c, respond.ToIndexerFileListResponse(files, nextCursor, hasMore))
}

// GetByCreatorMetaID get file list by creator MetaID
// @Summary      Get files by creator MetaID
// @Description  Query file list by creator MetaID with cursor pagination
// @Tags         Indexer File Query
// @Accept       json
// @Produce      json
// @Param        metaId   path   string  true   "Creator MetaID"
// @Param        cursor   query  int     false  "Cursor" default(0)
// @Param        size     query  int     false  "Page size"             default(20)
// @Success      200      {object}  respond.Response{data=respond.IndexerFileListResponse}
// @Failure      500      {object}  respond.Response
// @Router       /files/metaid/{metaId} [get]
func (h *IndexerQueryHandler) GetByCreatorMetaID(c *gin.Context) {
	metaID := c.Param("metaId")
	if metaID == "" {
		respond.InvalidParam(c, "metaId is required")
		return
	}

	// Get cursor and size parameters
	cursorStr := c.DefaultQuery("cursor", "0")
	sizeStr := c.DefaultQuery("size", "20")

	cursor, _ := strconv.ParseInt(cursorStr, 10, 64)
	size, _ := strconv.Atoi(sizeStr)

	// Query file list
	files, nextCursor, hasMore, err := h.indexerFileService.GetFilesByCreatorMetaID(metaID, cursor, size)
	if err != nil {
		respond.ServerError(c, err.Error())
		return
	}

	respond.Success(c, respond.ToIndexerFileListResponse(files, nextCursor, hasMore))
}

// ListFiles get file list with cursor pagination
// @Summary      Query file list
// @Description  Query file list with cursor pagination
// @Tags         Indexer File Query
// @Accept       json
// @Produce      json
// @Param        cursor  query  int  false  "Cursor" default(0)
// @Param        size    query  int  false  "Page size"             default(20)
// @Success      200     {object}  respond.Response{data=respond.IndexerFileListResponse}
// @Failure      500     {object}  respond.Response
// @Router       /files [get]
func (h *IndexerQueryHandler) ListFiles(c *gin.Context) {
	// Get cursor and size parameters
	cursorStr := c.DefaultQuery("cursor", "0")
	sizeStr := c.DefaultQuery("size", "20")

	cursor, _ := strconv.ParseInt(cursorStr, 10, 64)
	size, _ := strconv.Atoi(sizeStr)

	// Query file list
	files, nextCursor, hasMore, err := h.indexerFileService.ListFiles(cursor, size)
	if err != nil {
		respond.ServerError(c, err.Error())
		return
	}

	respond.Success(c, respond.ToIndexerFileListResponse(files, nextCursor, hasMore))
}

// GetLatestFileContentByFirstPinID get latest file content by first PIN ID
// @Summary      Get latest file content
// @Description  Get latest file content by first PIN ID
// @Tags         Indexer File Query
// @Accept       json
// @Produce      octet-stream
// @Param        firstPinId  path      string  true  "First PIN ID"
// @Success      200         {file}    binary
// @Failure      404         {object}  respond.Response
// @Router       /files/content/latest/{firstPinId} [get]
func (h *IndexerQueryHandler) GetLatestFileContentByFirstPinID(c *gin.Context) {
	firstPinID := c.Param("firstPinId")
	if firstPinID == "" {
		respond.InvalidParam(c, "firstPinId is required")
		return
	}

	content, contentType, fileName, err := h.indexerFileService.GetLatestFileContentByFirstPinID(firstPinID)
	if err != nil {
		respond.NotFound(c, err.Error())
		return
	}

	// Set response headers
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "inline; filename=\""+fileName+"\"")
	c.Data(200, contentType, content)
}

// GetFileContent get file content by PIN ID
// @Summary      Get file content
// @Description  Get file content by PIN ID
// @Tags         Indexer File Query
// @Accept       json
// @Produce      octet-stream
// @Param        pinId  path      string  true  "PIN ID"
// @Success      200    {file}    binary
// @Failure      404    {object}  respond.Response
// @Router       /files/content/{pinId} [get]
func (h *IndexerQueryHandler) GetFileContent(c *gin.Context) {
	pinID := c.Param("pinId")
	if pinID == "" {
		respond.InvalidParam(c, "pinId is required")
		return
	}

	content, contentType, fileName, err := h.indexerFileService.GetFileContent(pinID)
	if err != nil {
		respond.NotFound(c, err.Error())
		return
	}

	// Set response headers
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "inline; filename=\""+fileName+"\"")
	c.Data(200, contentType, content)
}

// GetSyncStatus get indexer sync status
// @Summary      Get sync status
// @Description  Get current sync status for all chains (current sync height and latest block height)
// @Tags         Indexer Status
// @Accept       json
// @Produce      json
// @Success      200  {object}  respond.Response{data=respond.IndexerMultiChainSyncStatusResponse}
// @Failure      500  {object}  respond.Response
// @Router       /status [get]
func (h *IndexerQueryHandler) GetSyncStatus(c *gin.Context) {
	// Get all chain sync statuses
	statuses, err := h.syncStatusService.GetAllSyncStatus()
	if err != nil {
		respond.ServerError(c, err.Error())
		return
	}

	// If no statuses found, return empty array
	if len(statuses) == 0 {
		respond.Success(c, respond.IndexerMultiChainSyncStatusResponse{
			Chains: []respond.IndexerSyncStatusResponse{},
		})
		return
	}

	// Get latest block height from node for each chain
	latestHeights, err := h.syncStatusService.GetLatestBlockHeightsForAllChains()
	if err != nil {
		// If failed to get latest heights, use empty map as fallback
		latestHeights = make(map[string]int64)
		for _, status := range statuses {
			latestHeights[status.ChainName] = 0
		}
	}

	respond.Success(c, respond.ToIndexerMultiChainSyncStatusResponse(statuses, latestHeights))
}

// GetStats get indexer statistics (supports per-chain breakdown)
// @Summary      Get statistics
// @Description  Get indexer statistics (total files count and per-chain breakdown)
// @Tags         Indexer Status
// @Accept       json
// @Produce      json
// @Success      200  {object}  respond.Response{data=respond.IndexerStatsResponse}
// @Failure      500  {object}  respond.Response
// @Router       /stats [get]
func (h *IndexerQueryHandler) GetStats(c *gin.Context) {
	// Get total files count
	filesCount, err := h.indexerFileService.GetFilesCount()
	if err != nil {
		respond.ServerError(c, err.Error())
		return
	}

	// Get per-chain file counts
	chainStats, err := h.indexerFileService.GetFilesCountByChains()
	if err != nil {
		// If failed to get chain stats, just return total count
		respond.Success(c, respond.ToIndexerStatsResponse(filesCount))
		return
	}

	respond.Success(c, respond.ToIndexerStatsResponseWithChains(filesCount, chainStats))
}

// ============================================================
// Old Avatar methods - DEPRECATED (commented out)
// ============================================================

// // ListAvatars get avatar list with cursor pagination
// // @Summary      Query avatar list
// // @Description  Query avatar list with cursor pagination
// // @Tags         Indexer Avatar Query
// // @Accept       json
// // @Produce      json
// // @Param        cursor  query  int  false  "Cursor (last avatar ID)" default(0)
// // @Param        size    query  int  false  "Page size"               default(20)
// // @Success      200     {object}  respond.Response{data=respond.IndexerAvatarListResponse}
// // @Failure      500     {object}  respond.Response
// // @Router       /avatars [get]
// func (h *IndexerQueryHandler) ListAvatars(c *gin.Context) {
// 	// Get cursor and size parameters
// 	cursorStr := c.DefaultQuery("cursor", "0")
// 	sizeStr := c.DefaultQuery("size", "20")
//
// 	cursor, _ := strconv.ParseInt(cursorStr, 10, 64)
// 	size, _ := strconv.Atoi(sizeStr)
//
// 	// Query avatar list
// 	avatars, nextCursor, hasMore, err := h.indexerFileService.ListAvatars(cursor, size)
// 	if err != nil {
// 		respond.ServerError(c, err.Error())
// 		return
// 	}
//
// 	respond.Success(c, respond.ToIndexerAvatarListResponse(avatars, nextCursor, hasMore))
// }

// // GetLatestAvatarByMetaID get latest avatar by MetaID
// // @Summary      Get latest avatar by MetaID
// // @Description  Query the latest avatar information by MetaID
// // @Tags         Indexer Avatar Query
// // @Accept       json
// // @Produce      json
// // @Param        metaId  path  string  true  "MetaID"
// // @Success      200     {object}  respond.Response{data=respond.IndexerAvatarResponse}
// // @Failure      404     {object}  respond.Response
// // @Router       /avatars/metaid/{metaId} [get]
// func (h *IndexerQueryHandler) GetLatestAvatarByMetaID(c *gin.Context) {
// 	metaID := c.Param("metaId")
// 	if metaID == "" {
// 		respond.InvalidParam(c, "metaId is required")
// 		return
// 	}
//
// 	avatar, err := h.indexerFileService.GetLatestAvatarByMetaID(metaID)
// 	if err != nil {
// 		respond.NotFound(c, err.Error())
// 		return
// 	}
//
// 	respond.Success(c, respond.ToIndexerAvatarResponse(avatar))
// }

// // GetLatestAvatarByAddress get latest avatar by address
// // @Summary      Get latest avatar by address
// // @Description  Query the latest avatar information by address
// // @Tags         Indexer Avatar Query
// // @Accept       json
// // @Produce      json
// // @Param        address  path  string  true  "Address"
// // @Success      200      {object}  respond.Response{data=respond.IndexerAvatarResponse}
// // @Failure      404      {object}  respond.Response
// // @Router       /avatars/address/{address} [get]
// func (h *IndexerQueryHandler) GetLatestAvatarByAddress(c *gin.Context) {
// 	address := c.Param("address")
// 	if address == "" {
// 		respond.InvalidParam(c, "address is required")
// 		return
// 	}
//
// 	avatar, err := h.indexerFileService.GetLatestAvatarByAddress(address)
// 	if err != nil {
// 		respond.NotFound(c, err.Error())
// 		return
// 	}
//
// 	respond.Success(c, respond.ToIndexerAvatarResponse(avatar))
// }

// // GetAvatarContent get avatar content by PIN ID
// // @Summary      Get avatar content
// // @Description  Get avatar content by PIN ID
// // @Tags         Indexer Avatar Query
// // @Accept       json
// // @Produce      octet-stream
// // @Param        pinId  path      string  true  "PIN ID"
// // @Success      200    {file}    binary
// // @Failure      404    {object}  respond.Response
// // @Router       /avatars/content/{pinId} [get]
// func (h *IndexerQueryHandler) GetAvatarContent(c *gin.Context) {
// 	pinID := c.Param("pinId")
// 	if pinID == "" {
// 		respond.InvalidParam(c, "pinId is required")
// 		return
// 	}
//
// 	content, contentType, fileName, err := h.indexerFileService.GetAvatarContent(pinID)
// 	if err != nil {
// 		respond.NotFound(c, err.Error())
// 		return
// 	}
//
// 	// Set response headers
// 	c.Header("Content-Type", contentType)
// 	c.Header("Content-Disposition", "inline; filename=\""+fileName+"\"")
// 	c.Data(200, contentType, content)
// }

// ============================================================
// New UserInfo methods
// ============================================================

// GetUserInfoByMetaID get user information by MetaID
// @Summary      Get user info by MetaID
// @Description  Query user information (name, avatar, chat public key) by MetaID
// @Tags         Indexer User Info
// @Accept       json
// @Produce      json
// @Param        metaId  path  string  true  "MetaID"
// @Success      200     {object}  respond.Response{data=model.IndexerUserInfo}
// @Failure      404     {object}  respond.Response
// @Router       /users/metaid/{metaId} [get]
func (h *IndexerQueryHandler) GetUserInfoByMetaID(c *gin.Context) {
	metaID := c.Param("metaId")
	if metaID == "" {
		respond.InvalidParam(c, "metaId is required")
		return
	}

	userInfo, err := h.indexerFileService.GetUserInfoByMetaID(metaID)
	if err != nil {
		respond.NotFound(c, err.Error())
		return
	}

	respond.Success(c, userInfo)
}

// GetUserInfoByAddress get user information by address
// @Summary      Get user info by address
// @Description  Query user information (name, avatar, chat public key) by address
// @Tags         Indexer User Info
// @Accept       json
// @Produce      json
// @Param        address  path  string  true  "Address"
// @Success      200      {object}  respond.Response{data=model.IndexerUserInfo}
// @Failure      404      {object}  respond.Response
// @Router       /users/address/{address} [get]
func (h *IndexerQueryHandler) GetUserInfoByAddress(c *gin.Context) {
	address := c.Param("address")
	if address == "" {
		respond.InvalidParam(c, "address is required")
		return
	}

	userInfo, err := h.indexerFileService.GetUserInfoByAddress(address)
	if err != nil {
		respond.NotFound(c, err.Error())
		return
	}

	respond.Success(c, userInfo)
}

// GetMetaIDUserInfoByMetaID get MetaID format user info by MetaID
// @Summary      Get MetaID user info by MetaID
// @Description  Query user information in MetaID format by MetaID
// @Tags         Indexer User Info
// @Accept       json
// @Produce      json
// @Param        metaid  path  string  true  "MetaID"
// @Success      200     {object}  respond.Response{data=respond.MetaIDUserInfo}
// @Failure      404     {object}  respond.Response
// @Router       /info/metaid/{metaid} [get]
func (h *IndexerQueryHandler) GetMetaIDUserInfoByMetaID(c *gin.Context) {
	metaID := c.Param("metaid")
	if metaID == "" {
		respond.InvalidParam(c, "metaid is required")
		return
	}

	userInfo, err := h.indexerFileService.GetUserInfoByMetaID(metaID)
	if err != nil {
		respond.NotFound(c, err.Error())
		return
	}

	// Convert to MetaIDUserInfo format
	respond.SuccessWithCode(c, 1, respond.ToMetaIDUserInfo(userInfo))
}

// GetMetaIDUserInfoByAddress get MetaID format user info by address
// @Summary      Get MetaID user info by address
// @Description  Query user information in MetaID format by address
// @Tags         Indexer User Info
// @Accept       json
// @Produce      json
// @Param        address  path  string  true  "Address"
// @Success      200      {object}  respond.Response{data=respond.MetaIDUserInfo}
// @Failure      404      {object}  respond.Response
// @Router       /info/address/{address} [get]
func (h *IndexerQueryHandler) GetMetaIDUserInfoByAddress(c *gin.Context) {
	address := c.Param("address")
	if address == "" {
		respond.InvalidParam(c, "address is required")
		return
	}

	userInfo, err := h.indexerFileService.GetUserInfoByAddress(address)
	if err != nil {
		respond.NotFound(c, err.Error())
		return
	}

	// Convert to MetaIDUserInfo format
	respond.SuccessWithCode(c, 1, respond.ToMetaIDUserInfo(userInfo))
}

// SearchMetaIDUserInfo search MetaID format user info (fuzzy search)
// @Summary      Search MetaID user info (fuzzy)
// @Description  Fuzzy search user information by keyword and keytype (metaid or name)
// @Tags         Indexer User Info
// @Accept       json
// @Produce      json
// @Param        keyword  query  string  true   "Search keyword (partial MetaID or Name)"
// @Param        keytype  query  string  true   "Key type: metaid (fuzzy) or name (fuzzy)"
// @Param        limit    query  int     false  "Result limit (default: 10, max: 100)"
// @Success      200      {object}  respond.Response{data=[]respond.MetaIDUserInfo}
// @Failure      404      {object}  respond.Response
// @Router       /info/search [get]
func (h *IndexerQueryHandler) SearchMetaIDUserInfo(c *gin.Context) {
	keyword := c.Query("keyword")
	keytype := c.Query("keytype")
	limitStr := c.DefaultQuery("limit", "10")

	if keyword == "" {
		respond.InvalidParam(c, "keyword is required")
		return
	}

	// Parse limit
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100 // Max limit
	}

	users, err := h.indexerFileService.SearchUserInfo(keyword, keytype, limit)
	if err != nil {
		respond.NotFound(c, err.Error())
		return
	}

	// Convert to MetaIDUserInfo format (list)
	var result []respond.MetaIDUserInfo
	for _, user := range users {
		result = append(result, *respond.ToMetaIDUserInfo(user))
	}

	// respond.Success(c, result)
	c.JSON(200, result)
}

// ListUserInfo get user info list with pagination
// @Summary      Query user info list
// @Description  Query user info list with cursor pagination
// @Tags         Indexer User Info
// @Accept       json
// @Produce      json
// @Param        cursor  query  int  false  "Cursor" default(0)
// @Param        size    query  int  false  "Page size" default(20)
// @Success      200     {object}  respond.Response{data=respond.UserInfoListResponse}
// @Failure      500     {object}  respond.Response
// @Router       /users [get]
func (h *IndexerQueryHandler) ListUserInfo(c *gin.Context) {
	// Get cursor and size parameters
	cursorStr := c.DefaultQuery("cursor", "0")
	sizeStr := c.DefaultQuery("size", "20")

	cursor, _ := strconv.ParseInt(cursorStr, 10, 64)
	size, _ := strconv.Atoi(sizeStr)

	// Query user info list with total count
	users, nextCursor, hasMore, total, err := h.indexerFileService.GetUserInfoList(cursor, size)
	if err != nil {
		respond.ServerError(c, err.Error())
		return
	}

	respond.Success(c, respond.ToUserInfoListResponse(users, nextCursor, hasMore, total))
}

// GetUserInfoHistory get user info history by MetaID or Address
// @Summary      Get user info history
// @Description  Get all user info history (name, avatar, chat public key) by MetaID or Address
// @Tags         Indexer User Info
// @Accept       json
// @Produce      json
// @Param        key  path  string  true  "MetaID or Address"
// @Success      200  {object}  respond.Response{data=model.UserInfoHistory}
// @Failure      404  {object}  respond.Response
// @Failure      500  {object}  respond.Response
// @Router       /users/history/{key} [get]
func (h *IndexerQueryHandler) GetUserInfoHistory(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		respond.InvalidParam(c, "key is required")
		return
	}

	history, err := h.indexerFileService.GetUserInfoHistoryByKey(key)
	if err != nil {
		respond.NotFound(c, err.Error())
		return
	}

	respond.Success(c, history)
}

// GetAvatarContentByMetaID get avatar content by MetaID
// @Summary      Get avatar content by MetaID
// @Description  Get avatar content by user MetaID, returns content from storage or redirects to OSS
// @Tags         Indexer User Info
// @Accept       json
// @Produce      octet-stream
// @Param        metaId  path  string  true  "User MetaID"
// @Success      200     {file}    binary  "Avatar content"
// @Success      307     {string}  string  "Redirect to OSS URL"
// @Failure      404     {object}  respond.Response
// @Router       /users/metaid/{metaId}/avatar [get]
func (h *IndexerQueryHandler) GetAvatarContentByMetaID(c *gin.Context) {
	metaID := c.Param("metaId")
	if metaID == "" {
		respond.InvalidParam(c, "metaId is required")
		return
	}

	// Get avatar OSS URL or content by MetaID
	ossURL, contentType, fileName, fileType, isOSS, err := h.indexerFileService.GetAvatarOSSURLByMetaID(metaID)
	if err != nil {
		respond.NotFound(c, err.Error())
		return
	}

	// If it's an OSS URL, redirect to OSS
	if isOSS {
		c.Redirect(307, ossURL)
		return
	}

	// If not OSS, get content from storage
	content, contentType, fileName, err := h.indexerFileService.GetAvatarContentByMetaID(metaID)
	if err != nil {
		respond.NotFound(c, err.Error())
		return
	}

	// Set response headers
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", fileName))
	c.Header("X-File-Type", fileType)
	c.Data(200, contentType, content)
}

// GetAvatarContentByPinID get avatar content by avatar PIN ID
// @Summary      Get avatar content by PIN ID
// @Description  Get specific avatar version content by avatar PIN ID
// @Tags         Indexer User Info
// @Accept       json
// @Produce      octet-stream
// @Param        pinId  path  string  true  "Avatar PIN ID"
// @Success      200    {file}    binary  "Avatar content"
// @Failure      404    {object}  respond.Response
// @Router       /users/avatar/content/{pinId} [get]
func (h *IndexerQueryHandler) GetAvatarContentByPinID(c *gin.Context) {
	pinID := c.Param("pinId")
	if pinID == "" {
		respond.InvalidParam(c, "pinId is required")
		return
	}

	// Get avatar content by PIN ID from collectionUserAvatarInfo
	content, contentType, fileName, err := h.indexerFileService.GetAvatarContentByPinID(pinID)
	if err != nil {
		respond.NotFound(c, err.Error())
		return
	}

	// Set response headers
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", fileName))
	c.Data(200, contentType, content)
}

// GetFastAvatarContentByPinID get accelerated avatar content redirect to OSS by avatar PIN ID
// @Summary      Get accelerated avatar content (redirect to OSS)
// @Description  Redirect to OSS URL for avatar content by avatar PIN ID, supports preview/thumbnail processing
// @Tags         Indexer User Info
// @Accept       json
// @Produce      json
// @Param        pinId       path   string  true   "Avatar PIN ID"
// @Param        process     query  string  false  "Process type: preview (640px), thumbnail (128px), empty for original"
// @Success      307         {string}  string  "Redirect to OSS URL"
// @Failure      404         {object}  respond.Response
// @Failure      500         {object}  respond.Response
// @Router       /users/avatar/accelerate/{pinId} [get]
func (h *IndexerQueryHandler) GetFastAvatarContentByPinID(c *gin.Context) {
	pinID := c.Param("pinId")
	if pinID == "" {
		respond.InvalidParam(c, "pinId is required")
		return
	}

	// Get process type from query parameter
	processType := c.DefaultQuery("process", "")

	// Get OSS URL for avatar
	ossURL, contentType, fileName, fileType, err := h.indexerFileService.GetFastAvatarOSSURLByPinID(pinID, processType)
	if err != nil {
		respond.NotFound(c, err.Error())
		return
	}

	// Set response headers
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "inline; filename=\""+fileName+"\"")
	c.Header("X-File-Type", fileType)

	// Redirect to OSS URL
	c.Redirect(307, ossURL)
}

// GetAvatarThumbnailByPinID get avatar thumbnail redirect to OSS by avatar PIN ID
// @Summary      Get avatar thumbnail (redirect to OSS)
// @Description  Redirect to OSS URL for avatar thumbnail (128x128) by avatar PIN ID using OSS built-in thumbnail processing
// @Tags         Indexer User Info
// @Accept       json
// @Produce      json
// @Param        pinId  path  string  true  "Avatar PIN ID"
// @Success      307    {string}  string  "Redirect to OSS URL with thumbnail processing"
// @Failure      404    {object}  respond.Response
// @Failure      500    {object}  respond.Response
// @Router       /thumbnail/{pinId} [get]
func (h *IndexerQueryHandler) GetAvatarThumbnailByPinID(c *gin.Context) {
	pinID := c.Param("pinId")
	if pinID == "" {
		respond.InvalidParam(c, "pinId is required")
		return
	}

	// Get OSS URL for avatar thumbnail (fixed processType as "thumbnail")
	ossURL, contentType, fileName, fileType, err := h.indexerFileService.GetFastAvatarOSSURLByPinID(pinID, "thumbnail")
	if err != nil {
		respond.NotFound(c, err.Error())
		return
	}

	// Set response headers
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "inline; filename=\""+fileName+"\"")
	c.Header("X-File-Type", fileType)

	// Redirect to OSS URL with thumbnail processing
	c.Redirect(307, ossURL)
}

// GetLatestFastFileContentByFirstPinID get latest accelerated file content redirect to OSS by first PIN ID
// @Summary      Get latest accelerated file content (redirect to OSS)
// @Description  Redirect to OSS URL for latest file content by first PIN ID, supports preview/thumbnail/video processing
// @Tags         Indexer File Query
// @Accept       json
// @Produce      json
// @Param        firstPinId  path   string  false  "First PIN ID"
// @Param        process     query  string  false  "Process type: preview (640px for image), thumbnail (235px for image), video (first frame for video), empty for original"
// @Success      307         {string}  string  "Redirect to OSS URL"
// @Failure      404         {object}  respond.Response
// @Failure      500         {object}  respond.Response
// @Router       /files/accelerate/content/latest/{firstPinId} [get]
func (h *IndexerQueryHandler) GetLatestFastFileContentByFirstPinID(c *gin.Context) {
	firstPinID := c.Param("firstPinId")
	if firstPinID == "" {
		respond.InvalidParam(c, "firstPinId is required")
		return
	}

	// Get process type from query parameter
	processType := c.DefaultQuery("process", "")

	// Get OSS URL, ContentType, FileName, and FileType for latest file
	ossURL, contentType, fileName, fileType, err := h.indexerFileService.GetLatestFastFileOSSURLByFirstPinID(firstPinID, processType)
	if err != nil {
		respond.NotFound(c, err.Error())
		return
	}

	// Determine if file should be previewed or downloaded
	// Images, videos, audio can be previewed, others should be downloaded
	shouldPreview := fileType == "image" || fileType == "video" || fileType == "audio" || fileType == "text"

	// Set response headers
	c.Header("Content-Type", contentType)
	if shouldPreview {
		c.Header("Content-Disposition", "inline; filename=\""+fileName+"\"")
	} else {
		c.Header("Content-Disposition", "attachment; filename=\""+fileName+"\"")
	}

	// Redirect to OSS URL (307 Temporary Redirect - preserves original request method)
	c.Redirect(307, ossURL)
}

// GetFastFileContent get accelerated file content redirect to OSS
// @Summary      Get accelerated file content (redirect to OSS)
// @Description  Redirect to OSS URL for file content by PIN ID, supports preview/thumbnail/video processing
// @Tags         Indexer File Query
// @Accept       json
// @Produce      json
// @Param        pinId       path   string  false  "PIN ID"
// @Param        process     query  string  false  "Process type: preview (640px for image), thumbnail (235px for image), video (first frame for video), empty for original"
// @Success      307         {string}  string  "Redirect to OSS URL"
// @Failure      404         {object}  respond.Response
// @Failure      500         {object}  respond.Response
// @Router       /files/accelerate/content/{pinId} [get]
func (h *IndexerQueryHandler) GetFastFileContent(c *gin.Context) {
	pinID := c.Param("pinId")
	if pinID == "" {
		respond.InvalidParam(c, "pinId is required")
		return
	}

	// Get process type from query parameter
	processType := c.DefaultQuery("process", "")

	// Get OSS URL, ContentType, FileName, and FileType
	ossURL, contentType, fileName, fileType, err := h.indexerFileService.GetFastFileOSSURL(pinID, processType)
	if err != nil {
		respond.NotFound(c, err.Error())
		return
	}

	// Determine if file should be previewed or downloaded
	// Images, videos, audio can be previewed, others should be downloaded
	shouldPreview := fileType == "image" || fileType == "video" || fileType == "audio" || fileType == "text"

	// Set response headers
	c.Header("Content-Type", contentType)
	if shouldPreview {
		c.Header("Content-Disposition", "inline; filename=\""+fileName+"\"")
	} else {
		c.Header("Content-Disposition", "attachment; filename=\""+fileName+"\"")
	}

	// Redirect to OSS URL (307 Temporary Redirect - preserves original request method)
	c.Redirect(307, ossURL)
}

// // GetFastAvatarContent get accelerated avatar content redirect to OSS
// // @Summary      Get accelerated avatar content (redirect to OSS)
// // @Description  Redirect to OSS URL for avatar content by PIN ID, supports preview/thumbnail processing
// // @Tags         Indexer Avatar Query
// // @Accept       json
// // @Produce      json
// // @Param        pinId       path   string  false  "PIN ID"
// // @Param        process     query  string  false  "Process type: preview (640px), thumbnail (128x128), empty for original"
// // @Success      307         {string}  string  "Redirect to OSS URL"
// // @Failure      404         {object}  respond.Response
// // @Failure      500         {object}  respond.Response
// // @Router       /avatars/accelerate/content/{pinId} [get]
// func (h *IndexerQueryHandler) GetFastAvatarContent(c *gin.Context) {
// 	pinID := c.Param("pinId")
// 	if pinID == "" {
// 		respond.InvalidParam(c, "pinId is required")
// 		return
// 	}
//
// 	// Get process type from query parameter
// 	processType := c.DefaultQuery("process", "")
//
// 	// Get OSS URL
// 	ossURL, err := h.indexerFileService.GetFastAvatarOSSURL(pinID, processType)
// 	if err != nil {
// 		respond.NotFound(c, err.Error())
// 		return
// 	}
//
// 	// Redirect to OSS URL (307 Temporary Redirect - preserves original request method)
// 	c.Redirect(307, ossURL)
// }

// // GetFastAvatarByMetaID get accelerated avatar redirect to OSS by MetaID
// // @Summary      Get accelerated avatar by MetaID (redirect to OSS)
// // @Description  Redirect to OSS URL for latest avatar by MetaID, supports preview/thumbnail processing
// // @Tags         Indexer Avatar Query
// // @Accept       json
// // @Produce      json
// // @Param        metaId      path   string  false  "MetaID"
// // @Param        process     query  string  false  "Process type: preview (640px), thumbnail (128x128), empty for original"
// // @Success      307         {string}  string  "Redirect to OSS URL"
// // @Failure      404         {object}  respond.Response
// // @Failure      500         {object}  respond.Response
// // @Router       /avatars/accelerate/metaid/{metaId} [get]
// func (h *IndexerQueryHandler) GetFastAvatarByMetaID(c *gin.Context) {
// 	metaID := c.Param("metaId")
// 	if metaID == "" {
// 		respond.InvalidParam(c, "metaId is required")
// 		return
// 	}
//
// 	// Get process type from query parameter
// 	processType := c.DefaultQuery("process", "")
//
// 	// Get OSS URL
// 	ossURL, err := h.indexerFileService.GetFastAvatarOSSURLByMetaID(metaID, processType)
// 	if err != nil {
// 		respond.NotFound(c, err.Error())
// 		return
// 	}
//
// 	// Redirect to OSS URL (307 Temporary Redirect - preserves original request method)
// 	c.Redirect(307, ossURL)
// }

// // GetFastAvatarByAddress get accelerated avatar redirect to OSS by address
// // @Summary      Get accelerated avatar by address (redirect to OSS)
// // @Description  Redirect to OSS URL for latest avatar by address, supports preview/thumbnail processing
// // @Tags         Indexer Avatar Query
// // @Accept       json
// // @Produce      json
// // @Param        address     path   string  false  "Address"
// // @Param        process     query  string  false  "Process type: preview (640px), thumbnail (128x128), empty for original"
// // @Success      307         {string}  string  "Redirect to OSS URL"
// // @Failure      404         {object}  respond.Response
// // @Failure      500         {object}  respond.Response
// // @Router       /avatars/accelerate/address/{address} [get]
// func (h *IndexerQueryHandler) GetFastAvatarByAddress(c *gin.Context) {
// 	address := c.Param("address")
// 	if address == "" {
// 		respond.InvalidParam(c, "address is required")
// 		return
// 	}
//
// 	// Get process type from query parameter
// 	processType := c.DefaultQuery("process", "")
//
// 	// Get OSS URL
// 	ossURL, err := h.indexerFileService.GetFastAvatarOSSURLByAddress(address, processType)
// 	if err != nil {
// 		respond.NotFound(c, err.Error())
// 		return
// 	}
//
// 	// Redirect to OSS URL (307 Temporary Redirect - preserves original request method)
// 	c.Redirect(307, ossURL)
// }

// RescanBlocks trigger asynchronous block rescan
// @Summary      Rescan blocks
// @Description  Trigger asynchronous rescan of blocks within specified height range for a specific chain
// @Tags         Indexer Admin
// @Accept       json
// @Produce      json
// @Param        request  body      respond.RescanRequest  true  "Rescan request parameters"
// @Success      200      {object}  respond.Response{data=respond.RescanResponse}
// @Failure      400      {object}  respond.Response
// @Failure      500      {object}  respond.Response
// @Router       /admin/rescan [post]
func (h *IndexerQueryHandler) RescanBlocks(c *gin.Context) {
	// Check if indexer service is available
	if h.indexerService == nil {
		respond.ServerError(c, "indexer service not available")
		return
	}

	// Parse request body
	var req respond.RescanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.InvalidParam(c, fmt.Sprintf("invalid request parameters: %v", err))
		return
	}

	// Validate height range
	if req.StartHeight <= 0 {
		respond.InvalidParam(c, "start_height must be greater than 0")
		return
	}
	if req.EndHeight < req.StartHeight {
		respond.InvalidParam(c, "end_height must be greater than or equal to start_height")
		return
	}

	// Trigger async rescan
	taskID, err := h.indexerService.RescanBlocksAsync(req.Chain, req.StartHeight, req.EndHeight)
	if err != nil {
		respond.ServerError(c, fmt.Sprintf("failed to start rescan: %v", err))
		return
	}

	// Return response
	response := respond.RescanResponse{
		Message:     "Block rescan task started successfully",
		Chain:       req.Chain,
		StartHeight: req.StartHeight,
		EndHeight:   req.EndHeight,
		TaskID:      taskID,
	}

	respond.Success(c, response)
}

// GetRescanStatus get rescan task status
// @Summary      Get rescan status
// @Description  Get current rescan task status
// @Tags         Indexer Admin
// @Accept       json
// @Produce      json
// @Success      200      {object}  respond.Response{data=respond.RescanStatusResponse}
// @Failure      500      {object}  respond.Response
// @Router       /admin/rescan/status [get]
func (h *IndexerQueryHandler) GetRescanStatus(c *gin.Context) {
	// Check if indexer service is available
	if h.indexerService == nil {
		respond.ServerError(c, "indexer service not available")
		return
	}

	// Get task status
	task := h.indexerService.GetRescanStatus()

	// Build response
	response := respond.RescanStatusResponse{
		TaskID:          task.TaskID,
		Chain:           task.Chain,
		Status:          string(task.Status),
		StartHeight:     task.StartHeight,
		EndHeight:       task.EndHeight,
		CurrentHeight:   task.CurrentHeight,
		ProcessedBlocks: task.ProcessedBlocks,
		TotalBlocks:     task.TotalBlocks,
		ErrorMessage:    task.ErrorMessage,
	}

	// Calculate progress, speed and time estimates
	if task.Status == "running" && task.TotalBlocks > 0 {
		response.Progress = float64(task.ProcessedBlocks) / float64(task.TotalBlocks) * 100
		response.StartTime = task.StartTime.Unix()

		elapsed := time.Since(task.StartTime)
		response.ElapsedTime = elapsed.Milliseconds()

		if task.ProcessedBlocks > 0 {
			response.Speed = float64(task.ProcessedBlocks) / elapsed.Seconds()

			// Estimate time left
			remainingBlocks := task.TotalBlocks - task.ProcessedBlocks
			if response.Speed > 0 {
				estimatedSeconds := float64(remainingBlocks) / response.Speed
				response.EstimatedTimeLeft = int64(estimatedSeconds * 1000) // Convert to milliseconds
			}
		}
	}

	respond.Success(c, response)
}

// StopRescan stop the current rescan task
// @Summary      Stop rescan
// @Description  Stop the current rescan task
// @Tags         Indexer Admin
// @Accept       json
// @Produce      json
// @Success      200      {object}  respond.Response{data=respond.RescanStopResponse}
// @Failure      400      {object}  respond.Response
// @Failure      500      {object}  respond.Response
// @Router       /admin/rescan/stop [post]
func (h *IndexerQueryHandler) StopRescan(c *gin.Context) {
	// Check if indexer service is available
	if h.indexerService == nil {
		respond.ServerError(c, "indexer service not available")
		return
	}

	// Stop the task
	err := h.indexerService.StopRescan()
	if err != nil {
		respond.InvalidParam(c, err.Error())
		return
	}

	// Get updated status
	task := h.indexerService.GetRescanStatus()

	// Build response
	response := respond.RescanStopResponse{
		Message: "Rescan task stopped successfully",
		TaskID:  task.TaskID,
		Status:  string(task.Status),
	}

	respond.Success(c, response)
}
