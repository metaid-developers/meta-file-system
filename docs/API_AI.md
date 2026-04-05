# Meta File System – AI‑Friendly API Manual

This document is designed for AI agents to call the Meta File System APIs directly. It is concise, explicit about request/response shapes, and highlights special behaviors and limitations.

## Overview

There are two services:

- **Uploader Service**: accepts files, builds/broadcasts transactions, manages chunked upload tasks, and multipart storage uploads.
- **Indexer Service**: queries indexed files/PINs/users and serves content.

### Base URLs

- `UPLOADER_BASE = https://<host>:<uploader_port>`
- `INDEXER_BASE = https://<host>:<indexer_port>`

The API base path for both services is:

- `BASE_PATH = /api/v1`

Indexer also exposes MetaID‑compatible routes under `/api/info/*` and legacy avatar content under `/content/:pinId` and `/thumbnail/:pinId`.

### Common Response Envelope

Most JSON responses are wrapped in the following envelope:

```json
{
  "code": 0,
  "message": "success",
  "processingTime": 123,
  "data": { }
}
```

- `code = 0` success
- `code = 40000` invalid parameters
- `code = 40400` not found
- `code = 50000` server error

Exceptions:

- `GET /api/v1/info/search` returns a **raw JSON array**, not the envelope.
- `GET /api/v1/info/*` uses `code = 1` on success (MetaID‑compatible).
- Binary content endpoints return bytes, not JSON.
- “accelerate” endpoints return **307 Redirect** to OSS URLs.

### Gzip Support (JSON requests)

These endpoints accept JSON bodies compressed with gzip if you set header:

- `Content-Encoding: gzip`

Supported endpoints:

- `POST /api/v1/files/estimate-chunked-upload`
- `POST /api/v1/files/chunked-upload`
- `POST /api/v1/files/chunked-upload-task`
- `POST /api/v1/files/multipart/upload-part`

### Chain & Storage Notes

- `chain` currently supports `mvc` and `doge` for chunked uploads. Default is `mvc`.
- “accelerate” endpoints only work when the file is stored in OSS and an OSS domain is configured.
- Indexer can use **Pebble** or **MySQL** as its DB. Some features are **not implemented** in MySQL (see “Limitations”).
- Multipart uploads store file data in the configured storage backend and return a `storageKey` that can be used by chunked upload endpoints.

---

# Uploader Service API (`UPLOADER_BASE`)

## 1) Pre‑Upload (build unsigned tx)

`POST /api/v1/files/pre-upload`

**Content‑Type:** `multipart/form-data`

**Form Fields:**

| Field | Type | Required | Notes |
|---|---|---|---|
| file | file | Yes | File content |
| path | string | Yes | MetaID path |
| operation | string | No | `create` (default) or `update` |
| contentType | string | No | MIME type; defaults to file Content‑Type |
| changeAddress | string | No | Change address |
| metaId | string | No | MetaID |
| address | string | No | User address |
| feeRate | int | No | Fee rate |
| outputs | string | No | JSON list of `{address,amount}` |
| otherOutputs | string | No | JSON list of `{address,amount}` |

**Response `data`:**

```json
{
  "fileId": "metaid_xxx",
  "fileMd5": "...",
  "filehash": "...",
  "txId": "...",
  "pinId": "...",
  "preTxRaw": "010000...",
  "status": "pending",
  "message": "success",
  "calTxFee": 1000,
  "calTxSize": 500
}
```

## 2) Commit Upload (broadcast signed tx)

`POST /api/v1/files/commit-upload`

**Content‑Type:** `application/json`

**Body:**

```json
{
  "fileId": "metaid_xxx",
  "signedRawTx": "010000..."
}
```

**Response `data`:**

```json
{
  "fileId": "...",
  "status": "success",
  "txId": "...",
  "pinId": "...",
  "message": "success"
}
```

## 3) Direct Upload (one‑step)

`POST /api/v1/files/direct-upload`

**Content‑Type:** `multipart/form-data`

**Form Fields:**

| Field | Type | Required | Notes |
|---|---|---|---|
| file | file | Yes | File content |
| path | string | Yes | MetaID path |
| preTxHex | string | Yes | Pre‑transaction hex (signed) |
| mergeTxHex | string | No | Merge tx hex (optional) |
| operation | string | No | `create` default |
| contentType | string | No | MIME type |
| metaId | string | No | MetaID |
| address | string | No | User address |
| changeAddress | string | No | Defaults to address |
| feeRate | int | No | Fee rate |
| totalInputAmount | int | No | Used to compute change |

