package handler

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"strconv"
	"strings"

	"meta-file-system/common"
	"meta-file-system/conf"
	"meta-file-system/controller/respond"
	"meta-file-system/service/upload_service"
	"meta-file-system/storage"

	"github.com/gin-gonic/gin"
)

// UploadHandler upload handler
type UploadHandler struct {
	uploadService *upload_service.UploadService
}

// NewUploadHandler create upload handler instance
func NewUploadHandler(uploadService *upload_service.UploadService) *UploadHandler {
	return &UploadHandler{
		uploadService: uploadService,
	}
}

// bindJSONWithOptionalGzip handles JSON payloads that may be gzip-compressed.
// If the request header specifies gzip encoding, the body is decompressed before binding.
func bindJSONWithOptionalGzip(c *gin.Context, obj interface{}) error {
	encoding := strings.ToLower(strings.TrimSpace(c.GetHeader("Content-Encoding")))
	if encoding == "gzip" || strings.Contains(encoding, "gzip") {
		defer c.Request.Body.Close()

		gzipReader, err := gzip.NewReader(c.Request.Body)
		if err != nil {
			return err
		}
		defer gzipReader.Close()

		bodyBytes, err := io.ReadAll(gzipReader)
		if err != nil {
			return err
		}

		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		c.Request.ContentLength = int64(len(bodyBytes))
		c.Request.Header.Del("Content-Encoding")
	}

	return c.ShouldBindJSON(obj)
}

// UploadFileRequest upload file request
type UploadFileRequest struct {
	Path          string `json:"path" binding:"required"`
	Operation     string `json:"operation"`
	ContentType   string `json:"content_type"`
	ChangeAddress string `json:"change_address" binding:"required"`
	// Inputs        []*TxInputUtxoRequest `json:"inputs" binding:"required"`
	Outputs      []*TxOutputRequest `json:"outputs"`
	OtherOutputs []*TxOutputRequest `json:"other_outputs"`
	FeeRate      int64              `json:"fee_rate"`
}

// TxInputUtxoRequest UTXO input request
type TxInputUtxoRequest struct {
	TxID     string `json:"txId" binding:"required"`
	TxIndex  int64  `json:"txIndex" binding:"required"`
	PkScript string `json:"pkScript" binding:"required"`
	Amount   uint64 `json:"amount" binding:"required"`
	PriHex   string `json:"priHex" binding:"required"`
	SignMode string `json:"signMode"`
}

// TxOutputRequest transaction output request
type TxOutputRequest struct {
	Address string `json:"address" binding:"required"`
	Amount  int64  `json:"amount" binding:"required"`
}

// PreUploadResponseData pre-upload response data
type PreUploadResponseData struct {
	FileId    string `json:"fileId" example:"metaid_abc123" description:"File ID (unique identifier)"`
	FileMd5   string `json:"fileMd5" example:"5d41402abc4b2a76b9719d911017c592" description:"File md5"`
	Filehash  string `json:"filehash" example:"2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae" description:"File sha256 hash"`
	TxId      string `json:"txId" example:"abc123..." description:"Transaction ID"`
	PinId     string `json:"pinId" example:"abc123...i0" description:"Pin ID"`
	PreTxRaw  string `json:"preTxRaw" example:"0100000..." description:"Pre-transaction raw data (hex)"`
	Status    string `json:"status" example:"pending" description:"Status: pending, success, failed"`
	Message   string `json:"message" example:"success" description:"Message"`
	CalTxFee  int64  `json:"calTxFee" example:"1000" description:"Calculated transaction fee (satoshis)"`
	CalTxSize int64  `json:"calTxSize" example:"500" description:"Calculated transaction size (bytes)"`
}

