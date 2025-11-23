package conf

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config application configuration structure
type Config struct {
	// Network configuration
	Net          string
	Port         string // Default port (backward compatible)
	IndexerPort  string // Indexer service port
	UploaderPort string // Uploader service port

	// Database configuration
	Database DatabaseConfig

	// Blockchain configuration
	Chain ChainConfig

	// Storage configuration
	Storage StorageConfig

	// Indexer configuration
	Indexer IndexerConfig

	// Uploader configuration
	Uploader UploaderConfig
}

// DatabaseConfig database configuration
type DatabaseConfig struct {
	IndexerType  string // Indexer database type: mysql, pebble
	Dsn          string // MySQL DSN
	MaxOpenConns int    // MySQL max open connections
	MaxIdleConns int    // MySQL max idle connections
	DataDir      string // PebbleDB data directory
}

// ChainConfig blockchain configuration
type ChainConfig struct {
	RpcUrl      string
	RpcUser     string
	RpcPass     string
	StartHeight int64
}

// StorageConfig storage configuration
type StorageConfig struct {
	Type  string
	Local LocalStorageConfig
	OSS   OSSStorageConfig
}

// LocalStorageConfig local storage configuration
type LocalStorageConfig struct {
	BasePath string
}

// OSSStorageConfig OSS storage configuration
type OSSStorageConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Domain    string
}

// IndexerConfig indexer configuration
type IndexerConfig struct {
	ScanInterval       int
	BatchSize          int
	StartHeight        int64
	MvcInitBlockHeight int64  // MVC chain initial block height to start scanning from
	BtcInitBlockHeight int64  // BTC chain initial block height to start scanning from
	SwaggerBaseUrl     string // Swagger API base URL (e.g., "example.com:7281")
	ZmqEnabled         bool   // Enable ZMQ real-time monitoring
	ZmqAddress         string // ZMQ server address (e.g., "tcp://127.0.0.1:28332")
}

// UploaderConfig uploader configuration
type UploaderConfig struct {
	MaxFileSize    int64
	FeeRate        int64
	ChunkSize      int64
	SwaggerBaseUrl string // Swagger API base URL (e.g., "example.com:7282")
}

// RpcConfig RPC configuration
type RpcConfig struct {
	Url      string
	Username string
	Password string
}

// RpcConfigMap RPC configuration mapping (for multi-chain support)
var RpcConfigMap = map[string]RpcConfig{}

// Cfg global configuration instance
var Cfg *Config

// InitConfig initialize configuration
func InitConfig() error {
	viper.SetConfigFile(GetYaml())
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("Fatal error config file: %s", err)
	}

	// Create configuration instance
	Cfg = &Config{
		Net:          viper.GetString("net"),
		Port:         viper.GetString("port"), // Retain for backward compatibility
		IndexerPort:  viper.GetString("indexer.port"),
		UploaderPort: viper.GetString("uploader.port"),

		Database: DatabaseConfig{
			IndexerType:  viper.GetString("database.indexer_type"),
			Dsn:          viper.GetString("database.dsn"),
			MaxOpenConns: viper.GetInt("database.max_open_conns"),
			MaxIdleConns: viper.GetInt("database.max_idle_conns"),
			DataDir:      viper.GetString("database.data_dir"),
		},

		Chain: ChainConfig{
			RpcUrl:      viper.GetString("chain.rpc_url"),
			RpcUser:     viper.GetString("chain.rpc_user"),
			RpcPass:     viper.GetString("chain.rpc_pass"),
			StartHeight: viper.GetInt64("chain.start_height"),
		},

		Storage: StorageConfig{
			Type: viper.GetString("storage.type"),
			Local: LocalStorageConfig{
				BasePath: viper.GetString("storage.local.base_path"),
			},
			OSS: OSSStorageConfig{
				Endpoint:  viper.GetString("storage.oss.endpoint"),
				AccessKey: viper.GetString("storage.oss.access_key"),
				SecretKey: viper.GetString("storage.oss.secret_key"),
				Bucket:    viper.GetString("storage.oss.bucket"),
				Domain:    viper.GetString("storage.oss.domain"),
			},
		},

		Indexer: IndexerConfig{
			ScanInterval:       viper.GetInt("indexer.scan_interval"),
			BatchSize:          viper.GetInt("indexer.batch_size"),
			StartHeight:        viper.GetInt64("indexer.start_height"),
			MvcInitBlockHeight: viper.GetInt64("indexer.mvc_init_block_height"),
			BtcInitBlockHeight: viper.GetInt64("indexer.btc_init_block_height"),
			SwaggerBaseUrl:     viper.GetString("indexer.swagger_base_url"),
			ZmqEnabled:         viper.GetBool("indexer.zmq_enabled"),
			ZmqAddress:         viper.GetString("indexer.zmq_address"),
		},

		Uploader: UploaderConfig{
			MaxFileSize:    viper.GetInt64("uploader.max_file_size") * 1024 * 1024, // MB to bytes
			FeeRate:        viper.GetInt64("uploader.fee_rate"),
			ChunkSize:      viper.GetInt64("uploader.chunk_size") * 1024 * 1024, // MB to bytes
			SwaggerBaseUrl: viper.GetString("uploader.swagger_base_url"),
		},
	}

	// Set default values
	if Cfg.IndexerPort == "" {
		Cfg.IndexerPort = "7281"
	}
	if Cfg.UploaderPort == "" {
		Cfg.UploaderPort = "7282"
	}
	// Retain Port for backward compatibility
	if Cfg.Port == "" {
		Cfg.Port = Cfg.IndexerPort
	}
	if Cfg.Storage.Type == "" {
		Cfg.Storage.Type = "local"
	}
	if Cfg.Storage.Local.BasePath == "" {
		Cfg.Storage.Local.BasePath = "./data/files"
	}
	if Cfg.Indexer.ScanInterval == 0 {
		Cfg.Indexer.ScanInterval = 10
	}
	if Cfg.Indexer.BatchSize == 0 {
		Cfg.Indexer.BatchSize = 100
	}
	if Cfg.Uploader.MaxFileSize == 0 {
		Cfg.Uploader.MaxFileSize = 10485760
	}
	if Cfg.Uploader.FeeRate == 0 {
		Cfg.Uploader.FeeRate = 1
	}
	if Cfg.Database.MaxOpenConns == 0 {
		Cfg.Database.MaxOpenConns = 100
	}
	if Cfg.Database.MaxIdleConns == 0 {
		Cfg.Database.MaxIdleConns = 10
	}
	if Cfg.Indexer.SwaggerBaseUrl == "" {
		Cfg.Indexer.SwaggerBaseUrl = "localhost:" + Cfg.IndexerPort
	}
	if Cfg.Uploader.SwaggerBaseUrl == "" {
		Cfg.Uploader.SwaggerBaseUrl = "localhost:" + Cfg.UploaderPort
	}

	// Initialize RpcConfigMap (use currently configured chain)
	RpcConfigMap[Cfg.Net] = RpcConfig{
		Url:      Cfg.Chain.RpcUrl,
		Username: Cfg.Chain.RpcUser,
		Password: Cfg.Chain.RpcPass,
	}

	return nil
}
