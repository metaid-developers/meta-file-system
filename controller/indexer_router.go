package controller

import (
	"meta-file-system/conf"
	"meta-file-system/controller/handler"
	"meta-file-system/controller/respond"
	indexerDocs "meta-file-system/docs/indexer"
	"meta-file-system/service/indexer_service"
	"meta-file-system/storage"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SetupIndexerRouter setup indexer service router
func SetupIndexerRouter(stor storage.Storage, indexerService *indexer_service.IndexerService) *gin.Engine {
	// Set Swagger host from config
	indexerDocs.SwaggerInfoindexer.Host = conf.Cfg.Indexer.SwaggerBaseUrl

	// Create Gin engine
	r := gin.Default()

	// Add CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Allow all origins, can be configured to specific domains
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization", "Accept", "Cache-Control", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * 3600, // 12 hours
	}))

	// Add timing middleware
	r.Use(respond.TimingMiddleware())

	// Create indexer file service instance
	indexerFileService := indexer_service.NewIndexerFileService(stor)

	// Create sync status service instance
	syncStatusService := indexer_service.NewSyncStatusService()
	// Set scanner or coordinator for getting latest block height
	if indexerService != nil {
		if indexerService.IsMultiChain() {
			// Multi-chain mode: set coordinator
			syncStatusService.SetMultiChainCoordinator(indexerService.GetCoordinator())
		} else {
			// Single-chain mode: set scanner
			syncStatusService.SetBlockScanner(indexerService.GetScanner())
		}
	}

	// Create handler
	indexerQueryHandler := handler.NewIndexerQueryHandler(indexerFileService, syncStatusService)
	// Set indexer service for admin operations (like rescan)
	if indexerService != nil {
		indexerQueryHandler.SetIndexerService(indexerService)
	}

	// API v1 route group
	v1 := r.Group("/api/v1")
	{
		// Indexer file query routes (using cursor pagination)
		files := v1.Group("/files")
		{
			// Get file list (cursor pagination)
			files.GET("", indexerQueryHandler.ListFiles)

			// Get file by PIN ID
			files.GET("/:pinId", indexerQueryHandler.GetByPinID)

			// Get file content by PIN ID
			files.GET("/content/:pinId", indexerQueryHandler.GetFileContent)

			// Get accelerated file content redirect to OSS
			files.GET("/accelerate/content/:pinId", indexerQueryHandler.GetFastFileContent)

			// Get latest file by first PIN ID
			files.GET("/latest/:firstPinId", indexerQueryHandler.GetLatestByFirstPinID)

			// Get latest file content by first PIN ID
			files.GET("/content/latest/:firstPinId", indexerQueryHandler.GetLatestFileContentByFirstPinID)

			// Get latest accelerated file content redirect to OSS by first PIN ID
			files.GET("/accelerate/content/latest/:firstPinId", indexerQueryHandler.GetLatestFastFileContentByFirstPinID)

			// Get files by creator address
			files.GET("/creator/:address", indexerQueryHandler.GetByCreatorAddress)

			// Get files by creator MetaID
			files.GET("/metaid/:metaidOrGlobalMetaId", indexerQueryHandler.GetByCreatorMetaID)
			// Get files by file extension (global), reverse time order; extension as query (array supported)
			files.GET("/extension", indexerQueryHandler.GetFilesByExtension)
			// Get files by globalMetaID and file extension; extension as query (array supported)
			files.GET("/metaid/:metaidOrGlobalMetaId/extension", indexerQueryHandler.GetFilesByGlobalMetaIDAndExtension)
		}

		// Indexer user info query routes
		users := v1.Group("/users")
		{
			// Get user info list (cursor pagination)
			users.GET("", indexerQueryHandler.ListUserInfo)

			// Get user info by MetaID
			users.GET("/metaid/:metaId", indexerQueryHandler.GetUserInfoByMetaID)

			// Get user info by address
			users.GET("/address/:address", indexerQueryHandler.GetUserInfoByAddress)

			// Get avatar content by MetaID (latest version)
			users.GET("/metaid/:metaId/avatar", indexerQueryHandler.GetAvatarContentByMetaID)

			// Get avatar content by avatar PIN ID (specific version)
			users.GET("/avatar/content/:pinId", indexerQueryHandler.GetAvatarContentByPinID)

			// Get accelerated avatar content redirect to OSS by avatar PIN ID
			users.GET("/avatar/accelerate/:pinId", indexerQueryHandler.GetFastAvatarContentByPinID)

			// Get user info history by MetaID or Address
			users.GET("/history/:key", indexerQueryHandler.GetUserInfoHistory)
		}

		// Indexer PIN info query routes
		pins := v1.Group("/pins")
		{
			// Get PIN info by PIN ID from collectionPinInfo
			pins.GET("/:pinId", indexerQueryHandler.GetPinInfoByPinID)
		}

		// Sync status route
		v1.GET("/status", indexerQueryHandler.GetSyncStatus)

		// Statistics route
		v1.GET("/stats", indexerQueryHandler.GetStats)

		// Info routes (MetaID format, same as /api/info for Swagger basePath /api/v1)
		infoV1 := v1.Group("/info")
		{
			infoV1.GET("/metaid/:metaidOrGlobalMetaId", indexerQueryHandler.GetMetaIDUserInfoByMetaID)
			infoV1.GET("/address/:address", indexerQueryHandler.GetMetaIDUserInfoByAddress)
			infoV1.GET("/globalmetaid/:globalMetaID", indexerQueryHandler.GetMetaIDUserInfoByGlobalMetaID)
			infoV1.GET("/search", indexerQueryHandler.SearchMetaIDUserInfo)
		}

		// Thumbnail (avatar) - Swagger documents /api/v1/thumbnail/{pinId}
		v1.GET("/thumbnail/:pinId", indexerQueryHandler.GetAvatarThumbnailByPinID)

		// Admin routes
		admin := v1.Group("/admin")
		{
			// Rescan blocks
			admin.POST("/rescan", indexerQueryHandler.RescanBlocks)

			// Get rescan status
			admin.GET("/rescan/status", indexerQueryHandler.GetRescanStatus)

			// Stop rescan
			admin.POST("/rescan/stop", indexerQueryHandler.StopRescan)
		}
	}

	api := r.Group("/api")
	{
		// MetaID compatible info routes
		info := api.Group("/info")
		{
			// Get user info by MetaID (MetaID format)
			info.GET("/metaid/:metaidOrGlobalMetaId", indexerQueryHandler.GetMetaIDUserInfoByMetaID)

			// Get user info by address (MetaID format)
			info.GET("/address/:address", indexerQueryHandler.GetMetaIDUserInfoByAddress)

			// Get user info by Global MetaID (MetaID format)
			info.GET("/globalmetaid/:globalMetaID", indexerQueryHandler.GetMetaIDUserInfoByGlobalMetaID)

			// Search user info (MetaID format)
			info.GET("/search", indexerQueryHandler.SearchMetaIDUserInfo)
		}
	}

	// Avatar (legacy root paths, kept for backward compatibility)
	r.GET("/content/:pinId", indexerQueryHandler.GetAvatarContentByPinID)
	r.GET("/thumbnail/:pinId", indexerQueryHandler.GetAvatarThumbnailByPinID)

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "indexer",
		})
	})

	// Swagger documentation
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler,
		ginSwagger.InstanceName("indexer")))

	// Static files and web pages
	r.Static("/static", "./web/static")
	r.StaticFile("/", "./web/indexer.html")
	r.StaticFile("/indexer.html", "./web/indexer.html")
	r.StaticFile("/indexer.js", "./web/indexer.js")

	return r
}
