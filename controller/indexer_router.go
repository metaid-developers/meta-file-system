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
	// Set scanner for getting latest block height
	if indexerService != nil {
		syncStatusService.SetBlockScanner(indexerService.GetScanner())
	}

	// Create handler
	indexerQueryHandler := handler.NewIndexerQueryHandler(indexerFileService, syncStatusService)

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

			// Get files by creator address
			files.GET("/creator/:address", indexerQueryHandler.GetByCreatorAddress)

			// Get files by creator MetaID
			files.GET("/metaid/:metaId", indexerQueryHandler.GetByCreatorMetaID)
		}

		// Indexer avatar query routes
		avatars := v1.Group("/avatars")
		{
			// Get avatar list (cursor pagination)
			avatars.GET("", indexerQueryHandler.ListAvatars)

			// Get avatar content by PIN ID
			avatars.GET("/content/:pinId", indexerQueryHandler.GetAvatarContent)

			// Get accelerated avatar content redirect to OSS by PIN ID
			avatars.GET("/accelerate/content/:pinId", indexerQueryHandler.GetFastAvatarContent)

			// Get latest avatar by MetaID
			avatars.GET("/metaid/:metaId", indexerQueryHandler.GetLatestAvatarByMetaID)

			// Get accelerated avatar redirect to OSS by MetaID
			avatars.GET("/accelerate/metaid/:metaId", indexerQueryHandler.GetFastAvatarByMetaID)

			// Get latest avatar by address
			avatars.GET("/address/:address", indexerQueryHandler.GetLatestAvatarByAddress)

			// Get accelerated avatar redirect to OSS by address
			avatars.GET("/accelerate/address/:address", indexerQueryHandler.GetFastAvatarByAddress)
		}

		// Sync status route
		v1.GET("/status", indexerQueryHandler.GetSyncStatus)

		// Statistics route
		v1.GET("/stats", indexerQueryHandler.GetStats)
	}

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
