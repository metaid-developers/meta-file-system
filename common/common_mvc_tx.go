package common

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"meta-file-system/tool"

	bsvec2 "github.com/bitcoinsv/bsvd/bsvec"
	chaincfg2 "github.com/bitcoinsv/bsvd/chaincfg"
	chainhash2 "github.com/bitcoinsv/bsvd/chaincfg/chainhash"
	txscript2 "github.com/bitcoinsv/bsvd/txscript"
	wire2 "github.com/bitcoinsv/bsvd/wire"
	bsvutil2 "github.com/bitcoinsv/bsvutil"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

type TxInputUtxo struct {
	TxId     string
	TxIndex  int64
	PkScript string
	Amount   uint64
	PriHex   string
	SignMode SignMode
}

type TxOutput struct {
	Address string
	Amount  int64
}

type SignMode string

const (
	SignModeSegwit  SignMode = "segwit"
	SignModeTaproot SignMode = "taproot"
	SignModeLegacy  SignMode = "legacy"
)

func BuildMvcCommonMetaIdTx(netParam *chaincfg2.Params, ins []*TxInputUtxo, outs, otherOuts []*TxOutput, operation, path string, content []byte, changeAddress string, feeRate int64, isUnSign bool) (*wire2.MsgTx, error) {
	tx := wire2.NewMsgTx(10)
	totalAmount := int64(0)
	outAmount := int64(0)
	for _, out := range outs {
		addr, err := bsvutil2.DecodeAddress(out.Address, netParam)
		if err != nil {
			return nil, err
		}
		pkScript, err := txscript2.PayToAddrScript(addr)
		if err != nil {
			return nil, err
		}
		tx.AddTxOut(wire2.NewTxOut(out.Amount, pkScript))
		outAmount = outAmount + out.Amount
	}

	if operation == "" {
		operation = "create"
	}
	inscriptionBuilder := txscript.NewScriptBuilder().
		AddOp(txscript.OP_0).
		AddOp(txscript.OP_RETURN).
		AddData([]byte("metaid")). //<metaid_flag>
		AddData([]byte(operation)) //<operation>

	//inscriptionBuilder.AddData([]byte("/file/my-bitcoin.png")) //<path>
	inscriptionBuilder.AddData([]byte(path))    //<path>
	inscriptionBuilder.AddData([]byte("0"))     //<Encryption>
	inscriptionBuilder.AddData([]byte("0.0.1")) //<version>
	//inscriptionBuilder.AddData([]byte("image/jpeg;binary"))    //<content-type>
	inscriptionBuilder.AddData([]byte("application/json")) //<content-type>

	maxChunkSize := 520
	bodySize := len(content)
	for i := 0; i < bodySize; i += maxChunkSize {
		end := i + maxChunkSize
		if end > bodySize {
			end = bodySize
		}
		inscriptionBuilder.AddFullData(content[i:end]) //<payload>
	}

	inscriptionScript, err := inscriptionBuilder.Script()
	if err != nil {
		return nil, err
	}

	tx.AddTxOut(wire2.NewTxOut(0, inscriptionScript))

	if otherOuts != nil {
		for _, out := range otherOuts {
			addr, err := bsvutil2.DecodeAddress(out.Address, netParam)
			if err != nil {
				return nil, err
			}
			pkScript, err := txscript2.PayToAddrScript(addr)
			if err != nil {
				return nil, err
			}
			tx.AddTxOut(wire2.NewTxOut(out.Amount, pkScript))
			outAmount = outAmount + out.Amount
		}
	}

	if changeAddress != "" {
		addr, err := bsvutil2.DecodeAddress(changeAddress, netParam)
		if err != nil {
			return nil, err
		}
		pkScriptByte, err := txscript2.PayToAddrScript(addr)
		tx.AddTxOut(wire2.NewTxOut(0, pkScriptByte))
	}

	for _, in := range ins {
		hash, err := chainhash2.NewHashFromStr(in.TxId)
		if err != nil {
			return nil, err
		}
		prevOut := wire2.NewOutPoint(hash, uint32(in.TxIndex))
		txIn := wire2.NewTxIn(prevOut, nil)
		tx.AddTxIn(txIn)
		totalAmount = totalAmount + int64(in.Amount)

	}
	txTotalSize := tx.SerializeSize()

	//txSize := tx.SerializeSize() + SpendSize*len(tx.TxIn)
	txFee := int64(txTotalSize) * feeRate

	fmt.Printf("txTotalSize:%d, txFee:%d, feeRate:%d, totalAmount:%d, outAmount:%d\n", txTotalSize, txFee, feeRate, totalAmount, outAmount)
	if totalAmount-outAmount < int64(txFee) {
		return nil, errors.New("insufficient fee")
	}

	changeVal := totalAmount - outAmount - int64(txFee)
	if changeVal >= 600 && changeAddress != "" {
		tx.TxOut[len(tx.TxOut)-1].Value = changeVal
	} else {
		tx.TxOut = tx.TxOut[:len(tx.TxOut)-1]
	}

	if !isUnSign {
		for i, in := range ins {
			privateKeyBytes, err := hex.DecodeString(in.PriHex)
			if err != nil {
				return nil, err
			}
			privateKey, _ := bsvec2.PrivKeyFromBytes(bsvec2.S256(), privateKeyBytes)

			pkScriptByte, err := hex.DecodeString(in.PkScript)
			if err != nil {
				return nil, err
			}

			var sigScript []byte
			sigScript, err = txscript2.SignatureScript(tx, i, int64(in.Amount), pkScriptByte, txscript2.SigHashAll, privateKey, true)
			if err != nil {
				fmt.Println(err)
				return nil, err
			}

			tx.TxIn[i].SignatureScript = sigScript
		}
	}

	return tx, nil
}

