package indexer

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/bitcoinsv/bsvd/wire"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	btcwire "github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil/base58"
	"github.com/metaid-developers/metaid-script-decoder/decoder"
	"github.com/metaid-developers/metaid-script-decoder/decoder/btc"
	"github.com/metaid-developers/metaid-script-decoder/decoder/mvc"
)

// ChainType represents the blockchain type
type ChainType string

const (
	ChainTypeBTC ChainType = "btc"
	ChainTypeMVC ChainType = "mvc"
)

type MetaIDDataTx struct {
	TxID       string // Transaction ID
	ChainName  string // Chain name: btc, mvc
	MetaIDData []*MetaIDData
}

// MetaIDData MetaID protocol data
type MetaIDData struct {
	PinID                string // PIN ID
	Operation            string // create/modify/revoke
	OriginalPath         string // Original path
	Host                 string // Host
	Path                 string // File path
	ParentPath           string // Parent path
	Encryption           string // Encryption method
	Version              string // Version
	ContentType          string // Content type
	Content              []byte // File content
	TxID                 string // Transaction ID
	Vout                 uint32 // Output index
	CreatorInputLocation string // Creator input location txId:vin
	CreatorAddress       string // Creator address
	OwnerAddress         string // Owner address
	ChainName            string // Chain name: btc, mvc
}

// MetaIDParser MetaID protocol parser
type MetaIDParser struct {
	btcParser    decoder.ChainParser
	mvcParser    decoder.ChainParser
	config       *decoder.ParserConfig
	blockScanner *BlockScanner // RPC client for fetching transactions
}

// NewMetaIDParser create a new MetaID parser
func NewMetaIDParser(protocolID string) *MetaIDParser {
	var config *decoder.ParserConfig
	if protocolID != "" {
		config = &decoder.ParserConfig{
			ProtocolID: protocolID,
		}
	}

	return &MetaIDParser{
		btcParser: btc.NewBTCParser(config),
		mvcParser: mvc.NewMVCParser(config),
		config:    config,
	}
}

// SetBlockScanner set block scanner for RPC calls
func (p *MetaIDParser) SetBlockScanner(scanner *BlockScanner) {
	p.blockScanner = scanner
}

// // ParseTransaction parse transaction and extract MetaID data with specified chain type
// // tx: can be *wire.MsgTx (MVC) or *btcwire.MsgTx (BTC)
// // chainType: ChainTypeBTC or ChainTypeMVC - specifies which parser to try first and how to interpret tx
// // It will try the specified chain parser first, then fallback to the other
// func (p *MetaIDParser) ParseTransaction(tx interface{}, chainType ChainType) (*MetaIDData, error) {
// 	var txBytes []byte
// 	var txID string
// 	var address string
// 	var err error

// 	// Type assertion based on chainType
// 	if chainType == ChainTypeBTC {
// 		// Expect BTC transaction
// 		btcTx, ok := tx.(*btcwire.MsgTx)
// 		if !ok {
// 			return nil, errors.New("invalid transaction type: expected *btcwire.MsgTx for BTC chain")
// 		}

// 		// Serialize BTC transaction
// 		var buf bytes.Buffer
// 		if err = btcTx.Serialize(&buf); err != nil {
// 			return nil, fmt.Errorf("failed to serialize BTC transaction: %w", err)
// 		}
// 		txBytes = buf.Bytes()
// 		txID = btcTx.TxHash().String()
// 		address = extractBTCAddress(btcTx)
// 	} else {
// 		// Expect MVC transaction
// 		mvcTx, ok := tx.(*wire.MsgTx)
// 		if !ok {
// 			return nil, errors.New("invalid transaction type: expected *wire.MsgTx for MVC chain")
// 		}

// 		// Serialize MVC transaction
// 		var buf bytes.Buffer
// 		if err = mvcTx.Serialize(&buf); err != nil {
// 			return nil, fmt.Errorf("failed to serialize MVC transaction: %w", err)
// 		}
// 		txBytes = buf.Bytes()
// 		txID = mvcTx.TxHash().String()
// 		address = extractMVCAddress(mvcTx)
// 	}

