package upload_service

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	bsvec2 "github.com/bitcoinsv/bsvd/bsvec"
	chaincfg2 "github.com/bitcoinsv/bsvd/chaincfg"
	chainhash2 "github.com/bitcoinsv/bsvd/chaincfg/chainhash"
	txscript2 "github.com/bitcoinsv/bsvd/txscript"
	wire2 "github.com/bitcoinsv/bsvd/wire"
	bsvutil2 "github.com/bitcoinsv/bsvutil"
	"github.com/btcsuite/btcd/txscript"
	"gorm.io/gorm"

	"meta-file-system/common"
	"meta-file-system/conf"
	"meta-file-system/database"
	"meta-file-system/indexer"
	"meta-file-system/model"
	"meta-file-system/model/dao"
	"meta-file-system/node"
	"meta-file-system/service/common_service/metaid_protocols"
	"meta-file-system/storage"
)

// UploadService upload service
type UploadService struct {
	fileDAO             *dao.FileDAO
	fileChunkDAO        *dao.FileChunkDAO
	fileAssistentDAO    *dao.FileAssistentDAO
	fileUploaderTaskDAO *dao.FileUploaderTaskDAO
	storage             storage.Storage
}

// NewUploadService create upload service instance
func NewUploadService(storage storage.Storage) *UploadService {
	return &UploadService{
		fileDAO:             dao.NewFileDAO(),
		fileChunkDAO:        dao.NewFileChunkDAO(),
		fileAssistentDAO:    dao.NewFileAssistentDAO(),
		fileUploaderTaskDAO: dao.NewFileUploaderTaskDAO(),
		storage:             storage,
	}
}

// UploadRequest upload request
type UploadRequest struct {
	MetaId        string                // MetaID
	Address       string                // Address
	FileName      string                // File name
	Content       []byte                // File content
	Path          string                // MetaID path
	Operation     string                // create/update
	ContentType   string                // Content type
	ChangeAddress string                // Change address
	Inputs        []*common.TxInputUtxo // Input UTXO
	Outputs       []*common.TxOutput    // Outputs
	OtherOutputs  []*common.TxOutput    // Other outputs
	FeeRate       int64                 // Fee rate
}

// DirectUploadRequest direct upload request (one-step upload with PreTxHex)
type DirectUploadRequest struct {
	MetaId           string // MetaID
	Address          string // Address (also used as change address if ChangeAddress is empty)
	FileName         string // File name
	Content          []byte // File content
	Path             string // MetaID path
	Operation        string // create/update
	ContentType      string // Content type
	MergeTxHex       string // Merge transaction hex (signed, with inputs and outputs)
	PreTxHex         string // Pre-transaction hex (signed, with inputs and outputs)
	ChangeAddress    string // Change address (optional, defaults to Address)
	FeeRate          int64  // Fee rate (satoshis per byte, optional, defaults to config)
	TotalInputAmount int64  // Total input amount in satoshis (optional, for change calculation)
}

// PreUploadResponse pre-upload response
type PreUploadResponse struct {
	FileId    string `json:"fileId"`    // File ID (unique identifier)
	FileMd5   string `json:"fileMd5"`   // File md5
	FileHash  string `json:"fileHash"`  // File hash
	TxId      string `json:"txId"`      // Transaction ID
	PinId     string `json:"pinId"`     // Pin ID
	PreTxRaw  string `json:"preTxRaw"`  // Pre-transaction raw data
	Status    string `json:"status"`    // Status
	Message   string `json:"message"`   // Message (e.g., exists, success, etc.)
	CalTxFee  int64  `json:"calTxFee"`  // Calculated transaction fee
	CalTxSize int64  `json:"calTxSize"` // Calculated transaction size
}

// UploadResponse upload response
type UploadResponse struct {
	FileId  string `json:"fileId"`  // File ID
	Status  string `json:"status"`  // Status
	TxId    string `json:"txId"`    // Transaction ID
	PinId   string `json:"pinId"`   // Pin ID
	Message string `json:"message"` // Message
}

// PreUpload pre-upload: build transaction and save file metadata
func (s *UploadService) PreUpload(req *UploadRequest) (*PreUploadResponse, error) {
	// Parameter validation
	if len(req.Content) == 0 {
		return nil, fmt.Errorf("file content is empty")
	}
	if req.Path == "" {
		return nil, fmt.Errorf("file path is required")
	}

	// Set default values
	if req.Operation == "" {
		req.Operation = "create"
	}
	if req.ContentType == "" {
		req.ContentType = "application/octet-stream"
	}
	if req.FeeRate == 0 {
		req.FeeRate = conf.Cfg.Uploader.FeeRate
	}

	// Get network parameters
	var netParam *chaincfg2.Params
	if conf.Cfg.Net == "mainnet" {
		netParam = &chaincfg2.MainNetParams
	} else {
		netParam = &chaincfg2.TestNet3Params
	}

	// Build transaction
	tx, err := common.BuildMvcCommonMetaIdTxForUnkwonInput(
		netParam,
		req.Inputs,
		req.Outputs,
		req.OtherOutputs,
		req.Operation,
		req.Path,
		req.Content,
		req.ContentType,
		req.ChangeAddress,
		req.FeeRate,
		true, // No signature needed
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build transaction: %w", err)
	}

	txSize := tx.SerializeSize()
	txFee := int64(txSize) * req.FeeRate

	// Get transaction ID and raw transaction
	// txID := tx.Txhash().String()
	preTxRaw, err := indexer.TxToHex(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize transaction: %w", err)
	}

	// Calculate file hash
	sha256hash := sha256.Sum256(req.Content)
	md5hash := md5.Sum(req.Content)
	filehashStr := hex.EncodeToString(sha256hash[:])
	md5hashStr := hex.EncodeToString(md5hash[:])

	// Generate FileId (ensure uniqueness)
	fileId := req.MetaId + "_" + filehashStr

	// Check if FileId already exists
	existingFile, err := s.fileDAO.GetByFileID(fileId)
	if err == nil && existingFile != nil {
		// File already exists, return different info based on status
		if existingFile.Status == model.StatusSuccess {
			// File already successfully uploaded to chain
			log.Printf("File already exists and uploaded successfully: FileId=%s", fileId)
			return &PreUploadResponse{
				TxId:     existingFile.TxID,
				PinId:    existingFile.PinId,
				FileId:   existingFile.FileId,
				FileMd5:  existingFile.FileMd5,
				FileHash: existingFile.FileHash,
				PreTxRaw: preTxRaw,
				Status:   string(existingFile.Status),
				Message:  "file already exists and uploaded",
			}, nil
		} else if existingFile.Status == model.StatusPending {
			// File is being processed, return existing PreTxRaw
			log.Printf("File already exists in pending status: FileId=%s", fileId)
			return &PreUploadResponse{
				FileId:   existingFile.FileId,
				FileMd5:  existingFile.FileMd5,
				FileHash: existingFile.FileHash,
				PreTxRaw: preTxRaw,
				Status:   string(existingFile.Status),
				Message:  "file already in pending, please commit",
			}, nil
		}
		// If status is failed, allow re-upload
		log.Printf("File exists but failed, allow re-upload: FileId=%s", fileId)
	}

	// Save file metadata (status pending)
	file := &model.File{
		FileId:          fileId,
		FileName:        req.FileName,
		FileType:        strings.ReplaceAll(req.ContentType, ";binary", ""),
		MetaId:          req.MetaId,
		Address:         req.Address,
		Path:            req.Path,
		ContentType:     req.ContentType,
		FileSize:        int64(len(req.Content)),
		FileHash:        filehashStr,
		FileMd5:         md5hashStr,
		FileContentType: strings.ReplaceAll(req.ContentType, ";binary", ""),
		ChunkType:       model.ChunkTypeSingle,
		Operation:       req.Operation,
		// PreTxRaw:        preTxRaw,
		Status: model.StatusPending, // Set status to pending
	}

	if err := s.fileDAO.Create(file); err != nil {
		return nil, fmt.Errorf("failed to save file metadata: %w", err)
	}

	log.Printf("File metadata saved successfully: FileId=%s, status=pending", file.FileId)

	return &PreUploadResponse{
		FileId:    file.FileId,
		FileMd5:   md5hashStr,
		FileHash:  filehashStr,
		PreTxRaw:  preTxRaw,
		Status:    string(file.Status),
		TxId:      file.TxID,
		PinId:     file.PinId,
		CalTxFee:  txFee,
		CalTxSize: int64(txSize),
		Message:   "success",
	}, nil
}

