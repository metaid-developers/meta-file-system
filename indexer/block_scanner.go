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
	btcwire "github.com/btcsuite/btcd/wire"
	"github.com/schollz/progressbar/v3"
)

// BlockScanner block scanner
type BlockScanner struct {
	rpcURL      string
	rpcUser     string
	rpcPassword string
	startHeight int64
	interval    time.Duration
	chainType   ChainType // Chain type: btc or mvc
	progressBar *progressbar.ProgressBar
	zmqClient   *ZMQClient    // ZMQ client for real-time transaction monitoring
	zmqEnabled  bool          // Whether ZMQ is enabled
	parser      *MetaIDParser // Shared parser to avoid repeated allocation
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
		if s.chainType == ChainTypeBTC {
			var btcTx btcwire.MsgTx
			if err := btcTx.Deserialize(bytes.NewReader(txBytes)); err != nil {
				log.Printf("[%s] Failed to deserialize BTC transaction: %v", s.chainType, err)
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

// GetBlockMsg get block message (MsgBlock) with all transactions
// Returns interface{} which can be *wire.MsgBlock (MVC) or *btcwire.MsgBlock (BTC)
func (s *BlockScanner) GetBlockMsg(height int64) (interface{}, int, error) {
	// Get block hash
	blockhash, err := s.GetBlockhash(height)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get block hash: %w", err)
	}

	// Get block hex
	blockHex, err := s.GetBlockHex(blockhash)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get block hex: %w", err)
	}

	// Decode hex to bytes
	blockBytes, err := hex.DecodeString(blockHex)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to decode block hex: %w", err)
	}

	// Deserialize based on chain type
	if s.chainType == ChainTypeBTC {
		// Parse as BTC block
		var msgBlock btcwire.MsgBlock
		if err := msgBlock.Deserialize(bytes.NewReader(blockBytes)); err != nil {
			return nil, 0, fmt.Errorf("failed to deserialize BTC block: %w", err)
		}
		txCount := len(msgBlock.Transactions)
		return &msgBlock, txCount, nil
	} else {
		// Parse as MVC block
		var msgBlock wire.MsgBlock
		if err := msgBlock.Deserialize(bytes.NewReader(blockBytes)); err != nil {
			return nil, 0, fmt.Errorf("failed to deserialize MVC block: %w", err)
		}
		txCount := len(msgBlock.Transactions)
		return &msgBlock, txCount, nil
	}
}

// ScanBlock scan specified block
// handler accepts interface{} for tx to support both BTC and MVC
// Returns the number of processed MetaID transactions
func (s *BlockScanner) ScanBlock(height int64, handler func(tx interface{}, metaDataTx *MetaIDDataTx, height, timestamp int64) error) (int, error) {
	// Get block message with all transactions
	msgBlockInterface, txCount, err := s.GetBlockMsg(height)
	if err != nil {
		return 0, fmt.Errorf("failed to get block message: %w", err)
	}

	// log.Printf("Scanning block at height %d, transaction count: %d (chain: %s)", height, txCount, s.chainType)

	processedCount := 0
	metaidPinCount := 0

	// Use shared parser to avoid repeated allocation
	// Note: parser := NewMetaIDParser("") removed to reduce memory allocation

	// Process transactions based on chain type
	if s.chainType == ChainTypeBTC {
		// BTC block
		btcBlock, ok := msgBlockInterface.(*btcwire.MsgBlock)
		if !ok {
			return 0, errors.New("invalid BTC block type")
		}
		timestamp := btcBlock.Header.Timestamp.UnixMilli()

		// Traverse transactions
		for _, tx := range btcBlock.Transactions {
			// Parse MetaID data using shared parser
			metaDataTx, err := s.parser.ParseAllPINs(tx, ChainTypeBTC)
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
			if err := handler(tx, metaDataTx, height, timestamp); err != nil {
				log.Printf("Failed to handle BTC transaction %s: %v", metaDataTx.TxID, err)
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
