# Meta File System

On-chain file service based on MetaID protocol, supporting file upload and indexing capabilities.

[中文版 / Chinese Version](README-ZH.md)

## Features

- 📤 **File Upload**: Upload files to blockchain via MetaID protocol
- 📥 **File Indexing**: Scan and index MetaID files from blockchain
- 🔗 **Multi-Chain Coordination**: Support BTC and MVC dual-chain indexing with timestamp-ordered processing
- ⚡ **ZMQ Real-time Monitoring**: Support mempool transaction listening for fast response to on-chain events
- 👥 **User Info Indexing**: Index network-wide user information (avatar, name, etc.) with Redis caching
- 🔄 **Full Operation Support**: Support complete lifecycle of create/modify/revoke operations
- 🌐 **Web Interface**: Provide visual file upload page with Metalet wallet integration
- 🚀 **OSS Accelerated Links**: Indexer exposes image/video/avatar accelerated access with preview parameters
- ☁️ **Multiple Storage Backends**: Support local storage, Alibaba Cloud OSS, AWS S3, MinIO

## Quick Start

### Prerequisites

- Go 1.23+
- MySQL 5.7+
- MVC Node (for indexing service)

### Install Dependencies

```bash
make deps
# or
go mod tidy
```

### Configuration

Copy and modify the configuration file:

```bash
cp conf/conf_example.yaml conf/conf_loc.yaml
```

Edit `conf/conf_loc.yaml` to configure database, blockchain node, storage, etc.

### Initialize Database

```bash
mysql -u root -p < scripts/init.sql
```

Or use Make command:

```bash
make init-db
```

### Build

```bash
# Build all services
make build

# Or use script
chmod +x scripts/build.sh
./scripts/build.sh
```

### Run

#### Run Indexer Service

The indexer service includes two functions:
1. Background indexing of blockchain data
2. Provide query and download API (port 7281)

```bash
# Use compiled binary
./bin/indexer --config=conf/conf_loc.yaml

# Or run directly
make run-indexer
```

#### Run Uploader Service

The uploader service provides file upload API (port 7282)

```bash
# Use compiled binary
./bin/uploader --config=conf/conf_loc.yaml

# Or run directly
make run-uploader
```

#### Run Both Services

```bash
# Terminal 1 - Indexer service
./bin/indexer --config=conf/conf_loc.yaml

# Terminal 2 - Uploader service
./bin/uploader --config=conf/conf_loc.yaml
```

### Web Upload Interface

After starting the Uploader service, you can access the visual upload page through browser:

```bash
# Access upload page
open http://localhost:7282
```

**Web Interface Preview:**

![MetaID File Upload Interface](static/image.png)

**Features**:
- 🔗 Connect to Metalet wallet
- 📁 Drag and drop file upload
- ⚙️ Configure on-chain parameters
- ✍️ Automatically call wallet for signing
- 📡 One-click upload to blockchain

## 📚 Documentation

- **[📤 Upload Flow Guide (English)](./UPLOAD_FLOW.md)** - Complete guide for uploading files to blockchain with detailed steps and flow diagrams, combined with wallet operations

### Docker Deployment

Docker Compose is recommended for quick deployment.

**Prerequisites**: Need to prepare MySQL database first (standalone deployment or use cloud database)

#### Full Deployment (Indexer + Uploader)

```bash
# Method 1: Use Makefile
make docker-up

# Method 2: Use docker-compose
cd deploy
docker-compose up -d
```

**Configure Database Connection**:

Edit `conf/conf_pro.yaml` to configure database DSN:

```yaml
rds:
  # Use Docker MySQL container
  dsn: "user:pass@tcp(mysql:3306)/metaid_file_system_db?charset=utf8mb4"

```

#### Deploy Uploader Only

```bash
# Use Makefile
make docker-up-uploader

# Use docker-compose
cd deploy
docker-compose -f docker-compose.uploader.yml up -d

# Use deployment script
cd deploy
./deploy.sh up uploader
```

#### Deploy Indexer Only

