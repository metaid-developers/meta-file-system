-- ============================================
-- MetaID Indexer Database Schema
-- ============================================
-- This file contains all table definitions for the Indexer service
-- Tables: tb_indexer_file, tb_indexer_file_chunk, tb_indexer_user_avatar, tb_indexer_sync_status
-- ============================================

-- --------------------------------------------
-- Table: tb_indexer_file
-- Description: Stores indexed file metadata from blockchain
-- --------------------------------------------
CREATE TABLE IF NOT EXISTS `tb_indexer_file` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT 'Primary key ID',
    
    -- MetaID related fields
    `pin_id` VARCHAR(255) NOT NULL COMMENT 'PIN ID (txid + i + vout)',
    `tx_id` VARCHAR(64) NOT NULL COMMENT 'Transaction ID',
    `vout` INT NOT NULL COMMENT 'Output index',
    `path` VARCHAR(500) NOT NULL COMMENT 'MetaID path',
    `operation` VARCHAR(20) NOT NULL COMMENT 'Operation: create/modify/revoke',
    `parent_path` VARCHAR(500) DEFAULT '' COMMENT 'Parent path',
    `encryption` VARCHAR(50) DEFAULT '0' COMMENT 'Encryption method',
    `version` VARCHAR(50) DEFAULT '0' COMMENT 'Version',
    `content_type` VARCHAR(100) DEFAULT '' COMMENT 'Content type',
    
    -- File related fields
    `file_type` VARCHAR(20) DEFAULT '' COMMENT 'File type: image/video/audio/document/text/archive/data/other',
    `file_extension` VARCHAR(10) DEFAULT '' COMMENT 'File extension: .jpg, .png, .mp4, .pdf, etc.',
    `file_name` VARCHAR(255) DEFAULT '' COMMENT 'File name (extracted from path)',
    `file_size` BIGINT DEFAULT 0 COMMENT 'File size (bytes)',
    `file_md5` VARCHAR(64) DEFAULT '' COMMENT 'File MD5 hash',
    `file_hash` VARCHAR(64) DEFAULT '' COMMENT 'File SHA256 hash',
    `is_gzip_compressed` TINYINT(1) DEFAULT 0 COMMENT 'Whether the original content was gzip compressed',
    
    -- Storage related fields
    `storage_type` VARCHAR(20) DEFAULT 'local' COMMENT 'Storage type: local/oss',
    `storage_path` VARCHAR(500) DEFAULT '' COMMENT 'Storage path',
    
    -- Blockchain related fields
    `chain_name` VARCHAR(20) NOT NULL COMMENT 'Chain name: btc/mvc',
    `block_height` BIGINT NOT NULL COMMENT 'Block height',
    `timestamp` BIGINT NOT NULL COMMENT 'Block timestamp (seconds since epoch)',
    `creator_meta_id` VARCHAR(64) DEFAULT '' COMMENT 'Creator MetaID (SHA256 of address)',
    `creator_address` VARCHAR(100) DEFAULT '' COMMENT 'Creator address',
    `owner_address` VARCHAR(100) DEFAULT '' COMMENT 'Owner address (current)',
    `owner_meta_id` VARCHAR(64) DEFAULT '' COMMENT 'Owner MetaID (SHA256 of owner address)',
    
    -- Status fields
    `status` VARCHAR(20) DEFAULT 'success' COMMENT 'Status: success/failed',
    `state` INT(11) DEFAULT 0 COMMENT 'State: 0=EXIST, 2=DELETED',
    
    -- Timestamps
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Creation time',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Update time',
    
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_pin_id` (`pin_id`),
    KEY `idx_tx_id` (`tx_id`),
    KEY `idx_path` (`path`(255)),
    KEY `idx_block_height` (`block_height`),
    KEY `idx_creator_address` (`creator_address`),
    KEY `idx_creator_meta_id` (`creator_meta_id`),
    KEY `idx_owner_address` (`owner_address`),
    KEY `idx_chain_name` (`chain_name`),
    KEY `idx_timestamp` (`timestamp`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Indexer file metadata table';

-- --------------------------------------------
-- Table: tb_indexer_file_chunk
-- Description: Stores indexed file chunk metadata (for large files split into chunks)
-- --------------------------------------------
CREATE TABLE IF NOT EXISTS `tb_indexer_file_chunk` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT 'Primary key ID',
    
    -- MetaID related fields
    `pin_id` VARCHAR(255) NOT NULL COMMENT 'PIN ID (txid + i + vout)',
    `tx_id` VARCHAR(64) NOT NULL COMMENT 'Transaction ID',
    `vout` INT NOT NULL COMMENT 'Output index',
    `path` VARCHAR(500) NOT NULL COMMENT 'MetaID path',
    `operation` VARCHAR(20) NOT NULL COMMENT 'Operation: create/modify/revoke',
    `content_type` VARCHAR(100) DEFAULT '' COMMENT 'Content type',
    
    -- Chunk related fields
    `chunk_index` INT NOT NULL COMMENT 'Chunk index (0-based)',
    `chunk_size` BIGINT DEFAULT 0 COMMENT 'Chunk size (bytes)',
    `chunk_md5` VARCHAR(64) DEFAULT '' COMMENT 'Chunk MD5 hash',
    `parent_pin_id` VARCHAR(255) NOT NULL COMMENT 'Parent file PIN ID',
    `is_gzip_compressed` TINYINT(1) DEFAULT 0 COMMENT 'Whether the original content was gzip compressed',
    
    -- Storage related fields
    `storage_type` VARCHAR(20) DEFAULT 'local' COMMENT 'Storage type: local/oss',
    `storage_path` VARCHAR(500) DEFAULT '' COMMENT 'Storage path',
    
    -- Blockchain related fields
    `chain_name` VARCHAR(20) NOT NULL COMMENT 'Chain name: btc/mvc',
    `block_height` BIGINT NOT NULL COMMENT 'Block height',
    
    -- Status fields
    `status` VARCHAR(20) DEFAULT 'success' COMMENT 'Status: success/failed',
    `state` INT(11) DEFAULT 0 COMMENT 'State: 0=EXIST, 2=DELETED',
    
    -- Timestamps
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Creation time',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Update time',
    
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_pin_id` (`pin_id`),
    KEY `idx_tx_id` (`tx_id`),
    KEY `idx_path` (`path`(255)),
    KEY `idx_parent_pin_id` (`parent_pin_id`),
    KEY `idx_block_height` (`block_height`),
    KEY `idx_chain_name` (`chain_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Indexer file chunk metadata table';

-- --------------------------------------------
-- Table: tb_indexer_user_avatar
-- Description: Stores user avatar metadata indexed from blockchain
-- --------------------------------------------
CREATE TABLE IF NOT EXISTS `tb_indexer_user_avatar` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT 'Primary key ID',
    
    -- PIN information
    `pin_id` VARCHAR(255) NOT NULL COMMENT 'PIN ID (unique identifier)',
    `tx_id` VARCHAR(100) NOT NULL COMMENT 'Transaction ID',
    
    -- MetaID information
    `meta_id` VARCHAR(100) NOT NULL COMMENT 'Meta ID (SHA256 of address)',
    `address` VARCHAR(100) NOT NULL COMMENT 'User address',
    
    -- Avatar information
    `avatar` VARCHAR(500) NOT NULL COMMENT 'Avatar storage path or URL',
    `content_type` VARCHAR(100) DEFAULT '' COMMENT 'Content type (e.g., image/jpeg)',
    `file_size` BIGINT DEFAULT 0 COMMENT 'File size (bytes)',
    `file_md5` VARCHAR(64) DEFAULT '' COMMENT 'File MD5 hash',
    `file_hash` VARCHAR(64) DEFAULT '' COMMENT 'File SHA256 hash',
    `file_extension` VARCHAR(10) DEFAULT '' COMMENT 'File extension: .jpg, .png, etc.',
    `file_type` VARCHAR(20) DEFAULT '' COMMENT 'File type: image/video/audio/other',
    
    -- Chain information
    `chain_name` VARCHAR(20) NOT NULL COMMENT 'Chain name: btc/mvc',
    `block_height` BIGINT NOT NULL COMMENT 'Block height',
    `timestamp` BIGINT NOT NULL COMMENT 'Block timestamp (seconds since epoch)',
    
    -- Timestamps
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Creation time',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Update time',
    
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_pin_id` (`pin_id`),
    KEY `idx_tx_id` (`tx_id`),
    KEY `idx_meta_id` (`meta_id`),
    KEY `idx_address` (`address`),
    KEY `idx_chain_name` (`chain_name`),
    KEY `idx_block_height` (`block_height`),
    KEY `idx_timestamp` (`timestamp`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Indexer user avatar table';

-- --------------------------------------------
-- Table: tb_indexer_sync_status
-- Description: Stores blockchain synchronization status for each chain
-- --------------------------------------------
CREATE TABLE IF NOT EXISTS `tb_indexer_sync_status` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT 'Primary key ID',
    
    -- Chain information
    `chain_name` VARCHAR(20) NOT NULL COMMENT 'Chain name: btc/mvc',
    
    -- Sync status
    `current_sync_height` BIGINT NOT NULL DEFAULT 0 COMMENT 'Current scanned block height',
    
    -- Timestamps
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Creation time',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Update time',
    
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_chain_name` (`chain_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Indexer synchronization status table';

-- --------------------------------------------
-- Initialize default sync status records
-- --------------------------------------------

-- Insert default status for MVC chain
INSERT INTO `tb_indexer_sync_status` (`chain_name`, `current_sync_height`)
VALUES ('mvc', 0)
ON DUPLICATE KEY UPDATE `chain_name` = `chain_name`;

-- Insert default status for BTC chain
INSERT INTO `tb_indexer_sync_status` (`chain_name`, `current_sync_height`)
VALUES ('btc', 0)
ON DUPLICATE KEY UPDATE `chain_name` = `chain_name`;

-- ============================================
-- Migration: Add is_gzip_compressed field
-- ============================================
-- Add is_gzip_compressed field to tb_indexer_file table
ALTER TABLE `tb_indexer_file` 
ADD COLUMN `is_gzip_compressed` TINYINT(1) DEFAULT 0 COMMENT 'Whether the original content was gzip compressed' 
AFTER `file_hash`;

-- Add is_gzip_compressed field to tb_indexer_file_chunk table
ALTER TABLE `tb_indexer_file_chunk` 
ADD COLUMN `is_gzip_compressed` TINYINT(1) DEFAULT 0 COMMENT 'Whether the original content was gzip compressed' 
AFTER `chunk_md5`;

-- ============================================
-- End of Indexer Database Schema
-- ============================================
