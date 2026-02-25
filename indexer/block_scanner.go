package indexer

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"meta-file-system/tool"

	"github.com/bitcoinsv/bsvd/wire"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	btcwire "github.com/btcsuite/btcd/wire"
	"github.com/schollz/progressbar/v3"
)

// DefaultLargeBlockThresholdBytes is the default block size threshold above which we use lazy tx-by-tx loading (50MB)
const DefaultLargeBlockThresholdBytes = 50 * 1024 * 1024

// LargeBlockTxCountFallback: when RPC does not return block size, use tx count as fallback for lazy path
const LargeBlockTxCountFallback = 50000

// BlockScanner block scanner
type BlockScanner struct {
	rpcURL                   string
	rpcUser                  string
	rpcPassword              string
	startHeight              int64
	interval                 time.Duration
	chainType                ChainType // Chain type: btc, mvc, or doge
	progressBar              *progressbar.ProgressBar
	zmqClient                *ZMQClient    // ZMQ client for real-time transaction monitoring
	zmqEnabled               bool          // Whether ZMQ is enabled
	parser                   *MetaIDParser // Shared parser to avoid repeated allocation
	largeBlockThresholdBytes int64         // Block size in bytes above which to use lazy loading; 0 = default
}

// NewBlockScanner create block scanner (default MVC)
func NewBlockScanner(rpcURL, rpcUser, rpcPassword string, startHeight int64, interval int) *BlockScanner {
	return &BlockScanner{
		rpcURL:      rpcURL,
		rpcUser:     rpcUser,
		rpcPassword: rpcPassword,
		startHeight: startHeight,
		interval:    time.Duration(interval) * time.Second,
		chainType:   ChainTypeMVC,
	}
}

// NewBlockScannerWithChain create block scanner with specified chain type
func NewBlockScannerWithChain(rpcURL, rpcUser, rpcPassword string, startHeight int64, interval int, chainType ChainType) *BlockScanner {
	return &BlockScanner{
		rpcURL:      rpcURL,
		rpcUser:     rpcUser,
		rpcPassword: rpcPassword,
		startHeight: startHeight,
		interval:    time.Duration(interval) * time.Second,
		chainType:   chainType,
		zmqEnabled:  false,
		parser:      NewMetaIDParser(""), // Create shared parser once
	}
}

// EnableZMQ enable ZMQ real-time transaction monitoring
func (s *BlockScanner) EnableZMQ(zmqAddress string) {
	s.zmqClient = NewZMQClient(zmqAddress, s.chainType)
	s.zmqEnabled = true
	log.Printf("ZMQ enabled for %s chain: %s", s.chainType, zmqAddress)
}

// SetZMQTransactionHandler set handler for ZMQ transactions
func (s *BlockScanner) SetZMQTransactionHandler(handler func(tx interface{}, metaDataTx *MetaIDDataTx) error) {
	if s.zmqClient != nil {
		s.zmqClient.SetTransactionHandler(handler)
	}
}

// SetLargeBlockThreshold sets the block size threshold in bytes above which blocks are loaded lazily (tx-by-tx).
// If bytes <= 0, DefaultLargeBlockThresholdBytes is used.
func (s *BlockScanner) SetLargeBlockThreshold(bytes int64) {
	if bytes <= 0 {
		s.largeBlockThresholdBytes = DefaultLargeBlockThresholdBytes
		return
	}
	s.largeBlockThresholdBytes = bytes
}