```bash
# Use Makefile
make docker-up-indexer

# Use docker-compose
cd deploy
docker-compose -f docker-compose.indexer.yml up -d

# Use deployment script
cd deploy
./deploy.sh up indexer
```

**View Logs**:
```bash
make docker-logs
# or
cd deploy && ./deploy.sh logs all
```

Detailed instructions: [Docker Deployment Documentation](deploy/README.md) | [Quick Start](deploy/QUICKSTART.md)

## API Documentation

### API Module Division

Two services provide different API endpoints:

| Service | Port | API Functions | Swagger Docs |
|---------|------|---------------|--------------|
| **Uploader** | 7282 | File upload, config query | http://localhost:7282/swagger/index.html |
| **Indexer** | 7281 | File query, download, accelerated links | http://localhost:7281/swagger/index.html |

### 📚 Swagger API Documentation

#### Uploader API Documentation (v1.0)

The Uploader service provides complete Swagger interactive API documentation.

**Access URL:**
```
http://localhost:7282/swagger/index.html
```

**API Endpoint List:**

1. **File Upload**
   - `POST /api/v1/files/pre-upload` - Pre-upload file, generate unsigned transaction
   - `POST /api/v1/files/commit-upload` - Submit signed transaction, broadcast to chain

2. **Config Query**
   - `GET /api/v1/config` - Get service configuration (e.g., max file size)

3. **Direct Upload**
   - `POST /api/v1/files/direct-upload` - Skip pre-upload and submit a signed transaction directly (DirectUpload flow)

**Response Structure:**

All APIs return a unified response format:
```json
{
  "code": 0,           // Response code: 0=success, 40000=param error, 40400=not found, 50000=server error
  "message": "success", // Response message
  "processingTime": 123, // Request processing time (milliseconds)
  "data": {}           // Response data (varies by endpoint)
}
```

#### Indexer API Documentation (v1.0)

The Indexer service now provides full query plus OSS acceleration capabilities with Swagger ready to use.

### Web Indexer Interface

After starting the Indexer service, you can access the visual indexer page through browser:

```bash
# Access indexer page
open http://localhost:7281
```

**Web Interface Preview:**

![MetaID File Indexer Interface](static/image-indexer.png)

**Access URL:**
```
http://localhost:7281/swagger/index.html
```

**Core Endpoints:**

1. **File Query**
   - `GET /api/v1/files`: Cursor-based list
   - `GET /api/v1/files/{pinId}`: Fetch file metadata by PinID
   - `GET /api/v1/files/content/{pinId}`: Return binary content from storage
   - `GET /api/v1/files/accelerate/content/{pinId}`: Return OSS link with optional processing

2. **Creator Lookup**
   - `GET /api/v1/files/creator/{address}`: Query files by address
   - `GET /api/v1/files/metaid/{metaId}`: Query files by MetaID

3. **User Info Query**
   - `GET /api/v1/users/info/metaid/{metaId}`: Get user info (name, avatar, etc.)
   - `GET /api/v1/users/info/address/{address}`: Get user info by address
   - Supports Redis caching for fast response

4. **Avatar Query**
   - `GET /api/v1/avatars`: Avatar pagination
   - `GET /api/v1/avatars/content/{pinId}`: Binary avatar
   - `GET /api/v1/avatars/accelerate/content/{pinId}`: Avatar OSS link
   - `GET /api/v1/avatars/accelerate/metaid/{metaId}`: Latest avatar by MetaID (OSS link)
   - `GET /api/v1/avatars/accelerate/address/{address}`: Latest avatar by address (OSS link)

5. **Sync & Stats**
   - `GET /api/v1/status`: Multi-chain sync status (supports MVC/BTC)
   - `GET /api/v1/stats`: Indexing statistics

**Accelerate Parameters**

`/accelerate` routes accept a `process` query parameter, e.g. `/api/v1/files/accelerate/content/{pinId}?process=preview`

| process | Type  | Description |
|---------|-------|-------------|
| `preview` | image | Resize width to 640px (keep aspect) |
| `thumbnail` | image | Files: width 235px; Avatars: 128x128 fill |
| `video` | video | Return snapshot at 1 second |
| *(empty)* | all | Return original OSS resource |