**Response `data`:** same shape as Commit Upload.

## 4) Estimate Chunked Upload Fees

`POST /api/v1/files/estimate-chunked-upload`

**Content‑Type:** `application/json` (gzip supported)

**Body:**

```json
{
  "fileName": "example.jpg",
  "content": "<base64>",
  "storageKey": "uploads/...",
  "path": "/file",
  "contentType": "image/jpeg",
  "chain": "mvc",
  "feeRate": 1
}
```

Rules:

- Provide either `content` (base64) **or** `storageKey`.

**Response `data`:**

```json
{
  "chain": "mvc",
  "chunkNumber": 10,
  "chunkSize": 2048000,
  "chunkFees": [123, 124],
  "chunkPerTxFee": 600,
  "chunkPreTxFee": 12000,
  "indexPreTxFee": 800,
  "totalFee": 12800,
  "perChunkFee": 1200,
  "message": "success"
}
```

## 5) Chunked Upload (build txs)

`POST /api/v1/files/chunked-upload`

**Content‑Type:** `application/json` (gzip supported)

**Body:**

```json
{
  "metaId": "metaid_xxx",
  "address": "1A1z...",
  "fileName": "example.jpg",
  "content": "<base64>",
  "storageKey": "uploads/...",
  "path": "/file",
  "operation": "create",
  "contentType": "image/jpeg",
  "chain": "mvc",
  "chunkPreTxHex": "010000...",
  "indexPreTxHex": "010000...",
  "mergeTxHex": "010000...",
  "feeRate": 1,
  "isBroadcast": false
}
```

Rules:

- Provide either `content` **or** `storageKey`.
- `chunkPreTxHex` and `indexPreTxHex` are required.
- `chain = mvc` by default.

**Response `data`:** (MVC example)

```json
{
  "fileId": "metaid_xxx",
  "fileHash": "...",
  "fileMd5": "...",
  "chunkNumber": 10,
  "chunkFundingTx": "010000...",
  "chunkTxs": ["010000..."],
  "chunkTxIds": ["..."] ,
  "indexTx": "010000...",
  "indexTxId": "...",
  "status": "pending",
  "message": "success"
}
```

If `isBroadcast=true`, response may omit raw txs and return `status=success` or `failed`.

## 6) Chunked Upload Task (async)

`POST /api/v1/files/chunked-upload-task`

Same body as chunked upload but returns a task ID.

**Response `data`:**

```json
{
  "taskId": "task_123",
  "status": "pending",
  "message": "task created"
}
```

## 7) Query Task Progress

`GET /api/v1/files/task/:taskId`

**Response `data`:**

```json
{
  "task": {
    "taskId": "task_123",
    "metaId": "...",
    "address": "...",
    "fileName": "...",
    "fileHash": "...",
    "fileMd5": "...",
    "fileSize": 123,
    "contentType": "image/jpeg",
    "path": "/file",
    "operation": "create",
    "status": "pending",
    "progress": 70,
    "totalChunks": 10,
    "processedChunks": 7,
    "currentStep": "...",
    "stage": "...",
    "fileId": "...",
    "chunkFundingTx": "...",
    "chunkTxIds": ["..."],
    "indexTxId": "...",
    "errorMessage": "",
    "createdAt": "...",
    "updatedAt": "...",
    "startedAt": "...",
    "finishedAt": "..."
  }
}
```

## 8) List Upload Tasks

`GET /api/v1/files/tasks?address=<address>&cursor=0&size=20`

**Response `data`:**

```json
{
  "tasks": [ ... ],
  "nextCursor": 123,
  "hasMore": true
}
```

## 9) Multipart Upload – Initiate

`POST /api/v1/files/multipart/initiate`

**Body:**

```json
{
  "fileName": "example.jpg",
  "fileSize": 12345,
  "metaId": "metaid_xxx",
  "address": "1A1z..."
}
```

**Response `data`:**

```json
{ "uploadId": "...", "key": "uploads/..." }
```

## 10) Multipart Upload – Upload Part

`POST /api/v1/files/multipart/upload-part`

**Body:**

