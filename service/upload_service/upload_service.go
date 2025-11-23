package upload_service

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
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
	fileDAO          *dao.FileDAO
	fileChunkDAO     *dao.FileChunkDAO
	fileAssistentDAO *dao.FileAssistentDAO
	storage          storage.Storage
}

// NewUploadService create upload service instance
func NewUploadService(storage storage.Storage) *UploadService {
	return &UploadService{
		fileDAO:          dao.NewFileDAO(),
		fileChunkDAO:     dao.NewFileChunkDAO(),
		fileAssistentDAO: dao.NewFileAssistentDAO(),
		storage:          storage,
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

// EstimateChunkedUploadRequest 估算分片上链费用请求
type EstimateChunkedUploadRequest struct {
	FileName    string // 文件名称
	Content     []byte // 文件内容
	Path        string // MetaID path (基础路径，会自动添加 /file/_chunk 和 /file/index)
	ContentType string // 文件内容类型 (如 image/jpeg, text/plain 等)
	FeeRate     int64  // 费率（可选，默认使用配置）
}

// EstimateChunkedUploadResponse 估算分片上链费用响应
type EstimateChunkedUploadResponse struct {
	ChunkNumber   int     `json:"chunkNumber"`   // 分片数量
	ChunkSize     int64   `json:"chunkSize"`     // 分片大小（字节）
	ChunkFees     []int64 `json:"chunkFees"`     // 每个 chunk 的费用
	ChunkPerTxFee int64   `json:"chunkPerTxFee"` // 每个 chunk 的费用
	ChunkPreTxFee int64   `json:"chunkPreTxFee"` // ChunkPreTxHex 需要的总费用（所有 chunk 费用之和）
	IndexPreTxFee int64   `json:"indexPreTxFee"` // IndexPreTxHex 需要的费用
	TotalFee      int64   `json:"totalFee"`      // 总费用（ChunkPreTxFee + IndexPreTxFee）
	PerChunkFee   int64   `json:"perChunkFee"`   // 每个 chunk 的平均费用
	Message       string  `json:"message"`       // 消息
}

// ChunkedUploadRequest 分片上链请求
type ChunkedUploadRequest struct {
	MetaId        string // MetaID
	Address       string // 用户地址
	FileName      string // 文件名称
	Content       []byte // 文件内容
	Path          string // MetaID path (基础路径，会自动添加 /file/_chunk 和 /file/index)
	Operation     string // create/update
	ContentType   string // 文件内容类型 (如 image/jpeg, text/plain 等)
	ChunkPreTxHex string // 预构建的托管地址充值交易（包含 inputs，signNull）
	IndexPreTxHex string // 预构建的 index 交易（包含 inputs，signNull）
	MergeTxHex    string // 合并交易 hex（用于创建两个 UTXO，需要先广播）
	FeeRate       int64  // 费率
	IsBroadcast   bool   // 是否广播
}

// EstimateChunkedUpload 估算分片上链费用
func (s *UploadService) EstimateChunkedUpload(req *EstimateChunkedUploadRequest) (*EstimateChunkedUploadResponse, error) {
	// 参数验证
	if len(req.Content) == 0 {
		return nil, fmt.Errorf("file content is empty")
	}
	if req.Path == "" {
		return nil, fmt.Errorf("file path is required")
	}

	// 设置默认值
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
		chunkSize = 2000 * 1024 // 默认 2000KB
	}

	// 文件分片
	chunks := splitFile(req.Content, chunkSize)
	chunkNumber := len(chunks)

	// 构建 chunk 路径
	chunkPath := fmt.Sprintf("%s/file/_chunk", req.Path)
	if !strings.HasPrefix(chunkPath, "/") {
		chunkPath = "/" + chunkPath
	}

	// 估算每个 chunk 的费用
	totalChunkFee := int64(0)
	perChunkFee := int64(0)
	chunkFees := make([]int64, 0, chunkNumber)

	for _, chunkData := range chunks {
		// 构建 chunk script 以估算大小
		chunkScript, err := buildChunkOpReturnScript(chunkPath, chunkData)
		if err != nil {
			return nil, fmt.Errorf("failed to build chunk script for estimation: %w", err)
		}
		chunkFee := estimateChunkFundingValue(chunkScript, feeRate)
		chunkFees = append(chunkFees, chunkFee)
		totalChunkFee += chunkFee
	}

	// 计算平均每个 chunk 的费用
	if chunkNumber > 0 {
		perChunkFee = totalChunkFee / int64(chunkNumber)
	}

	// 估算 chunkFundingTx 的 gas 费用
	// chunkFundingTx 包含：1 个 input + chunkNumber 个 outputs
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

	// ChunkPreTxFee 需要包含：所有 chunk outputs 的金额 + chunkFundingTx 的 gas 费用
	chunkPreTxFee := totalChunkFee + chunkFundingTxFee

	// 构建 index 路径
	indexPath := fmt.Sprintf("%s/file/index", req.Path)
	if !strings.HasPrefix(indexPath, "/") {
		indexPath = "/" + indexPath
	}

	// 估算 index 费用
	// 首先需要构建 index 数据来估算大小
	sha256hash := sha256.Sum256(req.Content)
	filehashStr := hex.EncodeToString(sha256hash[:])

	// 构建 chunkList（使用估算的 PinID）
	chunkList := make([]struct {
		Sha256 string `json:"sha256"`
		PinId  string `json:"pinId"`
	}, 0, chunkNumber)
	for i, chunkData := range chunks {
		chunkHash := sha256.Sum256(chunkData)
		chunkHashStr := hex.EncodeToString(chunkHash[:])
		// 使用占位符 PinID，实际值会在构建时确定
		chunkList = append(chunkList, struct {
			Sha256 string `json:"sha256"`
			PinId  string `json:"pinId"`
		}{
			Sha256: chunkHashStr,
			PinId:  fmt.Sprintf("placeholder_%d", i), // 占位符
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

	// 构建 index script
	indexScript, err := buildIndexOpReturnScript(indexPath, indexData)
	if err != nil {
		return nil, fmt.Errorf("failed to build index script for estimation: %w", err)
	}

	// 估算 index 交易费用
	// Index 交易包含：inputs（已存在）+ user output + OP_RETURN output
	const indexInputSize = 148 // P2PKH input with signature
	const indexOutputSize = 34 // P2PKH output (8 bytes value + ~26 bytes script)
	indexOpReturnSize := 8 + wire2.VarIntSerializeSize(uint64(len(indexScript))) + len(indexScript)
	// 假设有 1 个 input（用户需要提供）
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
		ChunkPreTxFee: chunkPreTxFee, // 包含所有 chunk outputs + chunkFundingTx gas 费用
		IndexPreTxFee: indexFee,
		TotalFee:      totalFee,
		PerChunkFee:   perChunkFee,
		Message:       "success",
	}, nil
}

// ChunkedUploadResponse 分片上链响应
type ChunkedUploadResponse struct {
	FileId         string   `json:"fileId"`         // 文件 ID
	FileHash       string   `json:"fileHash"`       // 文件 SHA256 hash
	FileMd5        string   `json:"fileMd5"`        // 文件 MD5 hash
	ChunkNumber    int      `json:"chunkNumber"`    // 分片数量
	ChunkFundingTx string   `json:"chunkFundingTx"` // 托管地址充值交易（构建完成，可直接签名/广播）
	ChunkTxs       []string `json:"chunkTxs"`       // Chunk 交易 hex 列表（按顺序）
	ChunkTxIds     []string `json:"chunkTxIds"`     // Chunk 交易 ID 列表（按顺序）
	IndexTx        string   `json:"indexTx"`        // Index 交易 hex
	IndexTxId      string   `json:"indexTxId"`      // Index 交易 ID
	Status         string   `json:"status"`         // 状态
	Message        string   `json:"message"`        // 消息
}

// ChunkedUpload 分片上链：将大文件分片并构建交易
func (s *UploadService) ChunkedUpload(req *ChunkedUploadRequest) (*ChunkedUploadResponse, error) {
	// 参数验证
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

	// 设置默认值
	if req.Operation == "" {
		req.Operation = "create"
	}
	if req.ContentType == "" {
		req.ContentType = "application/octet-stream"
	}
	if req.FeeRate == 0 {
		req.FeeRate = conf.Cfg.Uploader.FeeRate
	}

	// 获取网络参数
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

	// 获取或创建托管地址
	assistent, err := s.getOrCreateFileAssistent(req.MetaId, req.Address, netParam)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare assistent address: %w", err)
	}

	// 计算文件 hash
	sha256hash := sha256.Sum256(req.Content)
	md5hash := md5.Sum(req.Content)
	filehashStr := hex.EncodeToString(sha256hash[:])
	md5hashStr := hex.EncodeToString(md5hash[:])

	// 生成 FileId
	fileId := req.MetaId + "_" + filehashStr

	// 文件分片
	chunks := splitFile(req.Content, chunkSize)
	chunkNumber := len(chunks)

	log.Printf("File split into %d chunks, file size: %d bytes", chunkNumber, len(req.Content))

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

	// 构建 chunk 交易列表
	chunkTxs := make([]string, 0, chunkNumber)
	chunkTxIds := make([]string, 0, chunkNumber)
	chunkList := make([]struct {
		Sha256 string `json:"sha256"`
		PinId  string `json:"pinId"`
	}, 0, chunkNumber)

	// 为每个 chunk 构建交易
	chunkPath := "/file/_chunk"
	// chunkPath := fmt.Sprintf("%s/file/_chunk", req.Path)
	// if !strings.HasPrefix(chunkPath, "/") {
	// 	chunkPath = "/" + chunkPath
	// }

	// 先计算所有 chunk scripts 和 amounts
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

	// 计算 chunkFundingTx 的 gas 费用（与预估阶段保持一致）
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

	// 从合并交易中获取 chunkPreTx output 的金额
	var totalInputAmount int64 = 0
	if req.MergeTxHex != "" {
		mergeTx, err := decodeMvcTx(req.MergeTxHex)
		if err == nil {
			// 计算需要的总金额 = totalChunkOutputAmount + chunkFundingTxFee
			requiredAmount := totalChunkOutputAmount + chunkFundingTxFee
			// 从合并交易中找到匹配的 output（金额接近所需金额）
			for i, output := range mergeTx.TxOut {
				outputAmount := int64(output.Value)
				// 允许 1000 satoshis 的容差
				if outputAmount >= requiredAmount-1000 && outputAmount <= requiredAmount+1000 {
					totalInputAmount = outputAmount
					log.Printf("Found chunkPreTx output at index %d: %d satoshis (required: %d)", i, outputAmount, requiredAmount)
					break
				}
			}
		}
	}

	// 如果无法从合并交易获取，使用估算值（totalChunkOutputAmount + chunkFundingTxFee）
	if totalInputAmount == 0 {
		totalInputAmount = totalChunkOutputAmount + chunkFundingTxFee
		log.Printf("Using estimated totalInputAmount: %d satoshis (chunkOutputs: %d + fee: %d)",
			totalInputAmount, totalChunkOutputAmount, chunkFundingTxFee)
	}

	// 验证 input 金额是否足够
	availableAmount := totalInputAmount - chunkFundingTxFee
	if availableAmount < totalChunkOutputAmount {
		return nil, fmt.Errorf("insufficient input amount: need %d satoshis (outputs: %d + fee: %d), but only have %d satoshis available",
			totalChunkOutputAmount+chunkFundingTxFee, totalChunkOutputAmount, chunkFundingTxFee, availableAmount)
	}

	// 添加 outputs（使用原始金额，剩余金额作为 dust）
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
		// 计算 chunk hash
		chunkHash := sha256.Sum256(chunkData)
		chunkHashStr := hex.EncodeToString(chunkHash[:])

		// 构建 chunk 交易（使用托管地址的 UTXO）
		chunkTx, err := s.buildChunkTxWithFunding(
			chunkInputs[i],
			chunkScripts[i],
		)
		if err != nil {
			return nil, fmt.Errorf("failed to build chunk %d transaction: %w", i, err)
		}

		// 序列化交易
		chunkTxHex, err := common.MvcToRaw(chunkTx)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize chunk %d transaction: %w", i, err)
		}

		// 获取交易 hash（作为 PinID）
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

		// 计算 chunk MD5
		chunkMd5Hash := md5.Sum(chunkData)
		chunkMd5Str := hex.EncodeToString(chunkMd5Hash[:])

		// 保存 file_chunk 记录
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
			StoragePath: "", // 可选：如果需要保存到存储，可以设置路径
			Operation:   req.Operation,
			// TxRaw:       chunkTxHex,
			Status: model.StatusPending,
		}

		if err := s.fileChunkDAO.Create(fileChunk); err != nil {
			log.Printf("Failed to save file chunk %d: %v", i, err)
			// 继续处理其他 chunk，不中断流程
		} else {
			log.Printf("File chunk %d saved: hash=%s, txId=%s", i, chunkHashStr, chunkTxId)
		}

		log.Printf("Chunk %d/%d built: size=%d, hash=%s, pinId=%s", i+1, chunkNumber, len(chunkData), chunkHashStr, chunkPinId)
	}

	// 构建 index 数据
	metaFileIndex := metaid_protocols.MetaFileIndex{
		Sha256:      filehashStr,
		FileSize:    int64(len(req.Content)),
		ChunkNumber: chunkNumber,
		ChunkSize:   chunkSize,
		DataType:    req.ContentType,
		Name:        req.FileName,
		ChunkList:   chunkList,
	}

	// 序列化 index 数据
	indexData, err := json.Marshal(metaFileIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal index data: %w", err)
	}

	// 构建 index 路径
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

	// 序列化 index 交易
	indexTxHex, err := common.MvcToRaw(indexTx)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize index transaction: %w", err)
	}
	indexTxId := common.GetMvcTxhashFromRaw(indexTxHex)

	log.Printf("Index transaction built: fileHash=%s, chunkNumber=%d", filehashStr, chunkNumber)

	// 检查文件是否已存在
	existingFile, err := s.fileDAO.GetByFileID(fileId)
	if err == nil && existingFile != nil {
		// 文件已存在，根据状态处理
		if existingFile.Status == model.StatusSuccess {
			// 文件已成功上链，直接返回
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
		// 如果状态是 pending 或 failed，继续执行并更新记录
		if existingFile.Status == model.StatusPending {
			log.Printf("File already exists in pending status, will update and retry: FileId=%s", fileId)
		} else {
			log.Printf("File exists but failed, will update and retry: FileId=%s", fileId)
		}
	}

	// 准备文件元数据
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

	// 如果文件已存在，更新记录；否则创建新记录
	if existingFile != nil {
		// 更新现有记录
		file.ID = existingFile.ID // 保留原有 ID
		if err := s.fileDAO.Update(file); err != nil {
			return nil, fmt.Errorf("failed to update file metadata: %w", err)
		}
		log.Printf("File metadata updated: FileId=%s, status=pending", fileId)
	} else {
		// 创建新记录
		if err := s.fileDAO.Create(file); err != nil {
			return nil, fmt.Errorf("failed to save file metadata: %w", err)
		}
		log.Printf("File metadata saved: FileId=%s, status=pending", fileId)
	}

	// 如果 IsBroadcast 为 true，按顺序广播所有交易
	if req.IsBroadcast {
		chain := conf.Cfg.Net
		finalStatus := model.StatusSuccess
		finalMessage := "success"

		// 使用数据库事务确保数据一致性
		err := database.UploaderDB.Transaction(func(tx *gorm.DB) error {
			// 0. 如果有 MergeTxHex，先广播合并交易（必须在其他交易之前广播）
			if req.MergeTxHex != "" {
				log.Printf("Broadcasting merge transaction first...")
				mergeTxId, err := node.BroadcastTx(chain, req.MergeTxHex)
				if err != nil {
					log.Printf("Failed to broadcast merge transaction: %v", err)
					// 更新文件状态为失败
					if updateErr := tx.Model(&model.File{}).Where("file_id = ?", fileId).Update("status", model.StatusFailed).Error; updateErr != nil {
						return fmt.Errorf("failed to update file status: %w", updateErr)
					}
					// 更新所有 chunk 状态为失败
					if updateErr := tx.Model(&model.FileChunk{}).Where("file_hash = ?", filehashStr).Update("status", model.StatusFailed).Error; updateErr != nil {
						log.Printf("Failed to update chunk status: %v", updateErr)
					}
					return fmt.Errorf("failed to broadcast merge transaction: %w", err)
				}
				log.Printf("Merge transaction broadcasted successfully: %s", mergeTxId)
			}

			// 1. 广播 ChunkFundingTx
			log.Printf("Broadcasting chunk funding transaction: %s", chunkFundingTxHash)
			broadcastFundingTxID, err := node.BroadcastTx(chain, chunkFundingTxHex)
			if err != nil {
				fmt.Printf("tx hex: %s\n", chunkFundingTxHex)
				log.Printf("Failed to broadcast chunk funding transaction: %v", err)
				// 更新文件状态为失败
				if updateErr := tx.Model(&model.File{}).Where("file_id = ?", fileId).Update("status", model.StatusFailed).Error; updateErr != nil {
					return fmt.Errorf("failed to update file status: %w", updateErr)
				}
				// 更新所有 chunk 状态为失败
				if updateErr := tx.Model(&model.FileChunk{}).Where("file_hash = ?", filehashStr).Update("status", model.StatusFailed).Error; updateErr != nil {
					log.Printf("Failed to update chunk status: %v", updateErr)
				}
				return fmt.Errorf("failed to broadcast chunk funding transaction: %w", err)
			}
			log.Printf("Chunk funding transaction broadcasted successfully: %s", broadcastFundingTxID)

			// 2. 按顺序广播每个 ChunkTx
			for i, chunkTxHex := range chunkTxs {
				log.Printf("Broadcasting chunk transaction %d/%d: %s", i+1, chunkNumber, chunkTxIds[i])
				broadcastChunkTxID, err := node.BroadcastTx(chain, chunkTxHex)
				if err != nil {
					fmt.Printf("tx hex: %s\n", chunkTxHex)
					log.Printf("Failed to broadcast chunk transaction %d: %v", i, err)
					// 更新文件状态为失败
					if updateErr := tx.Model(&model.File{}).Where("file_id = ?", fileId).Update("status", model.StatusFailed).Error; updateErr != nil {
						return fmt.Errorf("failed to update file status: %w", updateErr)
					}
					// 更新所有 chunk 状态为失败
					if updateErr := tx.Model(&model.FileChunk{}).Where("file_hash = ?", filehashStr).Update("status", model.StatusFailed).Error; updateErr != nil {
						log.Printf("Failed to update chunk status: %v", updateErr)
					}
					return fmt.Errorf("failed to broadcast chunk transaction %d: %w", i, err)
				}
				log.Printf("Chunk transaction %d/%d broadcasted successfully: %s", i+1, chunkNumber, broadcastChunkTxID)

				// 更新对应 chunk 的状态
				if updateErr := tx.Model(&model.FileChunk{}).
					Where("pin_id = ?", fmt.Sprintf("%si0", chunkTxIds[i])).
					Update("status", model.StatusSuccess).Error; updateErr != nil {
					log.Printf("Failed to update chunk %d status: %v", i, updateErr)
					// 不中断流程，继续广播
				}
			}

			time.Sleep(5 * time.Second)
			// 3. 广播 IndexTx
			log.Printf("Broadcasting index transaction: %s", indexTxId)
			broadcastIndexTxID, err := node.BroadcastTx(chain, indexTxHex)
			if err != nil {
				fmt.Printf("tx hex: %s\n", indexTxHex)
				log.Printf("Failed to broadcast index transaction: %v", err)
				// 更新文件状态为失败
				if updateErr := tx.Model(&model.File{}).Where("file_id = ?", fileId).Update("status", model.StatusFailed).Error; updateErr != nil {
					return fmt.Errorf("failed to update file status: %w", updateErr)
				}
				// 更新所有 chunk 状态为失败
				if updateErr := tx.Model(&model.FileChunk{}).Where("file_hash = ?", filehashStr).Update("status", model.StatusFailed).Error; updateErr != nil {
					log.Printf("Failed to update chunk status: %v", updateErr)
				}
				return fmt.Errorf("failed to broadcast index transaction: %w", err)
			}
			log.Printf("Index transaction broadcasted successfully: %s", broadcastIndexTxID)

			// 4. 更新文件状态为成功
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
			ChunkTxs:       chunkTxs,
			ChunkTxIds:     chunkTxIds,
			IndexTx:        indexTxHex,
			IndexTxId:      indexTxId,
			Status:         string(finalStatus),
			Message:        finalMessage,
		}, nil
	}

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

// getOrCreateFileAssistent 获取或创建用户托管地址
func (s *UploadService) getOrCreateFileAssistent(metaID, address string, netParam *chaincfg2.Params) (*model.FileAssistent, error) {
	assistent, err := s.fileAssistentDAO.GetByAddress(address)
	if err != nil {
		return nil, err
	}
	if assistent != nil {
		return assistent, nil
	}

	// 生成新的私钥
	privateKey, err := bsvec2.NewPrivateKey(bsvec2.S256())
	if err != nil {
		return nil, fmt.Errorf("failed to generate assistent private key: %w", err)
	}
	privateKeyHex := hex.EncodeToString(privateKey.Serialize())

	// 生成托管地址
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

// splitFile 将文件分片
func splitFile(content []byte, chunkSize int64) [][]byte {
	if chunkSize <= 0 {
		chunkSize = 100 * 1024 // 默认 100KB
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