> Tip: Acceleration requires `storage.type=oss` and `storage.oss.domain` configured with the public CDN/custom domain.

### Pre-upload File (Uploader Service)

Step 1: Pre-upload, build unsigned transaction

```bash
POST http://localhost:7282/api/v1/files/pre-upload
Content-Type: multipart/form-data

Parameters:
- file: File content (binary)
- path: MetaID path
- metaId: MetaID (optional)
- address: Address (optional)
- operation: Operation type (create/modify/revoke, default: create)
- contentType: Content type (optional)
- changeAddress: Change address (optional)
- feeRate: Fee rate (optional, default: 1)
- outputs: Output list JSON (optional)
- otherOutputs: Other output list JSON (optional)

Response:
{
  "code": 0,
  "message": "success",
  "processingTime": 123,
  "data": {
    "fileId": "metaid_abc123...",        // File ID (unique identifier)
    "fileMd5": "5d41402abc4b2a76...",     // File MD5
    "fileHash": "2c26b46b68ffc68f...",    // File SHA256 hash
    "txId": "abc123...",                   // Transaction ID
    "pinId": "abc123...i0",                // Pin ID
    "preTxRaw": "0100000...",              // Pre-transaction raw data (hex, to be signed)
    "status": "pending",                   // Status: pending/success/failed
    "message": "success",                  // Message
    "calTxFee": 1000,                      // Calculated transaction fee (satoshi)
    "calTxSize": 500                       // Calculated transaction size (bytes)
  }
}
```

### Commit Upload (Uploader Service)

Step 2: Submit signed transaction

```bash
POST http://localhost:7282/api/v1/files/commit-upload
Content-Type: application/json

Request:
{
  "fileId": "metaid_abc123...",           // File ID (from pre-upload response)
  "signedRawTx": "0100000..."             // Signed raw transaction data (hex)
}

Response:
{
  "code": 0,
  "message": "success",
  "processingTime": 456,
  "data": {
    "fileId": "metaid_abc123...",         // File ID
    "status": "success",                   // Status: success/failed
    "txId": "abc123...",                   // Transaction ID
    "pinId": "abc123...i0",                // Pin ID
    "message": "success"                   // Message
  }
}
```


## Configuration

### Database Configuration

```yaml
rds:
  dsn: "user:password@tcp(host:3306)/database?charset=utf8mb4&parseTime=True"
  max_open_conns: 1000
  max_idle_conns: 50
```

### Redis Configuration (Optional)

For caching user information (avatar, name, etc.) to improve query performance:

```yaml
redis:
  enabled: true  # Enable Redis cache
  host: "localhost"
  port: 6379
  password: ""
  db: 1
  cache_ttl: 1800  # Cache expiration time (seconds, default 30 minutes)
```

### Storage Configuration

#### Local Storage

```yaml
storage:
  type: "local"
  local:
    base_path: "./data/files"
```

#### Alibaba Cloud OSS

```yaml
storage:
  type: "oss"
  oss:
    endpoint: "oss-cn-hangzhou.aliyuncs.com"
    access_key: "your-access-key"
    secret_key: "your-secret-key"
    bucket: "your-bucket"
    domain: "https://cdn.your-domain.com" # Public domain for accelerate links
```

#### AWS S3

```yaml
storage:
  type: "s3"
  s3:
    region: "us-east-1"
    endpoint: ""  # Optional: custom endpoint (leave empty for AWS S3)
    access_key: "your-access-key"
    secret_key: "your-secret-key"
    bucket: "your-bucket"
    domain: "https://cdn.your-domain.com" # Public domain for accelerate links
```

#### MinIO

```yaml
storage:
  type: "minio"
  minio:
    endpoint: "http://localhost:9000"
    access_key: "minioadmin"
    secret_key: "minioadmin"
    bucket: "meta-file-system"
    use_ssl: false
    domain: "https://minio.your-domain.com" # Public domain for accelerate links
```

### Indexer Configuration

#### Single-Chain Mode (Compatible with old version)