```json
{
  "uploadId": "...",
  "key": "uploads/...",
  "partNumber": 1,
  "content": "<base64>"
}
```

**Response `data`:**

```json
{ "etag": "...", "partNumber": 1 }
```

## 11) Multipart Upload – Complete

`POST /api/v1/files/multipart/complete`

**Body:**

```json
{
  "uploadId": "...",
  "key": "uploads/...",
  "parts": [
    { "partNumber": 1, "etag": "...", "size": 5242880 }
  ]
}
```

**Response `data`:**

```json
{ "key": "uploads/...", "uploadId": "...", "fileSize": 12345 }
```

## 12) Multipart Upload – List Parts

`POST /api/v1/files/multipart/list-parts`

**Body:**

```json
{ "uploadId": "...", "key": "uploads/..." }
```

**Response `data`:**

```json
{ "uploadId": "...", "parts": [ { "partNumber": 1, "etag": "...", "size": 5242880 } ] }
```

## 13) Multipart Upload – Abort

`POST /api/v1/files/multipart/abort`

**Body:**

```json
{ "uploadId": "...", "key": "uploads/..." }
```

**Response `data`:**

```json
{ "message": "Upload aborted successfully" }
```

## 14) Get Config

`GET /api/v1/config`

**Response `data`:**

```json
{
  "maxFileSize": 10485760,
  "swaggerBaseUrl": "host:port",
  "chains": {
    "mvc": { "maxFileSize": 10485760, "chunkSize": 2048000, "feeRate": 1 },
    "doge": { "maxFileSize": 5242880, "chunkSize": 1200, "feeRate": 1000 }
  }
}
```

## 15) Health

`GET /health`

**Response:**

```json
{ "status": "ok", "service": "uploader" }
```

---

# Indexer Service API (`INDEXER_BASE`)

## 1) Files – List

`GET /api/v1/files?cursor=0&size=20`

**Response `data`:**

```json
{
  "files": [ ... ],
  "next_cursor": 100,
  "has_more": true
}
```

## 2) Files – Get By PinID

`GET /api/v1/files/:pinId`

**Response `data`:**

```json
{
  "pin_id": "...",
  "tx_id": "...",
  "path": "/file/...",
  "operation": "create",
  "content_type": "image/jpeg",
  "file_type": "image",
  "file_extension": ".jpg",
  "file_name": "...",
  "file_size": 102400,
  "file_md5": "...",
  "file_hash": "...",
  "storage_path": "indexer/...",
  "chain_name": "mvc",
  "block_height": 12345,
  "timestamp": 1699999999,
  "creator_meta_id": "...",
  "creator_address": "...",
  "creator_global_meta_id": "...",
  "owner_meta_id": "...",
  "owner_address": "...",
  "content_url": "https://.../api/v1/files/content/<pinId>",
  "accelerate_content_url": "https://.../api/v1/files/accelerate/content/<pinId>"
}
```

## 3) Files – Content By PinID (binary)

`GET /api/v1/files/content/:pinId`

**Response:** bytes with `Content-Type` set. No JSON envelope.

## 4) Files – Accelerate Content (OSS redirect)

`GET /api/v1/files/accelerate/content/:pinId?process=preview|thumbnail|video`

**Response:** `307 Redirect` to OSS URL.

## 5) Files – Latest By FirstPinID

`GET /api/v1/files/latest/:firstPinId`

Same response shape as “Get By PinID”.

## 6) Files – Latest Content By FirstPinID

`GET /api/v1/files/content/latest/:firstPinId` (binary)

## 7) Files – Latest Accelerate Content

`GET /api/v1/files/accelerate/content/latest/:firstPinId?process=...`

## 8) Files – By Creator Address

`GET /api/v1/files/creator/:address?cursor=0&size=20`

## 9) Files – By Creator MetaID or GlobalMetaID

`GET /api/v1/files/metaid/:metaidOrGlobalMetaId?cursor=0&size=20`

## 10) Files – By Extension (global)

`GET /api/v1/files/extension?extension=.jpg&extension=.png&timestamp=<16-digit>&size=20`

Supports `extension=.jpg,.png` as CSV.

**Response `data`:**

```json
{ "files": [ ... ], "next_timestamp": "1699123456082917", "has_more": true }
```

## 11) Files – By GlobalMetaID + Extension

`GET /api/v1/files/metaid/:metaidOrGlobalMetaId/extension?extension=.jpg&timestamp=<16-digit>&size=20`

