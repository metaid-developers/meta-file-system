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

	// Redis configuration
	Redis RedisConfig
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
	S3    S3StorageConfig
	MinIO MinIOStorageConfig
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

// S3StorageConfig AWS S3 storage configuration
type S3StorageConfig struct {
	Region    string
	AccessKey string
	SecretKey string
	Bucket    string
	Domain    string
	Endpoint  string // Optional custom endpoint
}

// MinIOStorageConfig MinIO storage configuration
type MinIOStorageConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
	Domain    string
}

// ChainInstanceConfig single chain instance configuration
type ChainInstanceConfig struct {
	Name        string `mapstructure:"name"`         // Chain name: btc, mvc, etc.
	RpcUrl      string `mapstructure:"rpc_url"`      // RPC URL
	RpcUser     string `mapstructure:"rpc_user"`     // RPC username
	RpcPass     string `mapstructure:"rpc_pass"`     // RPC password
	StartHeight int64  `mapstructure:"start_height"` // Start height for this chain
	ZmqEnabled  bool   `mapstructure:"zmq_enabled"`  // Enable ZMQ for this chain
	ZmqAddress  string `mapstructure:"zmq_address"`  // ZMQ server address
}

// IndexerConfig indexer configuration
type IndexerConfig struct {
	ScanInterval        int
	BatchSize           int
	StartHeight         int64
	MvcInitBlockHeight  int64  // MVC chain initial block height to start scanning from
	BtcInitBlockHeight  int64  // BTC chain initial block height to start scanning from
	DogeInitBlockHeight int64  // DOGE chain initial block height to start scanning from
	SwaggerBaseUrl      string // Swagger API base URL (e.g., "example.com:7281")
	ZmqEnabled          bool   // Enable ZMQ real-time monitoring
	ZmqAddress          string // ZMQ server address (e.g., "tcp://127.0.0.1:28332")

	// LargeBlockSizeMB: blocks larger than this (MB) are loaded tx-by-tx to avoid OOM. 0 = use default (50)
	LargeBlockSizeMB int

	// Multi-chain support
	Chains              []ChainInstanceConfig // Multi-chain configurations
	TimeOrderingEnabled bool                  // Enable strict time ordering across chains
}

// RedisConfig redis configuration
type RedisConfig struct {
	Enabled  bool   // Enable Redis cache
	Host     string // Redis host
	Port     int    // Redis port
	Password string // Redis password (optional)
	DB       int    // Redis database number
	CacheTTL int    // Cache TTL in seconds (default: 300)
}

// UploaderChainConfig single chain configuration for uploader (RPC + per-chain params)
type UploaderChainConfig struct {
	Name           string `mapstructure:"name"`             // Chain name: mvc, doge, etc.
	RpcUrl         string `mapstructure:"rpc_url"`          // RPC URL
	RpcUser        string `mapstructure:"rpc_user"`         // RPC username
	RpcPass        string `mapstructure:"rpc_pass"`         // RPC password
	MaxFileSize    int64  `mapstructure:"max_file_size"`    // Max file size in MB, 0 = use global default
	ChunkSize      int64  `mapstructure:"chunk_size"`       // Chunk size in MB, 0 = use global default
	ChunkSizeBytes int64  `mapstructure:"chunk_size_bytes"` // Chunk size in bytes (for DOGE etc), 0 = use ChunkSize or chain default
	FeeRate        int64  `mapstructure:"fee_rate"`         // Fee rate: MVC sat/byte, DOGE sat/KB, 0 = use global default
}

