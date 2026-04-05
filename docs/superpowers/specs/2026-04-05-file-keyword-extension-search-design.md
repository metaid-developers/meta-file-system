# File Keyword + Extension Search Design

Date: 2026-04-05
Status: Draft for review

## Summary

Add a new public API:

- `GET /files/keyword/{keyword}/extension`

This API returns indexed metafiles whose:

- file extension matches the requested `extension` query parameter(s)
- file base name contains the requested `keyword`

The "file base name" is the stored `file_name` with only the last extension removed. For example:

- `周杰伦-夜曲.mp3` -> `周杰伦-夜曲`
- `jay.live.2004.mp3` -> `jay.live.2004`

This feature is intentionally implemented as an MVP on top of the existing extension index. It does not introduce a new search index, schema migration, or data backfill.

## Goals

- Preserve the current file query API style
- Reuse the current extension query response structure and pagination model
- Support searching by file base name keyword within one or more extensions
- Keep implementation small and safe for the current early-stage data volume

## Non-Goals

- Searching within a specific MetaID or GlobalMetaID scope
- General-purpose file name search across all files without extension filtering
- Full-text, fuzzy, phonetic, or relevance-ranked search
- New Pebble collections or MySQL schema/index changes
- Historical data migration or reindexing

## API Contract

### Route

- `GET /files/keyword/{keyword}/extension`

### Path Parameters

- `keyword`: required search keyword

### Query Parameters

- `extension`: required; supports the existing formats
  - repeated query: `extension=.mp3&extension=.wav`
  - csv query: `extension=.mp3,.wav`
- `timestamp`: optional; pagination cursor carrier, following the current extension API behavior and field naming
- `size`: optional; page size, same validation as the existing extension API

### Response

Reuse the existing `IndexerFileListByExtensionResponse`.

### Cursor Contract

This feature follows the current extension API contract rather than introducing a new cursor field name.

- request field name remains `timestamp`
- response field name remains `next_timestamp`
- the value is treated as an opaque pagination cursor carrier by the adapter path that produced it

For Pebble, this is the extracted `timestamp16` component from the underlying extension index key. For MySQL, the adapter may continue using its existing cursor style internally while still flowing through the same public field.

### Validation Rules

- `keyword` is required
- blank or whitespace-only `keyword` is invalid
- `extension` is required
- `size` keeps the current bounds and fallback behavior

### Scope Rules

- This API is global only
- It is not combined with MetaID or GlobalMetaID filtering

## Matching Semantics

For each indexed file candidate:

1. Match the normalized extension first
2. Read `file_name`
3. If `file_name` is empty, skip the record
4. Compute base name by removing only the last extension from `file_name`
5. Compare `strings.ToLower(baseName)` with `strings.ToLower(keyword)`
6. Record matches when `baseName` contains `keyword`

Examples:

- `周杰伦-夜曲.mp3` with keyword `周杰伦` matches
- `jay.live.2004.mp3` with keyword `live` matches
- `周杰伦` with no suffix keeps base name `周杰伦`
- `archive.tar.gz` with keyword `archive.tar` matches because only `.gz` is removed

## Architecture

### Routing

Add a new route in the existing files query group:

- `GET /keyword/:keyword/extension`

### Handler Layer

Add a new handler method:

- parse `keyword` from path
- parse `extension`, `timestamp`, and `size` with the same rules as the current extension handlers
- call a new service method
- return the existing extension-list response structure

### Service Layer

Add a new service method for keyword + extension search. The service remains responsible for:

- parameter normalization
- page size validation
- `hasMore` calculation
- merging multi-extension results using the current merge strategy

The service must not implement keyword filtering by repeatedly calling the existing extension API methods, because that would not preserve correct Pebble cursor semantics.

### DAO / Database Layer

Add dedicated methods for keyword + extension search:

- DAO method wrapping the database call
- database interface method
- Pebble implementation
- MySQL implementation