## 12) Users – List

`GET /api/v1/users?cursor=0&size=20`

## 13) Users – By MetaID

`GET /api/v1/users/metaid/:metaId`

## 14) Users – By Address

`GET /api/v1/users/address/:address`

## 15) Users – Avatar By MetaID (binary or redirect)

`GET /api/v1/users/metaid/:metaId/avatar`

- If avatar is OSS URL, returns **307 Redirect**.
- Otherwise returns binary content.

## 16) Users – Avatar Content By PinID (binary)

`GET /api/v1/users/avatar/content/:pinId`

## 17) Users – Avatar Accelerate

`GET /api/v1/users/avatar/accelerate/:pinId?process=preview|thumbnail`

Returns **307 Redirect**.

## 18) Users – History By Key

`GET /api/v1/users/history/:key`

## 19) Pins – By PinID

`GET /api/v1/pins/:pinId`

## 20) Indexer Status

`GET /api/v1/status`

**Response `data`:**

```json
{ "chains": [ { "chain_name": "mvc", "current_sync_height": 123, "latest_block_height": 124 } ] }
```

## 21) Indexer Stats

`GET /api/v1/stats`

**Response `data`:**

```json
{ "total_files": 12345, "chain_stats": { "mvc": 10000, "doge": 2345 } }
```

## 22) MetaID Info – MetaID Format

`GET /api/v1/info/metaid/:metaidOrGlobalMetaId`

`GET /api/v1/info/address/:address`

`GET /api/v1/info/globalmetaid/:globalMetaID`

**Response:** envelope with `code = 1` and `data`:

```json
{
  "globalMetaId": "...",
  "metaid": "...",
  "name": "...",
  "nameId": "...",
  "address": "...",
  "avatar": "/content/<pinId>",
  "avatarId": "...",
  "bio": { "text": "...", "links": [] },
  "chatpubkey": "...",
  "chatpubkeyId": "..."
}
```

## 23) MetaID Info – Search

`GET /api/v1/info/search?keyword=<kw>&keytype=metaid|name&limit=10`

**Response:** raw JSON array of `MetaIDUserInfo` (no envelope).

## 24) Thumbnail (Avatar)

`GET /api/v1/thumbnail/:pinId`

Returns **307 Redirect**.

## 25) Admin – Rescan

`POST /api/v1/admin/rescan`

**Body:**

```json
{ "chain": "mvc", "start_height": 100000, "end_height": 100100 }
```

## 26) Admin – Rescan Status

`GET /api/v1/admin/rescan/status`

## 27) Admin – Stop Rescan

`POST /api/v1/admin/rescan/stop`

## 28) Legacy & Compatibility Routes

- `GET /api/info/*` mirrors `/api/v1/info/*`.
- `GET /content/:pinId` and `GET /thumbnail/:pinId` are legacy root paths.

## 29) Health

`GET /health`

```json
{ "status": "ok", "service": "indexer" }
```

---

# Known Limitations

## Indexer + MySQL

When `database.indexer_type = mysql`, the following are **not implemented** and may return errors:

- Latest file by firstPinID and its content routes.
- User info, avatar info, chat public key info, and history routes.
- PIN info route.
- GlobalMetaID mapping routes.
- MetaID search & user list (depends on MetaID timestamp index and Redis cache).

To unlock full Indexer capabilities, set `database.indexer_type = pebble`.

## OSS “Accelerate” Endpoints

- Require OSS storage and a configured OSS domain.
- If a file is not stored in OSS, accelerate endpoints will error.

## Redis‑Backed Search

- `GET /api/v1/info/search` depends on Redis cache. If Redis is disabled or cache not built, it will fail.

---

# Suggested AI Calling Strategy

1. Use `/api/v1/config` on uploader to discover limits (`maxFileSize`, `chunkSize`, `feeRate`).
2. For large files, use **multipart upload** to storage, then pass `storageKey` into `estimate-chunked-upload` and `chunked-upload`.
3. For indexer queries, prefer `/api/v1/files` and `/api/v1/files/:pinId` first; check for 404/500 and fall back if using MySQL indexer.
4. For binary content, call `/api/v1/files/content/:pinId` and treat response as raw bytes.
5. For accelerated content, call `/api/v1/files/accelerate/content/:pinId` and follow the 307 redirect.
