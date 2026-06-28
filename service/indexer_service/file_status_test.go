package indexer_service

import (
	"testing"

	"meta-file-system/database"
	"meta-file-system/model"
	"meta-file-system/storage"
)

// newStatusTestService builds a minimal IndexerFileService over a temp Pebble
// DB + temp LocalStorage for GetFileStatus tests.
func newStatusTestService(t *testing.T) *IndexerFileService {
	t.Helper()
	setTestPebble(t)
	setTestConfig(t)
	stor, err := storage.NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}
	return NewIndexerFileService(stor)
}

// TestGetFileStatus_NotFound: pin never seen -> not_found.
func TestGetFileStatus_NotFound(t *testing.T) {
	s := newStatusTestService(t)
	status, err := s.GetFileStatus("never-seen-i0")
	if err != nil {
		t.Fatalf("GetFileStatus: %v", err)
	}
	if status.Status != FileStatusNotFound {
		t.Errorf("status = %q, want not_found", status.Status)
	}
}

// TestGetFileStatus_Merged: IndexerFile record exists -> merged.
func TestGetFileStatus_Merged(t *testing.T) {
	s := newStatusTestService(t)
	const pin = "merged-pin-i0"
	if err := s.indexerFileDAO.Create(&model.IndexerFile{
		PinID: pin, ChainName: "mvc", BlockHeight: 100, FileSize: 42, FileName: "x.png",
		Status: model.StatusSuccess,
	}); err != nil {
		t.Fatalf("seed IndexerFile: %v", err)
	}

	status, err := s.GetFileStatus(pin)
	if err != nil {
		t.Fatalf("GetFileStatus: %v", err)
	}
	if status.Status != FileStatusMerged {
		t.Errorf("status = %q, want merged", status.Status)
	}
	if status.FileSize != 42 || status.FileName != "x.png" {
		t.Errorf("merged metadata not surfaced: %+v", status)
	}
}

// TestGetFileStatus_PendingPendingRecord: a PendingIndexFile exists but no
// IndexerFile -> pending (deferred multi-chunk merge).
func TestGetFileStatus_PendingPendingRecord(t *testing.T) {
	s := newStatusTestService(t)
	const pin = "pending-merge-i0"
	if err := s.pendingIndexFileDAO.Create(&model.PendingIndexFile{
		PinID: pin, ChainName: "mvc", BlockHeight: 50,
	}); err != nil {
		t.Fatalf("seed pending: %v", err)
	}

	status, err := s.GetFileStatus(pin)
	if err != nil {
		t.Fatalf("GetFileStatus: %v", err)
	}
	if status.Status != FileStatusPending {
		t.Errorf("status = %q, want pending", status.Status)
	}
}

// TestGetFileStatus_PendingPinInfo: no IndexerFile, no PendingIndexFile, but a
// PinInfo exists -> pending (on chain, waiting for scan/merge).
func TestGetFileStatus_PendingPinInfo(t *testing.T) {
	s := newStatusTestService(t)
	const pin = "pininfo-only-i0"
	// PinInfo is written via the global DB (CreateOrUpdatePinInfo), not via the
	// IndexerFileService DAO. Add a minimal PinInfo record directly.
	pinInfo := &model.IndexerPinInfo{
		PinID: pin, ChainName: "mvc", BlockHeight: 77, Timestamp: 1, Operation: "create",
	}
	if err := database.DB.CreateOrUpdatePinInfo(pinInfo); err != nil {
		t.Fatalf("seed pinInfo: %v", err)
	}

	status, err := s.GetFileStatus(pin)
	if err != nil {
		t.Fatalf("GetFileStatus: %v", err)
	}
	if status.Status != FileStatusPending {
		t.Errorf("status = %q, want pending", status.Status)
	}
	if status.BlockHeight != 77 {
		t.Errorf("BlockHeight = %d, want 77", status.BlockHeight)
	}
}
