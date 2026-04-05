# File Keyword + Extension Search Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `GET /files/keyword/{keyword}/extension` so clients can search indexed files whose base name contains a keyword within one or more requested extensions.

**Architecture:** Reuse the existing extension query shape and response type. Add a shared file-name keyword helper, implement cursor-aware filtering inside the Pebble adapter so pagination stays correct, keep a compatibility MySQL implementation with the same semantics, then expose the new method through DAO, service, handler, router, and Swagger docs.

**Tech Stack:** Go, Gin, PebbleDB, GORM/MySQL, swaggo/swag, Go test

---

## Spec Reference

- `docs/superpowers/specs/2026-04-05-file-keyword-extension-search-design.md`

## File Map

**Create:**

- `database/file_keyword_search.go` - Shared base-name extraction and keyword-match helpers used by both database adapters
- `database/file_keyword_search_test.go` - Table-driven tests for helper semantics
- `database/pebble_keyword_search_test.go` - Temp-dir Pebble tests for keyword filtering and pagination
- `controller/handler/indexer_query_test.go` - Gin handler validation tests for the new endpoint

**Modify:**

- `database/interface.go:13-19` - Add the keyword + extension cursor query contract to the database interface
- `database/pebble_adapter.go:241-248,618-688` - Reuse extension index scanning and add keyword-aware cursor iteration
- `database/mysql_adapter.go:155-220` - Add compatibility keyword + extension query path with the shared helper
- `model/dao/indexer_file_dao.go:68-76` - Add DAO wrapper for the new database method
- `service/indexer_service/indexer_file_service.go:165-189` - Add service method and multi-extension merge flow for keyword search
- `controller/handler/indexer_query.go:235-441` - Add handler, reuse extension parsing helpers, and wire Swagger annotations
- `controller/indexer_router.go:91-96` - Register the new route next to the existing extension routes
- `docs/indexer/indexer_docs.go` - Generated Swagger docs after annotations are updated
- `docs/indexer/indexer_swagger.json` - Generated Swagger docs after annotations are updated
- `docs/indexer/indexer_swagger.yaml` - Generated Swagger docs after annotations are updated

## Task 1: Add Shared File-Keyword Helpers

**Files:**

- Create: `database/file_keyword_search.go`
- Test: `database/file_keyword_search_test.go`

- [ ] **Step 1: Write the failing helper tests**

```go
package database

import "testing"

func TestExtractFileBaseName(t *testing.T) {
	cases := []struct {
		name     string
		fileName string
		want     string
	}{
		{name: "single extension", fileName: "周杰伦-夜曲.mp3", want: "周杰伦-夜曲"},
		{name: "multiple dots", fileName: "jay.live.2004.mp3", want: "jay.live.2004"},
		{name: "no extension", fileName: "周杰伦", want: "周杰伦"},
		{name: "empty", fileName: "", want: ""},
		{name: "double suffix", fileName: "archive.tar.gz", want: "archive.tar"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractFileBaseName(tc.fileName); got != tc.want {
				t.Fatalf("extractFileBaseName(%q) = %q, want %q", tc.fileName, got, tc.want)
			}
		})
	}
}

func TestFileBaseNameContainsKeyword(t *testing.T) {
	cases := []struct {
		name     string
		fileName string
		keyword  string
		want     bool
	}{
		{name: "unicode match", fileName: "周杰伦-夜曲.mp3", keyword: "周杰伦", want: true},
		{name: "case insensitive", fileName: "JayChou.Live.mp3", keyword: "live", want: true},
		{name: "no extension still matches", fileName: "周杰伦", keyword: "周杰伦", want: true},
		{name: "empty file name", fileName: "", keyword: "周杰伦", want: false},
		{name: "blank keyword", fileName: "周杰伦-夜曲.mp3", keyword: "   ", want: false},
		{name: "miss", fileName: "周杰伦-夜曲.mp3", keyword: "林俊杰", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := fileBaseNameContainsKeyword(tc.fileName, tc.keyword); got != tc.want {
				t.Fatalf("fileBaseNameContainsKeyword(%q, %q) = %v, want %v", tc.fileName, tc.keyword, got, tc.want)
			}
		})
	}
}

func TestExtractTimestamp16FromCursorKey(t *testing.T) {
	cases := []struct {
		name string
		key  string
		want string
	}{
		{name: "extension key", key: ".mp3:0000000400123456", want: "0000000400123456"},
		{name: "global meta key", key: "globalMeta:.mp3:0000000400123456", want: "0000000400123456"},
		{name: "plain value", key: "0000000400123456", want: "0000000400123456"},
		{name: "empty", key: "", want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractTimestamp16FromCursorKey(tc.key); got != tc.want {
				t.Fatalf("extractTimestamp16FromCursorKey(%q) = %q, want %q", tc.key, got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run the helper tests to verify they fail**

Run: `go test ./database -run 'TestExtractFileBaseName|TestFileBaseNameContainsKeyword|TestExtractTimestamp16FromCursorKey' -v`

Expected: FAIL with `undefined: extractFileBaseName`, `undefined: fileBaseNameContainsKeyword`, and `undefined: extractTimestamp16FromCursorKey`

- [ ] **Step 3: Write the minimal helper implementation**

```go
package database