// PreUpload pre-upload file
// @Summary      Pre-upload file
// @Description  Upload file and generate unsigned transaction, return transaction for client signing
// @Tags         File Upload
// @Accept       multipart/form-data
// @Produce      json
// @Param        file           formData  file    true   "File to upload"
// @Param        path           formData  string  true   "File path"
// @Param        operation      formData  string  false  "Operation type"        default(create)
// @Param        contentType    formData  string  false  "Content type"
// @Param        changeAddress  formData  string  false  "Change address"
// @Param        metaId         formData  string  false  "MetaID"
// @Param        address        formData  string  false  "Address"
// @Param        feeRate        formData  int     false  "Fee rate"           default(1)
// @Param        outputs        formData  string  false  "Output list json"
// @Param        otherOutputs   formData  string  false  "Other output list json"
// @Success      200  {object}  respond.Response{data=PreUploadResponseData}  "Pre-upload successful, return transaction and file info"
// @Failure      400  {object}  respond.Response  "Parameter error"
// @Failure      500  {object}  respond.Response  "Server error"
// @Router       /files/pre-upload [post]
func (h *UploadHandler) PreUpload(c *gin.Context) {
	// Read file content
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		respond.InvalidParam(c, "file is required")
		return
	}
	defer file.Close()

	content, err := ioutil.ReadAll(file)
	if err != nil {
		respond.ServerError(c, "failed to read file")
		return
	}

	// Get other parameters
	path := c.PostForm("path")
	if path == "" {
		respond.InvalidParam(c, "path is required")
		return
	}

	operation := c.PostForm("operation")
	if operation == "" {
		operation = "create"
	}

	contentType := c.PostForm("contentType")
	if contentType == "" {
		contentType = header.Header.Get("Content-Type")
	}

	changeAddress := c.PostForm("changeAddress")
	// if changeAddress == "" {
	// 	respond.InvalidParam(c, "changeAddress is required")
	// 	return
	// }

	feeRateStr := c.PostForm("feeRate")
	feeRate := int64(1)
	if feeRateStr != "" {
		if rate, err := strconv.ParseInt(feeRateStr, 10, 64); err == nil {
			feeRate = rate
		}
	}

	// Get additional form parameters
	metaId := c.PostForm("metaId")
	address := c.PostForm("address")

	// Parse outputs and otherOutputs
	var outputs []*common.TxOutput
	var otherOutputs []*common.TxOutput

	outputsStr := c.PostForm("outputs")
	if outputsStr != "" && outputsStr != "[]" {
		var outputsReq []*TxOutputRequest
		if err := json.Unmarshal([]byte(outputsStr), &outputsReq); err == nil {
			for _, out := range outputsReq {
				outputs = append(outputs, &common.TxOutput{
					Address: out.Address,
					Amount:  out.Amount,
				})
			}
		}
	}

	otherOutputsStr := c.PostForm("otherOutputs")
	if otherOutputsStr != "" && otherOutputsStr != "[]" {
		var otherOutputsReq []*TxOutputRequest
		if err := json.Unmarshal([]byte(otherOutputsStr), &otherOutputsReq); err == nil {
			for _, out := range otherOutputsReq {
				otherOutputs = append(otherOutputs, &common.TxOutput{
					Address: out.Address,
					Amount:  out.Amount,
				})
			}
		}
	}

	// Build upload request
	req := &upload_service.UploadRequest{
		MetaId:        metaId,
		Address:       address,
		FileName:      header.Filename,
		Content:       content,
		Path:          path,
		Operation:     operation,
		ContentType:   contentType,
		ChangeAddress: changeAddress,
		Outputs:       outputs,
		OtherOutputs:  otherOutputs,
		FeeRate:       feeRate,
	}

	// Upload file
	resp, err := h.uploadService.PreUpload(req)
	if err != nil {
		respond.ServerError(c, err.Error())
		return
	}

	respond.Success(c, resp)
}