func BuildMvcCommonMetaIdTxForUnkwonInput(netParam *chaincfg2.Params, ins []*TxInputUtxo, outs, otherOuts []*TxOutput, operation, path string, content []byte, contentType string, changeAddress string, feeRate int64, isUnSign bool) (*wire2.MsgTx, error) {
	tx := wire2.NewMsgTx(10)
	totalAmount := int64(0)
	outAmount := int64(0)
	for _, out := range outs {
		addr, err := bsvutil2.DecodeAddress(out.Address, netParam)
		if err != nil {
			return nil, err
		}
		pkScript, err := txscript2.PayToAddrScript(addr)
		if err != nil {
			return nil, err
		}
		tx.AddTxOut(wire2.NewTxOut(out.Amount, pkScript))
		outAmount = outAmount + out.Amount
	}

	if operation == "" {
		operation = "create"
	}
	inscriptionBuilder := txscript.NewScriptBuilder().
		AddOp(txscript.OP_0).
		AddOp(txscript.OP_RETURN).
		AddData([]byte("metaid")). //<metaid_flag>
		AddData([]byte(operation)) //<operation>

	//inscriptionBuilder.AddData([]byte("/file/my-bitcoin.png")) //<path>
	inscriptionBuilder.AddData([]byte(path))    //<path>
	inscriptionBuilder.AddData([]byte("0"))     //<Encryption>
	inscriptionBuilder.AddData([]byte("1.0.0")) //<version>
	//inscriptionBuilder.AddData([]byte("image/jpeg;binary"))    //<content-type>
	inscriptionBuilder.AddData([]byte(contentType)) //<content-type>

	maxChunkSize := 520
	bodySize := len(content)
	for i := 0; i < bodySize; i += maxChunkSize {
		end := i + maxChunkSize
		if end > bodySize {
			end = bodySize
		}
		inscriptionBuilder.AddFullData(content[i:end]) //<payload>
	}

	inscriptionScript, err := inscriptionBuilder.Script()
	if err != nil {
		return nil, err
	}

	tx.AddTxOut(wire2.NewTxOut(0, inscriptionScript))

	if otherOuts != nil {
		for _, out := range otherOuts {
			addr, err := bsvutil2.DecodeAddress(out.Address, netParam)
			if err != nil {
				return nil, err
			}
			pkScript, err := txscript2.PayToAddrScript(addr)
			if err != nil {
				return nil, err
			}
			tx.AddTxOut(wire2.NewTxOut(out.Amount, pkScript))
			outAmount = outAmount + out.Amount
		}
	}

	if changeAddress != "" {
		addr, err := bsvutil2.DecodeAddress(changeAddress, netParam)
		if err != nil {
			return nil, err
		}
		pkScriptByte, err := txscript2.PayToAddrScript(addr)
		tx.AddTxOut(wire2.NewTxOut(0, pkScriptByte))
	}

	for _, in := range ins {
		hash, err := chainhash2.NewHashFromStr(in.TxId)
		if err != nil {
			return nil, err
		}
		prevOut := wire2.NewOutPoint(hash, uint32(in.TxIndex))
		txIn := wire2.NewTxIn(prevOut, nil)
		tx.AddTxIn(txIn)
		totalAmount = totalAmount + int64(in.Amount)
	}
	// txTotalSize := tx.SerializeSize()

	// //txSize := tx.SerializeSize() + SpendSize*len(tx.TxIn)
	// txFee := int64(txTotalSize) * feeRate

	// fmt.Printf("txTotalSize:%d, txFee:%d, feeRate:%d, totalAmount:%d, outAmount:%d\n", txTotalSize, txFee, feeRate, totalAmount, outAmount)
	// if totalAmount-outAmount < int64(txFee) {
	// 	return nil, errors.New("insufficient fee")
	// }

	// changeVal := totalAmount - outAmount - int64(txFee)
	// if changeVal >= 600 && changeAddress != "" {
	// 	tx.TxOut[len(tx.TxOut)-1].Value = changeVal
	// } else {
	// 	tx.TxOut = tx.TxOut[:len(tx.TxOut)-1]
	// }

	if !isUnSign {
		for i, in := range ins {
			privateKeyBytes, err := hex.DecodeString(in.PriHex)
			if err != nil {
				return nil, err
			}
			privateKey, _ := bsvec2.PrivKeyFromBytes(bsvec2.S256(), privateKeyBytes)

			pkScriptByte, err := hex.DecodeString(in.PkScript)
			if err != nil {
				return nil, err
			}

			var sigScript []byte
			sigScript, err = txscript2.SignatureScript(tx, i, int64(in.Amount), pkScriptByte, txscript2.SigHashAll, privateKey, true)
			if err != nil {
				fmt.Println(err)
				return nil, err
			}

			tx.TxIn[i].SignatureScript = sigScript
		}
	}

	return tx, nil
}

