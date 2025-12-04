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
	"meta-file-system/service/indexer_service"
	"meta-file-system/storage"
)

var ENV string

func init() {
	flag.StringVar(&ENV, "env", "mainnet", "Environment: loc/mainnet/testnet")
}

// @title           Meta File System Indexer API
// @version         1.0
// @description     Meta File System Indexer Service API, provides file query and download functionality
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:7281
// @BasePath  /api/v1

// @schemes https http

func main() {
	// Initialize all components
	indexerService, srv, cleanup := initAll()
	defer cleanup()

	// Start indexer service (in goroutine)
	go indexerService.Start()
	log.Println("Indexer service started successfully")

	// Start HTTP API service (in goroutine)
	go startServer(srv)
	log.Println("Indexer API service started successfully")

	// Wait for shutdown signal
	waitForShutdown()

	log.Println("Shutting down indexer service...")

	// Stop indexer service
	indexerService.Stop()

	// Gracefully shutdown HTTP service
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
func initAll() (*indexer_service.IndexerService, *http.Server, func()) {
	// Parse command line parameters
	flag.Parse()

	// Set environment
	initEnv()

	// Initialize configuration
	if err := conf.InitConfig(); err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}
	log.Printf("Configuration loaded: env=%s, net=%s, port=%s", ENV, conf.Cfg.Net, conf.Cfg.IndexerPort)

	// Initialize database
	if err := initDatabase(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize Redis (optional, won't fail if disabled or unavailable)
	if err := database.InitRedis(); err != nil {
		log.Printf("⚠️  Redis initialization failed (cache will be disabled): %v", err)
	}

	// Initialize storage
	stor, err := storage.NewStorage()
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	log.Printf("Storage initialized: type=%s", conf.Cfg.Storage.Type)

	// 🔧 执行数据修复（执行一次后可以注释掉）
	// 注意：需要先创建 BlockScanner 才能使用修复服务
	// 从多链配置中获取 MVC 和 BTC 配置
	// var mvcConfig, btcConfig *conf.ChainInstanceConfig
	// for i := range conf.Cfg.Indexer.Chains {
	// 	chain := &conf.Cfg.Indexer.Chains[i]
	// 	if chain.Name == "mvc" {
	// 		mvcConfig = chain
	// 	} else if chain.Name == "btc" {
	// 		btcConfig = chain
	// 	}
	// }

	// // 如果没有多链配置，使用单链配置作为 fallback
	// var mvcBlockScanner, btcBlockScanner *indexer.BlockScanner
	// if mvcConfig != nil {
	// 	mvcBlockScanner = indexer.NewBlockScannerWithChain(
	// 		mvcConfig.RpcUrl,
	// 		mvcConfig.RpcUser,
	// 		mvcConfig.RpcPass,
	// 		mvcConfig.StartHeight,
	// 		conf.Cfg.Indexer.ScanInterval,
	// 		indexer.ChainTypeMVC,
	// 	)
	// } else {
	// 	// Fallback: 使用单链配置
	// 	mvcBlockScanner = indexer.NewBlockScannerWithChain(
	// 		conf.Cfg.Chain.RpcUrl,
	// 		conf.Cfg.Chain.RpcUser,
	// 		conf.Cfg.Chain.RpcPass,
	// 		conf.Cfg.Chain.StartHeight,
	// 		conf.Cfg.Indexer.ScanInterval,
	// 		indexer.ChainTypeMVC,
	// 	)
	// }

	// if btcConfig != nil {
	// 	btcBlockScanner = indexer.NewBlockScannerWithChain(
	// 		btcConfig.RpcUrl,
	// 		btcConfig.RpcUser,
	// 		btcConfig.RpcPass,
	// 		btcConfig.StartHeight,
	// 		conf.Cfg.Indexer.ScanInterval,
	// 		indexer.ChainTypeBTC,
	// 	)
	// } else {
	// 	// Fallback: 使用单链配置（假设是 BTC）
	// 	btcBlockScanner = indexer.NewBlockScannerWithChain(
	// 		conf.Cfg.Chain.RpcUrl,
	// 		conf.Cfg.Chain.RpcUser,
	// 		conf.Cfg.Chain.RpcPass,
	// 		conf.Cfg.Chain.StartHeight,
	// 		conf.Cfg.Indexer.ScanInterval,
	// 		indexer.ChainTypeBTC,
	// 	)
	// }

	// // 创建修复服务（需要 MVC 和 BTC 两个扫描器）
	// fixService := indexer_service.NewFixService(mvcBlockScanner, btcBlockScanner)

	// // 修复用户头像信息（执行一次后可以注释掉）
	// // log.Println("🔧 Starting FixUserAvatarInfoCollection...")
	// // if err := fixService.FixUserAvatarInfoCollection(); err != nil {
	// // 	log.Printf("⚠️  FixUserAvatarInfoCollection failed: %v", err)
	// // } else {
	// // 	log.Println("✅ FixUserAvatarInfoCollection completed successfully")
	// // }

	// // 修复用户名称信息（执行一次后可以注释掉）
	// log.Println("[FIX]🔧 Starting FixUserNameInfoCollection...")
	// if err := fixService.FixUserNameInfoCollection(); err != nil {
	// 	log.Printf("[FIX]⚠️  FixUserNameInfoCollection failed: %v", err)
	// } else {
	// 	log.Println("[FIX]✅ FixUserNameInfoCollection completed successfully")
	// }

	// Create indexer service (multi-chain or single-chain)
	var indexerService *indexer_service.IndexerService
	if len(conf.Cfg.Indexer.Chains) > 0 {
		// Multi-chain mode
		log.Printf("Initializing in multi-chain mode with %d chains", len(conf.Cfg.Indexer.Chains))
		indexerService, err = indexer_service.NewMultiChainIndexerService(stor, conf.Cfg.Indexer.Chains)
		if err != nil {
			log.Fatalf("Failed to create multi-chain indexer service: %v", err)
		}
	} else {
		// Single-chain mode (backward compatible)
		log.Println("Initializing in single-chain mode")
		indexerService, err = indexer_service.NewIndexerService(stor)
		if err != nil {
			log.Fatalf("Failed to create indexer service: %v", err)
		}
	}

	// Setup indexer service router (pass indexerService for scanner access)
	router := controller.SetupIndexerRouter(stor, indexerService)

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + conf.Cfg.IndexerPort,
		Handler: router,
	}

	// Return service instance and cleanup function
	cleanup := func() {
		if database.DB != nil {
			database.DB.Close()
		}
		if err := database.CloseRedis(); err != nil {
			log.Printf("Failed to close Redis: %v", err)
		}
	}

	return indexerService, srv, cleanup
}

// initDatabase initialize database based on configuration
func initDatabase() error {
	dbType := database.DBType(conf.Cfg.Database.IndexerType)

	switch dbType {
	case database.DBTypeMySQL:
		config := &database.MySQLConfig{
			DSN:          conf.Cfg.Database.Dsn,
			MaxOpenConns: conf.Cfg.Database.MaxOpenConns,
			MaxIdleConns: conf.Cfg.Database.MaxIdleConns,
		}
		return database.InitDatabase(database.DBTypeMySQL, config)

	case database.DBTypePebble:
		config := &database.PebbleConfig{
			DataDir: conf.Cfg.Database.DataDir,
		}
		return database.InitDatabase(database.DBTypePebble, config)

	default:
		log.Printf("Indexer database type not specified, defaulting to MySQL")
		config := &database.MySQLConfig{
			DSN:          conf.Cfg.Database.Dsn,
			MaxOpenConns: conf.Cfg.Database.MaxOpenConns,
			MaxIdleConns: conf.Cfg.Database.MaxIdleConns,
		}
		return database.InitDatabase(database.DBTypeMySQL, config)
	}
}

// startServer start HTTP server
func startServer(srv *http.Server) {
	log.Printf("Indexer API service starting on port %s...", conf.Cfg.IndexerPort)
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
