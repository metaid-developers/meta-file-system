package database

import (
	"testing"

	"meta-file-system/model"
)

func newTestPebble(t *testing.T) *PebbleDatabase {
	t.Helper()
	dbi, err := NewPebbleDatabase(&PebbleConfig{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewPebbleDatabase: %v", err)
	}
	t.Cleanup(func() { _ = dbi.Close() })
	return dbi.(*PebbleDatabase)
}

func TestPendingIndexFile_CRUDRoundTrip(t *testing.T) {
	pdb := newTestPebble(t)

	pending := &model.PendingIndexFile{
		PinID:       "indexpin-mvc-1i0",
		FirstPinID:  "indexpin-mvc-1i0",
		TxID:        "abc123",
		ChainName:   "mvc",
		BlockHeight: 179400,
		Timestamp:   1700000000,
		MetaData:    `{"pinID":"indexpin-mvc-1i0"}`,
		IndexJSON:   `{"sha256":"deadbeef","chunkNumber":3,"chunkList":[{"pinId":"c1i0"},{"pinId":"c2i0"},{"pinId":"c3i0"}]}`,
	}

	// Create
	if err := pdb.CreatePendingIndexFile(pending); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Get
	got, err := pdb.GetPendingIndexFileByPinID(pending.PinID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.PinID != pending.PinID || got.ChainName != "mvc" || got.BlockHeight != 179400 {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
	if got.IndexJSON != pending.IndexJSON {
		t.Fatalf("IndexJSON mismatch: got %q", got.IndexJSON)
	}

	// List by chain
	list, err := pdb.ListPendingIndexFilesByChain("mvc")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].PinID != pending.PinID {
		t.Fatalf("list = %+v, want 1 mvc record", list)
	}

	// List by a different chain returns nothing
	other, err := pdb.ListPendingIndexFilesByChain("btc")
	if err != nil {
		t.Fatalf("List btc: %v", err)
	}
	if len(other) != 0 {
		t.Fatalf("btc list should be empty, got %d", len(other))
	}

	// Delete
	if err := pdb.DeletePendingIndexFile(pending.PinID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := pdb.GetPendingIndexFileByPinID(pending.PinID); err != ErrNotFound {
		t.Fatalf("after delete, Get should return ErrNotFound, got %v", err)
	}
}

func TestPendingIndexFile_CreateOverwrites(t *testing.T) {
	pdb := newTestPebble(t)

	v1 := &model.PendingIndexFile{PinID: "p1i0", ChainName: "mvc", BlockHeight: 1, IndexJSON: `{"v":1}`}
	v2 := &model.PendingIndexFile{PinID: "p1i0", ChainName: "mvc", BlockHeight: 2, IndexJSON: `{"v":2}`}

	if err := pdb.CreatePendingIndexFile(v1); err != nil {
		t.Fatal(err)
	}
	if err := pdb.CreatePendingIndexFile(v2); err != nil {
		t.Fatal(err)
	}
	got, err := pdb.GetPendingIndexFileByPinID("p1i0")
	if err != nil {
		t.Fatal(err)
	}
	if got.BlockHeight != 2 || got.IndexJSON != `{"v":2}` {
		t.Fatalf("overwrite failed: %+v", got)
	}
}

func TestPendingIndexFile_GetMissingReturnsErrNotFound(t *testing.T) {
	pdb := newTestPebble(t)
	if _, err := pdb.GetPendingIndexFileByPinID("never-saved-i0"); err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}