// DirectUpload direct upload file with existing PreTxHex (one-step upload)
// @Summary      Direct upload file (one-step)
// @Description  Upload file and add MetaID OP_RETURN output to existing PreTxHex, then broadcast immediately. This is a one-step upload process that combines building and broadcasting. Supports UTXO merge transaction for SIGHASH_SINGLE compatibility.
// @Tags         File Upload
// @Accept       multipart/form-data
// @Produce      json
// @Param        file             formData  file    true   "File to upload"
// @Param        path             formData  string  true   "File path"
// @Param        preTxHex         formData  string  true   "Pre-transaction hex (signed, with inputs and outputs)"
// @Param        mergeTxHex       formData  string  false  "Merge transaction hex (optional, broadcasted before main transaction)"
// @Param        operation        formData  string  false  "Operation type"        default(create)
// @Param        contentType      formData  string  false  "Content type"
// @Param        metaId           formData  string  false  "MetaID"
// @Param        address          formData  string  false  "Address (also used as change address if changeAddress is not provided)"
// @Param        changeAddress    formData  string  false  "Change address (optional, defaults to address)"
// @Param        feeRate          formData  int     false  "Fee rate (satoshis per byte, optional)"
// @Param        totalInputAmount formData  int     false  "Total input amount in satoshis (optional, for automatic change calculation)"
// @Success      200  {object}  respond.Response{data=CommitUploadResponseData}  "Upload successful, return transaction ID and Pin ID"
// @Failure      400  {object}  respond.Response  "Parameter error"
// @Failure      500  {object}  respond.Response  "Server error"
// @Router       /files/direct-upload [post]
func (h *UploadHandler) DirectUpload(c *gin.Context) {
	// Read file content
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		respond.InvalidParam(c, "file is required")
		return
	}
	defer file.Close()

	content, err := ioutil.ReadAll(file)
	if err != nil {
		respond.ServerError(c, "failed to read file")
		return
	}

	// Get required parameters
	path := c.PostForm("path")
	if path == "" {
		respond.InvalidParam(c, "path is required")
		return
	}

	preTxHex := c.PostForm("preTxHex")
	if preTxHex == "" {
		respond.InvalidParam(c, "preTxHex is required")
		return
	}

	// Get optional parameters
	operation := c.PostForm("operation")
	if operation == "" {
		operation = "create"
	}

	contentType := c.PostForm("contentType")
	if contentType == "" {
		contentType = header.Header.Get("Content-Type")
	}

	metaId := c.PostForm("metaId")
	address := c.PostForm("address")
	changeAddress := c.PostForm("changeAddress")
	mergeTxHex := c.PostForm("mergeTxHex") // Optional merge transaction hex

	// Parse optional numeric parameters
	feeRate := int64(0)
	feeRateStr := c.PostForm("feeRate")
	if feeRateStr != "" {
		if rate, err := strconv.ParseInt(feeRateStr, 10, 64); err == nil {
			feeRate = rate
		}
	}

	totalInputAmount := int64(0)
	totalInputAmountStr := c.PostForm("totalInputAmount")
	if totalInputAmountStr != "" {
		if amount, err := strconv.ParseInt(totalInputAmountStr, 10, 64); err == nil {
			totalInputAmount = amount
		}
	}

	// Build direct upload request
	req := &upload_service.DirectUploadRequest{
		MetaId:           metaId,
		Address:          address,
		FileName:         header.Filename,
		Content:          content,
		Path:             path,
		Operation:        operation,
		ContentType:      contentType,
		MergeTxHex:       mergeTxHex,
		PreTxHex:         preTxHex,
		ChangeAddress:    changeAddress,
		FeeRate:          feeRate,
		TotalInputAmount: totalInputAmount,
	}

	// Upload file (one-step: build + broadcast)
	resp, err := h.uploadService.DirectUpload(req)
	if err != nil {
		respond.ServerError(c, err.Error())
		return
	}

	respond.Success(c, resp)
}

// CommitUploadRequest commit upload request
type CommitUploadRequest struct {
	FileId      string `json:"fileId" binding:"required" example:"metaid_abc123" description:"File ID (from pre-upload response)"`
	SignedRawTx string `json:"signedRawTx" binding:"required" example:"0100000..." description:"Signed raw transaction data (hex)"`
}

// CommitUploadResponseData commit upload response data
type CommitUploadResponseData struct {
	FileId  string `json:"fileId" example:"metaid_abc123" description:"File ID"`
	Status  string `json:"status" example:"success" description:"Status: success, failed"`
	TxId    string `json:"txId" example:"abc123..." description:"Transaction ID"`
	PinId   string `json:"pinId" example:"abc123...i0" description:"Pin ID"`
	Message string `json:"message" example:"success" description:"Message"`
}

// CommitUpload commit upload: broadcast signed transaction
// @Summary      Commit upload
// @Description  Submit signed transaction for broadcast
// @Tags         File Upload
// @Accept       json
// @Produce      json
// @Param        request  body      CommitUploadRequest  true  "Commit upload request"
// @Success      200      {object}  respond.Response{data=CommitUploadResponseData}  "Upload successful, return transaction ID and Pin ID"
// @Failure      400      {object}  respond.Response  "Parameter error or file not found"
// @Failure      500      {object}  respond.Response  "Server error or broadcast failed"
// @Router       /files/commit-upload [post]
func (h *UploadHandler) CommitUpload(c *gin.Context) {
	var req CommitUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.InvalidParam(c, err.Error())
		return
	}

	// Commit upload
	resp, err := h.uploadService.CommitUpload(req.FileId, req.SignedRawTx)
	if err != nil {
		respond.ServerError(c, err.Error())
		return
	}

	respond.Success(c, resp)
}

// ChainConfigItem per-chain config for GetConfig response
type ChainConfigItem struct {
	MaxFileSize int64 `json:"maxFileSize" description:"Max file size in bytes"`
	ChunkSize   int64 `json:"chunkSize" description:"Chunk size in bytes"`
	FeeRate     int64 `json:"feeRate" description:"Fee rate (sat/byte or sat/KB)"`
}

