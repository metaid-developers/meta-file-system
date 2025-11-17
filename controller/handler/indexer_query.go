package handler

import (
	"strconv"

	"meta-file-system/controller/respond"
	"meta-file-system/service/indexer_service"

	"github.com/gin-gonic/gin"
)

// IndexerQueryHandler indexer query handler
type IndexerQueryHandler struct {
	indexerFileService *indexer_service.IndexerFileService
	syncStatusService  *indexer_service.SyncStatusService
}

// NewIndexerQueryHandler create indexer query handler instance
func NewIndexerQueryHandler(indexerFileService *indexer_service.IndexerFileService, syncStatusService *indexer_service.SyncStatusService) *IndexerQueryHandler {
	return &IndexerQueryHandler{
		indexerFileService: indexerFileService,
		syncStatusService:  syncStatusService,
	}
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

// GetByCreatorAddress get file list by creator address
// @Summary      Get files by creator address
// @Description  Query file list by creator address with cursor pagination
// @Tags         Indexer File Query
// @Accept       json
// @Produce      json
// @Param        address  path   string  true   "Creator address"
// @Param        cursor   query  int     false  "Cursor (last file ID)" default(0)
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
// @Param        cursor   query  int     false  "Cursor (last file ID)" default(0)
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
// @Param        cursor  query  int  false  "Cursor (last file ID)" default(0)
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
// @Description  Get indexer synchronization status (includes latest block height from node)
// @Tags         Indexer Status
// @Accept       json
// @Produce      json
// @Success      200  {object}  respond.Response{data=respond.IndexerSyncStatusResponse}
// @Failure      500  {object}  respond.Response
// @Router       /status [get]
func (h *IndexerQueryHandler) GetSyncStatus(c *gin.Context) {
	status, err := h.syncStatusService.GetSyncStatus()
	if err != nil {
		respond.ServerError(c, err.Error())
		return
	}

	// Get latest block height from node
	latestHeight, err := h.syncStatusService.GetLatestBlockHeight()
	if err != nil {
		// If failed to get from node, use 0 as fallback
		latestHeight = 0
	}

	respond.Success(c, respond.ToIndexerSyncStatusResponse(status, latestHeight))
}

// GetStats get indexer statistics
// @Summary      Get statistics
// @Description  Get indexer statistics (total files count, etc.)
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

	respond.Success(c, respond.ToIndexerStatsResponse(filesCount))
}

// ListAvatars get avatar list with cursor pagination
// @Summary      Query avatar list
// @Description  Query avatar list with cursor pagination
// @Tags         Indexer Avatar Query
// @Accept       json
// @Produce      json
// @Param        cursor  query  int  false  "Cursor (last avatar ID)" default(0)
// @Param        size    query  int  false  "Page size"               default(20)
// @Success      200     {object}  respond.Response{data=respond.IndexerAvatarListResponse}
// @Failure      500     {object}  respond.Response
// @Router       /avatars [get]
func (h *IndexerQueryHandler) ListAvatars(c *gin.Context) {
	// Get cursor and size parameters
	cursorStr := c.DefaultQuery("cursor", "0")
	sizeStr := c.DefaultQuery("size", "20")

	cursor, _ := strconv.ParseInt(cursorStr, 10, 64)
	size, _ := strconv.Atoi(sizeStr)

	// Query avatar list
	avatars, nextCursor, hasMore, err := h.indexerFileService.ListAvatars(cursor, size)
	if err != nil {
		respond.ServerError(c, err.Error())
		return
	}

	respond.Success(c, respond.ToIndexerAvatarListResponse(avatars, nextCursor, hasMore))
}

// GetLatestAvatarByMetaID get latest avatar by MetaID
// @Summary      Get latest avatar by MetaID
// @Description  Query the latest avatar information by MetaID
// @Tags         Indexer Avatar Query
// @Accept       json
// @Produce      json
// @Param        metaId  path  string  true  "MetaID"
// @Success      200     {object}  respond.Response{data=respond.IndexerAvatarResponse}
// @Failure      404     {object}  respond.Response
// @Router       /avatars/metaid/{metaId} [get]
func (h *IndexerQueryHandler) GetLatestAvatarByMetaID(c *gin.Context) {
	metaID := c.Param("metaId")
	if metaID == "" {
		respond.InvalidParam(c, "metaId is required")
		return
	}

	avatar, err := h.indexerFileService.GetLatestAvatarByMetaID(metaID)
	if err != nil {
		respond.NotFound(c, err.Error())
		return
	}

	respond.Success(c, respond.ToIndexerAvatarResponse(avatar))
}

// GetLatestAvatarByAddress get latest avatar by address
// @Summary      Get latest avatar by address
// @Description  Query the latest avatar information by address
// @Tags         Indexer Avatar Query
// @Accept       json
// @Produce      json
// @Param        address  path  string  true  "Address"
// @Success      200      {object}  respond.Response{data=respond.IndexerAvatarResponse}
// @Failure      404      {object}  respond.Response
// @Router       /avatars/address/{address} [get]
func (h *IndexerQueryHandler) GetLatestAvatarByAddress(c *gin.Context) {
	address := c.Param("address")
	if address == "" {
		respond.InvalidParam(c, "address is required")
		return
	}

	avatar, err := h.indexerFileService.GetLatestAvatarByAddress(address)
	if err != nil {
		respond.NotFound(c, err.Error())
		return
	}

	respond.Success(c, respond.ToIndexerAvatarResponse(avatar))
}

