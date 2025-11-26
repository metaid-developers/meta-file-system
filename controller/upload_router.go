package controller

import (
	"meta-file-system/conf"
	"meta-file-system/controller/handler"
	"meta-file-system/controller/respond"
	uploaderDocs "meta-file-system/docs/uploader"
	"meta-file-system/service/upload_service"
	"meta-file-system/storage"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SetupUploadRouter setup upload service router
func SetupUploadRouter(stor storage.Storage) (*gin.Engine, *upload_service.UploadService) {
	// Set Swagger host from config
	uploaderDocs.SwaggerInfouploader.Host = conf.Cfg.Uploader.SwaggerBaseUrl

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

	// Create upload service instance
	uploadService := upload_service.NewUploadService(stor)

	// Create handler instance
	uploadHandler := handler.NewUploadHandler(uploadService)

	// Static file service (upload page)
	// Map web directory directly to root path for direct access to app.js
	r.StaticFile("/", "./web/index.html")
	r.StaticFile("/app.js", "./web/app.js")
	r.Static("/static", "./web")

	// API v1 route group
	v1 := r.Group("/api/v1")
	{
		// File upload
		v1.POST("/files/pre-upload", uploadHandler.PreUpload)
		v1.POST("/files/commit-upload", uploadHandler.CommitUpload)
		v1.POST("/files/direct-upload", uploadHandler.DirectUpload)                    // One-step upload (recommended)
		v1.POST("/files/estimate-chunked-upload", uploadHandler.EstimateChunkedUpload) // Estimate chunked upload fee
		v1.POST("/files/chunked-upload", uploadHandler.ChunkedUpload)                  // Chunked file upload
		v1.POST("/files/chunked-upload-task", uploadHandler.ChunkedUploadForTask)      // Async chunked file upload (create task)
		v1.GET("/files/task/:taskId", uploadHandler.GetTaskProgress)                   // Get task progress
		v1.GET("/files/tasks", uploadHandler.ListUploadTasks)                          // List tasks by address

		// Configuration
		v1.GET("/config", uploadHandler.GetConfig)
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "uploader",
		})
	})

	// Ignore Chrome DevTools requests
	r.GET("/.well-known/*any", func(c *gin.Context) {
		c.Status(204) // No Content
	})

	// Swagger documentation
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler,
		ginSwagger.InstanceName("uploader")))

	return r, uploadService
}