// ConfigResponse configuration response
type ConfigResponse struct {
	MaxFileSize    int64                      `json:"maxFileSize" example:"10485760" description:"Max file size (bytes), min across chains for backward compat"`
	SwaggerBaseUrl string                     `json:"swaggerBaseUrl" example:"localhost:7282" description:"Swagger API base URL"`
	Chains         map[string]ChainConfigItem `json:"chains,omitempty" description:"Per-chain config (maxFileSize, chunkSize, feeRate)"`
}

// GetConfig get configuration information
// @Summary      Get configuration
// @Description  Get upload service configuration information, including max file size, swagger base URL, and per-chain config
// @Tags         Configuration
// @Accept       json
// @Produce      json
// @Success      200  {object}  respond.Response{data=ConfigResponse}
// @Router       /config [get]
func (h *UploadHandler) GetConfig(c *gin.Context) {
	chainsMap := make(map[string]ChainConfigItem)
	minMaxFileSize := conf.Cfg.Uploader.MaxFileSize
	for _, c := range conf.Cfg.Uploader.Chains {
		maxFileSize, chunkSize, feeRate := conf.GetUploaderChainParam(c.Name)
		chainsMap[c.Name] = ChainConfigItem{MaxFileSize: maxFileSize, ChunkSize: chunkSize, FeeRate: feeRate}
		if maxFileSize > 0 && (minMaxFileSize == 0 || maxFileSize < minMaxFileSize) {
			minMaxFileSize = maxFileSize
		}
	}
	respond.Success(c, ConfigResponse{
		MaxFileSize:    minMaxFileSize,
		SwaggerBaseUrl: conf.Cfg.Uploader.SwaggerBaseUrl,
		Chains:         chainsMap,
	})
}

// EstimateChunkedUploadRequest estimate chunked upload request
type EstimateChunkedUploadRequest struct {
	FileName    string `json:"fileName" binding:"required" example:"example.jpg" description:"File name"`
	Content     string `json:"content" description:"File content (base64 encoded string, optional if storageKey is provided)"`
	StorageKey  string `json:"storageKey" description:"Storage key from multipart upload (optional, if provided, file will be read from storage)"`
	Path        string `json:"path" binding:"required" example:"/file" description:"MetaID path (base path, will auto-add /file/_chunk and /file/index)"`
	ContentType string `json:"contentType" example:"image/jpeg" description:"File content type"`
	Chain       string `json:"chain" example:"mvc" description:"Blockchain: mvc or doge (default mvc)"`
	FeeRate     int64  `json:"feeRate" example:"1" description:"Fee rate (optional, defaults to chain config)"`
}

// EstimateChunkedUpload estimate chunked upload fee
// @Summary      Estimate chunked upload fee
// @Description  Estimate the fee required for chunked file upload, including chunk count and fees for ChunkPreTxHex and IndexPreTxHex
// @Tags         File Upload
// @Accept       json
// @Produce      json
// @Param        request  body      EstimateChunkedUploadRequest  true  "Estimate chunked upload request"
// @Success      200      {object}  respond.Response{data=upload_service.EstimateChunkedUploadResponse}  "Estimate successful"
// @Failure      400      {object}  respond.Response  "Parameter error"
// @Failure      500      {object}  respond.Response  "Server error"
// @Router       /files/estimate-chunked-upload [post]
func (h *UploadHandler) EstimateChunkedUpload(c *gin.Context) {
	var req EstimateChunkedUploadRequest
	if err := bindJSONWithOptionalGzip(c, &req); err != nil {
		respond.InvalidParam(c, err.Error())
		return
	}

	// Get content from storage or request body
	var content []byte
	var err error
	if req.StorageKey != "" {
		// Read from storage
		content, err = h.uploadService.GetFileFromStorage(req.StorageKey)
		if err != nil {
			respond.InvalidParam(c, "failed to read file from storage: "+err.Error())
			return
		}
	} else if req.Content != "" {
		// Decode base64 content
		content, err = base64.StdEncoding.DecodeString(req.Content)
		if err != nil {
			respond.InvalidParam(c, "invalid base64 content: "+err.Error())
			return
		}
	} else {
		respond.InvalidParam(c, "either content or storageKey must be provided")
		return
	}

	// Convert to service request
	chain := req.Chain
	if chain == "" {
		chain = "mvc"
	}
	serviceReq := &upload_service.EstimateChunkedUploadRequest{
		FileName:    req.FileName,
		Content:     content,
		Path:        req.Path,
		ContentType: req.ContentType,
		Chain:       chain,
		FeeRate:     req.FeeRate,
	}

	// Estimate fee
	resp, err := h.uploadService.EstimateChunkedUpload(serviceReq)
	if err != nil {
		respond.ServerError(c, err.Error())
		return
	}

	respond.Success(c, resp)
}