// UploaderConfig uploader configuration
type UploaderConfig struct {
	MaxFileSize    int64                  // Global default (MB), used when chain does not specify
	FeeRate        int64                  // Global default
	ChunkSize      int64                  // Global default (MB)
	SwaggerBaseUrl string                 // Swagger API base URL (e.g., "example.com:7282")
	Chains         []UploaderChainConfig  // Per-chain config (RPC + params), RpcConfigMap populated from here
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
			S3: S3StorageConfig{
				Region:    viper.GetString("storage.s3.region"),
				AccessKey: viper.GetString("storage.s3.access_key"),
				SecretKey: viper.GetString("storage.s3.secret_key"),
				Bucket:    viper.GetString("storage.s3.bucket"),
				Domain:    viper.GetString("storage.s3.domain"),
				Endpoint:  viper.GetString("storage.s3.endpoint"),
			},
			MinIO: MinIOStorageConfig{
				Endpoint:  viper.GetString("storage.minio.endpoint"),
				AccessKey: viper.GetString("storage.minio.access_key"),
				SecretKey: viper.GetString("storage.minio.secret_key"),
				Bucket:    viper.GetString("storage.minio.bucket"),
				UseSSL:    viper.GetBool("storage.minio.use_ssl"),
				Domain:    viper.GetString("storage.minio.domain"),
			},
		},

		Indexer: IndexerConfig{
			ScanInterval:        viper.GetInt("indexer.scan_interval"),
			BatchSize:           viper.GetInt("indexer.batch_size"),
			StartHeight:         viper.GetInt64("indexer.start_height"),
			MvcInitBlockHeight:  viper.GetInt64("indexer.mvc_init_block_height"),
			BtcInitBlockHeight:  viper.GetInt64("indexer.btc_init_block_height"),
			DogeInitBlockHeight: viper.GetInt64("indexer.doge_init_block_height"),
			SwaggerBaseUrl:      viper.GetString("indexer.swagger_base_url"),
			ZmqEnabled:          viper.GetBool("indexer.zmq_enabled"),
			ZmqAddress:          viper.GetString("indexer.zmq_address"),
			LargeBlockSizeMB:    viper.GetInt("indexer.large_block_size_mb"),
			TimeOrderingEnabled: viper.GetBool("indexer.time_ordering_enabled"),
		},

		Uploader: UploaderConfig{
			MaxFileSize:    viper.GetInt64("uploader.max_file_size") * 1024 * 1024, // MB to bytes
			FeeRate:        viper.GetInt64("uploader.fee_rate"),
			ChunkSize:      viper.GetInt64("uploader.chunk_size") * 1024 * 1024, // MB to bytes
			SwaggerBaseUrl: viper.GetString("uploader.swagger_base_url"),
			Chains:         nil, // populated below from uploader.chains
		},

		Redis: RedisConfig{
			Enabled:  viper.GetBool("redis.enabled"),
			Host:     viper.GetString("redis.host"),
			Port:     viper.GetInt("redis.port"),
			Password: viper.GetString("redis.password"),
			DB:       viper.GetInt("redis.db"),
			CacheTTL: viper.GetInt("redis.cache_ttl"),
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
	if Cfg.Indexer.LargeBlockSizeMB <= 0 {
		Cfg.Indexer.LargeBlockSizeMB = 50 // 50MB default
	}
	if Cfg.Indexer.SwaggerBaseUrl == "" {
		Cfg.Indexer.SwaggerBaseUrl = "localhost:" + Cfg.IndexerPort
	}
	if Cfg.Uploader.SwaggerBaseUrl == "" {
		Cfg.Uploader.SwaggerBaseUrl = "localhost:" + Cfg.UploaderPort
	}

	// Load multi-chain configurations if present
	if viper.IsSet("indexer.chains") {
		fmt.Println("🔍 Detected indexer.chains in config, loading multi-chain configuration...")

		// Debug: print raw config value
		rawChains := viper.Get("indexer.chains")
		fmt.Printf("📋 Raw chains config type: %T, value: %+v\n", rawChains, rawChains)

		var chains []ChainInstanceConfig
		if err := viper.UnmarshalKey("indexer.chains", &chains); err != nil {
			fmt.Printf("❌ Warning: failed to parse indexer.chains: %v\n", err)

			// Try alternative parsing method
			fmt.Println("🔄 Trying alternative parsing method...")
			chainsInterface := viper.Get("indexer.chains")
			if chainsList, ok := chainsInterface.([]interface{}); ok {
				fmt.Printf("📋 Found %d chains in config\n", len(chainsList))
				for i, chainInterface := range chainsList {
					if chainMap, ok := chainInterface.(map[string]interface{}); ok {
						chain := ChainInstanceConfig{
							Name:        getStringFromMap(chainMap, "name"),
							RpcUrl:      getStringFromMap(chainMap, "rpc_url"),
							RpcUser:     getStringFromMap(chainMap, "rpc_user"),
							RpcPass:     getStringFromMap(chainMap, "rpc_pass"),
							StartHeight: getInt64FromMap(chainMap, "start_height"),
							ZmqEnabled:  getBoolFromMap(chainMap, "zmq_enabled"),
							ZmqAddress:  getStringFromMap(chainMap, "zmq_address"),
						}
						chains = append(chains, chain)
						fmt.Printf("  ✅ Parsed chain %d: %s (RPC: %s)\n", i+1, chain.Name, chain.RpcUrl)
					}
				}
			}
		}

		if len(chains) > 0 {
			Cfg.Indexer.Chains = chains
			fmt.Printf("✅ Loaded %d chain configurations successfully\n", len(chains))
			for i, chain := range chains {
				fmt.Printf("  Chain %d: %s (RPC: %s, Start: %d, ZMQ: %v)\n",
					i+1, chain.Name, chain.RpcUrl, chain.StartHeight, chain.ZmqEnabled)
			}
		} else {
			fmt.Println("⚠️  No chains loaded, falling back to single-chain mode")
		}
	} else {
		fmt.Println("ℹ️  No multi-chain configuration found, using single-chain mode")
	}

	// Initialize RpcConfigMap from uploader.chains (not indexer)
	// Parse uploader.chains (structured config with RPC + per-chain params)
	if viper.IsSet("uploader.chains") {
		fmt.Println("🔍 Loading uploader.chains configuration...")
		var uploaderChains []UploaderChainConfig
		if err := viper.UnmarshalKey("uploader.chains", &uploaderChains); err != nil {
			fmt.Printf("❌ Warning: failed to parse uploader.chains: %v\n", err)
			// Try alternative parsing
			chainsInterface := viper.Get("uploader.chains")
			if chainsList, ok := chainsInterface.([]interface{}); ok {
				for i, ch := range chainsList {
					if m, ok := ch.(map[string]interface{}); ok {
						c := UploaderChainConfig{
							Name:           getStringFromMap(m, "name"),
							RpcUrl:         getStringFromMap(m, "rpc_url"),
							RpcUser:        getStringFromMap(m, "rpc_user"),
							RpcPass:        getStringFromMap(m, "rpc_pass"),
							MaxFileSize:    getInt64FromMap(m, "max_file_size"),
							ChunkSize:      getInt64FromMap(m, "chunk_size"),
							ChunkSizeBytes: getInt64FromMap(m, "chunk_size_bytes"),
							FeeRate:        getInt64FromMap(m, "fee_rate"),
						}
						if c.Name != "" && c.RpcUrl != "" {
							uploaderChains = append(uploaderChains, c)
							fmt.Printf("  ✅ Parsed uploader chain %d: %s (RPC: %s)\n", i+1, c.Name, c.RpcUrl)
						}
					}
				}
			}
		}
		if len(uploaderChains) > 0 {
			Cfg.Uploader.Chains = uploaderChains
			for _, c := range uploaderChains {
				RpcConfigMap[c.Name] = RpcConfig{Url: c.RpcUrl, Username: c.RpcUser, Password: c.RpcPass}
				fmt.Printf("  RpcConfigMap[%s] configured for broadcast (from uploader.chains)\n", c.Name)
			}
			// Legacy: map Cfg.Net (livenet/testnet) to first chain's RPC
			RpcConfigMap[Cfg.Net] = RpcConfig{Url: uploaderChains[0].RpcUrl, Username: uploaderChains[0].RpcUser, Password: uploaderChains[0].RpcPass}
		}
	}
	// Fallback: if uploader.chains empty, use top-level chain for single-chain mode
	if len(Cfg.Uploader.Chains) == 0 {
		fmt.Println("ℹ️  No uploader.chains, using single-chain mode from chain config")
		RpcConfigMap[Cfg.Net] = RpcConfig{
			Url:      Cfg.Chain.RpcUrl,
			Username: Cfg.Chain.RpcUser,
			Password: Cfg.Chain.RpcPass,
		}
		// Add mvc as default chain name when net is livenet/testnet
		chainName := Cfg.Net
		if chainName == "livenet" || chainName == "testnet" {
			chainName = "mvc"
		}
		Cfg.Uploader.Chains = []UploaderChainConfig{{
			Name:           chainName,
			RpcUrl:         Cfg.Chain.RpcUrl,
			RpcUser:        Cfg.Chain.RpcUser,
			RpcPass:        Cfg.Chain.RpcPass,
			MaxFileSize:    0,
			ChunkSize:      0,
			ChunkSizeBytes: 0,
			FeeRate:        0,
		}}
		RpcConfigMap[chainName] = RpcConfig{Url: Cfg.Chain.RpcUrl, Username: Cfg.Chain.RpcUser, Password: Cfg.Chain.RpcPass}
	}
	if len(Cfg.Uploader.Chains) > 0 {
		names := make([]string, 0, len(Cfg.Uploader.Chains))
		for _, c := range Cfg.Uploader.Chains {
			names = append(names, c.Name)
		}
		fmt.Printf("  Uploader supported chains: %v\n", names)
	}

	return nil
}