import (
	"path/filepath"
	"strings"
)

func extractFileBaseName(fileName string) string {
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		return ""
	}

	ext := filepath.Ext(fileName)
	if ext == "" {
		return fileName
	}

	return strings.TrimSuffix(fileName, ext)
}

func fileBaseNameContainsKeyword(fileName, keyword string) bool {
	baseName := extractFileBaseName(fileName)
	keyword = strings.TrimSpace(keyword)
	if baseName == "" || keyword == "" {
		return false
	}

	return strings.Contains(strings.ToLower(baseName), strings.ToLower(keyword))
}

func extractTimestamp16FromCursorKey(key string) string {
	if key == "" {
		return ""
	}

	idx := strings.LastIndex(key, ":")
	if idx < 0 {
		return key
	}

	return key[idx+1:]
}
```

- [ ] **Step 4: Run the helper tests to verify they pass**

Run: `go test ./database -run 'TestExtractFileBaseName|TestFileBaseNameContainsKeyword|TestExtractTimestamp16FromCursorKey' -v`

Expected: PASS for both tests

- [ ] **Step 5: Commit the helper work**

```bash
git add database/file_keyword_search.go database/file_keyword_search_test.go
git commit -m "test: add file keyword helper coverage"
```

## Task 2: Implement Pebble Keyword Search with Correct Pagination

**Files:**

- Modify: `database/interface.go`
- Modify: `database/pebble_adapter.go`
- Test: `database/pebble_keyword_search_test.go`

- [ ] **Step 1: Write the failing Pebble search test**

```go
package database

import (
	"testing"

	"meta-file-system/model"
)

