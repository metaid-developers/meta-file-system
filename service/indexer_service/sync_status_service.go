package indexer_service

import (
	"errors"
	"fmt"
	"log"

	"meta-file-system/conf"
	"meta-file-system/indexer"
	"meta-file-system/model"
	"meta-file-system/model/dao"

	"gorm.io/gorm"
)

// SyncStatusService sync status service
type SyncStatusService struct {
	syncStatusDAO *dao.IndexerSyncStatusDAO
	scanner       *indexer.BlockScanner
	coordinator   *indexer.MultiChainCoordinator
	isMultiChain  bool
}

// NewSyncStatusService create sync status service instance
func NewSyncStatusService() *SyncStatusService {
	return &SyncStatusService{
		syncStatusDAO: dao.NewIndexerSyncStatusDAO(),
	}
}

// SetBlockScanner set block scanner for getting latest block height (single-chain mode)
func (s *SyncStatusService) SetBlockScanner(scanner *indexer.BlockScanner) {
	s.scanner = scanner
	s.isMultiChain = false
}

// SetMultiChainCoordinator set multi-chain coordinator for getting latest block heights
func (s *SyncStatusService) SetMultiChainCoordinator(coordinator *indexer.MultiChainCoordinator) {
	s.coordinator = coordinator
	s.isMultiChain = true
}

// GetSyncStatus get sync status (default MVC chain)
func (s *SyncStatusService) GetSyncStatus() (*model.IndexerSyncStatus, error) {
	return s.GetSyncStatusByChain("mvc")
}

// GetSyncStatusByChain get sync status by chain name
func (s *SyncStatusService) GetSyncStatusByChain(chainName string) (*model.IndexerSyncStatus, error) {
	status, err := s.syncStatusDAO.GetByChainName(chainName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("sync status not found")
		}
		return nil, fmt.Errorf("failed to get sync status: %w", err)
	}
	return status, nil
}

// GetAllSyncStatus get all chain sync status
func (s *SyncStatusService) GetAllSyncStatus() ([]*model.IndexerSyncStatus, error) {
	statuses, err := s.syncStatusDAO.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get all sync status: %w", err)
	}
	return statuses, nil
}

// GetLatestBlockHeight get latest block height from node (single-chain mode)
func (s *SyncStatusService) GetLatestBlockHeight() (int64, error) {
	if s.scanner == nil {
		return 0, errors.New("scanner not available")
	}

	latestHeight, err := s.scanner.GetBlockCount()
	if err != nil {
		log.Printf("Failed to get latest block height from node: %v", err)
		return 0, fmt.Errorf("failed to get latest block height: %w", err)
	}

	return latestHeight, nil
}

// GetLatestBlockHeightsForAllChains get latest block heights for all chains (multi-chain mode)
func (s *SyncStatusService) GetLatestBlockHeightsForAllChains() (map[string]int64, error) {
	latestHeights := make(map[string]int64)

	if s.isMultiChain && s.coordinator != nil {
		// Get all scanners from coordinator
		// We need to access the scanners map in coordinator
		// For now, we'll try to get latest height from each chain's RPC

		// Get all chain sync statuses to know which chains exist
		statuses, err := s.syncStatusDAO.GetAll()
		if err != nil {
			return latestHeights, fmt.Errorf("failed to get sync statuses: %w", err)
		}

		// For each chain, we need to create a temporary scanner to get latest height
		// This is not ideal, but works for now
		// Better approach would be to expose scanners from coordinator
		for _, status := range statuses {
			// Get chain config from global config
			var chainConfig *conf.ChainInstanceConfig
			for i := range conf.Cfg.Indexer.Chains {
				if conf.Cfg.Indexer.Chains[i].Name == status.ChainName {
					chainConfig = &conf.Cfg.Indexer.Chains[i]
					break
				}
			}

			if chainConfig == nil {
				log.Printf("Chain config not found for %s, using 0 as latest height", status.ChainName)
				latestHeights[status.ChainName] = 0
				continue
			}

			// Determine chain type
			var chainType indexer.ChainType
			if status.ChainName == "btc" {
				chainType = indexer.ChainTypeBTC
			} else {
				chainType = indexer.ChainTypeMVC
			}

			// Create temporary scanner just to get block count
			tempScanner := indexer.NewBlockScannerWithChain(
				chainConfig.RpcUrl,
				chainConfig.RpcUser,
				chainConfig.RpcPass,
				0,
				10,
				chainType,
			)

			height, err := tempScanner.GetBlockCount()
			if err != nil {
				log.Printf("Failed to get latest block height for %s: %v", status.ChainName, err)
				latestHeights[status.ChainName] = 0
			} else {
				latestHeights[status.ChainName] = height
			}
		}
	} else if s.scanner != nil {
		// Single-chain mode - get from scanner
		height, err := s.scanner.GetBlockCount()
		if err != nil {
			log.Printf("Failed to get latest block height: %v", err)
		} else {
			// Need to determine chain name from scanner
			// For now, assume it's from the default chain
			latestHeights["mvc"] = height
		}
	}

	return latestHeights, nil
}