// ChunkedUploadRequest chunked upload request
type ChunkedUploadRequest struct {
	MetaId        string `json:"metaId" binding:"required" example:"metaid_abc123" description:"MetaID"`
	Address       string `json:"address" binding:"required" example:"1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa" description:"User address"`
	FileName      string `json:"fileName" binding:"required" example:"example.jpg" description:"File name"`
	Content       string `json:"content" description:"File content (base64 encoded string, optional if storageKey is provided)"`
	StorageKey    string `json:"storageKey" description:"Storage key from multipart upload (optional, if provided, file will be read from storage)"`
	Path          string `json:"path" binding:"required" example:"/file" description:"MetaID path (base path, will auto-add /file/_chunk and /file/index)"`
	Operation     string `json:"operation" example:"create" description:"Operation type (create/update)"`
	ContentType   string `json:"contentType" example:"image/jpeg" description:"File content type"`
	ChunkPreTxHex string `json:"chunkPreTxHex" binding:"required" example:"0100000..." description:"Pre-built chunk funding transaction (with inputs, signNull)"`
	IndexPreTxHex string `json:"indexPreTxHex" binding:"required" example:"0100000..." description:"Pre-built index transaction (with inputs, signNull)"`
	MergeTxHex    string `json:"mergeTxHex" example:"0100000..." description:"Merge transaction hex (creates two UTXOs, broadcasted first if IsBroadcast is true)"`
	FeeRate       int64  `json:"feeRate" example:"1" description:"Fee rate (optional, defaults to config)"`
	IsBroadcast   bool   `json:"isBroadcast" example:"false" description:"Whether to broadcast transactions automatically"`
}

// ChunkedUpload chunked file upload
// @Summary      Chunked file upload
// @Description  Upload large file by splitting it into chunks, build transactions for chunks and index, optionally broadcast all transactions in order
// @Tags         File Upload
// @Accept       json
// @Produce      json
// @Param        request  body      ChunkedUploadRequest  true  "Chunked upload request"
// @Success      200      {object}  respond.Response{data=upload_service.ChunkedUploadResponse}  "Upload successful"
// @Failure      400      {object}  respond.Response  "Parameter error"
// @Failure      500      {object}  respond.Response  "Server error"
// @Router       /files/chunked-upload [post]
func (h *UploadHandler) ChunkedUpload(c *gin.Context) {
	var req ChunkedUploadRequest
	if err := bindJSONWithOptionalGzip(c, &req); err != nil {
		respond.InvalidParam(c, err.Error())
		return
	}

	// Get content from storage or request body
	var content []byte
	var err error
	if req.StorageKey != "" {
		// Read from storage
		content, err = h.uploadService.GetFileFromStorage(req.StorageKey)
		if err != nil {
			respond.InvalidParam(c, "failed to read file from storage: "+err.Error())
			return
		}
	} else if req.Content != "" {
		// Decode base64 content
		content, err = base64.StdEncoding.DecodeString(req.Content)
		if err != nil {
			respond.InvalidParam(c, "invalid base64 content: "+err.Error())
			return
		}
	} else {
		respond.InvalidParam(c, "either content or storageKey must be provided")
		return
	}

	// Convert to service request
	serviceReq := &upload_service.ChunkedUploadRequest{
		MetaId:        req.MetaId,
		Address:       req.Address,
		FileName:      req.FileName,
		Content:       content,
		Path:          req.Path,
		Operation:     req.Operation,
		ContentType:   req.ContentType,
		ChunkPreTxHex: req.ChunkPreTxHex,
		IndexPreTxHex: req.IndexPreTxHex,
		MergeTxHex:    req.MergeTxHex,
		FeeRate:       req.FeeRate,
		IsBroadcast:   req.IsBroadcast,
	}

	// Upload file
	resp, err := h.uploadService.ChunkedUpload(serviceReq)
	if err != nil {
		respond.ServerError(c, err.Error())
		return
	}

	respond.Success(c, resp)
}

