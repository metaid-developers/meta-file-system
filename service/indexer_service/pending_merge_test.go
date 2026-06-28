package indexer_service

import (
	"testing"

	"meta-file-system/conf"
	"meta-file-system/database"
	"meta-file-system/indexer"
	"meta-file-system/model"
	"meta-file-system/model/dao"
	"meta-file-system/storage"
)

// setTestPebble swaps the global database.DB to a fresh temp Pebble and
// restores it on test end. The DAOs read database.DB at construction time.
func setTestPebble(t *testing.T) {
	t.Helper()
	dbi, err := database.NewPebbleDatabase(&database.PebbleConfig{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewPebbleDatabase: %v", err)
	}
	prev := database.DB
	database.DB = dbi
	t.Cleanup(func() {
		_ = dbi.Close()
		database.DB = prev
	})
}

// setTestConfig installs a minimal conf.Cfg so the merge path (which reads
// conf.Cfg.Storage.Type) does not dereference nil.
func setTestConfig(t *testing.T) {
	t.Helper()
	prev := conf.Cfg
	conf.Cfg = &conf.Config{
		Storage: conf.StorageConfig{Type: "local"},
		Net:     "livenet",
	}
	t.Cleanup(func() { conf.Cfg = prev })
}

// newMergeTestService builds a minimal IndexerService for retry/merge tests:
// real temp Pebble-backed DAOs + a temp LocalStorage. parser/chainType are set
// to defaults; the merge path does not touch the scanner.
func newMergeTestService(t *testing.T) (*IndexerService, *storage.LocalStorage) {
	t.Helper()
	setTestPebble(t)
	setTestConfig(t)

	tmpDir := t.TempDir()
	stor, err := storage.NewLocalStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	s := &IndexerService{
		fileDAO:             dao.NewFileDAO(),
		indexerFileDAO:      dao.NewIndexerFileDAO(),
		indexerFileChunkDAO: dao.NewIndexerFileChunkDAO(),
		pendingIndexFileDAO: dao.NewPendingIndexFileDAO(),
		storage:             stor,
		chainType:           indexer.ChainTypeMVC,
	}
	return s, stor
}

// seedChunks writes IndexerFileChunk records and their bytes to storage, and
// returns the chunkList JSON to embed in a pending record / index JSON.
func seedChunks(t *testing.T, s *IndexerService, stor *storage.LocalStorage, indexPinID string, contents []string) string {
	t.Helper()
	chunkList := ""
	for i, c := range contents {
		chunkPinID := indexPinID + "-chunk" + string(rune('1'+i))
		storagePath := "indexer/chunk/mvc/" + chunkPinID
		if err := stor.Save(storagePath, []byte(c)); err != nil {
			t.Fatalf("seed chunk storage: %v", err)
		}
		chunk := &model.IndexerFileChunk{
			PinID:          chunkPinID,
			StoragePath:    storagePath,
			ChunkSize:      int64(len(c)),
			ChunkMd5:       "deadbeef",
			ChainName:      "mvc",
		}
		if err := s.indexerFileChunkDAO.Create(chunk); err != nil {
			t.Fatalf("seed chunk DAO: %v", err)
		}
		if i > 0 {
			chunkList += ","
		}
		chunkList += `{"pinId":"` + chunkPinID + `","sha256":"x"}`
	}
	return chunkList
}

// TestRetryPendingIndexMerges_MergesWhenChunksAvailable proves the deferred
// merge completes once all chunks have landed: a pending record exists, the
// chunks are indexed, retryPendingIndexMerges merges the file and deletes the
// pending record.
func TestRetryPendingIndexMerges_MergesWhenChunksAvailable(t *testing.T) {
	s, stor := newMergeTestService(t)

	const indexPinID = "idxpin-retry-1i0"
	chunkList := seedChunks(t, s, stor, indexPinID, []string{"AAA", "BBB", "CCC"})

	metaData := `{"pinID":"` + indexPinID + `","chainName":"mvc","operation":"create","creatorAddress":"1BoatSLRHtKNngkdXEeobR76b53LETtpyT"}`
	indexJSON := `{"sha256":"ignored","fileSize":9,"chunkNumber":3,"chunkList":[` + chunkList + `],"dataType":"text/plain","name":"f.txt"}`
	pending := &model.PendingIndexFile{
		PinID:       indexPinID,
		FirstPinID:  indexPinID,
		ChainName:   "mvc",
		BlockHeight: 100,
		MetaData:    metaData,
		IndexJSON:   indexJSON,
	}
	if err := s.pendingIndexFileDAO.Create(pending); err != nil {
		t.Fatalf("seed pending: %v", err)
	}

	s.retryPendingIndexMerges("mvc")

	// File merged -> IndexerFile exists.
	file, err := s.indexerFileDAO.GetByPinID(indexPinID)
	if err != nil {
		t.Fatalf("get merged file: %v", err)
	}
	if file == nil || file.ChunkType != model.ChunkTypeMulti {
		t.Fatalf("merge did not produce an IndexerFile: %+v", file)
	}
	if file.FileSize != 9 {
		t.Errorf("FileSize = %d, want 9", file.FileSize)
	}

	// Pending record deleted after successful merge.
	got, err := s.pendingIndexFileDAO.GetByPinID(indexPinID)
	if err != nil {
		t.Fatalf("get pending after retry: %v", err)
	}
	if got != nil {
		t.Errorf("pending record should be deleted after merge, got %+v", got)
	}
}

// TestRetryPendingIndexMerges_LeavesRecordWhenChunksStillMissing proves a still-
// incomplete merge is NOT dropped: the pending record survives for a later block.
func TestRetryPendingIndexMerges_LeavesRecordWhenChunksStillMissing(t *testing.T) {
	s, stor := newMergeTestService(t)

	const indexPinID = "idxpin-missing-1i0"
	// Only 1 of 2 chunks exists.
	chunkList := seedChunks(t, s, stor, indexPinID, []string{"AAA"})
	// Add a second chunk entry to the index list that has no matching record.
	chunkList += `,{"pinId":"` + indexPinID + `-chunk2","sha256":"x"}`

	metaData := `{"pinID":"` + indexPinID + `","chainName":"mvc","operation":"create","creatorAddress":"1BoatSLRHtKNngkdXEeobR76b53LETtpyT"}`
	indexJSON := `{"sha256":"ignored","fileSize":6,"chunkNumber":2,"chunkList":[` + chunkList + `],"dataType":"text/plain","name":"f.txt"}`
	if err := s.pendingIndexFileDAO.Create(&model.PendingIndexFile{
		PinID: indexPinID, FirstPinID: indexPinID, ChainName: "mvc",
		MetaData: metaData, IndexJSON: indexJSON,
	}); err != nil {
		t.Fatalf("seed pending: %v", err)
	}

	s.retryPendingIndexMerges("mvc")

	// File NOT merged.
	if file, _ := s.indexerFileDAO.GetByPinID(indexPinID); file != nil {
		t.Errorf("file should not be merged with missing chunk, got %+v", file)
	}
	// Pending record survives.
	got, _ := s.pendingIndexFileDAO.GetByPinID(indexPinID)
	if got == nil {
		t.Error("pending record should survive when chunks are still missing")
	}
}