// CommitUpload commit upload: broadcast transaction and update file status
// Use database transaction to ensure data consistency
func (s *UploadService) CommitUpload(fileId string, signedRawTx string) (*UploadResponse, error) {

	var (
		txId   string
		status string
	)
	// Use database transaction
	err := database.UploaderDB.Transaction(func(tx *gorm.DB) error {
		// 1. Query file record
		var file model.File
		if err := tx.Where("file_id = ?", fileId).First(&file).Error; err != nil {
			return fmt.Errorf("failed to find file record: %w", err)
		}

		// Check file status
		if file.Status == model.StatusSuccess {
			log.Printf("File already committed: fileId=%s", fileId)
			return fmt.Errorf("file already committed: fileId=%s", fileId)
		}
		txhash := common.GetMvcTxhashFromRaw(signedRawTx)

		// 2. Update file record
		// file.TxRaw = signedRawTx
		file.TxID = txhash
		file.PinId = fmt.Sprintf("%si0", txhash)
		file.Status = model.StatusSuccess
		if err := tx.Save(&file).Error; err != nil {
			return fmt.Errorf("failed to update file record: %w", err)
		}
		status = string(file.Status)
		txId = file.TxID

		// 3. Broadcast transaction to blockchain network
		chain := conf.Cfg.Net // Use network type from configuration
		broadcastTxID, err := node.BroadcastTx(chain, signedRawTx)
		if err != nil {
			// Broadcast failed, update status to failed
			file.Status = model.StatusFailed
			if updateErr := tx.Save(&file).Error; updateErr != nil {
				return fmt.Errorf("failed to update file status to failed: %w", updateErr)
			}
			return fmt.Errorf("failed to broadcast transaction: %w", err)
		}

		log.Printf("Transaction broadcasted successfully: fileId=%s, broadcastTxID=%s", fileId, broadcastTxID)

		log.Printf("File status updated to success: fileId=%s", fileId)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &UploadResponse{
		FileId:  fileId,
		Status:  status,
		TxId:    txId,
		PinId:   fmt.Sprintf("%si0", txId),
		Message: "success",
	}, nil
}

// DirectUpload direct upload: one-step upload with PreTxHex (add MetaID output and broadcast)
func (s *UploadService) DirectUpload(req *DirectUploadRequest) (*UploadResponse, error) {
	// Parameter validation
	if len(req.Content) == 0 {
		return nil, fmt.Errorf("file content is empty")
	}
	if req.Path == "" {
		return nil, fmt.Errorf("file path is required")
	}
	// if req.MergeTxHex == "" {
	// 	return nil, fmt.Errorf("MergeTxHex is required")
	// }
	if req.PreTxHex == "" {
		return nil, fmt.Errorf("PreTxHex is required")
	}

	// Set default values
	if req.Operation == "" {
		req.Operation = "create"
	}
	if req.ContentType == "" {
		req.ContentType = "application/octet-stream"
	}
	if req.ChangeAddress == "" && req.Address != "" {
		req.ChangeAddress = req.Address
	}
	if req.FeeRate == 0 {
		req.FeeRate = conf.Cfg.Uploader.FeeRate
	}

	// Get network parameters
	var netParam *chaincfg2.Params
	if conf.Cfg.Net == "mainnet" {
		netParam = &chaincfg2.MainNetParams
	} else {
		netParam = &chaincfg2.TestNet3Params
	}

	// Parse PreTxHex to get transaction
	preTxBytes, err := hex.DecodeString(req.PreTxHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode PreTxHex: %w", err)
	}

	tx := wire2.NewMsgTx(10)
	err = tx.Deserialize(bytes.NewReader(preTxBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize transaction: %w", err)
	}

	// Calculate existing outputs amount
	outAmount := int64(0)
	for _, out := range tx.TxOut {
		outAmount += out.Value
	}

	// Build MetaID OP_RETURN output
	inscriptionBuilder := txscript.NewScriptBuilder().
		AddOp(txscript.OP_0).
		AddOp(txscript.OP_RETURN).
		AddData([]byte("metaid")).       // <metaid_flag>
		AddData([]byte(req.Operation)).  // <operation>
		AddData([]byte(req.Path)).       // <path>
		AddData([]byte("0")).            // <Encryption>
		AddData([]byte("1.0.0")).        // <version>
		AddData([]byte(req.ContentType)) // <content-type>

	// Split content into chunks (max 520 bytes per chunk)
	maxChunkSize := 520
	bodySize := len(req.Content)
	for i := 0; i < bodySize; i += maxChunkSize {
		end := i + maxChunkSize
		if end > bodySize {
			end = bodySize
		}
		inscriptionBuilder.AddFullData(req.Content[i:end]) // <payload>
	}

	inscriptionScript, err := inscriptionBuilder.Script()
	if err != nil {
		return nil, fmt.Errorf("failed to build inscription script: %w", err)
	}

	// Add MetaID OP_RETURN output to transaction
	tx.AddTxOut(wire2.NewTxOut(0, inscriptionScript))

	// Add change output if change address and total input amount are provided
	if req.ChangeAddress != "" && req.TotalInputAmount > 0 {
		addr, err := bsvutil2.DecodeAddress(req.ChangeAddress, netParam)
		if err != nil {
			return nil, fmt.Errorf("failed to decode change address: %w", err)
		}
		pkScriptByte, err := txscript2.PayToAddrScript(addr)
		if err != nil {
			return nil, fmt.Errorf("failed to create change script: %w", err)
		}
		// Add change output with initial value 0
		tx.AddTxOut(wire2.NewTxOut(0, pkScriptByte))

		// Calculate transaction size and fee
		txTotalSize := tx.SerializeSize()
		txFee := int64(txTotalSize) * req.FeeRate

		log.Printf("DirectUpload: txTotalSize=%d, txFee=%d, feeRate=%d, totalInputAmount=%d, outAmount=%d",
			txTotalSize, txFee, req.FeeRate, req.TotalInputAmount, outAmount)

		// Check if there's enough input amount
		if req.TotalInputAmount-outAmount < txFee {
			return nil, fmt.Errorf("insufficient fee: need %d, have %d", txFee, req.TotalInputAmount-outAmount)
		}

		// Calculate change value
		changeVal := req.TotalInputAmount - outAmount - txFee
		if changeVal >= 600 {
			// Set change output value
			tx.TxOut[len(tx.TxOut)-1].Value = changeVal
			log.Printf("DirectUpload: change output added with value=%d", changeVal)
		} else {
			// Remove change output if change is too small
			tx.TxOut = tx.TxOut[:len(tx.TxOut)-1]
			log.Printf("DirectUpload: change output removed (changeVal=%d < 600)", changeVal)
		}
	}

	// Serialize transaction to hex (final signed transaction with MetaID output)
	signedRawTx, err := indexer.TxToHex(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize transaction: %w", err)
	}

	// Get transaction hash
	txhash := common.GetMvcTxhashFromRaw(signedRawTx)

	// Calculate file hash
	sha256hash := sha256.Sum256(req.Content)
	md5hash := md5.Sum(req.Content)
	filehashStr := hex.EncodeToString(sha256hash[:])
	md5hashStr := hex.EncodeToString(md5hash[:])

	// Generate FileId (ensure uniqueness)
	fileId := req.MetaId + "_" + filehashStr

	var (
		finalTxId string
		pinId     string
		status    string
	)

	// Use database transaction to ensure data consistency
	err = database.UploaderDB.Transaction(func(dbTx *gorm.DB) error {
		// Check if FileId already exists
		var existingFile model.File
		err := dbTx.Where("file_id = ?", fileId).First(&existingFile).Error

		if err == nil {
			// File already exists
			if existingFile.Status == model.StatusSuccess {
				// File already successfully uploaded to chain
				log.Printf("File already exists and uploaded successfully: FileId=%s", fileId)
				finalTxId = existingFile.TxID
				pinId = existingFile.PinId
				status = string(existingFile.Status)
				return nil
			} else if existingFile.Status == model.StatusPending {
				// File is pending, update and broadcast
				log.Printf("File exists in pending status, updating and broadcasting: FileId=%s", fileId)
				existingFile.TxID = txhash
				existingFile.PinId = fmt.Sprintf("%si0", txhash)
				existingFile.Status = model.StatusSuccess
				if err := dbTx.Save(&existingFile).Error; err != nil {
					return fmt.Errorf("failed to update file record: %w", err)
				}
				finalTxId = existingFile.TxID
				pinId = existingFile.PinId
				status = string(existingFile.Status)

				// Broadcast transaction
				chain := conf.Cfg.Net
				if req.MergeTxHex != "" {
					broadcastMergeTxID, err := node.BroadcastTx(chain, req.MergeTxHex)
					if err != nil {
						// // Broadcast failed, update status to failed
						// existingFile.Status = model.StatusFailed
						// if updateErr := dbTx.Save(&existingFile).Error; updateErr != nil {
						// 	return fmt.Errorf("failed to update file status to failed: %w", updateErr)
						// }
						return fmt.Errorf("failed to broadcast merge transaction: %w", err)
					}
					log.Printf("Transaction broadcasted successfully: fileId=%s, broadcastMergeTxID=%s", fileId, broadcastMergeTxID)
				}

				broadcastTxID, err := node.BroadcastTx(chain, signedRawTx)
				if err != nil {
					// Broadcast failed, update status to failed
					// existingFile.Status = model.StatusFailed
					// if updateErr := dbTx.Save(&existingFile).Error; updateErr != nil {
					// 	return fmt.Errorf("failed to update file status to failed: %w", updateErr)
					// }
					return fmt.Errorf("failed to broadcast transaction: %w", err)
				}
				log.Printf("Transaction broadcasted successfully: fileId=%s, broadcastTxID=%s", fileId, broadcastTxID)
				return nil
			}
			// If status is failed, allow re-upload (continue to create new record)
			log.Printf("File exists but failed, allow re-upload: FileId=%s", fileId)
		}

		// File does not exist, create new record
		file := &model.File{
			FileId:          fileId,
			FileName:        req.FileName,
			FileType:        strings.ReplaceAll(req.ContentType, ";binary", ""),
			MetaId:          req.MetaId,
			Address:         req.Address,
			Path:            req.Path,
			ContentType:     req.ContentType,
			FileSize:        int64(len(req.Content)),
			FileHash:        filehashStr,
			FileMd5:         md5hashStr,
			FileContentType: strings.ReplaceAll(req.ContentType, ";binary", ""),
			ChunkType:       model.ChunkTypeSingle,
			Operation:       req.Operation,
			TxID:            txhash,
			PinId:           fmt.Sprintf("%si0", txhash),
			Status:          model.StatusSuccess,
		}

		if err := dbTx.Create(file).Error; err != nil {
			return fmt.Errorf("failed to create file metadata: %w", err)
		}

		finalTxId = file.TxID
		pinId = file.PinId
		status = string(file.Status)

		// Broadcast transaction
		chain := conf.Cfg.Net
		if req.MergeTxHex != "" {
			broadcastMergeTxID, err := node.BroadcastTx(chain, req.MergeTxHex)
			if err != nil {
				return fmt.Errorf("failed to broadcast merge transaction: %w", err)
			}
			log.Printf("Transaction broadcasted successfully: fileId=%s, broadcastMergeTxID=%s", fileId, broadcastMergeTxID)
		}

		broadcastTxID, err := node.BroadcastTx(chain, signedRawTx)
		if err != nil {
			// Broadcast failed, update status to failed
			// file.Status = model.StatusFailed
			// if updateErr := dbTx.Save(file).Error; updateErr != nil {
			// 	return fmt.Errorf("failed to update file status to failed: %w", updateErr)
			// }
			return fmt.Errorf("failed to broadcast transaction: %w", err)
		}

		log.Printf("File created and transaction broadcasted successfully: fileId=%s, broadcastTxID=%s", fileId, broadcastTxID)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &UploadResponse{
		FileId:  fileId,
		Status:  status,
		TxId:    finalTxId,
		PinId:   pinId,
		Message: "success",
	}, nil
}

// EstimateChunkedUploadRequest describes the payload used to estimate chunked upload fees.
type EstimateChunkedUploadRequest struct {
	FileName    string // File name
	Content     []byte // File content
	Path        string // Base MetaID path (will append /file/_chunk and /file/index)
	ContentType string // MIME type (e.g. image/jpeg, text/plain)
	FeeRate     int64  // Fee rate (optional, defaults to config)
}

// EstimateChunkedUploadResponse contains fee estimation details for chunked upload.
type EstimateChunkedUploadResponse struct {
	ChunkNumber   int     `json:"chunkNumber"`   // Number of chunks
	ChunkSize     int64   `json:"chunkSize"`     // Chunk size in bytes
	ChunkFees     []int64 `json:"chunkFees"`     // Fee per chunk
	ChunkPerTxFee int64   `json:"chunkPerTxFee"` // Fee required per chunk transaction
	ChunkPreTxFee int64   `json:"chunkPreTxFee"` // Total funding required for chunk transactions
	IndexPreTxFee int64   `json:"indexPreTxFee"` // Funding required for the index transaction
	TotalFee      int64   `json:"totalFee"`      // Total fee (ChunkPreTxFee + IndexPreTxFee)
	PerChunkFee   int64   `json:"perChunkFee"`   // Average fee per chunk
	Message       string  `json:"message"`       // Additional message
}

// ChunkedUploadRequest describes a chunked upload payload.
type ChunkedUploadRequest struct {
	MetaId        string                  // MetaID
	Address       string                  // User address
	FileName      string                  // File name
	Content       []byte                  // File content
	Path          string                  // Base MetaID path (auto appends /file/_chunk and /file/index)
	Operation     string                  // create/update
	ContentType   string                  // MIME type (e.g. image/jpeg, text/plain)
	ChunkPreTxHex string                  // Pre-built chunk funding transaction (contains inputs, signNull)
	IndexPreTxHex string                  // Pre-built index transaction (contains inputs, signNull)
	MergeTxHex    string                  // Optional merge transaction hex (creates two UTXOs, broadcast first)
	FeeRate       int64                   // Fee rate
	IsBroadcast   bool                    // Whether to broadcast automatically
	Task          *model.FileUploaderTask `json:"-"` // Associated async task (not exposed externally)
}

// EstimateChunkedUpload estimates fees for chunked upload.
func (s *UploadService) EstimateChunkedUpload(req *EstimateChunkedUploadRequest) (*EstimateChunkedUploadResponse, error) {
	// Validate parameters
	if len(req.Content) == 0 {
		return nil, fmt.Errorf("file content is empty")
	}
	if req.Path == "" {
		return nil, fmt.Errorf("file path is required")
	}

	// Apply defaults
	if req.ContentType == "" {
		req.ContentType = "application/octet-stream"
	}
	feeRate := req.FeeRate
	if feeRate == 0 {
		feeRate = conf.Cfg.Uploader.FeeRate
	}

	chunkSize := conf.Cfg.Uploader.ChunkSize
	fmt.Println("Conf - chunkSize:", chunkSize)
	if chunkSize <= 0 {
		chunkSize = 2000 * 1024 // default 2000 KB
	}

	// Split file
	chunks := splitFile(req.Content, chunkSize)
	chunkNumber := len(chunks)

	// Build chunk path
	chunkPath := fmt.Sprintf("%s/file/_chunk", req.Path)
	if !strings.HasPrefix(chunkPath, "/") {
		chunkPath = "/" + chunkPath
	}

	// Estimate fee per chunk
	totalChunkFee := int64(0)
	perChunkFee := int64(0)
	chunkFees := make([]int64, 0, chunkNumber)

	for _, chunkData := range chunks {
		// Build chunk script to estimate size
		chunkScript, err := buildChunkOpReturnScript(chunkPath, chunkData)
		if err != nil {
			return nil, fmt.Errorf("failed to build chunk script for estimation: %w", err)
		}
		chunkFee := estimateChunkFundingValue(chunkScript, feeRate)
		chunkFees = append(chunkFees, chunkFee)
		totalChunkFee += chunkFee
	}

	// Average fee per chunk
	if chunkNumber > 0 {
		perChunkFee = totalChunkFee / int64(chunkNumber)
	}

	// Estimate fee for chunk funding transaction
	// chunkFundingTx contains 1 input + chunkNumber outputs
	const chunkFundingInputSize = 148  // P2PKH input with signature
	const chunkFundingOutputSize = 34  // P2PKH output
	estimatedChunkFundingTxSize := 4 + // version
		1 + // input count (varint)
		chunkFundingInputSize + // input
		1 + // output count (varint)
		chunkFundingOutputSize*chunkNumber + // outputs
		4 // locktime
	chunkFundingTxFee := int64(estimatedChunkFundingTxSize) * feeRate
	if chunkFundingTxFee < 600 {
		chunkFundingTxFee = 600
	}

	// ChunkPreTxFee must cover every chunk output + chunkFundingTx fee
	chunkPreTxFee := totalChunkFee + chunkFundingTxFee

	// Build index path
	indexPath := fmt.Sprintf("%s/file/index", req.Path)
	if !strings.HasPrefix(indexPath, "/") {
		indexPath = "/" + indexPath
	}

	// Estimate index fee
	// Build index payload first to know its size
	sha256hash := sha256.Sum256(req.Content)
	filehashStr := hex.EncodeToString(sha256hash[:])

	// Build chunk list using placeholder PinIDs
	chunkList := make([]struct {
		Sha256 string `json:"sha256"`
		PinId  string `json:"pinId"`
	}, 0, chunkNumber)
	for i, chunkData := range chunks {
		chunkHash := sha256.Sum256(chunkData)
		chunkHashStr := hex.EncodeToString(chunkHash[:])
		chunkList = append(chunkList, struct {
			Sha256 string `json:"sha256"`
			PinId  string `json:"pinId"`
		}{
			Sha256: chunkHashStr,
			PinId:  fmt.Sprintf("placeholder_%d", i), // placeholder
		})
	}

	metaFileIndex := metaid_protocols.MetaFileIndex{
		Sha256:      filehashStr,
		FileSize:    int64(len(req.Content)),
		ChunkNumber: chunkNumber,
		ChunkSize:   chunkSize,
		DataType:    req.ContentType,
		Name:        req.FileName,
		ChunkList:   chunkList,
	}

	indexData, err := json.Marshal(metaFileIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal index data for estimation: %w", err)
	}

	// Build index script
	indexScript, err := buildIndexOpReturnScript(indexPath, indexData)
	if err != nil {
		return nil, fmt.Errorf("failed to build index script for estimation: %w", err)
	}

	// Estimate index transaction fee
	// Index tx contains existing inputs + a user output + OP_RETURN
	const indexInputSize = 148 // P2PKH input with signature
	const indexOutputSize = 34 // P2PKH output (8 bytes value + ~26 bytes script)
	indexOpReturnSize := 8 + wire2.VarIntSerializeSize(uint64(len(indexScript))) + len(indexScript)
	// Assume there is 1 input (provided by user)
	indexTxSize := 4 + 1 + indexInputSize + 1 + indexOutputSize + indexOpReturnSize + 4
	indexFee := int64(indexTxSize) * feeRate
	if indexFee < 600 {
		indexFee = 600
	}

	totalFee := chunkPreTxFee + indexFee

	return &EstimateChunkedUploadResponse{
		ChunkNumber:   chunkNumber,
		ChunkSize:     chunkSize,
		ChunkFees:     chunkFees,
		ChunkPreTxFee: chunkPreTxFee, // covers chunk outputs + chunkFundingTx fee
		IndexPreTxFee: indexFee,
		TotalFee:      totalFee,
		PerChunkFee:   perChunkFee,
		Message:       "success",
	}, nil
}

// ChunkedUploadResponse represents the result of a chunked upload build.
type ChunkedUploadResponse struct {
	FileId         string   `json:"fileId"`         // File ID
	FileHash       string   `json:"fileHash"`       // File SHA256 hash
	FileMd5        string   `json:"fileMd5"`        // File MD5 hash
	ChunkNumber    int      `json:"chunkNumber"`    // Number of chunks
	ChunkFundingTx string   `json:"chunkFundingTx"` // Funding transaction for chunk outputs
	ChunkTxs       []string `json:"chunkTxs"`       // Chunk transaction hex list (ordered)
	ChunkTxIds     []string `json:"chunkTxIds"`     // Chunk transaction IDs (ordered)
	IndexTx        string   `json:"indexTx"`        // Index transaction hex
	IndexTxId      string   `json:"indexTxId"`      // Index transaction ID
	Status         string   `json:"status"`         // Status string
	Message        string   `json:"message"`        // Additional message
}

// ChunkedUpload splits a large file, builds chunk and index transactions, and optionally broadcasts them.
func (s *UploadService) ChunkedUpload(req *ChunkedUploadRequest) (*ChunkedUploadResponse, error) {
	// Validate parameters
	if len(req.Content) == 0 {
		return nil, fmt.Errorf("file content is empty")
	}
	if req.Path == "" {
		return nil, fmt.Errorf("file path is required")
	}
	if req.Address == "" {
		return nil, fmt.Errorf("user address is required")
	}
	if req.ChunkPreTxHex == "" {
		return nil, fmt.Errorf("chunk pre-tx hex is required")
	}
	if req.IndexPreTxHex == "" {
		return nil, fmt.Errorf("index pre-tx hex is required")
	}

	// Apply defaults
	if req.Operation == "" {
		req.Operation = "create"
	}
	if req.ContentType == "" {
		req.ContentType = "application/octet-stream"
	}
	if req.FeeRate == 0 {
		req.FeeRate = conf.Cfg.Uploader.FeeRate
	}

	// Load network parameters
	var netParam *chaincfg2.Params
	if conf.Cfg.Net == "mainnet" {
		netParam = &chaincfg2.MainNetParams
	} else {
		netParam = &chaincfg2.TestNet3Params
	}

	chunkSize := conf.Cfg.Uploader.ChunkSize
	if chunkSize <= 0 {
		chunkSize = 2000 * 1024
	}

	chunkFundingTx, err := decodeMvcTx(req.ChunkPreTxHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode chunk pre-tx: %w", err)
	}

	indexBaseTx, err := decodeMvcTx(req.IndexPreTxHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode index pre-tx: %w", err)
	}

	// Obtain or create assistant address
	assistent, err := s.getOrCreateFileAssistent(req.MetaId, req.Address, netParam)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare assistent address: %w", err)
	}

	// Calculate file hashes
	sha256hash := sha256.Sum256(req.Content)
	md5hash := md5.Sum(req.Content)
	filehashStr := hex.EncodeToString(sha256hash[:])
	md5hashStr := hex.EncodeToString(md5hash[:])

	// Build file ID
	fileId := req.MetaId + "_" + filehashStr

	// Split file
	chunks := splitFile(req.Content, chunkSize)
	chunkNumber := len(chunks)

	log.Printf("File split into %d chunks, file size: %d bytes", chunkNumber, len(req.Content))
	s.updateUploadTaskProgress(req.Task, fmt.Sprintf("File split completed, %d chunks total", chunkNumber), 30, 0)

	assistentAddress, err := bsvutil2.DecodeAddress(assistent.AssistentAddress, netParam)
	if err != nil {
		return nil, fmt.Errorf("failed to decode assistent address: %w", err)
	}
	assistentPkScript, err := txscript2.PayToAddrScript(assistentAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to build assistent pkScript: %w", err)
	}

	chunkScripts := make([][]byte, 0, chunkNumber)
	chunkInputs := make([]*common.TxInputUtxo, 0, chunkNumber)

	// Prepare chunk transactions
	chunkTxs := make([]string, 0, chunkNumber)
	chunkTxIds := make([]string, 0, chunkNumber)
	chunkList := make([]struct {
		Sha256 string `json:"sha256"`
		PinId  string `json:"pinId"`
	}, 0, chunkNumber)

	// Build a transaction for each chunk
	chunkPath := "/file/_chunk"
	// chunkPath := fmt.Sprintf("%s/file/_chunk", req.Path)
	// if !strings.HasPrefix(chunkPath, "/") {
	// 	chunkPath = "/" + chunkPath
	// }

	// Calculate scripts and required amounts for all chunks
	totalChunkOutputAmount := int64(0)
	chunkAmounts := make([]int64, 0, chunkNumber)
	for _, chunkData := range chunks {
		chunkScript, err := buildChunkOpReturnScript(chunkPath, chunkData)
		if err != nil {
			return nil, fmt.Errorf("failed to build chunk script: %w", err)
		}
		chunkScripts = append(chunkScripts, chunkScript)
		chunkAmount := estimateChunkFundingValue(chunkScript, req.FeeRate)
		chunkAmounts = append(chunkAmounts, chunkAmount)
		totalChunkOutputAmount += chunkAmount
	}

	// Estimate chunkFundingTx fee (same logic as estimation)
	const inputSize = 148              // P2PKH input with signature
	const outputSize = 34              // P2PKH output
	estimatedChunkFundingTxSize := 4 + // version
		1 + // input count (varint)
		inputSize + // input
		1 + // output count (varint)
		outputSize*chunkNumber + // outputs
		4 // locktime
	chunkFundingTxFee := int64(estimatedChunkFundingTxSize) * req.FeeRate
	if chunkFundingTxFee < 600 {
		chunkFundingTxFee = 600
	}

	// Try to fetch funding amount from merge transaction if provided
	var totalInputAmount int64 = 0
	if req.MergeTxHex != "" {
		mergeTx, err := decodeMvcTx(req.MergeTxHex)
		if err == nil {
			// Required amount = totalChunkOutputAmount + chunkFundingTxFee
			requiredAmount := totalChunkOutputAmount + chunkFundingTxFee
			// Find an output that matches the required amount
			for i, output := range mergeTx.TxOut {
				outputAmount := int64(output.Value)
				// Allow a tolerance of 1000 satoshis
				if outputAmount >= requiredAmount-1000 && outputAmount <= requiredAmount+1000 {
					totalInputAmount = outputAmount
					log.Printf("Found chunkPreTx output at index %d: %d satoshis (required: %d)", i, outputAmount, requiredAmount)
					break
				}
			}
		}
	}

	// If merge tx not provided or not sufficient, fall back to estimated amount
	if totalInputAmount == 0 {
		totalInputAmount = totalChunkOutputAmount + chunkFundingTxFee
		log.Printf("Using estimated totalInputAmount: %d satoshis (chunkOutputs: %d + fee: %d)",
			totalInputAmount, totalChunkOutputAmount, chunkFundingTxFee)
	}

	// Validate available amount
	availableAmount := totalInputAmount - chunkFundingTxFee
	if availableAmount < totalChunkOutputAmount {
		return nil, fmt.Errorf("insufficient input amount: need %d satoshis (outputs: %d + fee: %d), but only have %d satoshis available",
			totalChunkOutputAmount+chunkFundingTxFee, totalChunkOutputAmount, chunkFundingTxFee, availableAmount)
	}

	// Add outputs (use original amount, leftover becomes dust)
	for _, chunkAmount := range chunkAmounts {
		chunkFundingTx.AddTxOut(wire2.NewTxOut(chunkAmount, assistentPkScript))

		chunkInputs = append(chunkInputs, &common.TxInputUtxo{
			TxId:     "", // filled later
			TxIndex:  int64(len(chunkFundingTx.TxOut) - 1),
			PkScript: hex.EncodeToString(assistentPkScript),
			Amount:   uint64(chunkAmount),
			PriHex:   assistent.AssistentPriHex,
		})
	}

	log.Printf("ChunkFundingTx: input=%d, fee=%d, outputs=%d (total=%d), remaining=%d",
		totalInputAmount, chunkFundingTxFee, totalChunkOutputAmount, totalChunkOutputAmount, availableAmount-totalChunkOutputAmount)

	chunkFundingTxHex, err := common.MvcToRaw(chunkFundingTx)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize chunk funding tx: %w", err)
	}
	chunkFundingTxHash := common.GetMvcTxhashFromRaw(chunkFundingTxHex)

	for i := range chunkInputs {
		chunkInputs[i].TxId = chunkFundingTxHash
	}

	for i, chunkData := range chunks {
		// Calculate chunk hash
		chunkHash := sha256.Sum256(chunkData)
		chunkHashStr := hex.EncodeToString(chunkHash[:])

		// Build chunk transaction using assistant UTXO
		chunkTx, err := s.buildChunkTxWithFunding(
			chunkInputs[i],
			chunkScripts[i],
		)
		if err != nil {
			return nil, fmt.Errorf("failed to build chunk %d transaction: %w", i, err)
		}

		// Serialize transaction
		chunkTxHex, err := common.MvcToRaw(chunkTx)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize chunk %d transaction: %w", i, err)
		}

		// Derive transaction hash (used as PinID)
		chunkTxId := common.GetMvcTxhashFromRaw(chunkTxHex)
		chunkPinId := fmt.Sprintf("%si0", chunkTxId)

		chunkTxs = append(chunkTxs, chunkTxHex)
		chunkTxIds = append(chunkTxIds, chunkTxId)
		chunkList = append(chunkList, struct {
			Sha256 string `json:"sha256"`
			PinId  string `json:"pinId"`
		}{
			Sha256: chunkHashStr,
			PinId:  chunkPinId,
		})

		// Calculate chunk MD5
		chunkMd5Hash := md5.Sum(chunkData)
		chunkMd5Str := hex.EncodeToString(chunkMd5Hash[:])

		// Check if chunk already exists before creating
		existingChunk, err := s.fileChunkDAO.GetByTxID(chunkTxId)
		if err != nil && err != gorm.ErrRecordNotFound {
			log.Printf("Failed to check existing chunk %d: %v", i, err)
			// Continue processing other chunks even if check fails
		} else if existingChunk != nil {
			log.Printf("File chunk %d already exists: hash=%s, txId=%s, skipping create", i, chunkHashStr, chunkTxId)
		} else {
			// Persist chunk metadata
			fileChunk := &model.FileChunk{
				ChunkHash:   chunkHashStr,
				ChunkSize:   int64(len(chunkData)),
				ChunkMd5:    chunkMd5Str,
				ChunkIndex:  int64(i),
				FileHash:    filehashStr,
				TxID:        chunkTxId,
				PinId:       chunkPinId,
				Path:        chunkPath,
				ContentType: metaid_protocols.MonitorMetaIdFileChunkContentType + ";binary",
				Size:        int64(len(chunkData)),
				StorageType: conf.Cfg.Storage.Type,
				StoragePath: "", // Optional: set if persisting chunk content externally
				Operation:   req.Operation,
				// TxRaw:       chunkTxHex,
				Status: model.StatusPending,
			}

			if err := s.fileChunkDAO.Create(fileChunk); err != nil {
				log.Printf("Failed to save file chunk %d: %v", i, err)
				// Continue processing other chunks even if persistence fails
			} else {
				log.Printf("File chunk %d saved: hash=%s, txId=%s", i, chunkHashStr, chunkTxId)
			}
		}

		log.Printf("Chunk %d/%d built: size=%d, hash=%s, pinId=%s", i+1, chunkNumber, len(chunkData), chunkHashStr, chunkPinId)
		progress := calcProgressRange(30, 70, i+1, chunkNumber)
		s.updateUploadTaskProgress(req.Task,
			fmt.Sprintf("Building chunk transactions (%d/%d)", i+1, chunkNumber),
			progress,
			i+1)
	}

	// Build index metadata payload
	s.updateUploadTaskProgress(req.Task, "Chunk transactions built, preparing index", 75, len(chunkTxIds))
	metaFileIndex := metaid_protocols.MetaFileIndex{
		Sha256:      filehashStr,
		FileSize:    int64(len(req.Content)),
		ChunkNumber: chunkNumber,
		ChunkSize:   chunkSize,
		DataType:    req.ContentType,
		Name:        req.FileName,
		ChunkList:   chunkList,
	}

	// Serialize index metadata
	indexData, err := json.Marshal(metaFileIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal index data: %w", err)
	}

	// Build index path
	indexPath := "/file/index"
	// indexPath := fmt.Sprintf("%s/file/index", req.Path)
	// if !strings.HasPrefix(indexPath, "/") {
	// 	indexPath = "/" + indexPath
	// }

	indexScript, err := buildIndexOpReturnScript(indexPath, indexData)
	if err != nil {
		return nil, fmt.Errorf("failed to build index script: %w", err)
	}

	indexTx, err := buildIndexTxFromPreTx(netParam, indexBaseTx, req.Address, indexScript)
	if err != nil {
		return nil, fmt.Errorf("failed to build index transaction: %w", err)
	}

	// Serialize index transaction
	indexTxHex, err := common.MvcToRaw(indexTx)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize index transaction: %w", err)
	}
	indexTxId := common.GetMvcTxhashFromRaw(indexTxHex)

	log.Printf("Index transaction built: fileHash=%s, chunkNumber=%d", filehashStr, chunkNumber)
	s.updateUploadTaskProgress(req.Task, "Index transaction built", 80, len(chunkTxIds))

	// Check if the file already exists
	existingFile, err := s.fileDAO.GetByFileID(fileId)
	if err == nil && existingFile != nil {
		// File exists; handle based on status
		if existingFile.Status == model.StatusSuccess {
			// Already on-chain, return existing info
			log.Printf("File already exists and uploaded successfully: FileId=%s", fileId)
			return &ChunkedUploadResponse{
				FileId:      existingFile.FileId,
				FileHash:    existingFile.FileHash,
				FileMd5:     existingFile.FileMd5,
				ChunkNumber: chunkNumber,
				IndexTxId:   existingFile.TxID,
				Status:      string(existingFile.Status),
				Message:     "file already exists and uploaded",
			}, nil
		}
		// Pending/failed -> continue processing and update record
		if existingFile.Status == model.StatusPending {
			log.Printf("File already exists in pending status, will update and retry: FileId=%s", fileId)
		} else {
			log.Printf("File exists but failed, will update and retry: FileId=%s", fileId)
		}
	}

	// Prepare file metadata entry
	file := &model.File{
		FileId:          fileId,
		FileName:        req.FileName,
		FileType:        strings.ReplaceAll(req.ContentType, ";binary", ""),
		MetaId:          req.MetaId,
		Address:         req.Address,
		Path:            indexPath,
		ContentType:     metaid_protocols.MonitorMetaIdFileIndexContentType + ";utf-8",
		FileSize:        int64(len(req.Content)),
		FileHash:        filehashStr,
		FileMd5:         md5hashStr,
		FileContentType: req.ContentType,
		ChunkType:       model.ChunkTypeMulti,
		Operation:       req.Operation,
		Status:          model.StatusPending,
	}

	// Update existing record or create a new one
	if existingFile != nil {
		// Update existing record
		file.ID = existingFile.ID // Preserve original ID
		if err := s.fileDAO.Update(file); err != nil {
			return nil, fmt.Errorf("failed to update file metadata: %w", err)
		}
		log.Printf("File metadata updated: FileId=%s, status=pending", fileId)
	} else {
		// Create a new record
		if err := s.fileDAO.Create(file); err != nil {
			return nil, fmt.Errorf("failed to save file metadata: %w", err)
		}
		log.Printf("File metadata saved: FileId=%s, status=pending", fileId)
	}

	// Broadcast all transactions when requested
	if req.IsBroadcast {
		chain := conf.Cfg.Net
		finalStatus := model.StatusSuccess
		finalMessage := "success"

		s.updateUploadTaskProgress(req.Task, "Broadcasting transactions", 82, len(chunkTxIds))

		// Use DB transaction to ensure consistency
		err := database.UploaderDB.Transaction(func(tx *gorm.DB) error {
			// 0. Broadcast merge transaction first if provided
			if req.MergeTxHex != "" {
				log.Printf("Broadcasting merge transaction first...")
				mergeTxId, err := node.BroadcastTx(chain, req.MergeTxHex)
				if err != nil {
					log.Printf("Failed to broadcast merge transaction: %v", err)
					// Mark file as failed
					if updateErr := tx.Model(&model.File{}).Where("file_id = ?", fileId).Update("status", model.StatusFailed).Error; updateErr != nil {
						return fmt.Errorf("failed to update file status: %w", updateErr)
					}
					// Mark all chunks as failed
					if updateErr := tx.Model(&model.FileChunk{}).Where("file_hash = ?", filehashStr).Update("status", model.StatusFailed).Error; updateErr != nil {
						log.Printf("Failed to update chunk status: %v", updateErr)
					}
					return fmt.Errorf("failed to broadcast merge transaction: %w", err)
				}
				log.Printf("Merge transaction broadcasted successfully: %s", mergeTxId)
			}

			// 1. Broadcast chunk funding transaction
			log.Printf("Broadcasting chunk funding transaction: %s", chunkFundingTxHash)
			broadcastFundingTxID, err := node.BroadcastTx(chain, chunkFundingTxHex)
			if err != nil {
				fmt.Printf("tx hex: %s\n", chunkFundingTxHex)
				log.Printf("Failed to broadcast chunk funding transaction: %v", err)
				// Mark file as failed
				if updateErr := tx.Model(&model.File{}).Where("file_id = ?", fileId).Update("status", model.StatusFailed).Error; updateErr != nil {
					return fmt.Errorf("failed to update file status: %w", updateErr)
				}
				// Mark all chunks as failed
				if updateErr := tx.Model(&model.FileChunk{}).Where("file_hash = ?", filehashStr).Update("status", model.StatusFailed).Error; updateErr != nil {
					log.Printf("Failed to update chunk status: %v", updateErr)
				}
				return fmt.Errorf("failed to broadcast chunk funding transaction: %w", err)
			}
			log.Printf("Chunk funding transaction broadcasted successfully: %s", broadcastFundingTxID)
			s.updateUploadTaskProgress(req.Task, "ChunkFunding transaction broadcasted", 85, len(chunkTxIds))

			// 2. Broadcast each chunk transaction sequentially
			for i, chunkTxHex := range chunkTxs {
				log.Printf("Broadcasting chunk transaction %d/%d: %s", i+1, chunkNumber, chunkTxIds[i])
				broadcastChunkTxID, err := node.BroadcastTx(chain, chunkTxHex)
				if err != nil {
					fmt.Printf("tx hex: %s\n", chunkTxHex)
					log.Printf("Failed to broadcast chunk transaction %d: %v", i, err)
					// Mark file as failed
					if updateErr := tx.Model(&model.File{}).Where("file_id = ?", fileId).Update("status", model.StatusFailed).Error; updateErr != nil {
						return fmt.Errorf("failed to update file status: %w", updateErr)
					}
					// Mark all chunks as failed
					if updateErr := tx.Model(&model.FileChunk{}).Where("file_hash = ?", filehashStr).Update("status", model.StatusFailed).Error; updateErr != nil {
						log.Printf("Failed to update chunk status: %v", updateErr)
					}
					return fmt.Errorf("failed to broadcast chunk transaction %d: %w", i, err)
				}
				log.Printf("Chunk transaction %d/%d broadcasted successfully: %s", i+1, chunkNumber, broadcastChunkTxID)
				progress := calcProgressRange(85, 95, i+1, chunkNumber)
				s.updateUploadTaskProgress(req.Task,
					fmt.Sprintf("Broadcasting chunk transactions (%d/%d)", i+1, chunkNumber),
					progress,
					len(chunkTxIds))

				// Update chunk status
				if updateErr := tx.Model(&model.FileChunk{}).
					Where("pin_id = ?", fmt.Sprintf("%si0", chunkTxIds[i])).
					Update("status", model.StatusSuccess).Error; updateErr != nil {
					log.Printf("Failed to update chunk %d status: %v", i, updateErr)
					// Ignore update failures and continue
				}
			}

			time.Sleep(5 * time.Second)
			// 3. Broadcast index transaction
			s.updateUploadTaskProgress(req.Task, "Preparing to broadcast index transaction", 96, len(chunkTxIds))
			log.Printf("Broadcasting index transaction: %s", indexTxId)
			broadcastIndexTxID, err := node.BroadcastTx(chain, indexTxHex)
			if err != nil {
				fmt.Printf("tx hex: %s\n", indexTxHex)
				log.Printf("Failed to broadcast index transaction: %v", err)
				// Mark file as failed
				if updateErr := tx.Model(&model.File{}).Where("file_id = ?", fileId).Update("status", model.StatusFailed).Error; updateErr != nil {
					return fmt.Errorf("failed to update file status: %w", updateErr)
				}
				// Mark all chunks as failed
				if updateErr := tx.Model(&model.FileChunk{}).Where("file_hash = ?", filehashStr).Update("status", model.StatusFailed).Error; updateErr != nil {
					log.Printf("Failed to update chunk status: %v", updateErr)
				}
				return fmt.Errorf("failed to broadcast index transaction: %w", err)
			}
			log.Printf("Index transaction broadcasted successfully: %s", broadcastIndexTxID)
			s.updateUploadTaskProgress(req.Task, "Index transaction broadcasted", 98, len(chunkTxIds))

			// 4. Mark file as success
			if updateErr := tx.Model(&model.File{}).
				Where("file_id = ?", fileId).
				Updates(map[string]interface{}{
					"status": model.StatusSuccess,
					"tx_id":  indexTxId,
					"pin_id": fmt.Sprintf("%si0", indexTxId),
				}).Error; updateErr != nil {
				return fmt.Errorf("failed to update file status: %w", updateErr)
			}

			log.Printf("All transactions broadcasted successfully for file: %s", fileId)
			s.updateUploadTaskProgress(req.Task, "Broadcast finished, finalizing task", 99, len(chunkTxIds))
			return nil
		})

		if err != nil {
			log.Printf("Failed to broadcast transactions: %v", err)
			finalStatus = model.StatusFailed
			finalMessage = fmt.Sprintf("broadcast failed: %v", err)
		} else {
			finalStatus = model.StatusSuccess
			finalMessage = "all transactions broadcasted successfully"
		}

		return &ChunkedUploadResponse{
			FileId:         fileId,
			FileHash:       filehashStr,
			FileMd5:        md5hashStr,
			ChunkNumber:    chunkNumber,
			ChunkFundingTx: chunkFundingTxHex,
			// ChunkTxs:       chunkTxs,
			ChunkTxIds: chunkTxIds,
			// IndexTx:        indexTxHex,
			IndexTxId: indexTxId,
			Status:    string(finalStatus),
			Message:   finalMessage,
		}, nil
	}

	s.updateUploadTaskProgress(req.Task, "Chunk transactions ready, waiting to broadcast", 85, len(chunkTxIds))

	return &ChunkedUploadResponse{
		FileId:         fileId,
		FileHash:       filehashStr,
		FileMd5:        md5hashStr,
		ChunkNumber:    chunkNumber,
		ChunkFundingTx: chunkFundingTxHex,
		ChunkTxs:       chunkTxs,
		ChunkTxIds:     chunkTxIds,
		IndexTx:        indexTxHex,
		IndexTxId:      indexTxId,
		Status:         string(model.StatusPending),
		Message:        "success",
	}, nil
}

// ChunkedUploadForTaskResponse describes the response when creating an async task.
type ChunkedUploadForTaskResponse struct {
	TaskId  string `json:"taskId"`  // Task ID
	Status  string `json:"status"`  // Task status
	Message string `json:"message"` // Additional message
}

// UploadTaskListResponse paginated task list
type UploadTaskListResponse struct {
	Tasks      []*model.FileUploaderTask `json:"tasks"`
	NextCursor int64                     `json:"nextCursor"`
	HasMore    bool                      `json:"hasMore"`
}

// ChunkedUploadForTask creates an async chunked upload task and returns its ID.
func (s *UploadService) ChunkedUploadForTask(req *ChunkedUploadRequest) (*ChunkedUploadForTaskResponse, error) {
	// Validate parameters
	if len(req.Content) == 0 {
		return nil, fmt.Errorf("file content is empty")
	}
	if req.Path == "" {
		return nil, fmt.Errorf("file path is required")
	}
	if req.Address == "" {
		return nil, fmt.Errorf("user address is required")
	}
	if req.ChunkPreTxHex == "" {
		return nil, fmt.Errorf("chunk pre-tx hex is required")
	}
	if req.IndexPreTxHex == "" {
		return nil, fmt.Errorf("index pre-tx hex is required")
	}

	// Apply defaults
	if req.Operation == "" {
		req.Operation = "create"
	}
	if req.ContentType == "" {
		req.ContentType = "application/octet-stream"
	}
	if req.FeeRate == 0 {
		req.FeeRate = conf.Cfg.Uploader.FeeRate
	}

	// Calculate file hashes
	sha256hash := sha256.Sum256(req.Content)
	md5hash := md5.Sum(req.Content)
	filehashStr := hex.EncodeToString(sha256hash[:])
	md5hashStr := hex.EncodeToString(md5hash[:])

	// Generate file ID
	fileId := req.MetaId + "_" + filehashStr

	// Determine chunk count
	chunkSize := conf.Cfg.Uploader.ChunkSize
	if chunkSize <= 0 {
		chunkSize = 2000 * 1024
	}
	chunks := splitFile(req.Content, chunkSize)
	chunkNumber := len(chunks)

	// Encode content as base64
	contentBase64 := base64.StdEncoding.EncodeToString(req.Content)

	// Generate task ID
	taskId := fmt.Sprintf("task_%s_%d", filehashStr[:16], time.Now().Unix())

	// Serialize empty chunk ID list placeholder
	chunkTxIdsJSON, _ := json.Marshal([]string{})

	// Create task record
	task := &model.FileUploaderTask{
		TaskId:          taskId,
		MetaId:          req.MetaId,
		Address:         req.Address,
		FileName:        req.FileName,
		FileHash:        filehashStr,
		FileMd5:         md5hashStr,
		FileSize:        int64(len(req.Content)),
		ContentType:     req.ContentType,
		Path:            req.Path,
		Operation:       req.Operation,
		ContentBase64:   contentBase64,
		ChunkPreTxHex:   req.ChunkPreTxHex,
		IndexPreTxHex:   req.IndexPreTxHex,
		MergeTxHex:      req.MergeTxHex,
		FeeRate:         req.FeeRate,
		Status:          model.StatusPending,
		Progress:        0,
		TotalChunks:     chunkNumber,
		ProcessedChunks: 0,
		CurrentStep:     "Task created, waiting to process",
		FileId:          fileId,
		ChunkTxIds:      string(chunkTxIdsJSON),
	}

	if err := s.fileUploaderTaskDAO.Create(task); err != nil {
		return nil, fmt.Errorf("failed to create upload task: %w", err)
	}

	log.Printf("Created chunked upload task: taskId=%s, fileId=%s, chunkNumber=%d", taskId, fileId, chunkNumber)

	return &ChunkedUploadForTaskResponse{
		TaskId:  taskId,
		Status:  string(model.StatusPending),
		Message: "Task created, processing",
	}, nil
}

// ListTasksByAddress returns paginated tasks for address.
func (s *UploadService) ListTasksByAddress(address string, cursor int64, size int) (*UploadTaskListResponse, error) {
	if address == "" {
		return nil, fmt.Errorf("address is required")
	}
	if size <= 0 || size > 100 {
		size = 20
	}

	tasks, nextCursor, err := s.fileUploaderTaskDAO.ListByAddressWithCursor(address, cursor, size)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	return &UploadTaskListResponse{
		Tasks:      tasks,
		NextCursor: nextCursor,
		HasMore:    len(tasks) == size,
	}, nil
}

// ProcessUploadTask executes the async upload task in the background.
func (s *UploadService) ProcessUploadTask(task *model.FileUploaderTask) error {
	// Mark as processing
	now := time.Now()
	task.Status = "processing"
	task.StartedAt = &now
	task.CurrentStep = "Processing started"
	task.Progress = 5
	if err := s.fileUploaderTaskDAO.Update(task); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	// Decode file content
	content, err := base64.StdEncoding.DecodeString(task.ContentBase64)
	if err != nil {
		task.Status = model.StatusFailed
		task.ErrorMessage = fmt.Sprintf("failed to decode content: %v", err)
		task.Progress = 0
		s.clearTaskPayload(task)
		s.fileUploaderTaskDAO.Update(task)
		return fmt.Errorf("failed to decode content: %w", err)
	}

	// Build service request
	chunkedReq := &ChunkedUploadRequest{
		MetaId:        task.MetaId,
		Address:       task.Address,
		FileName:      task.FileName,
		Content:       content,
		Path:          task.Path,
		Operation:     task.Operation,
		ContentType:   task.ContentType,
		ChunkPreTxHex: task.ChunkPreTxHex,
		IndexPreTxHex: task.IndexPreTxHex,
		MergeTxHex:    task.MergeTxHex,
		FeeRate:       task.FeeRate,
		IsBroadcast:   false, // chunkedUploadOnTask will drive broadcasting
	}

	// Update progress
	task.CurrentStep = "Starting chunk transaction build"
	task.Progress = 10
	s.fileUploaderTaskDAO.Update(task)

	// Build/broadcast with resumable logic
	resp, err := s.chunkedUploadOnTask(chunkedReq, task)
	if err != nil {
		task.Status = model.StatusFailed
		task.ErrorMessage = err.Error()
		// task.Progress = 0
		finishedAt := time.Now()
		task.FinishedAt = &finishedAt
		s.clearTaskPayload(task)
		s.fileUploaderTaskDAO.Update(task)
		return fmt.Errorf("failed to process chunked upload: %w", err)
	}

	// Persist task result
	task.Status = model.StatusSuccess
	task.Progress = 100
	task.CurrentStep = "Task completed"
	task.FileId = resp.FileId
	task.ChunkFundingTx = resp.ChunkFundingTx
	chunkTxIdsJSON, _ := json.Marshal(resp.ChunkTxIds)
	task.ChunkTxIds = string(chunkTxIdsJSON)
	task.IndexTxId = resp.IndexTxId
	finishedAt := time.Now()
	task.FinishedAt = &finishedAt
	s.clearTaskPayload(task)

	if err := s.fileUploaderTaskDAO.Update(task); err != nil {
		return fmt.Errorf("failed to update task result: %w", err)
	}

	log.Printf("Task processed successfully: taskId=%s, fileId=%s", task.TaskId, resp.FileId)
	return nil
}

// chunkedUploadOnTask executes chunked upload steps with resumable stages.
func (s *UploadService) chunkedUploadOnTask(req *ChunkedUploadRequest, task *model.FileUploaderTask) (*ChunkedUploadResponse, error) {
	if task == nil {
		return nil, fmt.Errorf("task is nil")
	}
	if task.Stage == "" {
		task.Stage = model.TaskStageCreated
	}

	// Stage 1: build transactions
	if task.Stage == model.TaskStageCreated {
		if err := s.prepareChunkedUploadForTask(req, task); err != nil {
			return nil, err
		}
	}

	chunkTxHexes, err := decodeStringArray(task.ChunkTxHexes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse chunk tx hex cache: %w", err)
	}
	chunkTxIds, err := decodeStringArray(task.ChunkTxIds)
	if err != nil {
		return nil, fmt.Errorf("failed to parse chunk tx id cache: %w", err)
	}

	// Stage 2: broadcast merge tx (if any)
	if task.Stage == model.TaskStagePrepared {
		if err := s.broadcastMergeTxForTask(task); err != nil {
			return nil, err
		}
	}

	// Stage 3: broadcast chunk funding tx
	if task.Stage == model.TaskStageMergeBroadcast {
		if err := s.broadcastFundingTxForTask(task); err != nil {
			return nil, err
		}
	}

	// Stage 4: broadcast chunk tx
	if task.Stage == model.TaskStageFundingBroadcast {
		if err := s.broadcastChunkTransactionsForTask(task, chunkTxHexes, chunkTxIds); err != nil {
			return nil, err
		}
	}

	// Stage 5: broadcast index tx
	if task.Stage == model.TaskStageChunkBroadcast {
		return s.broadcastIndexTxForTask(req, task, chunkTxIds)
	}

	// Already broadcast index tx; return success
	if task.Stage == model.TaskStageIndexBroadcast || task.Stage == model.TaskStageCompleted {
		return &ChunkedUploadResponse{
			FileId:      task.FileId,
			FileHash:    task.FileHash,
			FileMd5:     task.FileMd5,
			ChunkNumber: task.TotalChunks,
			ChunkTxIds:  chunkTxIds,
			IndexTxId:   task.IndexTxId,
			Status:      string(model.StatusSuccess),
			Message:     "success",
		}, nil
	}

	return nil, fmt.Errorf("unknown task stage: %s", task.Stage)
}

func (s *UploadService) prepareChunkedUploadForTask(req *ChunkedUploadRequest, task *model.FileUploaderTask) error {
	reqCopy := *req
	reqCopy.IsBroadcast = false
	reqCopy.Task = task

	s.updateUploadTaskProgress(task, "Building chunk transactions", 20, 0)

	resp, err := s.ChunkedUpload(&reqCopy)
	if err != nil {
		return err
	}

	chunkTxHexJSON, err := json.Marshal(resp.ChunkTxs)
	if err != nil {
		return fmt.Errorf("failed to marshal chunk tx hex list: %w", err)
	}
	chunkTxIdsJSON, err := json.Marshal(resp.ChunkTxIds)
	if err != nil {
		return fmt.Errorf("failed to marshal chunk tx ids: %w", err)
	}

	task.ChunkFundingTx = resp.ChunkFundingTx
	task.ChunkTxHexes = string(chunkTxHexJSON)
	task.ChunkTxIds = string(chunkTxIdsJSON)
	task.FileId = resp.FileId
	task.FileHash = resp.FileHash
	task.FileMd5 = resp.FileMd5
	task.TotalChunks = resp.ChunkNumber
	task.ProcessedChunks = 0
	task.Stage = model.TaskStagePrepared

	s.updateUploadTaskProgress(task, "Chunk transactions prepared", 40, 0)
	return s.fileUploaderTaskDAO.Update(task)
}

func (s *UploadService) broadcastMergeTxForTask(task *model.FileUploaderTask) error {
	chain := conf.Cfg.Net
	mergeHex := strings.TrimSpace(task.MergeTxHex)

	return database.UploaderDB.Transaction(func(tx *gorm.DB) error {
		task.Stage = model.TaskStageMergeBroadcast
		if err := tx.Model(&model.FileUploaderTask{}).
			Where("id = ?", task.ID).
			Updates(map[string]interface{}{
				"stage":        task.Stage,
				"merge_tx_hex": task.MergeTxHex,
			}).Error; err != nil {
			return err
		}

		if mergeHex != "" {
			if _, err := node.BroadcastTx(chain, mergeHex); err != nil {
				if !isDuplicateBroadcastError(err) {
					return fmt.Errorf("failed to broadcast merge transaction: %w", err)
				}
			}
			task.MergeTxHex = ""
		} else {
			log.Printf("Merge transaction already broadcasted or not required for taskId=%s", task.TaskId)
		}

		return nil
	})
}

func (s *UploadService) broadcastFundingTxForTask(task *model.FileUploaderTask) error {
	fundingHex := strings.TrimSpace(task.ChunkFundingTx)
	if fundingHex == "" {
		return fmt.Errorf("chunk funding transaction missing")
	}

	chain := conf.Cfg.Net
	return database.UploaderDB.Transaction(func(tx *gorm.DB) error {
		task.Stage = model.TaskStageFundingBroadcast
		if err := tx.Model(&model.FileUploaderTask{}).
			Where("id = ?", task.ID).
			Update("stage", task.Stage).Error; err != nil {
			return err
		}

		if _, err := node.BroadcastTx(chain, fundingHex); err != nil {
			if !isDuplicateBroadcastError(err) {
				return fmt.Errorf("failed to broadcast chunk funding transaction: %w", err)
			}
		}

		return nil
	})
}

func (s *UploadService) broadcastChunkTransactionsForTask(task *model.FileUploaderTask, chunkTxHexes, chunkTxIds []string) error {
	if len(chunkTxHexes) == 0 {
		return fmt.Errorf("chunk transaction cache empty")
	}
	if len(chunkTxIds) < len(chunkTxHexes) {
		missing := make([]string, len(chunkTxHexes)-len(chunkTxIds))
		chunkTxIds = append(chunkTxIds, missing...)
	}

	total := len(chunkTxHexes)
	start := task.ProcessedChunks
	if start < 0 {
		start = 0
	}

	for i := start; i < total; i++ {
		if err := s.broadcastSingleChunkTx(task, chunkTxHexes, chunkTxIds, i, total); err != nil {
			return err
		}
	}

	task.Stage = model.TaskStageChunkBroadcast
	if encoded, err := json.Marshal(chunkTxIds); err == nil {
		task.ChunkTxIds = string(encoded)
	}
	s.updateUploadTaskProgress(task, "Chunk transactions broadcasted", 95, task.ProcessedChunks)
	return s.fileUploaderTaskDAO.Update(task)
}

func (s *UploadService) broadcastSingleChunkTx(task *model.FileUploaderTask, chunkTxHexes, chunkTxIds []string, index, total int) error {
	chain := conf.Cfg.Net
	txHex := chunkTxHexes[index]

	return database.UploaderDB.Transaction(func(tx *gorm.DB) error {
		txID := common.GetMvcTxhashFromRaw(txHex)
		chunkTxIds[index] = txID

		pinID := fmt.Sprintf("%si0", txID)
		if err := tx.Model(&model.FileChunk{}).
			Where("pin_id = ?", pinID).
			Update("status", model.StatusSuccess).Error; err != nil {
			return err
		}

		processed := index + 1
		progress := calcProgressRange(85, 95, processed, total)
		step := fmt.Sprintf("Broadcasting chunk transactions (%d/%d)", processed, total)

		chunkTxIdsJSON, err := json.Marshal(chunkTxIds)
		if err != nil {
			return err
		}

		if err := tx.Model(&model.FileUploaderTask{}).
			Where("id = ?", task.ID).
			Updates(map[string]interface{}{
				"processed_chunks": processed,
				"progress":         progress,
				"current_step":     step,
				"chunk_tx_ids":     string(chunkTxIdsJSON),
			}).Error; err != nil {
			return err
		}

		_, err = node.BroadcastTx(chain, txHex)
		if err != nil {
			if !isDuplicateBroadcastError(err) {
				return fmt.Errorf("failed to broadcast chunk transaction %d: %w", index, err)
			}

		}

		task.ProcessedChunks = processed
		task.Progress = progress
		task.CurrentStep = step
		task.ChunkTxIds = string(chunkTxIdsJSON)
		return nil
	})
}

func (s *UploadService) broadcastIndexTxForTask(req *ChunkedUploadRequest, task *model.FileUploaderTask, chunkTxIds []string) (*ChunkedUploadResponse, error) {
	if len(req.Content) == 0 {
		return nil, fmt.Errorf("task content is empty")
	}
	if len(chunkTxIds) == 0 {
		return nil, fmt.Errorf("chunk transaction ID cache empty")
	}

	chunkSize := conf.Cfg.Uploader.ChunkSize
	if chunkSize <= 0 {
		chunkSize = 2000 * 1024
	}
	chunks := splitFile(req.Content, chunkSize)
	if len(chunks) != len(chunkTxIds) {
		return nil, fmt.Errorf("chunk tx count mismatch: have %d ids, expect %d chunks", len(chunkTxIds), len(chunks))
	}

	filehash := task.FileHash
	if filehash == "" {
		sha := sha256.Sum256(req.Content)
		filehash = hex.EncodeToString(sha[:])
	}

	chunkList := make([]struct {
		Sha256 string `json:"sha256"`
		PinId  string `json:"pinId"`
	}, 0, len(chunks))
	for i, chunkData := range chunks {
		chunkHash := sha256.Sum256(chunkData)
		chunkHashStr := hex.EncodeToString(chunkHash[:])
		pinID := fmt.Sprintf("%si0", chunkTxIds[i])
		chunkList = append(chunkList, struct {
			Sha256 string `json:"sha256"`
			PinId  string `json:"pinId"`
		}{
			Sha256: chunkHashStr,
			PinId:  pinID,
		})
	}

	metaFileIndex := metaid_protocols.MetaFileIndex{
		Sha256:      filehash,
		FileSize:    int64(len(req.Content)),
		ChunkNumber: len(chunks),
		ChunkSize:   chunkSize,
		DataType:    req.ContentType,
		Name:        req.FileName,
		ChunkList:   chunkList,
	}
	indexData, err := json.Marshal(metaFileIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal index metadata: %w", err)
	}

	indexScript, err := buildIndexOpReturnScript("/file/index", indexData)
	if err != nil {
		return nil, fmt.Errorf("failed to build index script: %w", err)
	}

	indexPreTx, err := decodeMvcTx(req.IndexPreTxHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode index pre-tx: %w", err)
	}

	var netParam *chaincfg2.Params
	if conf.Cfg.Net == "mainnet" {
		netParam = &chaincfg2.MainNetParams
	} else {
		netParam = &chaincfg2.TestNet3Params
	}

	indexTx, err := buildIndexTxFromPreTx(netParam, indexPreTx, req.Address, indexScript)
	if err != nil {
		return nil, fmt.Errorf("failed to build index transaction: %w", err)
	}

	indexTxHex, err := common.MvcToRaw(indexTx)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize index tx: %w", err)
	}
	indexTxId := common.GetMvcTxhashFromRaw(indexTxHex)
	fileId := task.FileId
	if fileId == "" {
		return nil, fmt.Errorf("file ID missing in task")
	}

	// Use transaction to ensure atomicity of broadcast and updates
	err = database.UploaderDB.Transaction(func(tx *gorm.DB) error {
		// Broadcast index transaction
		if _, err := node.BroadcastTx(conf.Cfg.Net, indexTxHex); err != nil {
			if !isDuplicateBroadcastError(err) {
				// Mark file as failed on broadcast error
				if updateErr := tx.Model(&model.File{}).Where("file_id = ?", fileId).Update("status", model.StatusFailed).Error; updateErr != nil {
					log.Printf("Failed to update file status: %v", updateErr)
				}
				return fmt.Errorf("failed to broadcast index transaction: %w", err)
			}
		}

		// Update task
		task.IndexTxId = indexTxId
		task.Stage = model.TaskStageIndexBroadcast
		task.Progress = 98
		task.CurrentStep = "Index transaction broadcasted"
		if err := tx.Model(&model.FileUploaderTask{}).
			Where("id = ?", task.ID).
			Updates(map[string]interface{}{
				"index_tx_id":  task.IndexTxId,
				"stage":        task.Stage,
				"progress":     task.Progress,
				"current_step": task.CurrentStep,
			}).Error; err != nil {
			return fmt.Errorf("failed to update task: %w", err)
		}

		// Update file entity to mark as success
		if err := tx.Model(&model.File{}).
			Where("file_id = ?", fileId).
			Updates(map[string]interface{}{
				"status": model.StatusSuccess,
				"tx_id":  indexTxId,
				"pin_id": fmt.Sprintf("%si0", indexTxId),
			}).Error; err != nil {
			return fmt.Errorf("failed to update file status: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to broadcast index transaction and update records: %w", err)
	}

	// Update task object for response
	task.IndexTxId = indexTxId
	task.Stage = model.TaskStageIndexBroadcast
	task.Progress = 98
	task.CurrentStep = "Index transaction broadcasted"

	return &ChunkedUploadResponse{
		FileId:         task.FileId,
		FileHash:       filehash,
		FileMd5:        task.FileMd5,
		ChunkNumber:    len(chunks),
		ChunkFundingTx: task.ChunkFundingTx,
		ChunkTxIds:     chunkTxIds,
		IndexTx:        indexTxHex,
		IndexTxId:      indexTxId,
		Status:         string(model.StatusSuccess),
		Message:        "success",
	}, nil
}

func decodeStringArray(data string) ([]string, error) {
	if strings.TrimSpace(data) == "" {
		return []string{}, nil
	}
	var list []string
	if err := json.Unmarshal([]byte(data), &list); err != nil {
		return nil, err
	}
	return list, nil
}

func isDuplicateBroadcastError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already") || strings.Contains(msg, "known") || strings.Contains(msg, "exists") || strings.Contains(msg, "spent")
}

func (s *UploadService) updateUploadTaskProgress(task *model.FileUploaderTask, step string, progress int, processedChunks int) {
	if task == nil {
		return
	}

	if step != "" {
		task.CurrentStep = step
	}

	if progress >= 0 {
		if progress > 100 {
			progress = 100
		}
		if progress > task.Progress {
			task.Progress = progress
		}
	}

	if processedChunks >= 0 {
		task.ProcessedChunks = processedChunks
	}

	if err := s.fileUploaderTaskDAO.Update(task); err != nil {
		log.Printf("Failed to update task progress (taskId=%s): %v", task.TaskId, err)
	}
}

func (s *UploadService) clearTaskPayload(task *model.FileUploaderTask) {
	if task == nil {
		return
	}

	task.ChunkPreTxHex = ""
	task.IndexPreTxHex = ""
	task.MergeTxHex = ""
	task.ChunkFundingTx = ""
	task.ContentBase64 = ""
	task.ChunkTxHexes = ""
}

func calcProgressRange(start, end, processed, total int) int {
	if end <= start {
		return end
	}
	if total <= 0 {
		return end
	}
	if processed < 0 {
		processed = 0
	}
	if processed > total {
		processed = total
	}

	span := end - start
	return start + span*processed/total
}

// GetTaskProgress fetches a task by task ID.
func (s *UploadService) GetTaskProgress(taskId string) (*model.FileUploaderTask, error) {
	return s.fileUploaderTaskDAO.GetByTaskID(taskId)
}

// getOrCreateFileAssistent creates or retrieves the assistant address for a user.
func (s *UploadService) getOrCreateFileAssistent(metaID, address string, netParam *chaincfg2.Params) (*model.FileAssistent, error) {
	assistent, err := s.fileAssistentDAO.GetByAddress(address)
	if err != nil {
		return nil, err
	}
	if assistent != nil {
		return assistent, nil
	}

	// Generate private key
	privateKey, err := bsvec2.NewPrivateKey(bsvec2.S256())
	if err != nil {
		return nil, fmt.Errorf("failed to generate assistent private key: %w", err)
	}
	privateKeyHex := hex.EncodeToString(privateKey.Serialize())

	// Derive assistant address
	pubKeyBytes := privateKey.PubKey().SerializeCompressed()
	addressPubKey, err := bsvutil2.NewAddressPubKey(pubKeyBytes, netParam)
	if err != nil {
		return nil, fmt.Errorf("failed to derive assistent address: %w", err)
	}

	newAssistent := &model.FileAssistent{
		MetaId:           metaID,
		Address:          address,
		AssistentAddress: addressPubKey.EncodeAddress(),
		AssistentPriHex:  privateKeyHex,
		Status:           model.StatusSuccess,
	}

	if err := s.fileAssistentDAO.Create(newAssistent); err != nil {
		return nil, fmt.Errorf("failed to create assistent: %w", err)
	}

	log.Printf("Created new file assistent for user address %s, assistent address: %s", address, newAssistent.AssistentAddress)
	return newAssistent, nil
}

// splitFile splits content into chunks of the provided size.
func splitFile(content []byte, chunkSize int64) [][]byte {
	if chunkSize <= 0 {
		chunkSize = 100 * 1024 // default 100 KB
	}

	chunks := make([][]byte, 0)
	fileSize := int64(len(content))

	for i := int64(0); i < fileSize; i += chunkSize {
		end := i + chunkSize
		if end > fileSize {
			end = fileSize
		}
		chunks = append(chunks, content[i:end])
	}

	return chunks
}

func decodeMvcTx(txHex string) (*wire2.MsgTx, error) {
	raw, err := hex.DecodeString(strings.TrimSpace(txHex))
	if err != nil {
		return nil, fmt.Errorf("failed to decode tx hex: %w", err)
	}
	var tx wire2.MsgTx
	if err := tx.Deserialize(bytes.NewReader(raw)); err != nil {
		return nil, fmt.Errorf("failed to deserialize tx: %w", err)
	}
	return &tx, nil
}

func buildChunkOpReturnScript(path string, chunkData []byte) ([]byte, error) {
	builder := txscript.NewScriptBuilder().
		AddOp(txscript.OP_0).
		AddOp(txscript.OP_RETURN).
		AddData([]byte("metaid")).
		AddData([]byte("create")).
		AddData([]byte(path)).
		AddData([]byte("0")).
		AddData([]byte("1.0.0")).
		AddData([]byte(metaid_protocols.MonitorMetaIdFileChunkContentType + ";binary"))

	maxChunkSize := 520
	for i := 0; i < len(chunkData); i += maxChunkSize {
		end := i + maxChunkSize
		if end > len(chunkData) {
			end = len(chunkData)
		}
		builder.AddFullData(chunkData[i:end])
	}

	return builder.Script()
}

func estimateChunkFundingValue(chunkScript []byte, feeRate int64) int64 {
	const inputSize = 148 // Approximate size of a P2PKH input with signature
	opReturnSize := 8 + wire2.VarIntSerializeSize(uint64(len(chunkScript))) + len(chunkScript)
	txSize := 4 + 1 + inputSize + 1 + opReturnSize + 4
	fee := int64(txSize) * feeRate
	if fee < 600 {
		fee = 600
	}
	return fee
}

func (s *UploadService) buildChunkTxWithFunding(
	input *common.TxInputUtxo,
	chunkScript []byte,
) (*wire2.MsgTx, error) {
	tx := wire2.NewMsgTx(10)

	hash, err := chainhash2.NewHashFromStr(input.TxId)
	if err != nil {
		return nil, fmt.Errorf("failed to parse chunk funding txid: %w", err)
	}
	prevOut := wire2.NewOutPoint(hash, uint32(input.TxIndex))
	txIn := wire2.NewTxIn(prevOut, nil)
	tx.AddTxIn(txIn)

	tx.AddTxOut(wire2.NewTxOut(0, chunkScript))

	privateKeyBytes, err := hex.DecodeString(input.PriHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode assistent private key: %w", err)
	}
	privateKey, _ := bsvec2.PrivKeyFromBytes(bsvec2.S256(), privateKeyBytes)

	pkScriptBytes, err := hex.DecodeString(input.PkScript)
	if err != nil {
		return nil, fmt.Errorf("failed to decode pkScript: %w", err)
	}

	sigScript, err := txscript2.SignatureScript(tx, 0, int64(input.Amount), pkScriptBytes, txscript2.SigHashAll, privateKey, true)
	if err != nil {
		return nil, fmt.Errorf("failed to sign chunk tx: %w", err)
	}

	tx.TxIn[0].SignatureScript = sigScript
	return tx, nil
}

func buildIndexOpReturnScript(path string, indexData []byte) ([]byte, error) {
	builder := txscript.NewScriptBuilder().
		AddOp(txscript.OP_0).
		AddOp(txscript.OP_RETURN).
		AddData([]byte("metaid")).
		AddData([]byte("create")).
		AddData([]byte(path)).
		AddData([]byte("0")).
		AddData([]byte("1.0.0")).
		AddData([]byte(metaid_protocols.MonitorMetaIdFileIndexContentType + ";utf-8"))

	maxChunkSize := 520
	for i := 0; i < len(indexData); i += maxChunkSize {
		end := i + maxChunkSize
		if end > len(indexData) {
			end = len(indexData)
		}
		builder.AddFullData(indexData[i:end])
	}

	return builder.Script()
}

func buildIndexTxFromPreTx(netParam *chaincfg2.Params, baseTx *wire2.MsgTx, userAddress string, indexScript []byte) (*wire2.MsgTx, error) {
	userAddressBytes, err := bsvutil2.DecodeAddress(userAddress, netParam)
	if err != nil {
		return nil, fmt.Errorf("failed to decode user address: %w", err)
	}
	userPkScript, err := txscript2.PayToAddrScript(userAddressBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to build user pkScript: %w", err)
	}
	//add uesr output
	baseTx.AddTxOut(wire2.NewTxOut(1, userPkScript))

	//add opreturn output
	baseTx.AddTxOut(wire2.NewTxOut(0, indexScript))
	return baseTx, nil
}
