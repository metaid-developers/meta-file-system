package handler

import (
	"encoding/json"
	"io/ioutil"
	"strconv"

	"meta-file-system/common"
	"meta-file-system/conf"
	"meta-file-system/controller/respond"
	"meta-file-system/service/upload_service"

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

// ConfigResponse configuration response
type ConfigResponse struct {
	MaxFileSize    int64  `json:"maxFileSize" example:"10485760" description:"Max file size (bytes)"`
	SwaggerBaseUrl string `json:"swaggerBaseUrl" example:"localhost:7282" description:"Swagger API base URL"`
}

// GetConfig get configuration information
// @Summary      Get configuration
// @Description  Get upload service configuration information, including max file size and swagger base URL
// @Tags         Configuration
// @Accept       json
// @Produce      json
// @Success      200  {object}  respond.Response{data=ConfigResponse}
// @Router       /config [get]
func (h *UploadHandler) GetConfig(c *gin.Context) {
	respond.Success(c, ConfigResponse{
		MaxFileSize:    conf.Cfg.Uploader.MaxFileSize,
		SwaggerBaseUrl: conf.Cfg.Uploader.SwaggerBaseUrl,
	})
}