// 	// Try to parse with specified chain type first
// 	var pins []*decoder.Pin
// 	var chainName string

// 	if chainType == ChainTypeBTC {
// 		// Try BTC parser first
// 		pins, err = p.btcParser.ParseTransaction(txBytes, &chaincfg.MainNetParams)
// 		if err == nil && len(pins) > 0 {
// 			chainName = "btc"
// 		}
// 	} else {
// 		// Try MVC parser first
// 		pins, err = p.mvcParser.ParseTransaction(txBytes, &chaincfg.MainNetParams)
// 		if err == nil && len(pins) > 0 {
// 			chainName = "mvc"
// 		} else {
// 		}
// 	}

// 	// Check if any PIN data was found
// 	if err != nil || len(pins) == 0 {
// 		return nil, nil
// 	}

// 	// Use the first PIN data found
// 	pin := pins[0]

// 	// Convert PIN to MetaIDData (address already extracted above)
// 	data := &MetaIDData{
// 		Operation:   pin.Operation,
// 		Path:        pin.Path,
// 		ParentPath:  pin.ParentPath,
// 		Encryption:  pin.Encryption,
// 		Version:     pin.Version,
// 		ContentType: pin.ContentType,
// 		Content:     pin.ContentBody,
// 		TxID:        txID,
// 		Vout:        pin.Vout,
// 		Address:     address,
// 		ChainName:   chainName,
// 	}

// 	return data, nil
// }

// ParseTransactionWithTxID parse transaction with explicit transaction ID and chain type
// tx: can be *wire.MsgTx (MVC) or *btcwire.MsgTx (BTC)
func (p *MetaIDParser) ParseTransactionWithTxID(tx interface{}, txID string, chainType ChainType) (*MetaIDDataTx, error) {
	return p.ParseAllPINs(tx, chainType)
}

// ParseAllPINs parse all PIN data from transaction with specified chain type (for MVC)
func (p *MetaIDParser) ParseAllPINs(tx interface{}, chainType ChainType) (*MetaIDDataTx, error) {
	var txBytes []byte
	var txID string
	var address string
	var err error
	_ = address

	// Type assertion based on chainType
	if chainType == ChainTypeBTC {
		// Expect BTC transaction
		btcTx, ok := tx.(*btcwire.MsgTx)
		if !ok {
			return nil, errors.New("invalid transaction type: expected *btcwire.MsgTx for BTC chain")
		}

		// Serialize BTC transaction
		var buf bytes.Buffer
		if err = btcTx.Serialize(&buf); err != nil {
			return nil, fmt.Errorf("failed to serialize BTC transaction: %w", err)
		}
		txBytes = buf.Bytes()
		txID = btcTx.TxHash().String()
		address = extractBTCCreatorAddress(btcTx)
	} else {
		// Expect MVC transaction
		mvcTx, ok := tx.(*wire.MsgTx)
		if !ok {
			return nil, errors.New("invalid transaction type: expected *wire.MsgTx for MVC chain")
		}

		// Serialize MVC transaction
		var buf bytes.Buffer
		if err = mvcTx.Serialize(&buf); err != nil {
			return nil, fmt.Errorf("failed to serialize MVC transaction: %w", err)
		}
		txBytes = buf.Bytes()
		txID = mvcTx.TxHash().String()
		address = extractMVCCreatorAddress(mvcTx)
	}

	// Try to parse with specified chain type first
	var pins []*decoder.Pin
	var chainName string

	if chainType == ChainTypeBTC {
		// Try BTC parser first
		pins, err = p.btcParser.ParseTransaction(txBytes, &chaincfg.MainNetParams)
		if err == nil && len(pins) > 0 {
			chainName = "btc"
		}
	} else {
		// Try MVC parser first
		pins, err = p.mvcParser.ParseTransaction(txBytes, nil)
		if err == nil && len(pins) > 0 {
			chainName = "mvc"
		}
	}

	// Check if any PIN data was found
	if err != nil || len(pins) == 0 {
		return nil, nil
	}

	// Convert all PINs to MetaIDData (address already extracted above)
	var results []*MetaIDData
	for _, pin := range pins {
		data := &MetaIDData{
			PinID:                pin.Id,
			Operation:            pin.Operation,
			OriginalPath:         pin.OriginalPath,
			Host:                 pin.Host,
			Path:                 pin.Path,
			ParentPath:           pin.ParentPath,
			Encryption:           pin.Encryption,
			Version:              pin.Version,
			ContentType:          pin.ContentType,
			Content:              pin.ContentBody,
			TxID:                 txID,
			Vout:                 pin.Vout,
			CreatorAddress:       pin.OwnerAddress,
			CreatorInputLocation: pin.CreatorInputLocation,
			OwnerAddress:         pin.OwnerAddress,
			ChainName:            chainName,
		}
		results = append(results, data)
	}

	return &MetaIDDataTx{
		TxID:       txID,
		ChainName:  chainName,
		MetaIDData: results,
	}, nil
}

