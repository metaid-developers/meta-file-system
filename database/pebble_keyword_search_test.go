package database

import (
	"encoding/json"
	"testing"

	"meta-file-system/model"

	"github.com/cockroachdb/pebble"
)

func TestPebble_GetIndexerFilesByKeywordAndExtensionWithCursor_Pagination(t *testing.T) {
	dbi, err := NewPebbleDatabase(&PebbleConfig{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewPebbleDatabase: %v", err)
	}
	defer func() { _ = dbi.Close() }()

	pdb := dbi.(*PebbleDatabase)

	olderMatch := &model.IndexerFile{
		PinID:          "pin_mp3_old_match",
		FirstPinID:     "pin_mp3_old_match",
		CreatorAddress: "addr1",
		CreatorMetaId:  "meta1",
		FileExtension:  ".mp3",
		FileName:       "周杰伦-夜曲.mp3",
		FileMd5:        "md5_old_match",
		Timestamp:      1000,
		Status:         model.StatusSuccess,
	}
	betweenNonMatch := &model.IndexerFile{
		PinID:          "pin_mp3_between_nonmatch",
		FirstPinID:     "pin_mp3_between_nonmatch",
		CreatorAddress: "addr1",
		CreatorMetaId:  "meta1",
		FileExtension:  ".mp3",
		FileName:       "林俊杰-江南.mp3",
		FileMd5:        "md5_between_nonmatch",
		Timestamp:      2000,
		Status:         model.StatusSuccess,
	}
	newerMatch := &model.IndexerFile{
		PinID:          "pin_mp3_new_match",
		FirstPinID:     "pin_mp3_new_match",
		CreatorAddress: "addr1",
		CreatorMetaId:  "meta1",
		FileExtension:  ".mp3",
		FileName:       "周杰伦-稻香.mp3",
		FileMd5:        "md5_new_match",
		Timestamp:      3000,
		Status:         model.StatusSuccess,
	}
	txtRecord := &model.IndexerFile{
		PinID:          "pin_txt_keyword",
		FirstPinID:     "pin_txt_keyword",
		CreatorAddress: "addr1",
		CreatorMetaId:  "meta1",
		FileExtension:  ".txt",
		FileName:       "周杰伦-说明.txt",
		FileMd5:        "md5_txt",
		Timestamp:      4000,
		Status:         model.StatusSuccess,
	}

	for _, f := range []*model.IndexerFile{olderMatch, betweenNonMatch, newerMatch, txtRecord} {
		if err := dbi.CreateIndexerFile(f); err != nil {
			t.Fatalf("CreateIndexerFile(%s): %v", f.PinID, err)
		}
	}

	wantCursor := findExtensionTimestampCursorForPinID(t, pdb, ".mp3", newerMatch.PinID)

	page1, cursor1, err := dbi.GetIndexerFilesByKeywordAndExtensionWithCursor("周杰伦", "mp3", "", 1)
	if err != nil {
		t.Fatalf("GetIndexerFilesByKeywordAndExtensionWithCursor(page1): %v", err)
	}
	if len(page1) != 1 {
		t.Fatalf("page1 len = %d, want 1", len(page1))
	}
	if page1[0].PinID != newerMatch.PinID {
		t.Fatalf("page1[0].PinID = %q, want %q", page1[0].PinID, newerMatch.PinID)
	}
	if cursor1 == "" {
		t.Fatalf("page1 cursor empty, want non-empty")
	}
	if cursor1 != wantCursor {
		t.Fatalf("page1 cursor = %q, want %q", cursor1, wantCursor)
	}

	page2, cursor2, err := dbi.GetIndexerFilesByKeywordAndExtensionWithCursor("周杰伦", ".MP3", cursor1, 1)
	if err != nil {
		t.Fatalf("GetIndexerFilesByKeywordAndExtensionWithCursor(page2): %v", err)
	}
	if len(page2) != 1 {
		t.Fatalf("page2 len = %d, want 1", len(page2))
	}
	if page2[0].PinID != olderMatch.PinID {
		t.Fatalf("page2[0].PinID = %q, want %q", page2[0].PinID, olderMatch.PinID)
	}
	if cursor2 != "" {
		t.Fatalf("page2 cursor = %q, want empty", cursor2)
	}
}

func findExtensionTimestampCursorForPinID(t *testing.T, pdb *PebbleDatabase, extension, pinID string) string {
	t.Helper()

	extNorm := normalizeFileExtension(extension)
	prefix := extNorm + ":"
	db := pdb.collections[collectionFileExtensionTimestamp]

	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte(prefix),
		UpperBound: []byte(prefix + "~"),
	})
	if err != nil {
		t.Fatalf("NewIter: %v", err)
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		var f model.IndexerFile
		if err := json.Unmarshal(iter.Value(), &f); err != nil {
			continue
		}
		if f.PinID == pinID {
			return extractTimestamp16FromCursorKey(string(iter.Key()))
		}
	}

	t.Fatalf("did not find pinID %q in extension index for %q", pinID, extension)
	return ""
}