// ChunkedUploadForTaskRequest defines the payload for creating an async chunked upload task.
type ChunkedUploadForTaskRequest struct {
	MetaId        string `json:"metaId" binding:"required" example:"metaid_abc123" description:"MetaID"`
	Address       string `json:"address" binding:"required" example:"1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa" description:"User address"`
	FileName      string `json:"fileName" binding:"required" example:"example.jpg" description:"File name"`
	Content       string `json:"content" description:"Base64 encoded file content (optional if storageKey is provided)"`
	StorageKey    string `json:"storageKey" description:"Storage key from multipart upload (optional, if provided, file will be read from storage)"`
	Path          string `json:"path" binding:"required" example:"/file" description:"Base MetaID path (will append /file/_chunk and /file/index)"`
	Operation     string `json:"operation" example:"create" description:"Operation type (create/update)"`
	ContentType   string `json:"contentType" example:"image/jpeg" description:"MIME type"`
	Chain         string `json:"chain" example:"mvc" description:"Blockchain: mvc or doge (default mvc)"`
	ChunkPreTxHex string `json:"chunkPreTxHex" binding:"required" example:"0100000..." description:"Pre-built chunk transaction (contains inputs, signNull)"`
	IndexPreTxHex string `json:"indexPreTxHex" example:"0100000..." description:"Pre-built index transaction (required for mvc, optional for doge - index funded by chunk change)"`
	MergeTxHex    string `json:"mergeTxHex" example:"0100000..." description:"Merge transaction hex (optional, broadcast first)"`
	FeeRate       int64  `json:"feeRate" example:"1" description:"Fee rate (optional, defaults to config)"`
}

// ChunkedUploadForTask creates an async chunked upload task.
// @Summary      Async chunked upload (create task)
// @Description  Create an async chunked upload task and return the task ID so the client can poll for progress
// @Tags         File Upload
// @Accept       json
// @Produce      json
// @Param        request  body      ChunkedUploadForTaskRequest  true  "Async chunked upload request"
// @Success      200      {object}  respond.Response{data=respond.ChunkedUploadTaskResponse}
// @Failure      400      {object}  respond.Response  "Invalid parameter"
// @Failure      500      {object}  respond.Response  "Server error"
// @Router       /files/chunked-upload-task [post]
func (h *UploadHandler) ChunkedUploadForTask(c *gin.Context) {
	var req ChunkedUploadForTaskRequest
	if err := bindJSONWithOptionalGzip(c, &req); err != nil {
		respond.InvalidParam(c, err.Error())
		return
	}

	// Get content from storage or request body
	var content []byte
	var err error
	if req.StorageKey != "" {
		// Read from storage
		content, err = h.uploadService.GetFileFromStorage(req.StorageKey)
		if err != nil {
			respond.InvalidParam(c, "failed to read file from storage: "+err.Error())
			return
		}
	} else if req.Content != "" {
		// Decode base64 payload
		content, err = base64.StdEncoding.DecodeString(req.Content)
		if err != nil {
			respond.InvalidParam(c, "invalid base64 content: "+err.Error())
			return
		}
	} else {
		respond.InvalidParam(c, "either content or storageKey must be provided")
		return
	}

	chain := req.Chain
	if chain == "" {
		chain = "mvc"
	}
	if !conf.IsChainSupportedForUpload(chain) {
		supported := "none configured"
		if names := conf.GetUploaderChainNames(); len(names) > 0 {
			supported = strings.Join(names, ", ")
		}
		respond.InvalidParam(c, "chain not supported: "+chain+", supported: "+supported)
		return
	}
	if chain == "mvc" && req.IndexPreTxHex == "" {
		respond.InvalidParam(c, "indexPreTxHex is required for mvc chain")
		return
	}

	// Convert to service request
	serviceReq := &upload_service.ChunkedUploadRequest{
		MetaId:        req.MetaId,
		Address:       req.Address,
		FileName:      req.FileName,
		Content:       content,
		Path:          req.Path,
		Operation:     req.Operation,
		ContentType:   req.ContentType,
		Chain:         chain,
		ChunkPreTxHex: req.ChunkPreTxHex,
		IndexPreTxHex: req.IndexPreTxHex,
		MergeTxHex:    req.MergeTxHex,
		FeeRate:       req.FeeRate,
		IsBroadcast:   false, // handled asynchronously by background worker
	}

	// Create async task
	resp, err := h.uploadService.ChunkedUploadForTask(serviceReq)
	if err != nil {
		respond.ServerError(c, err.Error())
		return
	}

	respond.Success(c, respond.ChunkedUploadTaskResponse{
		TaskId:  resp.TaskId,
		Status:  resp.Status,
		Message: resp.Message,
	})
}

// GetTaskProgress returns detailed task info by task ID.
// @Summary      Query task progress
// @Description  Get async upload task progress and status by task ID
// @Tags         File Upload
// @Accept       json
// @Produce      json
// @Param        taskId  path      string  true  "Task ID"
// @Success      200     {object}  respond.Response{data=respond.UploadTaskDetailResponse}
// @Failure      400     {object}  respond.Response  "Invalid parameter"
// @Failure      404     {object}  respond.Response  "Task not found"
// @Failure      500     {object}  respond.Response  "Server error"
// @Router       /files/task/{taskId} [get]
func (h *UploadHandler) GetTaskProgress(c *gin.Context) {
	taskId := c.Param("taskId")
	if taskId == "" {
		respond.InvalidParam(c, "task ID is required")
		return
	}

	task, err := h.uploadService.GetTaskProgress(taskId)
	if err != nil {
		respond.ServerError(c, err.Error())
		return
	}

	respond.Success(c, task)
}