The keyword filtering logic belongs in the database adapter layer for this feature because the adapter owns the real cursor format.

## Pebble Design

### Existing Index Reuse

Reuse the current extension timestamp collection:

- `file_extension_timestamp`
- key format: `{extension}:{timestamp16}`

No new collection is added.

### Why Filtering Must Happen in Pebble Adapter

The current extension index uses a 16-digit timestamp key:

- first 10 digits: unix seconds
- last 6 digits: random suffix for uniqueness

The public API exposes only `next_timestamp`, not the full internal key. If keyword filtering were done above the Pebble adapter, the implementation could not precisely resume after the last returned match. That would make pagination unreliable.

### Pebble Pagination Algorithm

For each requested extension:

1. Normalize the extension
2. Seek using the existing extension range and incoming cursor
3. Iterate in reverse chronological order
4. For each record, apply the base-name keyword match
5. Keep scanning until either:
   - `size + 1` matching records are found, or
   - the iterator is exhausted
6. Return the first `size` matching records
7. If an extra matching record exists, set `hasMore = true`
8. Set `nextCursor` from the underlying extension index key of the last returned matching record

This ensures:

- no matching records are skipped
- pagination stays stable for matching records
- non-matching records may be rescanned on later pages, which is acceptable for this MVP

## MySQL Design

MySQL should expose the same method and the same matching semantics as Pebble.

For MVP:

- query by normalized `file_extension`
- order by `timestamp DESC, id DESC`
- scan batches in application code
- apply the same base-name keyword filter in Go
- use the last returned matching row id as the adapter cursor

The implementation should not use SQL `LIKE` in this MVP. The goal is to keep Pebble and MySQL behavior aligned and avoid diverging semantics between backends.

This project currently deploys the indexer on Pebble by configuration, so MySQL support here is compatibility coverage, not the primary runtime path for this feature.

## Multi-Extension Behavior

The new API should preserve the existing multi-extension style.

Recommended behavior:

- single extension: use the dedicated keyword + extension query path directly
- multiple extensions: query each extension independently with the same keyword
- use the same fetch sizing strategy as the current multi-extension endpoint: `fetchSize = size * len(extensions)`, capped at `500`
- merge using the existing `mergeFilesByExtension` behavior

This keeps the implementation consistent with the current `/files/extension` endpoint and avoids introducing a new global merge cursor design.

## Error Handling

- missing `keyword` -> invalid parameter response
- blank `keyword` after trimming -> invalid parameter response
- missing `extension` -> invalid parameter response
- adapter errors -> existing server error response path

## Risks

### Acceptable MVP Risks

- query cost grows with the number of files in the requested extension range
- low-match keywords may require scanning many non-matching records
- empty or poor-quality `file_name` values cannot be matched

### Deferred Risks

- if one extension grows large, response latency may degrade
- future requests for fuzzy matching, pinyin matching, or ranking will not fit this design well

## Why This Is Acceptable Now

- current data volume is expected to be small enough
- feature value is high relative to implementation size
- no schema change or data migration is required
- this can ship quickly and validate real usage before building a true search index

## Testing Plan

### Unit Tests

- base-name extraction
- case-insensitive keyword match
- last-extension-only removal
- empty `file_name` skip behavior

### Adapter Tests

- Pebble keyword search returns only matching records
- Pebble pagination does not skip matching records across pages
- Pebble handles many non-matching records between matches
- MySQL adapter follows the same matching semantics

### Handler Tests

- missing `keyword`
- blank `keyword`
- missing `extension`
- invalid `size` fallback behavior
- multi-extension request shape

## Implementation Notes

- Reuse the current response type and existing extension parsing helpers
- Keep helper logic for base-name extraction local and explicit
- Avoid changing the existing `/files/extension` behavior
- Do not expand the scope to MetaID-specific keyword search in this change

## Open Questions

None. The current scope and semantics are already confirmed:

- keyword applies to file base name
- extension remains mandatory
- no MetaID + keyword combined search