// extractBTCAddress extract address from BTC transaction first input
func extractBTCCreatorAddress(tx *btcwire.MsgTx) string {
	// In Bitcoin, the address is typically extracted from the first input's previous output
	// This is a simplified implementation - in production you may need to query the previous transaction
	// to get the actual address
	if len(tx.TxIn) > 0 {
		// Return the previous transaction hash as a placeholder
		// In a real implementation, you would need to:
		// 1. Get the previous transaction using tx.TxIn[0].PreviousOutPoint.Hash
		// 2. Extract the address from that transaction's output
		return tx.TxIn[0].PreviousOutPoint.Hash.String()
	}
	return ""
}

// extractMVCAddress extract address from MVC transaction first input
func extractMVCCreatorAddress(tx *wire.MsgTx) string {
	// In MVC, the address is typically extracted from the first input's previous output
	// This is a simplified implementation - in production you may need to query the previous transaction
	// to get the actual address
	if len(tx.TxIn) > 0 {
		// Return the previous transaction hash as a placeholder
		// In a real implementation, you would need to:
		// 1. Get the previous transaction using tx.TxIn[0].PreviousOutPoint.Hash
		// 2. Extract the address from that transaction's output
		return tx.TxIn[0].PreviousOutPoint.Hash.String()
	}
	return ""
}

// TxToHex convert MVC transaction to hexadecimal string (backward compatibility)
func TxToHex(tx *wire.MsgTx) (string, error) {
	var buf bytes.Buffer
	if err := tx.Serialize(&buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf.Bytes()), nil
}

// FindCreatorAddressFromCreatorInputLocation find creator address from CreatorInputLocation
// CreatorInputLocation format: "txid:vin" (e.g., "abc123def456:0")
// Returns the address from the specified input of the referenced transaction
func (p *MetaIDParser) FindCreatorAddressFromCreatorInputLocation(creatorInputLocation string, chainType ChainType) (string, error) {
	if creatorInputLocation == "" {
		return "", errors.New("creatorInputLocation is empty")
	}

	if p.blockScanner == nil {
		return "", errors.New("blockScanner not set, cannot fetch transaction from node")
	}

	// Parse CreatorInputLocation: "txid:vin"
	parts := bytes.Split([]byte(creatorInputLocation), []byte(":"))
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid creatorInputLocation format: %s (expected txid:vin)", creatorInputLocation)
	}

	txid := string(parts[0])
	voutStr := string(parts[1])

	// Parse vin (input index)
	var vout int
	if _, err := fmt.Sscanf(voutStr, "%d", &vout); err != nil {
		return "", fmt.Errorf("invalid vout in creatorInputLocation: %s", voutStr)
	}

	// Get raw transaction from node
	txHex, err := p.blockScanner.GetRawTransaction(txid)
	if err != nil {
		return "", fmt.Errorf("failed to get transaction %s: %w", txid, err)
	}

	// Decode hex to bytes
	txBytes, err := hex.DecodeString(txHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode transaction hex: %w", err)
	}

	// Deserialize transaction based on chain type
	var address string
	if chainType == ChainTypeBTC {
		// Parse as BTC transaction
		var btcTx btcwire.MsgTx
		if err := btcTx.Deserialize(bytes.NewReader(txBytes)); err != nil {
			return "", fmt.Errorf("failed to deserialize BTC transaction: %w", err)
		}

		// Get address from the specified input
		address, err = extractAddressFromBTCInput(&btcTx, vout)
		if err != nil {
			return "", fmt.Errorf("failed to extract address from BTC input: %w", err)
		}
	} else {
		// Parse as MVC transaction
		var mvcTx wire.MsgTx
		if err := mvcTx.Deserialize(bytes.NewReader(txBytes)); err != nil {
			return "", fmt.Errorf("failed to deserialize MVC transaction: %w", err)
		}

		// Get address from the specified input
		address, err = extractAddressFromMVCInput(&mvcTx, vout)
		if err != nil {
			return "", fmt.Errorf("failed to extract address from MVC input: %w", err)
		}
	}

	return address, nil
}