// InitiateMultipartUploadRequest request for initiating multipart upload
type InitiateMultipartUploadRequest struct {
	FileName string `json:"fileName" binding:"required"`
	FileSize int64  `json:"fileSize" binding:"required"`
	MetaId   string `json:"metaId"`  // Optional, for file organization
	Address  string `json:"address"` // Optional, user address
}

// InitiateMultipartUpload initiates a multipart upload session
// @Summary      Initiate multipart upload
// @Description  Start a multipart upload session for large file uploads with resume support
// @Tags         File Upload
// @Accept       json
// @Produce      json
// @Param        request  body      InitiateMultipartUploadRequest  true  "Initiate multipart upload request"
// @Success      200      {object}  respond.Response{data=upload_service.InitiateMultipartUploadResponse}
// @Failure      400      {object}  respond.Response  "Parameter error"
// @Failure      500      {object}  respond.Response  "Server error"
// @Router       /files/multipart/initiate [post]
func (h *UploadHandler) InitiateMultipartUpload(c *gin.Context) {
	var req InitiateMultipartUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.InvalidParam(c, err.Error())
		return
	}

	serviceReq := &upload_service.MultipartUploadRequest{
		FileName: req.FileName,
		FileSize: req.FileSize,
		MetaId:   req.MetaId,
		Address:  req.Address,
	}

	resp, err := h.uploadService.InitiateMultipartUpload(serviceReq)
	if err != nil {
		respond.ServerError(c, err.Error())
		return
	}

	respond.Success(c, resp)
}

// UploadPartRequest request for uploading a part
type UploadPartRequest struct {
	UploadId   string `json:"uploadId" binding:"required"`
	Key        string `json:"key" binding:"required"` // Storage key from initiate
	PartNumber int    `json:"partNumber" binding:"required"`
	Content    string `json:"content" binding:"required"` // Base64 encoded part data
}

// UploadPart uploads a part of the file
// @Summary      Upload part
// @Description  Upload a single part of the file in a multipart upload session
// @Tags         File Upload
// @Accept       json
// @Produce      json
// @Param        request  body      UploadPartRequest  true  "Upload part request"
// @Success      200      {object}  respond.Response{data=upload_service.UploadPartResponse}
// @Failure      400      {object}  respond.Response  "Parameter error"
// @Failure      500      {object}  respond.Response  "Server error"
// @Router       /files/multipart/upload-part [post]
func (h *UploadHandler) UploadPart(c *gin.Context) {
	var req UploadPartRequest
	if err := bindJSONWithOptionalGzip(c, &req); err != nil {
		respond.InvalidParam(c, err.Error())
		return
	}

	// Decode base64 content
	content, err := base64.StdEncoding.DecodeString(req.Content)
	if err != nil {
		respond.InvalidParam(c, "invalid base64 content: "+err.Error())
		return
	}

	serviceReq := &upload_service.UploadPartRequest{
		UploadId:   req.UploadId,
		Key:        req.Key,
		PartNumber: req.PartNumber,
		Data:       content,
	}

	resp, err := h.uploadService.UploadPart(serviceReq)
	if err != nil {
		respond.ServerError(c, err.Error())
		return
	}

	respond.Success(c, resp)
}

// CompleteMultipartUploadRequest request for completing multipart upload
type CompleteMultipartUploadRequest struct {
	UploadId string `json:"uploadId" binding:"required"`
	Key      string `json:"key" binding:"required"`
	Parts    []struct {
		PartNumber int    `json:"partNumber"`
		ETag       string `json:"etag"`
		Size       int64  `json:"size"`
	} `json:"parts" binding:"required"`
}

