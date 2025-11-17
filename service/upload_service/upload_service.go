package upload_service

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"strings"

	chaincfg2 "github.com/bitcoinsv/bsvd/chaincfg"
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
	"meta-file-system/storage"
)

// UploadService upload service
type UploadService struct {
	fileDAO *dao.FileDAO
	storage storage.Storage
}

// NewUploadService create upload service instance
func NewUploadService(storage storage.Storage) *UploadService {
	return &UploadService{
		fileDAO: dao.NewFileDAO(),
		storage: storage,
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