// extractAddressFromBTCInput extract address from BTC transaction output
func extractAddressFromBTCInput(tx *btcwire.MsgTx, outputIndex int) (string, error) {
	if outputIndex < 0 || outputIndex >= len(tx.TxOut) {
		return "", fmt.Errorf("output index %d out of range (total outputs: %d)", outputIndex, len(tx.TxOut))
	}

	output := tx.TxOut[outputIndex]

	// Extract address from scriptPubKey (P2PKH)
	scriptPubKey := output.PkScript
	if len(scriptPubKey) == 0 {
		return "", errors.New("empty script pubkey")
	}

	_, addresses, _, err := txscript.ExtractPkScriptAddrs(scriptPubKey, &chaincfg.MainNetParams)
	if err != nil {
		return "", fmt.Errorf("failed to extract addresses from script pubkey: %w", err)
	}
	if len(addresses) == 0 {
		return "", errors.New("no addresses found in script pubkey")
	}
	return addresses[0].EncodeAddress(), nil
}

// extractAddressFromMVCInput extract address from MVC transaction output
func extractAddressFromMVCInput(tx *wire.MsgTx, outputIndex int) (string, error) {
	if outputIndex < 0 || outputIndex >= len(tx.TxOut) {
		return "", fmt.Errorf("output index %d out of range (total outputs: %d)", outputIndex, len(tx.TxOut))
	}

	output := tx.TxOut[outputIndex]

	// Extract address from scriptPubKey (P2PKH)
	scriptPubKey := output.PkScript
	if len(scriptPubKey) == 0 {
		return "", errors.New("empty script pubkey")
	}

	_, addresses, _, err := txscript.ExtractPkScriptAddrs(scriptPubKey, &chaincfg.MainNetParams)
	if err != nil {
		return "", fmt.Errorf("failed to extract addresses from script pubkey: %w", err)
	}
	if len(addresses) == 0 {
		return "", errors.New("no addresses found in script pubkey")
	}
	return addresses[0].EncodeAddress(), nil
}

// pubKeyHashToAddress convert pubKeyHash to address
func pubKeyHashToAddress(pubKeyHash []byte, chainType ChainType) string {
	if len(pubKeyHash) != 20 {
		return ""
	}

	// For both BTC and MVC, we use the same address format (P2PKH)
	// 1. Add version byte (0x00 for mainnet)
	// 2. Calculate checksum (double SHA256)
	// 3. Base58 encode

	// Step 1: Add version byte (0x00 for mainnet P2PKH)
	versionedPayload := append([]byte{0x00}, pubKeyHash...)

	// Step 2: Calculate checksum (first 4 bytes of double SHA256)
	firstSHA := sha256.Sum256(versionedPayload)
	secondSHA := sha256.Sum256(firstSHA[:])
	checksum := secondSHA[:4]

	// Step 3: Append checksum and Base58 encode
	fullPayload := append(versionedPayload, checksum...)
	address := base58.Encode(fullPayload)

	return address
}