```yaml
indexer:
  port: "7281"
  scan_interval: 10  # Scan interval (seconds)
  batch_size: 100    # Batch processing size
  start_height: 0    # Start height (0 = start from max height in database)
  zmq_enabled: true  # Enable ZMQ real-time monitoring
  zmq_address: "tcp://127.0.0.1:28332"  # ZMQ server address

# Single-chain blockchain configuration
chain:
  rpc_url: "http://127.0.0.1:9882"
  rpc_user: "rpcuser"
  rpc_pass: "rpcpassword"
```

#### Multi-Chain Coordination Mode (Recommended)

```yaml
indexer:
  port: "7281"
  scan_interval: 10
  time_ordering_enabled: true  # Enable cross-chain timestamp ordering
  mvc_init_block_height: 350000  # MVC initial block height
  btc_init_block_height: 800000  # BTC initial block height
  
  # Multi-chain configuration (auto-enables multi-chain mode)
  chains:
    - name: "mvc"
      rpc_url: "http://127.0.0.1:9882"
      rpc_user: "rpcuser"
      rpc_pass: "rpcpassword"
      start_height: 350000
      zmq_enabled: true  # MVC chain ZMQ monitoring
      zmq_address: "tcp://127.0.0.1:28332"
    
    - name: "btc"
      rpc_url: "http://127.0.0.1:8332"
      rpc_user: "btcuser"
      rpc_pass: "btcpass"
      start_height: 800000
      zmq_enabled: true  # BTC chain ZMQ monitoring
      zmq_address: "tcp://127.0.0.1:28333"
```

**Multi-Chain Mode Features:**
- ✅ Index BTC and MVC chains simultaneously
- ✅ Process cross-chain transactions in timestamp order (optional)
- ✅ Independent ZMQ real-time monitoring for each chain
- ✅ Automatic sync status management and resume capability
- ✅ Prevent single-chain blocking with smart queue scheduling

### Uploader Configuration

```yaml
uploader:
  enabled: true
  max_file_size: 10  # Max file size (10MB)
  fee_rate: 1              # Default fee rate
```

## Development

### Run Tests

```bash
make test
```

### Clean Build Artifacts

```bash
make clean
```

## License

MIT License

## Version Information

**Current Version: v0.3.0**

### Changelog

#### v0.3.0 (2025-12-05)

**Indexer Service - Major Update**
- 🎉 **Multi-Chain Coordination**: Support BTC and MVC dual-chain indexing with timestamp-ordered processing
- ⚡ **ZMQ Real-time Monitoring**: Support mempool transaction listening, auto-scan mempool before starting monitoring
- 👥 **User Info Indexing**: Index network-wide user information (avatar, name, bio, etc.)
- 🔄 **Modify Operation Support**: Full support for file create/modify/revoke lifecycle
- ☁️ **New Storage Backends**: Support AWS S3 and MinIO (S3-compatible)
- 💾 **Redis Caching**: User info Redis cache to improve query performance
- 📊 **Multi-Chain Status**: Independent tracking of sync status for each chain
- 🛡️ **Smart Queue Scheduling**: Prevent single-chain blocking, optimize memory usage

**Configuration Changes**
- Added `indexer.chains[]` for multi-chain configuration
- Added `indexer.time_ordering_enabled` for timestamp ordering
- Added `storage.s3` and `storage.minio` configurations
- Added `redis` cache configuration

#### v0.2.0 (2025-11-17)

**Indexer Service**
- ✅ Added OSS accelerate routes (`/accelerate`) with image preview, thumbnail, video snapshot
- ✅ Avatar accelerate endpoints for MetaID / address
- ✅ Swagger available at `http://localhost:7281/swagger/index.html`

**Uploader Service**
- ✅ Added DirectUpload flow (submit signed tx directly)
- ✅ Swagger exposes `POST /api/v1/files/direct-upload`

#### v0.1.0 (2025-10-16)

**Uploader Service**
- ✅ Complete file upload functionality (pre-upload + commit upload)
- ✅ Comprehensive Swagger API documentation
- ✅ Web visual upload interface (Metalet wallet integration)