// CompleteMultipartUpload completes the multipart upload
// @Summary      Complete multipart upload
// @Description  Complete a multipart upload session and merge all parts into a single file
// @Tags         File Upload
// @Accept       json
// @Produce      json
// @Param        request  body      CompleteMultipartUploadRequest  true  "Complete multipart upload request"
// @Success      200      {object}  respond.Response{data=upload_service.CompleteMultipartUploadResponse}
// @Failure      400      {object}  respond.Response  "Parameter error"
// @Failure      500      {object}  respond.Response  "Server error"
// @Router       /files/multipart/complete [post]
func (h *UploadHandler) CompleteMultipartUpload(c *gin.Context) {
	var req CompleteMultipartUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.InvalidParam(c, err.Error())
		return
	}

	// Convert parts
	parts := make([]storage.PartInfo, 0, len(req.Parts))
	for _, p := range req.Parts {
		parts = append(parts, storage.PartInfo{
			PartNumber: p.PartNumber,
			ETag:       p.ETag,
			Size:       p.Size,
		})
	}

	serviceReq := &upload_service.CompleteMultipartUploadRequest{
		UploadId: req.UploadId,
		Parts:    parts,
	}

	resp, err := h.uploadService.CompleteMultipartUpload(req.Key, serviceReq)
	if err != nil {
		respond.ServerError(c, err.Error())
		return
	}

	respond.Success(c, resp)
}

// ListPartsRequest request for listing parts
type ListPartsRequest struct {
	UploadId string `json:"uploadId" binding:"required"`
	Key      string `json:"key" binding:"required"`
}

// ListParts lists all uploaded parts for resuming upload
// @Summary      List uploaded parts
// @Description  List all uploaded parts in a multipart upload session for resume support
// @Tags         File Upload
// @Accept       json
// @Produce      json
// @Param        request  body      ListPartsRequest  true  "List parts request"
// @Success      200      {object}  respond.Response{data=upload_service.ListPartsResponse}
// @Failure      400      {object}  respond.Response  "Parameter error"
// @Failure      500      {object}  respond.Response  "Server error"
// @Router       /files/multipart/list-parts [post]
func (h *UploadHandler) ListParts(c *gin.Context) {
	var req ListPartsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.InvalidParam(c, err.Error())
		return
	}

	resp, err := h.uploadService.ListParts(req.Key, req.UploadId)
	if err != nil {
		respond.ServerError(c, err.Error())
		return
	}

	respond.Success(c, resp)
}

// AbortMultipartUploadRequest request for aborting multipart upload
type AbortMultipartUploadRequest struct {
	UploadId string `json:"uploadId" binding:"required"`
	Key      string `json:"key" binding:"required"`
}

// AbortMultipartUpload aborts a multipart upload
// @Summary      Abort multipart upload
// @Description  Abort a multipart upload session and clean up resources
// @Tags         File Upload
// @Accept       json
// @Produce      json
// @Param        request  body      AbortMultipartUploadRequest  true  "Abort multipart upload request"
// @Success      200      {object}  respond.Response  "Abort successful"
// @Failure      400      {object}  respond.Response  "Parameter error"
// @Failure      500      {object}  respond.Response  "Server error"
// @Router       /files/multipart/abort [post]
func (h *UploadHandler) AbortMultipartUpload(c *gin.Context) {
	var req AbortMultipartUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.InvalidParam(c, err.Error())
		return
	}

	err := h.uploadService.AbortMultipartUpload(req.Key, req.UploadId)
	if err != nil {
		respond.ServerError(c, err.Error())
		return
	}

	respond.Success(c, gin.H{"message": "Upload aborted successfully"})
}

// ListUploadTasks list upload tasks by address with cursor pagination
// @Summary      List upload tasks
// @Description  List chunked upload tasks for a given address with cursor-based pagination
// @Tags         File Upload
// @Accept       json
// @Produce      json
// @Param        address  query     string  true   "User address"
// @Param        cursor   query     int     false  "Cursor (last task ID)"  default(0)
// @Param        size     query     int     false  "Page size"              default(20)
// @Success      200      {object}  respond.Response{data=respond.UploadTaskListResponse}
// @Failure      400      {object}  respond.Response  "Parameter error"
// @Failure      500      {object}  respond.Response  "Server error"
// @Router       /files/tasks [get]
func (h *UploadHandler) ListUploadTasks(c *gin.Context) {
	address := c.Query("address")
	if strings.TrimSpace(address) == "" {
		respond.InvalidParam(c, "address is required")
		return
	}

	cursorStr := c.DefaultQuery("cursor", "0")
	sizeStr := c.DefaultQuery("size", "20")

	cursor, err := strconv.ParseInt(cursorStr, 10, 64)
	if err != nil {
		respond.InvalidParam(c, "invalid cursor")
		return
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		respond.InvalidParam(c, "invalid size")
		return
	}

	resp, err := h.uploadService.ListTasksByAddress(address, cursor, size)
	if err != nil {
		respond.ServerError(c, err.Error())
		return
	}

	respond.Success(c, respond.UploadTaskListResponse{
		Tasks:      respond.ToUploadTaskList(resp.Tasks),
		NextCursor: resp.NextCursor,
		HasMore:    resp.HasMore,
	})
}