// GetAvatarContent get avatar content by PIN ID
// @Summary      Get avatar content
// @Description  Get avatar content by PIN ID
// @Tags         Indexer Avatar Query
// @Accept       json
// @Produce      octet-stream
// @Param        pinId  path      string  true  "PIN ID"
// @Success      200    {file}    binary
// @Failure      404    {object}  respond.Response
// @Router       /avatars/content/{pinId} [get]
func (h *IndexerQueryHandler) GetAvatarContent(c *gin.Context) {
	pinID := c.Param("pinId")
	if pinID == "" {
		respond.InvalidParam(c, "pinId is required")
		return
	}

	content, contentType, fileName, err := h.indexerFileService.GetAvatarContent(pinID)
	if err != nil {
		respond.NotFound(c, err.Error())
		return
	}

	// Set response headers
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "inline; filename=\""+fileName+"\"")
	c.Data(200, contentType, content)
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

	// Get OSS URL
	ossURL, err := h.indexerFileService.GetFastFileOSSURL(pinID, processType)
	if err != nil {
		respond.NotFound(c, err.Error())
		return
	}

	// Redirect to OSS URL (307 Temporary Redirect - preserves original request method)
	c.Redirect(307, ossURL)
}

// GetFastAvatarContent get accelerated avatar content redirect to OSS
// @Summary      Get accelerated avatar content (redirect to OSS)
// @Description  Redirect to OSS URL for avatar content by PIN ID, supports preview/thumbnail processing
// @Tags         Indexer Avatar Query
// @Accept       json
// @Produce      json
// @Param        pinId       path   string  false  "PIN ID"
// @Param        process     query  string  false  "Process type: preview (640px), thumbnail (128x128), empty for original"
// @Success      307         {string}  string  "Redirect to OSS URL"
// @Failure      404         {object}  respond.Response
// @Failure      500         {object}  respond.Response
// @Router       /avatars/accelerate/content/{pinId} [get]
func (h *IndexerQueryHandler) GetFastAvatarContent(c *gin.Context) {
	pinID := c.Param("pinId")
	if pinID == "" {
		respond.InvalidParam(c, "pinId is required")
		return
	}

	// Get process type from query parameter
	processType := c.DefaultQuery("process", "")

	// Get OSS URL
	ossURL, err := h.indexerFileService.GetFastAvatarOSSURL(pinID, processType)
	if err != nil {
		respond.NotFound(c, err.Error())
		return
	}

	// Redirect to OSS URL (307 Temporary Redirect - preserves original request method)
	c.Redirect(307, ossURL)
}

// GetFastAvatarByMetaID get accelerated avatar redirect to OSS by MetaID
// @Summary      Get accelerated avatar by MetaID (redirect to OSS)
// @Description  Redirect to OSS URL for latest avatar by MetaID, supports preview/thumbnail processing
// @Tags         Indexer Avatar Query
// @Accept       json
// @Produce      json
// @Param        metaId      path   string  false  "MetaID"
// @Param        process     query  string  false  "Process type: preview (640px), thumbnail (128x128), empty for original"
// @Success      307         {string}  string  "Redirect to OSS URL"
// @Failure      404         {object}  respond.Response
// @Failure      500         {object}  respond.Response
// @Router       /avatars/accelerate/metaid/{metaId} [get]
func (h *IndexerQueryHandler) GetFastAvatarByMetaID(c *gin.Context) {
	metaID := c.Param("metaId")
	if metaID == "" {
		respond.InvalidParam(c, "metaId is required")
		return
	}

	// Get process type from query parameter
	processType := c.DefaultQuery("process", "")

	// Get OSS URL
	ossURL, err := h.indexerFileService.GetFastAvatarOSSURLByMetaID(metaID, processType)
	if err != nil {
		respond.NotFound(c, err.Error())
		return
	}

	// Redirect to OSS URL (307 Temporary Redirect - preserves original request method)
	c.Redirect(307, ossURL)
}

// GetFastAvatarByAddress get accelerated avatar redirect to OSS by address
// @Summary      Get accelerated avatar by address (redirect to OSS)
// @Description  Redirect to OSS URL for latest avatar by address, supports preview/thumbnail processing
// @Tags         Indexer Avatar Query
// @Accept       json
// @Produce      json
// @Param        address     path   string  false  "Address"
// @Param        process     query  string  false  "Process type: preview (640px), thumbnail (128x128), empty for original"
// @Success      307         {string}  string  "Redirect to OSS URL"
// @Failure      404         {object}  respond.Response
// @Failure      500         {object}  respond.Response
// @Router       /avatars/accelerate/address/{address} [get]
func (h *IndexerQueryHandler) GetFastAvatarByAddress(c *gin.Context) {
	address := c.Param("address")
	if address == "" {
		respond.InvalidParam(c, "address is required")
		return
	}

	// Get process type from query parameter
	processType := c.DefaultQuery("process", "")

	// Get OSS URL
	ossURL, err := h.indexerFileService.GetFastAvatarOSSURLByAddress(address, processType)
	if err != nil {
		respond.NotFound(c, err.Error())
		return
	}

	// Redirect to OSS URL (307 Temporary Redirect - preserves original request method)
	c.Redirect(307, ossURL)
}