// RPCRequest RPC request structure
type RPCRequest struct {
	Jsonrpc string        `json:"jsonrpc"`
	ID      string        `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

// RPCResponse RPC response structure
type RPCResponse struct {
	Result interface{} `json:"result"`
	Error  *RPCError   `json:"error"`
	ID     string      `json:"id"`
}

// RPCError RPC error structure
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// GetBlockCount get current block height
func (s *BlockScanner) GetBlockCount() (int64, error) {
	request := RPCRequest{
		Jsonrpc: "1.0",
		ID:      "getblockcount",
		Method:  "getblockcount",
		Params:  []interface{}{},
	}

	response, err := s.rpcCall(request)
	if err != nil {
		return 0, err
	}

	if response.Error != nil {
		return 0, fmt.Errorf("rpc error: %s", response.Error.Message)
	}

	height, ok := response.Result.(float64)
	if !ok {
		return 0, errors.New("invalid block height response")
	}

	return int64(height), nil
}

// GetBlockHash get block hash
func (s *BlockScanner) GetBlockhash(height int64) (string, error) {
	request := RPCRequest{
		Jsonrpc: "1.0",
		ID:      "getblockhash",
		Method:  "getblockhash",
		Params:  []interface{}{height},
	}

	response, err := s.rpcCall(request)
	if err != nil {
		return "", err
	}

	if response.Error != nil {
		return "", fmt.Errorf("rpc error: %s", response.Error.Message)
	}

	hash, ok := response.Result.(string)
	if !ok {
		return "", errors.New("invalid block hash response")
	}

	return hash, nil
}

// GetBlockHex get block hex data
// verbosity=0 returns raw block hex
func (s *BlockScanner) GetBlockHex(blockhash string) (string, error) {
	request := RPCRequest{
		Jsonrpc: "1.0",
		ID:      "getblock",
		Method:  "getblock",
		Params:  []interface{}{blockhash, 0}, // verbosity=0 return raw hex
	}

	response, err := s.rpcCall(request)
	if err != nil {
		return "", err
	}

	if response.Error != nil {
		return "", fmt.Errorf("rpc error: %s", response.Error.Message)
	}

	blockHex, ok := response.Result.(string)
	if !ok {
		return "", errors.New("invalid block hex response")
	}

	return blockHex, nil
}

// GetRawTransaction get raw transaction by txid
// verbosity=0 returns raw transaction hex
func (s *BlockScanner) GetRawTransaction(txid string) (string, error) {
	request := RPCRequest{
		Jsonrpc: "1.0",
		ID:      "getrawtransaction",
		Method:  "getrawtransaction",
		Params:  []interface{}{txid, 0}, // verbosity=0 return raw hex
	}

	response, err := s.rpcCall(request)
	if err != nil {
		return "", err
	}

	if response.Error != nil {
		return "", fmt.Errorf("rpc error: %s", response.Error.Message)
	}

	txHex, ok := response.Result.(string)
	if !ok {
		return "", errors.New("invalid transaction hex response")
	}

	return txHex, nil
}

// GetAndDeserializeTx fetches raw transaction by txid and deserializes it to *btcwire.MsgTx or *wire.MsgTx.
// Used for large-block path to avoid holding the full block in memory.
func (s *BlockScanner) GetAndDeserializeTx(txid string) (interface{}, error) {
	txHex, err := s.GetRawTransaction(txid)
	if err != nil {
		return nil, err
	}
	txBytes, err := hex.DecodeString(txHex)
	if err != nil {
		return nil, fmt.Errorf("decode tx hex: %w", err)
	}
	if s.chainType == ChainTypeBTC || s.chainType == ChainTypeDOGE {
		var btcTx btcwire.MsgTx
		if err := btcTx.Deserialize(bytes.NewReader(txBytes)); err != nil {
			return nil, fmt.Errorf("deserialize btc/doge tx: %w", err)
		}
		return &btcTx, nil
	}
	var mvcTx wire.MsgTx
	if err := mvcTx.Deserialize(bytes.NewReader(txBytes)); err != nil {
		return nil, fmt.Errorf("deserialize mvc tx: %w", err)
	}
	return &mvcTx, nil
}

// BlockVerboseResult represents the result of getblock RPC with verbosity=1
// (Bitcoin Core returns size, strippedsize, time, tx; some nodes may omit size)
type BlockVerboseResult struct {
	Hash         string   `json:"hash"`
	Version      int32    `json:"version"`
	PreviousHash string   `json:"previousblockhash"`
	MerkleRoot   string   `json:"merkleroot"`
	Time         int64    `json:"time"`
	Bits         string   `json:"bits"`
	Nonce        uint32   `json:"nonce"`
	Tx           []string `json:"tx"`
	Size         int      `json:"size"`         // Block size in bytes (from getblock verbosity=1)
	StrippedSize int      `json:"strippedsize"` // Block size excluding witness (optional)
}

// TxVerboseResult represents the result of getrawtransaction RPC with verbosity=1
type TxVerboseResult struct {
	TxID     string          `json:"txid"`
	Hex      string          `json:"hex"`
	Version  int32           `json:"version"`
	LockTime uint32          `json:"locktime"`
	Vin      []TxVerboseVin  `json:"vin"`
	Vout     []TxVerboseVout `json:"vout"`
}

// TxVerboseVin represents a transaction input in verbose format
type TxVerboseVin struct {
	TxID      string              `json:"txid"`
	Vout      uint32              `json:"vout"`
	ScriptSig *TxVerboseScriptSig `json:"scriptSig,omitempty"`
	Sequence  uint32              `json:"sequence"`
	Coinbase  string              `json:"coinbase,omitempty"`
}

// TxVerboseScriptSig represents scriptSig in verbose format
type TxVerboseScriptSig struct {
	Asm string `json:"asm"`
	Hex string `json:"hex"`
}

// TxVerboseVout represents a transaction output in verbose format
type TxVerboseVout struct {
	Value        float64               `json:"value"`
	N            uint32                `json:"n"`
	ScriptPubKey TxVerboseScriptPubKey `json:"scriptPubKey"`
}

// TxVerboseScriptPubKey represents scriptPubKey in verbose format
type TxVerboseScriptPubKey struct {
	Asm       string   `json:"asm"`
	Hex       string   `json:"hex"`
	ReqSigs   int      `json:"reqSigs,omitempty"`
	Type      string   `json:"type"`
	Addresses []string `json:"addresses,omitempty"`
}

// GetBlockVerbose get block with verbosity=1 (returns transaction IDs only)
func (s *BlockScanner) GetBlockVerbose(blockhash string) (*BlockVerboseResult, error) {
	request := RPCRequest{
		Jsonrpc: "1.0",
		ID:      "getblock",
		Method:  "getblock",
		Params:  []interface{}{blockhash, 1}, // verbosity=1 returns transaction IDs
	}

	response, err := s.rpcCall(request)
	if err != nil {
		return nil, err
	}

	if response.Error != nil {
		return nil, fmt.Errorf("rpc error: %s", response.Error.Message)
	}

	// Parse JSON response
	resultJSON, err := json.Marshal(response.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var blockVerbose BlockVerboseResult
	if err := json.Unmarshal(resultJSON, &blockVerbose); err != nil {
		return nil, fmt.Errorf("failed to unmarshal block verbose result: %w", err)
	}

	return &blockVerbose, nil
}

// GetRawTransactionVerbose get raw transaction with verbosity=1
func (s *BlockScanner) GetRawTransactionVerbose(txid string) (*TxVerboseResult, error) {
	request := RPCRequest{
		Jsonrpc: "1.0",
		ID:      "getrawtransaction",
		Method:  "getrawtransaction",
		Params:  []interface{}{txid, 1}, // verbosity=1 returns decoded transaction
	}

	response, err := s.rpcCall(request)
	if err != nil {
		return nil, err
	}

	if response.Error != nil {
		return nil, fmt.Errorf("rpc error: %s", response.Error.Message)
	}

	// Parse JSON response
	resultJSON, err := json.Marshal(response.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var txVerbose TxVerboseResult
	if err := json.Unmarshal(resultJSON, &txVerbose); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transaction verbose result: %w", err)
	}

	return &txVerbose, nil
}

// parseHexUint32 parses a hex string to uint32
func parseHexUint32(hexStr string) (uint32, error) {
	var result uint32
	_, err := fmt.Sscanf(hexStr, "%x", &result)
	return result, err
}

// mustParseChainhash parses a hex string to chainhash.Hash, panics on error
func mustParseChainhash(hexStr string) chainhash.Hash {
	hash, err := chainhash.NewHashFromStr(hexStr)
	if err != nil {
		panic(fmt.Sprintf("failed to parse chainhash %s: %v", hexStr, err))
	}
	return *hash
}

// parseDOGETxFromVerbose parses a DOGE transaction from verbose format
func parseDOGETxFromVerbose(txVerbose *TxVerboseResult) (*btcwire.MsgTx, error) {
	tx := btcwire.NewMsgTx(btcwire.TxVersion)
	tx.Version = txVerbose.Version
	tx.LockTime = txVerbose.LockTime

	// Parse inputs
	for _, vin := range txVerbose.Vin {
		var txIn *btcwire.TxIn

		if vin.Coinbase != "" {
			// Coinbase transaction
			coinbaseScript, err := hex.DecodeString(vin.Coinbase)
			if err != nil {
				return nil, fmt.Errorf("failed to decode coinbase script: %w", err)
			}
			// Coinbase uses nil OutPoint
			txIn = btcwire.NewTxIn(&btcwire.OutPoint{}, coinbaseScript, nil)
		} else {
			// Regular transaction input
			txHash, err := chainhash.NewHashFromStr(vin.TxID)
			if err != nil {
				return nil, fmt.Errorf("failed to parse txid: %w", err)
			}
			outPoint := btcwire.OutPoint{
				Hash:  *txHash,
				Index: vin.Vout,
			}

			var scriptSig []byte
			if vin.ScriptSig != nil {
				scriptSig, err = hex.DecodeString(vin.ScriptSig.Hex)
				if err != nil {
					return nil, fmt.Errorf("failed to decode scriptSig: %w", err)
				}
			}

			txIn = btcwire.NewTxIn(&outPoint, scriptSig, nil)
		}

		txIn.Sequence = vin.Sequence
		tx.AddTxIn(txIn)
	}

	// Parse outputs
	for _, vout := range txVerbose.Vout {
		scriptPubKey, err := hex.DecodeString(vout.ScriptPubKey.Hex)
		if err != nil {
			return nil, fmt.Errorf("failed to decode scriptPubKey: %w", err)
		}

		// Convert value from float64 to int64 (satoshis)
		// DOGE uses 8 decimal places like BTC
		value := int64(vout.Value * 1e8)

		txOut := btcwire.NewTxOut(value, scriptPubKey)
		tx.AddTxOut(txOut)
	}

	return tx, nil
}

// getDOGEBlockByRPC manually fetches block and individual transactions using verbose RPC to avoid parsing issues
func (s *BlockScanner) getDOGEBlockByRPC(blockhash string) (*btcwire.MsgBlock, error) {
	// Get block verbose first to get tx list and header info
	blockVerbose, err := s.GetBlockVerbose(blockhash)
	if err != nil {
		return nil, fmt.Errorf("GetBlockVerbose failed: %w", err)
	}

	// Parse bits
	bits, err := parseHexUint32(blockVerbose.Bits)
	if err != nil {
		return nil, fmt.Errorf("failed to parse bits: %w", err)
	}

	// Build MsgBlock
	msgBlock := &btcwire.MsgBlock{
		Header: btcwire.BlockHeader{
			Version:    blockVerbose.Version,
			PrevBlock:  mustParseChainhash(blockVerbose.PreviousHash),
			MerkleRoot: mustParseChainhash(blockVerbose.MerkleRoot),
			Timestamp:  time.Unix(blockVerbose.Time, 0),
			Bits:       bits,
			Nonce:      blockVerbose.Nonce,
		},
		Transactions: make([]*btcwire.MsgTx, 0, len(blockVerbose.Tx)),
	}

	// Get each transaction individually by txid using verbose format
	for _, txid := range blockVerbose.Tx {
		txhash, err := chainhash.NewHashFromStr(txid)
		if err != nil {
			continue
		}

		txVerbose, err := s.GetRawTransactionVerbose(txhash.String())
		if err != nil {
			continue
		}

		tx, err := parseDOGETxFromVerbose(txVerbose)
		if err != nil {
			log.Printf("[DOGE] Failed to parse transaction %s: %v", txid, err)
			continue
		}
		msgBlock.Transactions = append(msgBlock.Transactions, tx)
	}

	return msgBlock, nil
}

// GetRawMempool get all transaction IDs in mempool
func (s *BlockScanner) GetRawMempool() ([]string, error) {
	request := RPCRequest{
		Jsonrpc: "1.0",
		ID:      "getrawmempool",
		Method:  "getrawmempool",
		Params:  []interface{}{false}, // verbose=false, return array of txids
	}

	response, err := s.rpcCall(request)
	if err != nil {
		return nil, err
	}

	if response.Error != nil {
		return nil, fmt.Errorf("rpc error: %s", response.Error.Message)
	}

	// Response should be an array of transaction IDs
	txidsInterface, ok := response.Result.([]interface{})
	if !ok {
		return nil, errors.New("invalid mempool response format")
	}

	txids := make([]string, 0, len(txidsInterface))
	for _, txidInterface := range txidsInterface {
		if txid, ok := txidInterface.(string); ok {
			txids = append(txids, txid)
		}
	}

	return txids, nil
}

// ScanMempool scan all transactions in mempool and process MetaID transactions
// Returns the number of processed MetaID transactions
func (s *BlockScanner) ScanMempool(handler func(tx interface{}, metaDataTx *MetaIDDataTx, height, timestamp int64) error) (int, error) {
	// Get all transaction IDs in mempool
	txids, err := s.GetRawMempool()
	if err != nil {
		return 0, fmt.Errorf("failed to get mempool: %w", err)
	}

	if len(txids) == 0 {
		log.Printf("[%s] Mempool is empty", s.chainType)
		return 0, nil
	}

	log.Printf("[%s] Scanning %d transactions in mempool...", s.chainType, len(txids))

	processedCount := 0
	metaidPinCount := 0
	currentTime := time.Now().UnixMilli()

	for i, txid := range txids {
		// Get raw transaction hex
		txHex, err := s.GetRawTransaction(txid)
		if err != nil {
			log.Printf("[%s] Failed to get transaction %s from mempool: %v", s.chainType, txid, err)
			continue
		}

		// Decode hex to bytes
		txBytes, err := hex.DecodeString(txHex)
		if err != nil {
			log.Printf("[%s] Failed to decode transaction hex: %v", s.chainType, err)
			continue
		}

		// Deserialize transaction based on chain type
		var tx interface{}
		if s.chainType == ChainTypeBTC || s.chainType == ChainTypeDOGE {
			var btcTx btcwire.MsgTx
			chainName := "BTC"
			if s.chainType == ChainTypeDOGE {
				chainName = "DOGE"
			}
			if err := btcTx.Deserialize(bytes.NewReader(txBytes)); err != nil {
				log.Printf("[%s] Failed to deserialize %s transaction: %v", s.chainType, chainName, err)
				continue
			}
			tx = &btcTx
		} else {
			var mvcTx wire.MsgTx
			if err := mvcTx.Deserialize(bytes.NewReader(txBytes)); err != nil {
				log.Printf("[%s] Failed to deserialize MVC transaction: %v", s.chainType, err)
				continue
			}
			tx = &mvcTx
		}

		// Parse MetaID data using shared parser
		metaDataTx, err := s.parser.ParseAllPINs(tx, s.chainType)
		if err != nil || metaDataTx == nil {
			// not MetaID transaction, skip
			continue
		}

		metaidPinCount += len(metaDataTx.MetaIDData)

		// Call handler with height=0 (mempool transaction)
		if err := handler(tx, metaDataTx, 0, currentTime); err != nil {
			log.Printf("[%s] Failed to handle mempool transaction %s: %v", s.chainType, metaDataTx.TxID, err)
		} else {
			processedCount++
		}

		// Log progress every 100 transactions
		if (i+1)%100 == 0 {
			log.Printf("[%s] Mempool scan progress: %d/%d transactions processed", s.chainType, i+1, len(txids))
		}
	}

	log.Printf("[%s] Mempool scan complete: processed %d MetaID transactions with %d PINs from %d total transactions",
		s.chainType, processedCount, metaidPinCount, len(txids))

	return processedCount, nil
}

// GetBlockMsg get block message (MsgBlock) with all transactions, or *LazyBlock for large blocks.
// Returns interface{} which can be *wire.MsgBlock (MVC), *btcwire.MsgBlock (BTC/DOGE), or *LazyBlock.
// For large blocks (size > largeBlockThresholdBytes), returns LazyBlock to avoid loading full block into memory.
func (s *BlockScanner) GetBlockMsg(height int64) (interface{}, int, error) {
	blockhash, err := s.GetBlockhash(height)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get block hash: %w", err)
	}

	verbose, err := s.GetBlockVerbose(blockhash)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get block verbose: %w", err)
	}

	threshold := s.largeBlockThresholdBytes
	if threshold <= 0 {
		threshold = DefaultLargeBlockThresholdBytes
	}

	useLazy := false
	if verbose.Size > 0 {
		useLazy = int64(verbose.Size) > threshold
	} else if len(verbose.Tx) > LargeBlockTxCountFallback {
		// RPC did not return size; use tx count as fallback
		useLazy = true
	}

	if useLazy {
		timestampMs := verbose.Time * 1000
		if verbose.Time > 1e12 {
			timestampMs = verbose.Time
		}
		lazy := &LazyBlock{
			TxIDs:     verbose.Tx,
			Timestamp: timestampMs,
			Blockhash: blockhash,
		}
		log.Printf("Using lazy block loading for block %s at height %d, size: %dMB, tx count: %d", blockhash, height, verbose.Size/1024/1024, len(verbose.Tx))
		return lazy, len(verbose.Tx), nil
	}

	// Normal path: load full block
	if s.chainType == ChainTypeDOGE {
		msgBlock, err := s.getDOGEBlockByRPC(blockhash)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get DOGE block by RPC: %w", err)
		}
		return msgBlock, len(msgBlock.Transactions), nil
	}

	blockHex, err := s.GetBlockHex(blockhash)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get block hex: %w", err)
	}
	blockBytes, err := hex.DecodeString(blockHex)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to decode block hex: %w", err)
	}

	if s.chainType == ChainTypeBTC {
		var msgBlock btcwire.MsgBlock
		if err := msgBlock.Deserialize(bytes.NewReader(blockBytes)); err != nil {
			return nil, 0, fmt.Errorf("failed to deserialize BTC block: %w", err)
		}
		return &msgBlock, len(msgBlock.Transactions), nil
	}
	var msgBlock wire.MsgBlock
	if err := msgBlock.Deserialize(bytes.NewReader(blockBytes)); err != nil {
		return nil, 0, fmt.Errorf("failed to deserialize MVC block: %w", err)
	}
	return &msgBlock, len(msgBlock.Transactions), nil
}

// ScanBlock scan specified block
// handler accepts interface{} for tx to support both BTC and MVC
// Returns the number of processed MetaID transactions
func (s *BlockScanner) ScanBlock(height int64, handler func(tx interface{}, metaDataTx *MetaIDDataTx, height, timestamp int64) error) (int, error) {
	// Get block message with all transactions (or LazyBlock for large blocks)
	msgBlockInterface, txCount, err := s.GetBlockMsg(height)
	if err != nil {
		return 0, fmt.Errorf("failed to get block message: %w", err)
	}

	processedCount := 0
	metaidPinCount := 0

	// Large block path: iterate txids and fetch each tx on demand
	if lazy, ok := msgBlockInterface.(*LazyBlock); ok {
		for _, txid := range lazy.TxIDs {
			tx, err := s.GetAndDeserializeTx(txid)
			if err != nil {
				log.Printf("[%s] Failed to get tx %s in block %d: %v", s.chainType, txid, height, err)
				continue
			}
			metaDataTx, err := s.parser.ParseAllPINs(tx, s.chainType)
			if err != nil || metaDataTx == nil {
				continue
			}
			metaidPinCount += len(metaDataTx.MetaIDData)
			if err := handler(tx, metaDataTx, height, lazy.Timestamp); err != nil {
				log.Printf("[%s] Failed to handle transaction %s: %v", s.chainType, metaDataTx.TxID, err)
			} else {
				processedCount++
			}
		}
		log.Printf("Scanned block at height %d (lazy), transaction count: %d (chain: %s), MetaID PIN count: %d", height, txCount, s.chainType, metaidPinCount)
		return processedCount, nil
	}

	// Process transactions based on chain type (full block in memory)
	if s.chainType == ChainTypeBTC || s.chainType == ChainTypeDOGE {
		// BTC/DOGE block
		btcBlock, ok := msgBlockInterface.(*btcwire.MsgBlock)
		if !ok {
			chainName := "BTC"
			if s.chainType == ChainTypeDOGE {
				chainName = "DOGE"
			}
			return 0, fmt.Errorf("invalid %s block type", chainName)
		}
		timestamp := btcBlock.Header.Timestamp.UnixMilli()

		// Traverse transactions
		for _, tx := range btcBlock.Transactions {
			// Parse MetaID data using shared parser
			metaDataTx, err := s.parser.ParseAllPINs(tx, s.chainType)
			if err != nil {
				// not MetaID transaction, skip
				continue
			}
			if metaDataTx == nil {
				// not MetaID transaction, skip
				continue
			}
			metaidPinCount += len(metaDataTx.MetaIDData)

			// Call handler
			chainName := "BTC"
			if s.chainType == ChainTypeDOGE {
				chainName = "DOGE"
			}
			if err := handler(tx, metaDataTx, height, timestamp); err != nil {
				log.Printf("Failed to handle %s transaction %s: %v", chainName, metaDataTx.TxID, err)
			} else {
				processedCount++
			}
		}
	} else {
		// MVC block
		mvcBlock, ok := msgBlockInterface.(*wire.MsgBlock)
		if !ok {
			return 0, errors.New("invalid MVC block type")
		}
		timestamp := mvcBlock.Header.Timestamp.UnixMilli()

		// Traverse transactions
		for _, tx := range mvcBlock.Transactions {
			// Parse MetaID data using shared parser
			metaDataTx, err := s.parser.ParseAllPINs(tx, ChainTypeMVC)
			if err != nil {
				// not MetaID transaction, skip
				continue
			}
			if metaDataTx == nil {
				// not MetaID transaction, skip
				continue
			}
			// has
			// for _, pin := range metaDataTx.MetaIDData {
			// 	if
			// }

			// Call handler
			if err := handler(tx, metaDataTx, height, timestamp); err != nil {
				log.Printf("Failed to handle MVC transaction %s: %v", metaDataTx.TxID, err)
			} else {
				processedCount++
			}
			metaidPinCount += len(metaDataTx.MetaIDData)
		}
	}
	log.Printf("Scanned block at height %d, transaction count: %d (chain: %s), MetaID PIN count: %d", height, txCount, s.chainType, metaidPinCount)

	return processedCount, nil
}

// Start start scanner
// handler accepts interface{} for tx to support both BTC and MVC
// onBlockComplete is called after each block is successfully scanned
func (s *BlockScanner) Start(
	handler func(tx interface{}, metaDataTx *MetaIDDataTx, height, timestamp int64) error,
	onBlockComplete func(height int64) error,
) {
	currentHeight := s.startHeight
	log.Printf("Block scanner started from height %d (chain: %s)", currentHeight, s.chainType)

	zmqStarted := false // Track if ZMQ has been started

	for {
		// get latest block height
		latestHeight, err := s.GetBlockCount()
		if err != nil {
			log.Printf("Failed to get block count: %v", err)
			time.Sleep(s.interval)
			continue
		}

		// if new blocks exist, start scan
		if currentHeight <= latestHeight {
			blocksToScan := latestHeight - currentHeight + 1

			// Create progress bar for this batch
			s.progressBar = progressbar.NewOptions64(
				blocksToScan,
				progressbar.OptionSetDescription(fmt.Sprintf("[%s] Scanning blocks", s.chainType)),
				progressbar.OptionSetWidth(50),
				progressbar.OptionShowCount(),
				progressbar.OptionShowIts(),
				progressbar.OptionSetItsString("blocks"),
				progressbar.OptionThrottle(100*time.Millisecond),
				progressbar.OptionShowElapsedTimeOnFinish(),
				progressbar.OptionSetPredictTime(true),
				progressbar.OptionFullWidth(),
				progressbar.OptionSetRenderBlankState(true),
			)

			log.Printf("Starting to scan %d blocks (from %d to %d)", blocksToScan, currentHeight, latestHeight)

			for currentHeight <= latestHeight {
				_, err := s.ScanBlock(currentHeight, handler)
				if err != nil {
					log.Printf("\nFailed to scan block %d: %v", currentHeight, err)
					time.Sleep(s.interval)
					continue
				}

				// Call onBlockComplete callback to update sync status
				if onBlockComplete != nil {
					if err := onBlockComplete(currentHeight); err != nil {
						log.Printf("Failed to update sync status for block %d: %v", currentHeight, err)
					}
				}

				// Update progress bar
				s.progressBar.Add(1)
				currentHeight++
			}

			// Finish progress bar
			s.progressBar.Finish()
			s.progressBar = nil // Clear reference to help GC
			log.Printf("\nCompleted scanning to block %d", latestHeight)

			// Start ZMQ client after catching up to latest block (only once)
			if !zmqStarted && s.zmqEnabled && s.zmqClient != nil {
				log.Printf("✅ Caught up to latest block, starting ZMQ real-time monitoring...")

				// Set ZMQ transaction handler (without height parameter for mempool txs)
				s.zmqClient.SetTransactionHandler(func(tx interface{}, metaDataTx *MetaIDDataTx) error {
					// Call the same handler but with height = 0 (mempool transaction)
					return handler(tx, metaDataTx, 0, time.Now().UnixMilli())
				})

				// Start ZMQ client
				if err := s.zmqClient.StartWithRawTx(); err != nil {
					log.Printf("Failed to start ZMQ client: %v", err)
				} else {
					zmqStarted = true
					log.Printf("✅ ZMQ real-time monitoring started successfully")
				}
			}
		} else {
			// Already at latest block
			if !zmqStarted {
				log.Printf("Already at latest block %d", currentHeight-1)

				// Start ZMQ if enabled and not started yet
				if s.zmqEnabled && s.zmqClient != nil {
					log.Printf("✅ At latest block, starting ZMQ real-time monitoring...")

					// Set ZMQ transaction handler
					s.zmqClient.SetTransactionHandler(func(tx interface{}, metaDataTx *MetaIDDataTx) error {
						return handler(tx, metaDataTx, 0, time.Now().UnixMilli())
					})

					// Start ZMQ client
					if err := s.zmqClient.StartWithRawTx(); err != nil {
						log.Printf("Failed to start ZMQ client: %v", err)
					} else {
						zmqStarted = true
						log.Printf("✅ ZMQ real-time monitoring started successfully")
					}
				}
			} else {
				log.Printf("ZMQ real-time monitoring already started")
			}
		}

		// wait for next scan
		time.Sleep(s.interval)
	}
}

// Stop stop scanner and ZMQ client
func (s *BlockScanner) Stop() {
	log.Println("Stopping block scanner...")

	// Stop ZMQ client if running
	if s.zmqClient != nil {
		s.zmqClient.Stop()
	}

	log.Println("Block scanner stopped")
}

// rpcCall execute RPC call
func (s *BlockScanner) rpcCall(request RPCRequest) (*RPCResponse, error) {
	// set authentication header
	headers := map[string]string{
		"Authorization": "Basic " + tool.Base64Encode(s.rpcUser+":"+s.rpcPassword),
	}

	// Send request
	respStr, err := tool.PostUrl(s.rpcURL, request, headers)
	if err != nil {
		return nil, fmt.Errorf("rpc call failed: %w", err)
	}

	// Parse response
	var response RPCResponse
	if err := json.Unmarshal([]byte(respStr), &response); err != nil {
		return nil, fmt.Errorf("failed to parse rpc response: %w", err)
	}

	return &response, nil
}