func ToRaw(tx *wire.MsgTx) (string, error) {
	buf := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
	if err := tx.Serialize(buf); err != nil {
		return "", err
	}
	txHex := hex.EncodeToString(buf.Bytes())
	return txHex, nil
}

func MvcToRaw(tx *wire2.MsgTx) (string, error) {
	buf := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
	if err := tx.Serialize(buf); err != nil {
		return "", err
	}
	txHex := hex.EncodeToString(buf.Bytes())
	tx.TxHash()
	return txHex, nil
}

func GetMvcTxhashFromRaw(rawTx string) string {
	rawTxByte, err := hex.DecodeString(rawTx)
	if err != nil {
		return ""
	}
	return GetMvcTxhash(rawTxByte)
}

func GetMvcTxhash(rawTxByte []byte) string {
	limit := len(rawTxByte)
	if limit == 0 {
		return ""
	}
	index := 0
	if index+4 > limit {
		return ""
	}
	versionByte := rawTxByte[index : index+4]
	version := uint64(binary.LittleEndian.Uint32(versionByte))
	if version >= 10 {
		rawTxByte = GetTxNewhash(rawTxByte)
	}

	txhash := tool.DoubleSHA256(rawTxByte)
	for i := 0; i < len(txhash)/2; i++ {
		h := txhash[len(txhash)-1-i]
		txhash[len(txhash)-1-i] = txhash[i]
		txhash[i] = h
	}
	return hex.EncodeToString(txhash)
}

func GetTxNewhash(rawTxByte []byte) []byte {
	var (
		newRawTxByte []byte
		tx           *wire2.MsgTx = wire2.NewMsgTx(10)
	)
	err := tx.Deserialize(bytes.NewReader(rawTxByte))
	if err != nil {
		return nil
	}
	newRawTxByte = getTxNewRawByte(tx)
	return newRawTxByte
}

func getTxNewRawByte(tx *wire2.MsgTx) []byte {
	var (
		bVersion  [4]byte
		bLockTime [4]byte

		newRawTxByte   []byte
		newInputsByte  []byte
		newInputs2Byte []byte
		newOutputsByte []byte
	)
	binary.LittleEndian.PutUint32(bVersion[:], uint32(tx.Version))
	binary.LittleEndian.PutUint32(bLockTime[:], tx.LockTime)

	newRawTxByte = append(newRawTxByte, bVersion[:]...)
	newRawTxByte = append(newRawTxByte, bLockTime[:]...)
	newRawTxByte = append(newRawTxByte, tool.Uint32ToLittleEndianBytes(uint32(len(tx.TxIn)))...)
	newRawTxByte = append(newRawTxByte, tool.Uint32ToLittleEndianBytes(uint32(len(tx.TxOut)))...)

	for _, in := range tx.TxIn {
		var indexBuf [4]byte
		binary.LittleEndian.PutUint32(indexBuf[:], in.PreviousOutPoint.Index)

		var sequenceBuf [4]byte
		binary.LittleEndian.PutUint32(sequenceBuf[:], in.Sequence)

		newInputsByte = append(newInputsByte, in.PreviousOutPoint.Hash.CloneBytes()...)
		newInputsByte = append(newInputsByte, indexBuf[:]...)
		newInputsByte = append(newInputsByte, sequenceBuf[:]...)

		newInputs2Byte = append(newInputs2Byte, tool.SHA256(in.SignatureScript)...)
	}
	newRawTxByte = append(newRawTxByte, tool.SHA256(newInputsByte)...)
	newRawTxByte = append(newRawTxByte, tool.SHA256(newInputs2Byte)...)

	for _, out := range tx.TxOut {
		var valueBuf []byte
		valueBuf = tool.Uint64ToLittleEndianBytes(uint64(out.Value))

		newOutputsByte = append(newOutputsByte, valueBuf[:]...)
		newOutputsByte = append(newOutputsByte, tool.SHA256(out.PkScript)...)
	}
	newRawTxByte = append(newRawTxByte, tool.SHA256(newOutputsByte)...)
	return newRawTxByte
}
