-- Meta File System - Uploader Service Database Initialization Script

-- =============================================
-- File table (tb_file)
-- =============================================
CREATE TABLE IF NOT EXISTS `tb_file` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT 'Primary key ID',
    
    -- File identifiers
    `file_id` VARCHAR(150) DEFAULT NULL COMMENT 'File unique ID (metaid_fileHash)',
    `file_name` VARCHAR(255) DEFAULT NULL COMMENT 'File name',
    
    -- File information
    `file_hash` VARCHAR(80) DEFAULT NULL COMMENT 'File hash (SHA256)',
    `file_size` BIGINT DEFAULT NULL COMMENT 'File size (bytes))',
    `file_type` VARCHAR(20) DEFAULT NULL COMMENT 'File type (image/video/audio/document/other)',
    `file_md5` VARCHAR(191) DEFAULT NULL COMMENT 'File MD5',
    `file_content_type` VARCHAR(100) DEFAULT NULL COMMENT 'File content type (MIME Type)',
    `chunk_type` VARCHAR(20) DEFAULT NULL COMMENT 'Chunk type (single/multi)',
    
    -- Content
    `content_hex` TEXT COMMENT 'Content hexadecimal',
    
    -- MetaID information
    `meta_id` VARCHAR(100) DEFAULT NULL COMMENT 'MetaID',
    `address` VARCHAR(100) DEFAULT NULL COMMENT 'Address',
    
    -- Transaction information
    `tx_id` VARCHAR(64) DEFAULT NULL COMMENT 'On-chain transaction ID',
    `pin_id` VARCHAR(80) NOT NULL COMMENT 'Pin ID',
    `path` VARCHAR(191) NOT NULL COMMENT 'MetaID path',
    `content_type` VARCHAR(100) DEFAULT NULL COMMENT 'Content type',
    `operation` VARCHAR(20) DEFAULT NULL COMMENT 'Operation type (create/modify/revoke)',
    
    -- Storage information
    `storage_type` VARCHAR(20) DEFAULT NULL COMMENT 'Storage type (local/oss)',
    `storage_path` VARCHAR(500) DEFAULT NULL COMMENT 'Storage path',
    
    -- Transaction data
    `pre_tx_raw` TEXT COMMENT 'Pre-transaction raw data',
    `tx_raw` TEXT COMMENT 'Transaction raw data',
    `status` VARCHAR(20) DEFAULT NULL COMMENT 'Status (pending/success/failed)',
    
    -- Block information
    `block_height` BIGINT DEFAULT NULL COMMENT 'Block height',
    
    -- Timestamps
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Creation time',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Update time',
    `state` INT(11) DEFAULT 0 COMMENT 'Status (0:EXIST, 2:DELETED)',
    
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_file_id` (`file_id`),
    KEY `idx_pin_id` (`pin_id`),
    KEY `idx_meta_id` (`meta_id`),
    KEY `idx_address` (`address`),
    KEY `idx_status` (`status`),
    KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='File metadata table';

-- =============================================
-- File chunk table (tb_file_chunk)
-- =============================================
CREATE TABLE IF NOT EXISTS `tb_file_chunk` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT 'Primary key ID',
    
    -- Chunk information
    `chunk_hash` VARCHAR(80) DEFAULT NULL COMMENT 'Chunk hash',
    `chunk_size` BIGINT DEFAULT NULL COMMENT 'Chunk size',
    `chunk_md5` VARCHAR(191) DEFAULT NULL COMMENT 'Chunk MD5',
    `chunk_index` BIGINT DEFAULT NULL COMMENT 'Chunk index',
    `file_hash` VARCHAR(80) DEFAULT NULL COMMENT 'Belonging file hash',
    
    -- Content
    `content_hex` TEXT COMMENT 'Content hexadecimal',
    
    -- Transaction information
    `tx_id` VARCHAR(64) NOT NULL COMMENT 'On-chain transaction ID',
    `pin_id` VARCHAR(80) NOT NULL COMMENT 'Pin ID',
    `path` VARCHAR(191) NOT NULL COMMENT 'MetaID path',
    `content_type` VARCHAR(100) DEFAULT NULL COMMENT 'Content type',
    `size` BIGINT DEFAULT NULL COMMENT 'Size',
    `operation` VARCHAR(20) DEFAULT NULL COMMENT 'Operation type (create/modify/revoke)',
    
    -- Storage information
    `storage_type` VARCHAR(20) DEFAULT NULL COMMENT 'Storage type (local/oss)',
    `storage_path` VARCHAR(500) DEFAULT NULL COMMENT 'Storage path',
    
    -- Transaction data
    `tx_raw` TEXT COMMENT 'Transaction raw data',
    `status` VARCHAR(20) DEFAULT NULL COMMENT 'Status (pending/success/failed)',
    
    -- Block information
    `block_height` BIGINT DEFAULT NULL COMMENT 'Block height',
    
    -- Timestamps
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Creation time',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Update time',
    `state` INT(11) DEFAULT 0 COMMENT 'Status (0:EXIST, 1:DELETED)',
    
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_tx_id` (`tx_id`),
    KEY `idx_pin_id` (`pin_id`),
    KEY `idx_file_hash` (`file_hash`),
    KEY `idx_chunk_index` (`chunk_index`),
    KEY `idx_block_height` (`block_height`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='File chunk table';

-- =============================================
-- File assistent table (tb_file_assistent)
-- =============================================
CREATE TABLE IF NOT EXISTS `tb_file_assistent` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT 'Primary key ID',
    
    -- User information
    `meta_id` VARCHAR(255) NOT NULL COMMENT 'User MetaID',
    `address` VARCHAR(100) NOT NULL COMMENT 'User address',
    
    -- Assistent information
    `assistent_address` VARCHAR(100) NOT NULL COMMENT 'Assistent address (托管地址)',
    `assistent_pri_hex` TEXT NOT NULL COMMENT 'Assistent private key (hex format, 托管地址私钥)',
    
    -- Status
    `status` VARCHAR(20) DEFAULT 'success' COMMENT 'Status (success/failed)',
    
    -- Timestamps
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Creation time',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Update time',
    
    PRIMARY KEY (`id`),
    KEY `idx_meta_id` (`meta_id`),
    KEY `idx_address` (`address`),
    KEY `idx_assistent_address` (`assistent_address`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='File assistent table (文件托管助手表)';

-- =============================================
-- Index notes
-- =============================================
-- Note:
-- 1. VARCHAR field length optimized，ensure index does not exceed 767 byte limit
-- 2. file_id: VARCHAR(150) - file unique identifier
-- 3. file_hash, file_md5: VARCHAR(80) - hash value fixed length
-- 4. pin_id: VARCHAR(80) - PinID
-- 5. path: VARCHAR(191) - MetaID path
-- 6. meta_id, address: VARCHAR(100) - MetaID and address

-- =============================================
-- Composite index optimization(optional, add based on query needs)
-- =============================================
-- query files by user(by time descending)
-- ALTER TABLE tb_file ADD INDEX idx_meta_id_created (meta_id, created_at DESC);

-- statistics by status and type
-- ALTER TABLE tb_file ADD INDEX idx_status_type (status, file_type);

-- query by path and status
-- ALTER TABLE tb_file ADD INDEX idx_path_status (path, status);



