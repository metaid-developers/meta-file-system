package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"meta-file-system/conf"
	"meta-file-system/controller"
	"meta-file-system/database"
	"meta-file-system/service/upload_service"
	"meta-file-system/storage"
)

var ENV string

func init() {
	flag.StringVar(&ENV, "env", "loc", "Environment: loc/mainnet/testnet")
}

// @title           Meta File System Uploader API
// @version         1.0
// @description     Meta File System Upload Service API, provides file upload functionality
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:7282
// @BasePath  /api/v1

// @schemes https http

func main() {
	// Initialize all components
	srv, cleanup := initAll()
	defer cleanup()

	// Start server (in goroutine)
	go startServer(srv)

	log.Println("Uploader service started successfully")

	// Wait for shutdown signal
	waitForShutdown()

	log.Println("Shutting down uploader service...")

	// Graceful shutdown
	shutdownServer(srv)

	log.Println("Server exited")
}

// initEnv initialize environment
func initEnv() {
	if ENV == "loc" {
		conf.SystemEnvironmentEnum = conf.LocalEnvironmentEnum
	} else if ENV == "mainnet" {
		conf.SystemEnvironmentEnum = conf.MainnetEnvironmentEnum
	} else if ENV == "testnet" {
		conf.SystemEnvironmentEnum = conf.TestnetEnvironmentEnum
	} else if ENV == "example" {
		conf.SystemEnvironmentEnum = conf.ExampleEnvironmentEnum
	}
	fmt.Printf("Environment: %s\n", ENV)
}

// initAll initialize all components
func initAll() (*http.Server, func()) {
	// Parse command line parameters
	flag.Parse()

	// Set environment
	initEnv()

	// Initialize configuration
	if err := conf.InitConfig(); err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}
	log.Printf("Configuration loaded: env=%s, net=%s, port=%s", ENV, conf.Cfg.Net, conf.Cfg.UploaderPort)

	// Initialize database (Uploader always uses MySQL)
	if err := database.InitUploaderDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize storage
	stor, err := storage.NewStorage()
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	log.Printf("Storage initialized: type=%s", conf.Cfg.Storage.Type)

	// Setup upload service router
	router, uploadService := controller.SetupUploadRouter(stor)

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + conf.Cfg.UploaderPort,
		Handler: router,
	}

	// Start task processor
	taskProcessor := upload_service.NewTaskProcessor(uploadService)
	taskProcessor.Start()
	log.Println("Task processor started")

	// Return server instance and cleanup function
	cleanup := func() {
		database.CloseUploaderDB()
	}

	return srv, cleanup
}

// startServer start HTTP server
func startServer(srv *http.Server) {
	log.Printf("Uploader service starting on port %s...", conf.Cfg.UploaderPort)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// waitForShutdown wait for shutdown signal
func waitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
}

// shutdownServer gracefully shutdown server
func shutdownServer(srv *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}
}