// Helper functions for parsing chain config from map
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getInt64FromMap(m map[string]interface{}, key string) int64 {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case int:
			return int64(v)
		case int64:
			return v
		case float64:
			return int64(v)
		}
	}
	return 0
}

func getBoolFromMap(m map[string]interface{}, key string) bool {
	if val, ok := m[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

// GetUploaderChainConfig returns the uploader chain config for the given chain name
func GetUploaderChainConfig(chain string) *UploaderChainConfig {
	if chain == "" || Cfg == nil {
		return nil
	}
	for i := range Cfg.Uploader.Chains {
		if Cfg.Uploader.Chains[i].Name == chain {
			return &Cfg.Uploader.Chains[i]
		}
	}
	return nil
}

// GetUploaderChainParam returns maxFileSize (bytes), chunkSize (bytes), feeRate for the given chain
func GetUploaderChainParam(chain string) (maxFileSize, chunkSize, feeRate int64) {
	if Cfg == nil {
		return 0, 0, 0
	}
	defMax := Cfg.Uploader.MaxFileSize
	defChunk := Cfg.Uploader.ChunkSize
	defFee := Cfg.Uploader.FeeRate
	c := GetUploaderChainConfig(chain)
	if c == nil {
		if chain == "doge" {
			return defMax, 1200, defFee // DOGE default chunk size in bytes
		}
		return defMax, defChunk, defFee
	}
	maxFileSize = defMax
	if c.MaxFileSize > 0 {
		maxFileSize = c.MaxFileSize * 1024 * 1024 // MB to bytes
	}
	chunkSize = defChunk
	if c.ChunkSizeBytes > 0 {
		chunkSize = c.ChunkSizeBytes // Use bytes directly (for DOGE etc)
	} else if c.ChunkSize > 0 && chain != "doge" {
		chunkSize = c.ChunkSize * 1024 * 1024 // MB to bytes (DOGE uses 10KB script limit, must use ChunkSizeBytes)
	} else if chain == "doge" {
		chunkSize = 1200 // DOGE default when ChunkSizeBytes not set (inscription script max 10KB)
	}
	feeRate = defFee
	if c.FeeRate > 0 {
		feeRate = c.FeeRate
	}
	return maxFileSize, chunkSize, feeRate
}

// GetUploaderChainNames returns the list of supported chain names
func GetUploaderChainNames() []string {
	if Cfg == nil {
		return nil
	}
	names := make([]string, 0, len(Cfg.Uploader.Chains))
	for _, c := range Cfg.Uploader.Chains {
		names = append(names, c.Name)
	}
	return names
}

// IsChainSupportedForUpload returns true if the chain is configured for upload
func IsChainSupportedForUpload(chain string) bool {
	if chain == "" {
		return false
	}
	c := GetUploaderChainConfig(chain)
	if c == nil {
		return false
	}
	_, ok := RpcConfigMap[chain]
	return ok && c.RpcUrl != ""
}
