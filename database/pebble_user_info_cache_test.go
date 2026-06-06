package database

import (
	"testing"

	"meta-file-system/model"
	common_service "meta-file-system/service/common_service"
)

func TestBuildUserInfoCachePayloadIncludesGlobalMetaID(t *testing.T) {
	dbi, err := NewPebbleDatabase(&PebbleConfig{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewPebbleDatabase: %v", err)
	}
	defer func() { _ = dbi.Close() }()

	pdb := dbi.(*PebbleDatabase)
	metaID := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	address := "1BoatSLRHtKNngkdXEeobR76b53LETtpyT"
	wantGlobalMetaID := common_service.ConvertToGlobalMetaId(address)
	if wantGlobalMetaID == "" {
		t.Fatalf("ConvertToGlobalMetaId(%q) returned empty", address)
	}

	if err := pdb.SaveMetaIdAddress(metaID, address); err != nil {
		t.Fatalf("SaveMetaIdAddress: %v", err)
	}
	nameInfo := &model.UserNameInfo{
		Name:        "Alice Next",
		PinID:       "name-pin-2",
		ChainName:   "btc",
		BlockHeight: 12345,
		Timestamp:   2000,
	}
	if err := pdb.CreateOrUpdateLatestUserNameInfo(nameInfo, metaID); err != nil {
		t.Fatalf("CreateOrUpdateLatestUserNameInfo: %v", err)
	}

	userInfo, latestNameInfo := pdb.buildUserInfoCachePayload(metaID)
	if userInfo == nil {
		t.Fatalf("buildUserInfoCachePayload returned nil userInfo")
	}
	if latestNameInfo == nil {
		t.Fatalf("buildUserInfoCachePayload returned nil latestNameInfo")
	}
	if userInfo.GlobalMetaId != wantGlobalMetaID {
		t.Fatalf("GlobalMetaId = %q, want %q", userInfo.GlobalMetaId, wantGlobalMetaID)
	}
	if userInfo.MetaId != metaID {
		t.Fatalf("MetaId = %q, want %q", userInfo.MetaId, metaID)
	}
	if userInfo.Address != address {
		t.Fatalf("Address = %q, want %q", userInfo.Address, address)
	}
	if userInfo.Name != nameInfo.Name {
		t.Fatalf("Name = %q, want %q", userInfo.Name, nameInfo.Name)
	}
}

func TestUserInfoCacheKeysIncludeGlobalMetaID(t *testing.T) {
	userInfo := &model.IndexerUserInfo{
		GlobalMetaId: "idq1exampleglobalmetaid",
		MetaId:       "meta-id",
		Address:      "1BoatSLRHtKNngkdXEeobR76b53LETtpyT",
	}

	got := userInfoCacheKeys(userInfo)
	wantKeys := []string{
		"user:metaid:" + userInfo.MetaId,
		"user:globalmetaid:" + userInfo.GlobalMetaId,
		"user:address:" + userInfo.Address,
	}

	seen := make(map[string]bool, len(got))
	for _, key := range got {
		seen[key] = true
	}
	for _, want := range wantKeys {
		if !seen[want] {
			t.Fatalf("userInfoCacheKeys missing %q in %#v", want, got)
		}
	}
}
