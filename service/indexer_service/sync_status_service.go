package indexer_service

import (
	"errors"
	"fmt"
	"log"

	"meta-file-system/indexer"
	"meta-file-system/model"
	"meta-file-system/model/dao"

	"gorm.io/gorm"
)

// SyncStatusService sync status service
type SyncStatusService struct {
	syncStatusDAO *dao.IndexerSyncStatusDAO
	scanner       *indexer.BlockScanner
}

// NewSyncStatusService create sync status service instance
func NewSyncStatusService() *SyncStatusService {
	return &SyncStatusService{
		syncStatusDAO: dao.NewIndexerSyncStatusDAO(),
	}
}

// SetBlockScanner set block scanner for getting latest block height
func (s *SyncStatusService) SetBlockScanner(scanner *indexer.BlockScanner) {
	s.scanner = scanner
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

// GetLatestBlockHeight get latest block height from node
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