func TestPebbleGetIndexerFilesByKeywordAndExtensionWithCursor(t *testing.T) {
	dbi, err := NewPebbleDatabase(&PebbleConfig{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewPebbleDatabase() error = %v", err)
	}
	defer dbi.Close()

	db := dbi.(*PebbleDatabase)
	files := []*model.IndexerFile{
		{PinID: "pin-4", FirstPinID: "first-4", FileName: "周杰伦-最伟大的作品.mp3", FileExtension: ".mp3", Timestamp: 400, Status: model.StatusSuccess},
		{PinID: "pin-3", FirstPinID: "first-3", FileName: "not-match.mp3", FileExtension: ".mp3", Timestamp: 300, Status: model.StatusSuccess},
		{PinID: "pin-2", FirstPinID: "first-2", FileName: "周杰伦-夜曲.mp3", FileExtension: ".mp3", Timestamp: 200, Status: model.StatusSuccess},
		{PinID: "pin-1", FirstPinID: "first-1", FileName: "older.txt", FileExtension: ".txt", Timestamp: 100, Status: model.StatusSuccess},
	}

	for _, file := range files {
		if err := db.CreateIndexerFile(file); err != nil {
			t.Fatalf("CreateIndexerFile(%s) error = %v", file.PinID, err)
		}
	}

	page1, cursor1, err := db.GetIndexerFilesByKeywordAndExtensionWithCursor("周杰伦", ".mp3", "", 1)
	if err != nil {
		t.Fatalf("page1 query error = %v", err)
	}
	if len(page1) != 1 || page1[0].PinID != "pin-4" {
		t.Fatalf("page1 = %#v, want pin-4", page1)
	}
	if cursor1 == "" {
		t.Fatal("cursor1 should not be empty")
	}

	page2, cursor2, err := db.GetIndexerFilesByKeywordAndExtensionWithCursor("周杰伦", ".mp3", cursor1, 1)
	if err != nil {
		t.Fatalf("page2 query error = %v", err)
	}
	if len(page2) != 1 || page2[0].PinID != "pin-2" {
		t.Fatalf("page2 = %#v, want pin-2", page2)
	}
	if cursor2 != "" {
		t.Fatalf("cursor2 = %q, want empty", cursor2)
	}
}
```

- [ ] **Step 2: Run the Pebble test to verify it fails**

Run: `go test ./database -run TestPebbleGetIndexerFilesByKeywordAndExtensionWithCursor -v`

Expected: FAIL with `db.GetIndexerFilesByKeywordAndExtensionWithCursor undefined`

- [ ] **Step 3: Add the database contract and Pebble implementation**

```go
// database/interface.go
GetIndexerFilesByKeywordAndExtensionWithCursor(keyword string, extension string, cursor string, size int) ([]*model.IndexerFile, string, error)
```

```go
// database/pebble_adapter.go
func (p *PebbleDatabase) GetIndexerFilesByKeywordAndExtensionWithCursor(keyword string, extension string, cursor string, size int) ([]*model.IndexerFile, string, error) {
	if size < 1 || size > 100 {
		size = 20
	}

	extNorm := normalizeFileExtension(extension)
	prefix := extNorm + ":"
	lowerBound := []byte(prefix)
	upperBound := []byte(prefix + "~")
	if cursor != "" {
		upperBound = []byte(prefix + cursor)
	}

	iter, err := p.collections[collectionFileExtensionTimestamp].NewIter(&pebble.IterOptions{
		LowerBound: lowerBound,
		UpperBound: upperBound,
	})
	if err != nil {
		return nil, "", err
	}
	defer iter.Close()

	var files []*model.IndexerFile
	var matchKeys [][]byte

	for ok := iter.Last(); ok; ok = iter.Prev() {
		var file model.IndexerFile
		if err := json.Unmarshal(iter.Value(), &file); err != nil {
			continue
		}
		if file.Status != model.StatusSuccess {
			continue
		}
		if !fileBaseNameContainsKeyword(file.FileName, keyword) {
			continue
		}

		fileCopy := file
		files = append(files, &fileCopy)
		matchKeys = append(matchKeys, append([]byte(nil), iter.Key()...))
		if len(files) >= size+1 {
			break
		}
	}

	nextCursor := ""
	if len(files) > size {
		nextCursor = extractTimestamp16FromCursorKey(string(matchKeys[size-1]))
		files = files[:size]
	}

	return files, nextCursor, nil
}
```

- [ ] **Step 4: Run the Pebble tests to verify they pass**

Run: `go test ./database -run 'TestExtractFileBaseName|TestFileBaseNameContainsKeyword|TestExtractTimestamp16FromCursorKey|TestPebbleGetIndexerFilesByKeywordAndExtensionWithCursor' -v`

Expected: PASS for all three tests

- [ ] **Step 5: Commit the Pebble search work**

```bash
git add database/interface.go database/pebble_adapter.go database/pebble_keyword_search_test.go
git commit -m "feat: add pebble file keyword extension search"
```

## Task 3: Wire DAO, Service, MySQL Compatibility, and HTTP Surface

**Files:**

- Modify: `database/mysql_adapter.go`
- Modify: `model/dao/indexer_file_dao.go`
- Modify: `service/indexer_service/indexer_file_service.go`
- Modify: `controller/handler/indexer_query.go`
- Modify: `controller/indexer_router.go`
- Test: `controller/handler/indexer_query_test.go`

- [ ] **Step 1: Write the failing handler validation tests**

```go
package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetFilesByKeywordAndExtensionValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("blank keyword", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "keyword", Value: "   "}}
		c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/files/keyword/%20%20%20/extension?extension=.mp3", nil)

		handler := &IndexerQueryHandler{}
		handler.GetFilesByKeywordAndExtension(c)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("missing extension", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "keyword", Value: "周杰伦"}}
		c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/files/keyword/%E5%91%A8%E6%9D%B0%E4%BC%A6/extension", nil)

		handler := &IndexerQueryHandler{}
		handler.GetFilesByKeywordAndExtension(c)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}

		var resp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
	})
}
```

- [ ] **Step 2: Run the handler test to verify it fails**

Run: `go test ./controller/handler -run TestGetFilesByKeywordAndExtensionValidation -v`

Expected: FAIL with `handler.GetFilesByKeywordAndExtension undefined`

- [ ] **Step 3: Add the new query path through MySQL, DAO, service, handler, and router**

```go
// database/mysql_adapter.go
func (m *MySQLDatabase) GetIndexerFilesByKeywordAndExtensionWithCursor(keyword string, extension string, cursor string, size int) ([]*model.IndexerFile, string, error) {
	if size < 1 || size > 100 {
		size = 20
	}

	extNorm := strings.TrimSpace(strings.ToLower(extension))
	if extNorm != "" && !strings.HasPrefix(extNorm, ".") {
		extNorm = "." + extNorm
	}

	var files []*model.IndexerFile
	nextCursor := ""

	lastSeenID := int64(0)
	if cursor != "" {
		lastSeenID, _ = strconv.ParseInt(cursor, 10, 64)
	}

	for {
		query := m.db.Where("file_extension = ? AND status = ? AND state = 0", extNorm, model.StatusSuccess)
		if lastSeenID > 0 {
			query = query.Where("id < ?", lastSeenID)
		}

		var rows []*model.IndexerFile
		if err := query.Order("timestamp DESC, id DESC").Limit(size * 4).Find(&rows).Error; err != nil {
			return nil, "", err
		}
		if len(rows) == 0 {
			break
		}

		for _, file := range rows {
			lastSeenID = file.ID
			if !fileBaseNameContainsKeyword(file.FileName, keyword) {
				continue
			}
			files = append(files, file)
			if len(files) == size+1 {
				nextCursor = strconv.FormatInt(files[size-1].ID, 10)
				files = files[:size]
				return files, nextCursor, nil
			}
		}
	}

	return files, nextCursor, nil
}
```

```go
// model/dao/indexer_file_dao.go
func (dao *IndexerFileDAO) GetByKeywordAndExtensionWithCursor(keyword string, extension string, cursor string, size int) ([]*model.IndexerFile, string, error) {
	return dao.db.GetIndexerFilesByKeywordAndExtensionWithCursor(keyword, extension, cursor, size)
}
```

```go
// service/indexer_service/indexer_file_service.go
func (s *IndexerFileService) ListFilesByKeywordAndExtension(keyword string, extension string, cursor string, size int) ([]*model.IndexerFile, string, bool, error) {
	if size < 1 || size > 100 {
		size = 20
	}
	files, nextCursor, err := s.indexerFileDAO.GetByKeywordAndExtensionWithCursor(keyword, extension, cursor, size)
	if err != nil {
		return nil, "", false, fmt.Errorf("failed to list files by keyword and extension: %w", err)
	}
	return files, nextCursor, nextCursor != "", nil
}
```

```go
// controller/handler/indexer_query.go
func (h *IndexerQueryHandler) GetFilesByKeywordAndExtension(c *gin.Context) {
	keyword := strings.TrimSpace(c.Param("keyword"))
	if keyword == "" {
		respond.InvalidParam(c, "keyword is required")
		return
	}

	extensions := parseExtensionsQuery(c)
	if len(extensions) == 0 {
		respond.InvalidParam(c, "extension is required (query, supports: extension=.jpg&extension=.png or extension=.jpg,.png)")
		return
	}

	timestamp := c.DefaultQuery("timestamp", "")
	sizeStr := c.DefaultQuery("size", "20")
	size, _ := strconv.Atoi(sizeStr)
	if size < 1 || size > 100 {
		size = 20
	}

	var files []*model.IndexerFile
	var nextTimestamp string
	var hasMore bool
	if len(extensions) == 1 {
		list, nextCursor, more, err := h.indexerFileService.ListFilesByKeywordAndExtension(keyword, extensions[0], timestamp, size)
		if err != nil {
			respond.ServerError(c, err.Error())
			return
		}
		files = list
		hasMore = more
		nextTimestamp = extractTimestamp16FromKey(nextCursor)
	} else {
		fetchSize := size * len(extensions)
		if fetchSize > 500 {
			fetchSize = 500
		}
		var filesByExt [][]*model.IndexerFile
		for _, ext := range extensions {
			list, _, _, err := h.indexerFileService.ListFilesByKeywordAndExtension(keyword, ext, timestamp, fetchSize)
			if err != nil {
				respond.ServerError(c, err.Error())
				return
			}
			filesByExt = append(filesByExt, list)
		}
		files, nextTimestamp, hasMore = mergeFilesByExtension(filesByExt, size)
	}

	respond.Success(c, respond.ToIndexerFileListByExtensionResponse(files, nextTimestamp, hasMore, h.indexerFileService, getIndexerBaseUrl()))
}
```

```go
// controller/indexer_router.go
files.GET("/keyword/:keyword/extension", indexerQueryHandler.GetFilesByKeywordAndExtension)
```

- [ ] **Step 4: Run the targeted tests and compile checks**

Run: `go test ./controller/handler ./database ./model/dao ./service/indexer_service -run 'TestGetFilesByKeywordAndExtensionValidation|TestExtractFileBaseName|TestFileBaseNameContainsKeyword|TestExtractTimestamp16FromCursorKey|TestPebbleGetIndexerFilesByKeywordAndExtensionWithCursor' -v`

Expected: PASS for the new tests and clean package compilation for DAO and service

- [ ] **Step 5: Commit the HTTP and compatibility wiring**

```bash
git add database/mysql_adapter.go model/dao/indexer_file_dao.go service/indexer_service/indexer_file_service.go controller/handler/indexer_query.go controller/indexer_router.go controller/handler/indexer_query_test.go
git commit -m "feat: expose file keyword extension search"
```

## Task 4: Regenerate Swagger Docs and Run Final Verification

**Files:**

- Modify: `controller/handler/indexer_query.go`
- Modify: `docs/indexer/indexer_docs.go`
- Modify: `docs/indexer/indexer_swagger.json`
- Modify: `docs/indexer/indexer_swagger.yaml`

- [ ] **Step 1: Add the Swagger annotation block for the new handler**

```go
// GetFilesByKeywordAndExtension get file list by file base-name keyword and file extension (global), reverse time order; extension as query (array supported)
// @Summary      Get files by keyword and extension
// @Description  Query file list by base-name keyword and file extension(s), reverse time order; extension can be repeated. Paginate with timestamp (16-digit).
// @Tags         Indexer File Query
// @Accept       json
// @Produce      json
// @Param        keyword    path     string    true   "Base-name keyword (case-insensitive contains match)"
// @Param        extension  query    []string  true   "File extension(s), supports multi (extension=.jpg&extension=.png) and csv (extension=.jpg,.png)"
// @Param        timestamp  query    string    false  "Next page: cursor value from previous response next_timestamp"
// @Param        size       query    int       false  "Page size" default(20)
// @Success      200        {object}  respond.Response{data=respond.IndexerFileListByExtensionResponse}
// @Failure      500        {object}  respond.Response
// @Router       /files/keyword/{keyword}/extension [get]
```

- [ ] **Step 2: Regenerate the indexer Swagger docs**

Run: `make swagger-indexer`

Expected: `Indexer Swagger docs generated at docs/indexer/`

- [ ] **Step 3: Verify the generated docs include the new path**

Run: `rg -n '"/files/keyword/\\{keyword\\}/extension"|/files/keyword/{keyword}/extension' docs/indexer/indexer_docs.go docs/indexer/indexer_swagger.json docs/indexer/indexer_swagger.yaml`

Expected: hits in all generated indexer Swagger artifacts

- [ ] **Step 4: Run final verification**

Run: `go test ./...`

Expected: PASS for the full repository test suite

- [ ] **Step 5: Commit the docs and verification pass**

```bash
git add controller/handler/indexer_query.go docs/indexer/indexer_docs.go docs/indexer/indexer_swagger.json docs/indexer/indexer_swagger.yaml
git commit -m "docs: add keyword extension search api"
```

## Implementation Notes

- Keep the new base-name helper in `database/` so both adapters share the exact same matching semantics
- Do not change the behavior of the existing `/files/extension` and `/files/metaid/{metaidOrGlobalMetaId}/extension` endpoints
- In the Pebble implementation, derive `nextCursor` from the underlying matched index key, not from `file.Timestamp`
- The MySQL path is compatibility coverage; prioritize correct semantics and clean compilation over aggressive optimization
- Keep multi-extension merging consistent with the current `mergeFilesByExtension` helper

## Verification Checklist

- New endpoint returns the same response shape as the extension endpoint
- Blank keywords fail fast with a parameter error
- Missing extensions fail fast with a parameter error
- `周杰伦-夜曲.mp3` matches keyword `周杰伦`
- `jay.live.2004.mp3` matches keyword `live`
- Pebble second-page queries do not skip older matches when non-matching records exist between matches
- Swagger docs show the new path and parameters
