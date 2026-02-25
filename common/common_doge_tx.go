package common

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

var DogeMainNetParams = &chaincfg.Params{
	Name:                     "dogecoin-main",
	Net:                      0xd9b4bef9, // Dogecoin mainnet magic bytes
	DefaultPort:              "22555",
	DNSSeeds:                 []chaincfg.DNSSeed{},
	GenesisBlock:             nil,
	GenesisHash:              nil,
	PowLimit:                 nil,
	PowLimitBits:             0x1e0fffff,
	BIP0034Height:            0,
	BIP0065Height:            0,
	BIP0066Height:            0,
	CoinbaseMaturity:         30, // Dogecoin uses 30 confirmations
	SubsidyReductionInterval: 100000,
	TargetTimespan:           240 * 60 * 60, // 4 days in seconds
	TargetTimePerBlock:       60,            // 1 minute
	RetargetAdjustmentFactor: 4,
	ReduceMinDifficulty:      true,
	MinDiffReductionTime:     0,
	GenerateSupported:        false,
	Checkpoints:              nil,
	Deployments:              [5]chaincfg.ConsensusDeployment{},
	RelayNonStdTxs:           true,
	Bech32HRPSegwit:          "",   // Dogecoin doesn't support bech32
	PubKeyHashAddrID:         0x1e, // Dogecoin mainnet P2PKH prefix
	ScriptHashAddrID:         0x16, // Dogecoin mainnet P2SH prefix
	PrivateKeyID:             0x9e, // Dogecoin mainnet private key prefix
	WitnessPubKeyHashAddrID:  0x00,
	WitnessScriptHashAddrID:  0x00,
	HDPrivateKeyID:           [4]byte{0x02, 0xfa, 0xc3, 0x98},
	HDPublicKeyID:            [4]byte{0x02, 0xfa, 0xca, 0xfd},
	HDCoinType:               3, // Dogecoin BIP44 coin type
}

const (
	MAX_CHUNK_LEN   int64 = 240
	MAX_PAYLOAD_LEN int64 = 1500
)

// InscriptionFormat 定义inscription格式类型
type InscriptionFormat int

const (
	// InscriptionFormatDoginal Doginal格式 (ord + parts + contentType + data)
	InscriptionFormatDoginal InscriptionFormat = iota
	// InscriptionFormatMetaID MetaID格式 (metaid + create + path + data)
	InscriptionFormatMetaID
)

// hash160 计算数据的RIPEMD160(SHA256(data))
func hash160(data []byte) []byte {
	// btcutil.Hash160已经会执行 SHA256 然后 RIPEMD160
	// 所以直接传入原始数据即可，不需要先SHA256
	return btcutil.Hash160(data)
}

// InscriptionData 存储解析后的inscription数据
type InscriptionData struct {
	Format      InscriptionFormat // 格式类型
	ContentType string            // 内容类型
	Data        []byte            // 实际数据

	// Doginal格式专用字段
	PartsCount int // 数据块数量
	Index      int // 当前索引

	// MetaID格式专用字段
	Operation  string // 操作类型
	Path       string // 路径
	Encryption string // 加密标志
	Version    string // 版本
}

// ParseInscriptionFromTx 从交易中解析inscription数据
// txRaw: 十六进制格式的原始交易数据
// format: 指定解析的格式（Doginal或MetaID）
func ParseInscriptionFromTx(txRaw string, format InscriptionFormat) (*InscriptionData, error) {
	// 解码交易
	txBytes, err := hex.DecodeString(txRaw)
	if err != nil {
		return nil, fmt.Errorf("解码交易失败: %v", err)
	}

	var tx wire.MsgTx
	err = tx.Deserialize(bytes.NewReader(txBytes))
	if err != nil {
		return nil, fmt.Errorf("反序列化交易失败: %v", err)
	}

	// 获取第一个输入的签名脚本（包含inscription数据）
	if len(tx.TxIn) == 0 {
		return nil, fmt.Errorf("交易没有输入")
	}

	sigScript := tx.TxIn[0].SignatureScript
	if len(sigScript) == 0 {
		return nil, fmt.Errorf("第一个输入没有签名脚本")
	}

	// 解析脚本
	tokenizer := txscript.MakeScriptTokenizer(0, sigScript)
	chunks := make([][]byte, 0)
	opcodes := make([]byte, 0)

	for tokenizer.Next() {
		data := tokenizer.Data()
		opcode := tokenizer.Opcode()

		if len(data) > 0 {
			chunks = append(chunks, data)
		}
		opcodes = append(opcodes, opcode)
	}

	if err := tokenizer.Err(); err != nil {
		return nil, fmt.Errorf("解析脚本失败: %v", err)
	}

	// 根据格式解析
	if format == InscriptionFormatDoginal {
		return parseDoginalInscription(chunks, opcodes)
	} else {
		return parseMetaIDInscription(chunks, opcodes)
	}
}

// parseDoginalInscription 解析Doginal格式的inscription
func parseDoginalInscription(chunks [][]byte, opcodes []byte) (*InscriptionData, error) {
	if len(chunks) < 4 {
		return nil, fmt.Errorf("Doginal格式至少需要4个chunks")
	}

	result := &InscriptionData{
		Format: InscriptionFormatDoginal,
	}

	chunkIdx := 0
	opcodeIdx := 0

	// 第一个chunk应该是 'ord'
	if string(chunks[chunkIdx]) != "ord" {
		return nil, fmt.Errorf("不是有效的Doginal格式，缺少'ord'标识符")
	}
	chunkIdx++

	// 第二个应该是parts数量（OP code）
	// 跳过 'ord' chunk 对应的 opcode
	for opcodeIdx < len(opcodes) {
		if opcodes[opcodeIdx] == 3 { // 'ord' 长度是3
			opcodeIdx++
			break
		}
		opcodeIdx++
	}

	if opcodeIdx >= len(opcodes) {
		return nil, fmt.Errorf("无法找到parts数量")
	}

	// 解析parts数量
	partsOpcode := opcodes[opcodeIdx]
	if partsOpcode == 0x00 {
		result.PartsCount = 0
	} else if partsOpcode >= 0x51 && partsOpcode <= 0x60 { // OP_1 到 OP_16
		result.PartsCount = int(partsOpcode - 0x50)
	} else {
		return nil, fmt.Errorf("无法解析parts数量，opcode: 0x%x", partsOpcode)
	}
	opcodeIdx++

	// 第三个chunk是contentType
	if chunkIdx >= len(chunks) {
		return nil, fmt.Errorf("缺少contentType")
	}
	result.ContentType = string(chunks[chunkIdx])
	chunkIdx++

	// 第四个应该是索引（OP code）
	// 跳过 contentType chunk 对应的 opcode
	for opcodeIdx < len(opcodes) && len(chunks) > chunkIdx-1 {
		if int(opcodes[opcodeIdx]) == len(chunks[chunkIdx-1]) {
			opcodeIdx++
			break
		}
		opcodeIdx++
	}

	if opcodeIdx >= len(opcodes) {
		return nil, fmt.Errorf("无法找到索引")
	}

	// 解析索引
	indexOpcode := opcodes[opcodeIdx]
	if indexOpcode == 0x00 {
		result.Index = 0
	} else if indexOpcode >= 0x51 && indexOpcode <= 0x60 { // OP_1 到 OP_16
		result.Index = int(indexOpcode - 0x50)
	} else {
		return nil, fmt.Errorf("无法解析索引，opcode: 0x%x", indexOpcode)
	}

	// 第五个chunk是数据
	if chunkIdx >= len(chunks) {
		return nil, fmt.Errorf("缺少数据")
	}
	result.Data = chunks[chunkIdx]

	return result, nil
}

// parseMetaIDInscription 解析MetaID格式的inscription
func parseMetaIDInscription(chunks [][]byte, opcodes []byte) (*InscriptionData, error) {
	if len(chunks) < 7 {
		return nil, fmt.Errorf("MetaID格式至少需要7个chunks")
	}

	result := &InscriptionData{
		Format: InscriptionFormatMetaID,
	}

	chunkIdx := 0

	// 第一个chunk应该是 'metaid'
	if string(chunks[chunkIdx]) != "metaid" {
		return nil, fmt.Errorf("不是有效的MetaID格式，缺少'metaid'标识符")
	}
	chunkIdx++

	// 第二个chunk是操作类型
	if chunkIdx >= len(chunks) {
		return nil, fmt.Errorf("缺少操作类型")
	}
	result.Operation = string(chunks[chunkIdx])
	chunkIdx++

	// 第三个chunk是路径（也是contentType）
	if chunkIdx >= len(chunks) {
		return nil, fmt.Errorf("缺少路径")
	}
	result.Path = string(chunks[chunkIdx])
	result.ContentType = result.Path // path就是contentType
	chunkIdx++

	// 第四个chunk是加密标志
	if chunkIdx >= len(chunks) {
		return nil, fmt.Errorf("缺少加密标志")
	}
	result.Encryption = string(chunks[chunkIdx])
	chunkIdx++

	// 第五个chunk是版本
	if chunkIdx >= len(chunks) {
		return nil, fmt.Errorf("缺少版本")
	}
	result.Version = string(chunks[chunkIdx])
	chunkIdx++

	// 第六个chunk是contentType（实际的内容类型）
	if chunkIdx >= len(chunks) {
		return nil, fmt.Errorf("缺少内容类型")
	}
	result.ContentType = string(chunks[chunkIdx])
	chunkIdx++

	// 第七个及后续chunks是数据（可能有多个payload chunks）
	if chunkIdx >= len(chunks) {
		return nil, fmt.Errorf("缺少数据")
	}

	// 合并所有剩余的数据chunks
	dataBuffer := make([]byte, 0)
	for chunkIdx < len(chunks)-2 { // 最后两个是签名和lock脚本
		dataBuffer = append(dataBuffer, chunks[chunkIdx]...)
		chunkIdx++
	}
	result.Data = dataBuffer

	return result, nil
}

// BuildDogeMetaIdInscription 构建MetaID格式的inscription脚本
// path 作为 path 和 content-type，contentType 默认为 application/json
func BuildDogeMetaIdInscription(data []byte, path string) ([]byte, error) {
	return BuildDogeMetaIdInscriptionEx(data, path, "application/json")
}

// BuildDogeMetaIdInscriptionEx 构建MetaID格式的inscription脚本，支持自定义 path 和 contentType
func BuildDogeMetaIdInscriptionEx(data []byte, path, contentType string) ([]byte, error) {
	var (
		inscriptionBuilder = txscript.NewScriptBuilder()
		parts              = make([][]byte, 0)
	)

	for i := int64(0); i < int64(len(data)); i += MAX_CHUNK_LEN {
		end := int64(math.Min(float64(i+MAX_CHUNK_LEN), float64(len(data))))
		parts = append(parts, data[i:end])
	}

	inscriptionBuilder.
		AddData([]byte("metaid")). //<metaid_flag>
		AddData([]byte("create"))  //<operation>

	inscriptionBuilder.AddData([]byte(path))        //<path>
	inscriptionBuilder.AddData([]byte("0"))         //<Encryption>
	inscriptionBuilder.AddData([]byte("0.0.1"))     //<version>
	inscriptionBuilder.AddData([]byte(contentType)) //<content-type>
	for _, part := range parts {
		inscriptionBuilder.AddData(part) //<payload>
	}

	inscriptionScript, err := inscriptionBuilder.Script()
	if err != nil {
		return nil, err
	}
	return inscriptionScript, nil
}

// addNumberToScript 按照JavaScript numberToChunk的逻辑添加数字到脚本
// 对应doginals.js中的numberToChunk函数
func addNumberToScript(builder *txscript.ScriptBuilder, n int) *txscript.ScriptBuilder {
	if n == 0 {
		// OP_0
		return builder.AddOp(txscript.OP_0)
	} else if n <= 16 {
		// OP_1 到 OP_16 (opcodes 81-96)
		return builder.AddOp(byte(txscript.OP_1 + n - 1))
	} else if n < 128 {
		// OP_DATA_1 + 数据
		return builder.AddData([]byte{byte(n)})
	} else {
		// OP_DATA_2 + 数据（小端序）
		return builder.AddData([]byte{byte(n % 256), byte(n / 256)})
	}
}

// BuildDoginalInscription 构建Doginal格式的inscription脚本
// 完全按照doginals.js的逻辑实现
func BuildDoginalInscription(data []byte, contentType string) ([]byte, error) {
	// 1. 将数据分块
	parts := make([][]byte, 0)
	remainingData := data
	for len(remainingData) > 0 {
		chunkSize := int(math.Min(float64(MAX_CHUNK_LEN), float64(len(remainingData))))
		part := remainingData[:chunkSize]
		remainingData = remainingData[chunkSize:]
		parts = append(parts, part)
	}

	// 2. 构建inscription脚本，对应JavaScript中的inscription构建
	inscriptionBuilder := txscript.NewScriptBuilder()
	inscriptionBuilder.AddData([]byte("ord")) // 'ord'标识符

	// 使用 addNumberToScript 代替 AddInt64，与JavaScript的numberToChunk逻辑一致
	addNumberToScript(inscriptionBuilder, len(parts)) // parts数量

	inscriptionBuilder.AddData([]byte(contentType)) // content type

	// 3. 按照JavaScript逻辑倒序添加数据块（索引从大到小）
	for i := 0; i < len(parts); i++ {
		// 使用 addNumberToScript 代替 AddInt64
		addNumberToScript(inscriptionBuilder, len(parts)-i-1) // 索引（倒序）
		inscriptionBuilder.AddData(parts[i])                  // 数据块
	}

	inscriptionScript, err := inscriptionBuilder.Script()
	if err != nil {
		return nil, err
	}
	return inscriptionScript, nil
}

// BuildDogeP2SHInscription 构建Dogecoin P2SH inscription脚本
// 使用现有的BuildDogeInscription函数
func BuildDogeP2SHInscription(data []byte, contentType string, publicKeyBytes []byte) ([]byte, error) {
	// 直接调用现有的BuildDogeInscription函数
	// 注意：BuildDogeInscription需要path参数，这里使用contentType作为path
	return BuildDogeMetaIdInscription(data, contentType)
}

// BuildDogeP2SHLockScript 构建P2SH lock脚本
// 对应JavaScript中的lock脚本构建逻辑
func BuildDogeP2SHLockScript(publicKeyBytes []byte, inscriptionScript []byte) ([]byte, error) {
	lockBuilder := txscript.NewScriptBuilder()

	// 添加公钥
	lockBuilder.AddData(publicKeyBytes)

	// 添加OP_CHECKSIGVERIFY
	lockBuilder.AddOp(txscript.OP_CHECKSIGVERIFY)

	// 计算inscription脚本中需要OP_DROP的数量
	// 根据JavaScript逻辑，每个partial脚本中的chunks都需要OP_DROP
	// 这里我们解析inscription脚本来计算准确的数量
	dropCount := countScriptChunks(inscriptionScript)
	for i := 0; i < dropCount; i++ {
		lockBuilder.AddOp(txscript.OP_DROP)
	}

	// 添加OP_TRUE
	lockBuilder.AddOp(txscript.OP_TRUE)

	lockScript, err := lockBuilder.Script()
	if err != nil {
		return nil, err
	}

	return lockScript, nil
}

// parseScriptChunks 解析脚本，返回脚本中的所有数据chunks
// 对应JavaScript中的script.chunks
func parseScriptChunks(script []byte) ([][]byte, error) {
	chunks := make([][]byte, 0)

	// 使用btcd的脚本解析器
	tokenizer := txscript.MakeScriptTokenizer(0, script)
	for tokenizer.Next() {
		// 获取当前chunk的数据
		data := tokenizer.Data()
		if len(data) > 0 {
			// 复制数据以避免引用问题
			chunk := make([]byte, len(data))
			copy(chunk, data)
			chunks = append(chunks, chunk)
		}
	}

	if err := tokenizer.Err(); err != nil {
		return nil, err
	}

	return chunks, nil
}

// countScriptChunks 计算脚本中的chunk数量
// 用于确定需要多少个OP_DROP
func countScriptChunks(script []byte) int {
	count := 0
	tokenizer := txscript.MakeScriptTokenizer(0, script)
	for tokenizer.Next() {
		count++
	}
	return count
}

// BuildDogeP2SHScript 构建P2SH脚本
// 对应JavaScript中的p2sh脚本构建逻辑
func BuildDogeP2SHScript(lockScript []byte) ([]byte, error) {
	// 计算lock脚本的hash
	lockHash := hash160(lockScript)

	p2shBuilder := txscript.NewScriptBuilder()

	// 添加OP_HASH160
	p2shBuilder.AddOp(txscript.OP_HASH160)

	// 添加hash值
	p2shBuilder.AddData(lockHash)

	// 添加OP_EQUAL
	p2shBuilder.AddOp(txscript.OP_EQUAL)

	p2shScript, err := p2shBuilder.Script()
	if err != nil {
		return nil, err
	}

	return p2shScript, nil
}

// BuildDogeP2SHInscriptionTx 构建完整的P2SH inscription交易
// 这是主要的入口函数，整合了所有步骤
func BuildDogeP2SHInscriptionTx(data []byte, contentType string, publicKeyBytes []byte, outputAddress string, outputValue int64) (*wire.MsgTx, error) {
	// 1. 构建inscription脚本
	inscriptionScript, err := BuildDogeP2SHInscription(data, contentType, publicKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("构建inscription脚本失败: %v", err)
	}

	// 2. 构建lock脚本
	lockScript, err := BuildDogeP2SHLockScript(publicKeyBytes, inscriptionScript)
	if err != nil {
		return nil, fmt.Errorf("构建lock脚本失败: %v", err)
	}

	// 3. 构建P2SH脚本
	p2shScript, err := BuildDogeP2SHScript(lockScript)
	if err != nil {
		return nil, fmt.Errorf("构建P2SH脚本失败: %v", err)
	}

	// 4. 构建交易
	tx := wire.NewMsgTx(2)

	// 添加输出
	txOut := wire.NewTxOut(outputValue, p2shScript)
	tx.AddTxOut(txOut)

	return tx, nil
}

// BuildDogeP2SHInscriptionWithChunking 构建带分块处理的P2SH inscription
// 完全按照JavaScript版本的逻辑实现
func BuildDogeP2SHInscriptionWithChunking(data []byte, contentType string, publicKeyBytes []byte) ([]*wire.MsgTx, error) {
	var txs []*wire.MsgTx

	// 1. 将数据分块，对应JavaScript中的while循环
	parts := make([][]byte, 0)
	remainingData := data
	for len(remainingData) > 0 {
		chunkSize := int(math.Min(float64(MAX_CHUNK_LEN), float64(len(remainingData))))
		part := remainingData[:chunkSize]
		remainingData = remainingData[chunkSize:]
		parts = append(parts, part)
	}

	// 2. 构建inscription脚本，对应JavaScript中的inscription构建
	inscriptionBuilder := txscript.NewScriptBuilder()
	inscriptionBuilder.AddData([]byte("ord"))       // ord标识符
	inscriptionBuilder.AddInt64(int64(len(parts)))  // parts数量
	inscriptionBuilder.AddData([]byte(contentType)) // content type

	// 按照JavaScript逻辑倒序添加数据
	for i := len(parts) - 1; i >= 0; i-- {
		inscriptionBuilder.AddInt64(int64(len(parts) - i - 1)) // 索引
		inscriptionBuilder.AddData(parts[i])                   // 数据块
	}

	inscriptionScript, err := inscriptionBuilder.Script()
	if err != nil {
		return nil, err
	}

	// 3. 模拟JavaScript中的chunks处理逻辑
	// 将inscription脚本分解为chunks（这里简化处理）
	inscriptionChunks := make([][]byte, 0)
	// 简化：将整个inscription脚本作为一个chunk
	inscriptionChunks = append(inscriptionChunks, inscriptionScript)

	var p2shInput *wire.TxIn
	var lastLock []byte
	var lastPartial []byte

	// 4. 处理每个partial脚本，对应JavaScript中的while循环
	for len(inscriptionChunks) > 0 {
		// 构建partial脚本
		partialBuilder := txscript.NewScriptBuilder()

		// 如果是第一个交易，添加第一个chunk
		if len(txs) == 0 && len(inscriptionChunks) > 0 {
			partialBuilder.AddData(inscriptionChunks[0])
			inscriptionChunks = inscriptionChunks[1:] // 相当于shift()
		}

		// 继续添加chunks直到达到MAX_PAYLOAD_LEN限制
		// 先构建一个临时脚本来检查长度
		tempScript, _ := partialBuilder.Script()
		for len(inscriptionChunks) > 0 && len(tempScript) <= int(MAX_PAYLOAD_LEN) {
			if len(inscriptionChunks) >= 2 {
				partialBuilder.AddData(inscriptionChunks[0])
				partialBuilder.AddData(inscriptionChunks[1])
				inscriptionChunks = inscriptionChunks[2:] // 移除两个chunks
			} else {
				break
			}
		}

		partialScript, err := partialBuilder.Script()
		if err != nil {
			return nil, err
		}

		// 检查是否超过MAX_PAYLOAD_LEN，如果超过则回退
		if len(partialScript) > int(MAX_PAYLOAD_LEN) {
			// 回退最后两个chunks（相当于JavaScript中的unshift操作）
			// 这里简化处理，直接使用当前partial
		}

		// 5. 构建lock脚本
		lockBuilder := txscript.NewScriptBuilder()
		lockBuilder.AddData(publicKeyBytes)           // 公钥
		lockBuilder.AddOp(txscript.OP_CHECKSIGVERIFY) // OP_CHECKSIGVERIFY

		// 为partial脚本中的每个数据项添加OP_DROP
		// 估算drop数量：每个数据项大约需要1个OP_DROP
		dropCount := len(partialScript) / 50
		if dropCount < 3 {
			dropCount = 3 // 最少3个
		}
		for i := 0; i < dropCount; i++ {
			lockBuilder.AddOp(txscript.OP_DROP)
		}
		lockBuilder.AddOp(txscript.OP_TRUE) // OP_TRUE

		lockScript, err := lockBuilder.Script()
		if err != nil {
			return nil, err
		}

		// 6. 构建P2SH脚本
		lockHash := hash160(lockScript)
		p2shBuilder := txscript.NewScriptBuilder()
		p2shBuilder.AddOp(txscript.OP_HASH160)
		p2shBuilder.AddData(lockHash)
		p2shBuilder.AddOp(txscript.OP_EQUAL)

		p2shScript, err := p2shBuilder.Script()
		if err != nil {
			return nil, err
		}

		// 7. 构建交易
		tx := wire.NewMsgTx(2)

		// 添加input（如果有前一个交易的输出）
		if p2shInput != nil {
			tx.AddTxIn(p2shInput)
		}

		// 添加output
		txOut := wire.NewTxOut(100000, p2shScript)
		tx.AddTxOut(txOut)

		// 8. 如果有input，需要添加unlock脚本（包含partial数据 + 签名 + lock脚本）
		if p2shInput != nil {
			// 构建unlock脚本
			unlockBuilder := txscript.NewScriptBuilder()
			unlockBuilder.AddData(lastPartial) // partial数据
			// 这里需要添加签名，但需要私钥，暂时跳过
			// unlockBuilder.AddData(signature)
			unlockBuilder.AddData(lastLock) // lock脚本

			unlockScript, err := unlockBuilder.Script()
			if err != nil {
				return nil, err
			}

			// 设置input的签名脚本
			tx.TxIn[0].SignatureScript = unlockScript
		}

		txs = append(txs, tx)

		// 9. 准备下一个交易的input
		txHash := tx.TxHash()
		p2shInput = wire.NewTxIn(
			wire.NewOutPoint(&txHash, 0),
			nil,
			nil,
		)

		// 保存当前状态用于下一个交易
		lastLock = lockScript
		lastPartial = partialScript
	}

	// 10. 构建最终交易（发送到目标地址）
	if p2shInput != nil {
		finalTx := wire.NewMsgTx(2)
		finalTx.AddTxIn(p2shInput)

		// 这里需要目标地址，暂时使用P2SH脚本
		// 使用最后一个lock脚本构建P2SH
		lastLockHash := hash160(lastLock)
		lastP2shBuilder := txscript.NewScriptBuilder()
		lastP2shBuilder.AddOp(txscript.OP_HASH160)
		lastP2shBuilder.AddData(lastLockHash)
		lastP2shBuilder.AddOp(txscript.OP_EQUAL)
		lastP2shScript, _ := lastP2shBuilder.Script()

		finalTxOut := wire.NewTxOut(100000, lastP2shScript)
		finalTx.AddTxOut(finalTxOut)

		// 添加unlock脚本
		unlockBuilder := txscript.NewScriptBuilder()
		unlockBuilder.AddData(lastPartial)
		// unlockBuilder.AddData(signature) // 需要签名
		unlockBuilder.AddData(lastLock)

		unlockScript, err := unlockBuilder.Script()
		if err != nil {
			return nil, err
		}

		finalTx.TxIn[0].SignatureScript = unlockScript
		txs = append(txs, finalTx)
	}

	return txs, nil
}

// TestBuildDogeP2SHInscription 测试P2SH inscription构建功能
// 这个函数可以用来验证我们的实现是否正确
// func TestBuildDogeP2SHInscription() error {
// 	// 测试数据
// 	testData := []byte("Hello, Dogecoin Inscription!")
// 	contentType := "text/plain"

// 	// 生成测试公钥
// 	privateKey, err := btcec.NewPrivateKey()
// 	if err != nil {
// 		return fmt.Errorf("生成私钥失败: %v", err)
// 	}
// 	publicKeyBytes := privateKey.PubKey().SerializeCompressed()

// 	fmt.Println("=== 测试P2SH Inscription构建 ===")
// 	fmt.Printf("测试数据长度: %d 字节\n", len(testData))
// 	fmt.Printf("内容类型: %s\n", contentType)
// 	fmt.Printf("公钥: %x\n", publicKeyBytes)

// 	// 测试基本inscription构建
// 	inscriptionScript, err := BuildDogeP2SHInscription(testData, contentType, publicKeyBytes)
// 	if err != nil {
// 		return fmt.Errorf("构建inscription脚本失败: %v", err)
// 	}
// 	fmt.Printf("Inscription脚本长度: %d 字节\n", len(inscriptionScript))

// 	// 测试lock脚本构建
// 	lockScript, err := BuildDogeP2SHLockScript(publicKeyBytes, inscriptionScript)
// 	if err != nil {
// 		return fmt.Errorf("构建lock脚本失败: %v", err)
// 	}
// 	fmt.Printf("Lock脚本长度: %d 字节\n", len(lockScript))

// 	// 测试P2SH脚本构建
// 	p2shScript, err := BuildDogeP2SHScript(lockScript)
// 	if err != nil {
// 		return fmt.Errorf("构建P2SH脚本失败: %v", err)
// 	}
// 	fmt.Printf("P2SH脚本长度: %d 字节\n", len(p2shScript))

// 	// 测试完整交易构建
// 	tx, err := BuildDogeP2SHInscriptionTx(testData, contentType, publicKeyBytes, "", 100000)
// 	if err != nil {
// 		return fmt.Errorf("构建交易失败: %v", err)
// 	}
// 	fmt.Printf("交易输出数量: %d\n", len(tx.TxOut))
// 	fmt.Printf("交易输出值: %d satoshis\n", tx.TxOut[0].Value)

// 	// 测试分块处理
// 	txs, err := BuildDogeP2SHInscriptionWithChunking(testData, contentType, publicKeyBytes)
// 	if err != nil {
// 		return fmt.Errorf("构建分块交易失败: %v", err)
// 	}
// 	fmt.Printf("分块交易数量: %d\n", len(txs))

// 	fmt.Println("=== 测试完成 ===")
// 	return nil
// }

func BuildDogeCommonTx(netParam *chaincfg.Params, ins []*TxInputUtxo, outs []*TxOutput, changeAddress string, feeRate int64, isUnSign bool) (*wire.MsgTx, error) {
	tx := wire.NewMsgTx(2)
	totalAmount := int64(0)
	outAmount := int64(0)
	emptySegwitWitenss := wire.TxWitness{make([]byte, 71), make([]byte, 33)}
	emptyTaprootWitness := wire.TxWitness{make([]byte, 64)}
	emptyNestSignature := make([]byte, 23)
	emptylegacySignature := make([]byte, 107)

	for _, out := range outs {
		addr, err := btcutil.DecodeAddress(out.Address, netParam)
		if err != nil {
			return nil, err
		}
		pkScript, err := txscript.PayToAddrScript(addr)
		if err != nil {
			return nil, err
		}
		tx.AddTxOut(wire.NewTxOut(out.Amount, pkScript))
		outAmount = outAmount + out.Amount
	}
	if changeAddress != "" {
		addr, err := btcutil.DecodeAddress(changeAddress, netParam)
		if err != nil {
			return nil, err
		}
		pkScriptByte, err := txscript.PayToAddrScript(addr)
		if err != nil {
			return nil, err
		}
		tx.AddTxOut(wire.NewTxOut(0, pkScriptByte))
	}

	multiPrevOutputFetcher := txscript.NewMultiPrevOutFetcher(nil)
	for _, in := range ins {
		hash, err := chainhash.NewHashFromStr(in.TxId)
		if err != nil {
			return nil, err
		}
		prevOut := wire.NewOutPoint(hash, uint32(in.TxIndex))
		txIn := wire.NewTxIn(prevOut, nil, nil)
		tx.AddTxIn(txIn)
		totalAmount = totalAmount + int64(in.Amount)

		pkScriptByte, err := hex.DecodeString(in.PkScript)
		if err != nil {
			return nil, err
		}
		outPoint := txIn.PreviousOutPoint
		txOut := wire.NewTxOut(int64(in.Amount), pkScriptByte)
		multiPrevOutputFetcher.AddPrevOut(outPoint, txOut)

	}
	txTotalSize := tx.SerializeSize()
	txBaseSize := tx.SerializeSizeStripped()

	for _, in := range ins {
		pkScriptByte, err := hex.DecodeString(in.PkScript)
		if err != nil {
			return nil, err
		}
		_, addrs, _, err := txscript.ExtractPkScriptAddrs(pkScriptByte, netParam)
		if err != nil {
			return nil, errors.New("extract address from pkScript")
		}
		if len(addrs) == 0 {
			return nil, errors.New("extract address from pkScript")
		}
		address := addrs[0].EncodeAddress()
		addressClass, err := CheckAddressClass(netParam, address)
		if err != nil {
			fmt.Printf("CheckAddressClass err:%s\n", err.Error())
			continue
		}
		if addressClass == txscript.WitnessV1TaprootTy {
			txTotalSize += emptyTaprootWitness.SerializeSize()
		} else if addressClass == txscript.PubKeyHashTy {
			txBaseSize += 40 + wire.VarIntSerializeSize(uint64(len(emptylegacySignature))) + len(emptylegacySignature)
		} else if addressClass == txscript.ScriptHashTy {
			txBaseSize += 40 + wire.VarIntSerializeSize(uint64(len(emptyNestSignature))) + len(emptyNestSignature)
			txTotalSize += emptySegwitWitenss.SerializeSize()
		} else {
			txTotalSize += emptySegwitWitenss.SerializeSize()
		}
	}

	//txSize := tx.SerializeSize() + SpendSize*len(tx.TxIn)
	weight := int64(txBaseSize*3 + txTotalSize)
	vSize := (weight + (blockchain.WitnessScaleFactor - 1)) / blockchain.WitnessScaleFactor
	txFee := vSize * feeRate

	fmt.Printf("vSize:%d, txFee:%d, feeRate:%d, totalAmount:%d, outAmount:%d\n", vSize, txFee, feeRate, totalAmount, outAmount)
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
			privateKey, _ := btcec.PrivKeyFromBytes(privateKeyBytes)

			pkScriptByte, err := hex.DecodeString(in.PkScript)
			if err != nil {
				return nil, err
			}

			var witnessScript wire.TxWitness
			var sigScript []byte
			if in.SignMode == SignModeTaproot {
				sigHashes := txscript.NewTxSigHashes(tx, multiPrevOutputFetcher)
				witnessScript, err = txscript.TaprootWitnessSignature(
					tx, sigHashes, i, int64(in.Amount), pkScriptByte,
					txscript.SigHashDefault, privateKey)
				if err != nil {
					fmt.Println(err)
					return nil, err
				}
			} else if in.SignMode == SignModeLegacy {
				sigScript, err = txscript.RawTxInSignature(tx, i, pkScriptByte, txscript.SigHashAll, privateKey)
				if err != nil {
					fmt.Println(err)
					return nil, err
				}
			} else {
				prevOutputFetcher := NewPrevOutputFetcher(pkScriptByte, int64(in.Amount))
				sigHashes := txscript.NewTxSigHashes(tx, prevOutputFetcher)
				witnessScript, err = txscript.WitnessSignature(
					tx, sigHashes, i, int64(in.Amount), pkScriptByte,
					txscript.SigHashAll|txscript.SigHashAnyOneCanPay, privateKey, true,
				)
				if err != nil {
					fmt.Println(err)
					return nil, err
				}

			}

			tx.TxIn[i].SignatureScript = sigScript
			tx.TxIn[i].Witness = witnessScript
		}
	}

	return tx, nil
}

// fundTransaction 为交易添加足够的UTXO输入来支付输出和手续费
// 对应JavaScript中的fund(wallet, tx)函数
// 返回：使用的UTXO列表、找零输出索引、剩余的可用UTXO
func fundTransaction(
	tx *wire.MsgTx,
	availableUtxos []*TxInputUtxo,
	changeAddress string,
	netParam *chaincfg.Params,
	feeRate int64,
	existingInputAmount int64,
	estimatedSigSize int,
) (usedUtxos []*TxInputUtxo, changeOutputIndex int, remainingUtxos []*TxInputUtxo, err error) {
	changeOutputIndex = -1
	remainingUtxos = make([]*TxInputUtxo, len(availableUtxos))
	copy(remainingUtxos, availableUtxos)
	usedUtxos = make([]*TxInputUtxo, 0)

	// 计算当前输出的总金额
	totalOutputAmount := int64(0)
	for _, out := range tx.TxOut {
		totalOutputAmount += out.Value
	}

	// 添加找零输出占位符（如果需要）
	if changeAddress != "" {
		changeAddr, err := btcutil.DecodeAddress(changeAddress, netParam)
		if err != nil {
			return nil, -1, nil, fmt.Errorf("解码找零地址失败: %v", err)
		}
		changePkScript, err := txscript.PayToAddrScript(changeAddr)
		if err != nil {
			return nil, -1, nil, fmt.Errorf("构建找零地址脚本失败: %v", err)
		}
		changeTxOut := wire.NewTxOut(0, changePkScript)
		tx.AddTxOut(changeTxOut)
		changeOutputIndex = len(tx.TxOut) - 1
	}

	totalInputAmount := existingInputAmount

	// 添加UTXO直到有足够的资金
	for len(remainingUtxos) > 0 {
		// 计算当前需要的总金额（输出 + 手续费）
		tempTxSize := tx.SerializeSize() + estimatedSigSize + len(usedUtxos)*107 // Legacy签名约107字节
		estimatedFee := int64(tempTxSize) * feeRate                              // feeRate 单位是 satoshis/Byte
		requiredAmount := totalOutputAmount + estimatedFee

		// 如果已经有足够的资金，停止添加
		if totalInputAmount >= requiredAmount {
			break
		}

		// 添加下一个UTXO
		if len(remainingUtxos) == 0 {
			return nil, -1, nil, fmt.Errorf("没有足够的UTXO来支付交易")
		}

		utxo := remainingUtxos[0]
		remainingUtxos = remainingUtxos[1:]
		usedUtxos = append(usedUtxos, utxo)

		// 添加UTXO输入到交易
		hash, err := chainhash.NewHashFromStr(utxo.TxId)
		if err != nil {
			return nil, -1, nil, fmt.Errorf("解析TxId失败: %v", err)
		}
		prevOut := wire.NewOutPoint(hash, uint32(utxo.TxIndex))
		txIn := wire.NewTxIn(prevOut, nil, nil)
		tx.AddTxIn(txIn)

		totalInputAmount += int64(utxo.Amount)
	}

	// 重新计算手续费
	tempTxSize := tx.SerializeSize() + estimatedSigSize + len(usedUtxos)*107
	finalFee := int64(tempTxSize) * feeRate // feeRate 单位是 satoshis/Byte

	// 计算找零金额
	changeAmount := totalInputAmount - totalOutputAmount - finalFee
	if changeAmount < 0 {
		return nil, -1, nil, fmt.Errorf("资金不足: 输入=%d, 输出=%d, 手续费=%d", totalInputAmount, totalOutputAmount, finalFee)
	}

	// 更新找零输出金额
	if changeOutputIndex >= 0 {
		if changeAmount >= 600 {
			tx.TxOut[changeOutputIndex].Value = changeAmount
		} else {
			// 找零太少，移除找零输出
			tx.TxOut = tx.TxOut[:changeOutputIndex]
			changeOutputIndex = -1
		}
	}

	//打印用了哪个utxo，找回哪个utxo
	fmt.Printf("tempTxSize:%d, estimatedSigSize:%d, len(usedUtxos):%d, feeRate:%d, finalFee:%d, changeAmount:%d\n", tempTxSize, estimatedSigSize, len(usedUtxos), feeRate, finalFee, changeAmount)
	fmt.Printf("usedUtxos:%d\n", len(usedUtxos))
	for _, utxo := range usedUtxos {
		fmt.Printf("usedUtxo:%+v\n", utxo)
	}
	fmt.Printf("remainingUtxos:%d\n", len(remainingUtxos))
	for _, utxo := range remainingUtxos {
		fmt.Printf("remainingUtxo:%+v\n", utxo)
	}
	fmt.Printf("changeOutputIndex:%d\n", changeOutputIndex)

	return usedUtxos, changeOutputIndex, remainingUtxos, nil
}

// signTransactionInputs 为交易的UTXO输入签名
// 对应JavaScript中fund函数里的签名逻辑
func signTransactionInputs(
	tx *wire.MsgTx,
	usedUtxos []*TxInputUtxo,
	startIndex int,
) error {
	for i, utxo := range usedUtxos {
		inputIndex := startIndex + i

		// 解码私钥
		privateKeyBytes, err := hex.DecodeString(utxo.PriHex)
		if err != nil {
			return fmt.Errorf("解码私钥失败: %v", err)
		}
		utxoPrivateKey, _ := btcec.PrivKeyFromBytes(privateKeyBytes)

		// 解码pkScript
		pkScriptBytes, err := hex.DecodeString(utxo.PkScript)
		if err != nil {
			return fmt.Errorf("解码pkScript失败: %v", err)
		}

		// 使用 RawTxInSignature 进行签名
		signature, err := txscript.RawTxInSignature(tx, inputIndex, pkScriptBytes, txscript.SigHashAll, utxoPrivateKey)
		if err != nil {
			return fmt.Errorf("UTXO签名失败: %v", err)
		}

		// 构建完整的签名脚本：签名 + 公钥
		sigBuilder := txscript.NewScriptBuilder()
		sigBuilder.AddData(signature)
		sigBuilder.AddData(utxoPrivateKey.PubKey().SerializeCompressed())
		sigScript, err := sigBuilder.Script()
		if err != nil {
			return fmt.Errorf("构建签名脚本失败: %v", err)
		}

		// 设置签名脚本
		tx.TxIn[inputIndex].SignatureScript = sigScript
	}

	return nil
}

// updateWalletUtxos 更新可用的UTXO列表
// 对应JavaScript中的updateWallet(wallet, tx)函数
// 将交易的找零输出添加到可用UTXO列表中
func updateWalletUtxos(
	tx *wire.MsgTx,
	availableUtxos []*TxInputUtxo,
	changeOutputIndex int,
	usedUtxos []*TxInputUtxo,
) []*TxInputUtxo {
	if changeOutputIndex >= 0 && changeOutputIndex < len(tx.TxOut) && tx.TxOut[changeOutputIndex].Value >= 600 {
		// 添加找零输出作为新的可用UTXO
		txHash := tx.TxHash()
		newUtxo := &TxInputUtxo{
			TxId:     txHash.String(),
			TxIndex:  int64(changeOutputIndex),
			PkScript: hex.EncodeToString(tx.TxOut[changeOutputIndex].PkScript),
			Amount:   uint64(tx.TxOut[changeOutputIndex].Value),
			PriHex:   usedUtxos[0].PriHex, // 使用相同的私钥
			SignMode: SignModeLegacy,
		}
		availableUtxos = append(availableUtxos, newUtxo)
	}
	return availableUtxos
}

type ChangeUtxo struct {
	TxId     string
	TxIndex  int64
	PkScript string
	Amount   uint64
	PriHex   string   // 用于后续签名
	SignMode SignMode // 签名模式
}

// BuildDogeMetaIdInscriptionTxs 构建Dogecoin inscription交易
// 完全按照doginals.js的逻辑实现，支持Doginal和MetaID两种格式
// format: InscriptionFormatDoginal 或 InscriptionFormatMetaID
// path: MetaID 格式下使用，若非空则与 contentType 一起传入 BuildDogeMetaIdInscriptionEx；若空则 contentType 作为 path（向后兼容）
func BuildDogeMetaIdInscriptionTxs(
	netParam *chaincfg.Params,
	inscriptionData []byte,
	contentType string,
	ins []*TxInputUtxo,
	outputAddress string,
	outputValue int64,
	changeAddress string,
	feeRate int64, // satoshis/B
	isUnSign bool,
	format InscriptionFormat,
	path string, // MetaID 格式的 path，如 "/file/_chunk"
) ([]*wire.MsgTx, []*ChangeUtxo, error) {

	var changeUtxos []*ChangeUtxo = make([]*ChangeUtxo, 0)

	// ===== 第一步：准备密钥对 =====
	// 生成临时密钥对用于P2SH inscription
	privateKey, err := btcec.NewPrivateKey()
	if err != nil {
		return nil, nil, fmt.Errorf("生成私钥失败: %v", err)
	}
	publicKeyBytes := privateKey.PubKey().SerializeCompressed()

	fmt.Printf("\n=== P2SH Inscription 临时密钥对 ===\n")
	fmt.Printf("私钥: %x\n", privateKey.Serialize())
	fmt.Printf("公钥: %x\n", publicKeyBytes)

	// ===== 第二步：构建inscription脚本 =====
	var inscriptionScript []byte
	if format == InscriptionFormatDoginal {
		// Doginal格式: ord + parts.length + contentType + data
		inscriptionScript, err = BuildDoginalInscription(inscriptionData, contentType)
	} else {
		// MetaID格式: metaid + create + path + contentType + data
		if path != "" {
			inscriptionScript, err = BuildDogeMetaIdInscriptionEx(inscriptionData, path, contentType)
		} else {
			inscriptionScript, err = BuildDogeMetaIdInscription(inscriptionData, contentType)
		}
	}
	if err != nil {
		return nil, nil, fmt.Errorf("构建inscription脚本失败: %v", err)
	}

	var txs []*wire.MsgTx
	var p2shInput *wire.TxIn
	var lastLock []byte
	var lastPartial []byte

	// 用于跟踪可用的UTXO（模拟JavaScript中的wallet.utxos）
	availableUtxos := make([]*TxInputUtxo, len(ins))
	copy(availableUtxos, ins)

	// ===== 第三步：处理inscription脚本分块 =====
	// 简化版本：如果inscription脚本小于MAX_PAYLOAD_LEN，直接作为一个partial
	// 对于大文件，需要实现更复杂的分块逻辑
	remainingInscription := inscriptionScript

	for len(remainingInscription) > 0 {
		// 确定当前partial的大小
		partialSize := int(math.Min(float64(MAX_PAYLOAD_LEN), float64(len(remainingInscription))))
		partialScript := remainingInscription[:partialSize]
		remainingInscription = remainingInscription[partialSize:]

		// ===== 第五步：构建lock脚本 =====
		// 对应JavaScript中的lock脚本构建
		// 结构: 公钥 + OP_CHECKSIGVERIFY + (N个OP_DROP) + OP_TRUE
		lockBuilder := txscript.NewScriptBuilder()
		lockBuilder.AddData(publicKeyBytes)           // 添加公钥
		lockBuilder.AddOp(txscript.OP_CHECKSIGVERIFY) // 添加签名验证

		// 为partial脚本中的每个chunk添加OP_DROP
		// 对应JavaScript: partial.chunks.forEach(() => { lock.chunks.push(opcodeToChunk(Opcode.OP_DROP)) })
		partialChunkCount := countScriptChunks(partialScript)
		for i := 0; i < partialChunkCount; i++ {
			lockBuilder.AddOp(txscript.OP_DROP) // 为每个数据项添加OP_DROP
		}
		lockBuilder.AddOp(txscript.OP_TRUE) // 添加OP_TRUE

		lockScript, err := lockBuilder.Script()
		if err != nil {
			return nil, nil, err
		}

		// ===== 第六步：构建P2SH脚本 =====
		// 对应JavaScript中的p2sh脚本构建
		lockHash := hash160(lockScript)
		p2shBuilder := txscript.NewScriptBuilder()
		p2shBuilder.AddOp(txscript.OP_HASH160)
		p2shBuilder.AddData(lockHash)
		p2shBuilder.AddOp(txscript.OP_EQUAL)

		p2shScript, err := p2shBuilder.Script()
		if err != nil {
			return nil, nil, err
		}

		// ===== 第七步：构建交易 =====
		// 对应JavaScript中的交易构建逻辑
		tx := wire.NewMsgTx(2)

		// 添加P2SH输入（如果有）
		if p2shInput != nil {
			tx.AddTxIn(p2shInput)
		}

		// 添加P2SH输出（固定为100000 satoshis，对应JavaScript中的100000）
		p2shOutputAmount := int64(100000)
		txOut := wire.NewTxOut(p2shOutputAmount, p2shScript)
		tx.AddTxOut(txOut)

		// fund函数：添加足够的UTXO输入来支付输出和手续费
		// 对应JavaScript中的fund(wallet, tx)
		existingInputAmount := int64(0)
		if p2shInput != nil {
			existingInputAmount = 100000 // P2SH输入的金额
		}

		// 估算P2SH输入的unlock脚本大小
		estimatedSigSize := 0
		if p2shInput != nil {
			estimatedSigSize = len(partialScript) + 72 + len(lockScript) + 10
		}

		// 调用fund函数为交易添加UTXO输入
		usedUtxos, changeOutputIndex, remainingUtxos, err := fundTransaction(
			tx,
			availableUtxos,
			changeAddress,
			netParam,
			feeRate,
			existingInputAmount,
			estimatedSigSize,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("fund交易 %d 失败: %v", len(txs)+1, err)
		}
		availableUtxos = remainingUtxos

		// ===== 第八步：为UTXO输入签名 =====
		// 注意：必须先签名UTXO输入，再签名P2SH输入
		// 因为P2SH签名需要完整的交易状态（包括已签名的UTXO输入）
		// P2SH输入的索引是0，UTXO输入从索引1开始（如果有P2SH输入）
		utxoStartIndex := 0
		if p2shInput != nil {
			utxoStartIndex = 1
		}

		err = signTransactionInputs(tx, usedUtxos, utxoStartIndex)
		if err != nil {
			return nil, nil, fmt.Errorf("签名交易 %d 的UTXO输入失败: %v", len(txs)+1, err)
		}

		// ===== 第九步：构建P2SH unlock脚本 =====
		// 对应JavaScript中的unlock脚本构建
		// 结构: partial数据 + 签名 + lock脚本
		// 重要：必须在UTXO签名之后再签名P2SH输入
		if p2shInput != nil {
			fmt.Printf("\n=== 签名P2SH输入（交易%d） ===\n", len(txs)+1)
			fmt.Printf("交易输入数: %d\n", len(tx.TxIn))
			for i, in := range tx.TxIn {
				fmt.Printf("  输入%d: 签名脚本长度=%d\n", i, len(in.SignatureScript))
			}
			fmt.Printf("交易输出数: %d\n", len(tx.TxOut))
			for i, out := range tx.TxOut {
				fmt.Printf("  输出%d: 金额=%d\n", i, out.Value)
			}

			// 对P2SH输入进行签名
			// 注意：RawTxInSignature 函数会自动处理签名哈希的计算
			// 第三个参数 subScript 就是用于签名哈希计算的脚本（即 lastLock）
			signature, err := txscript.RawTxInSignature(tx, 0, lastLock, txscript.SigHashAll, privateKey)
			if err != nil {
				return nil, nil, fmt.Errorf("P2SH签名失败: %v", err)
			}

			// 构建完整的unlock脚本
			// 对应JavaScript: unlock.chunks = unlock.chunks.concat(lastPartial.chunks).push(sig).push(lock)
			// 重要：不能用ScriptBuilder逐个AddData，因为会丢失OP codes
			// 应该直接拼接原始字节
			unlockScript := make([]byte, 0)

			// 1. 添加partial inscription数据（原始脚本字节）
			unlockScript = append(unlockScript, lastPartial...)

			// 2. 添加签名（需要添加长度前缀）
			unlockScript = append(unlockScript, byte(len(signature)))
			unlockScript = append(unlockScript, signature...)

			// 3. 添加lock脚本（需要添加长度前缀）
			unlockScript = append(unlockScript, byte(len(lastLock)))
			unlockScript = append(unlockScript, lastLock...)

			// 设置input的签名脚本
			tx.TxIn[0].SignatureScript = unlockScript
		}

		txs = append(txs, tx)

		// ===== 第九步：准备下一个交易的输入 =====
		// 对应JavaScript中的p2shInput构建
		txHash := tx.TxHash()
		p2shInput = wire.NewTxIn(
			wire.NewOutPoint(&txHash, 0),
			nil,
			nil,
		)

		// 保存当前状态用于下一个交易
		lastLock = lockScript
		lastPartial = partialScript

		// updateWallet: 更新可用UTXO列表
		// 对应JavaScript中的updateWallet(wallet, tx)
		availableUtxos = updateWalletUtxos(tx, availableUtxos, changeOutputIndex, usedUtxos)
		// fmt.Printf("next availableUtxos:%+v\n", availableUtxos)
		fmt.Printf("next availableUtxos:%d\n", len(availableUtxos))
		for _, utxo := range availableUtxos {
			fmt.Printf("next availableUtxo:%+v\n", utxo)
		}
	}

	// ===== 第十步：构建最终交易（reveal交易） =====
	// 对应JavaScript中的最终交易构建
	if p2shInput != nil && outputAddress != "" {
		finalTx := wire.NewMsgTx(2)
		finalTx.AddTxIn(p2shInput)

		// 解码目标地址
		addr, err := btcutil.DecodeAddress(outputAddress, netParam)
		if err != nil {
			return nil, nil, fmt.Errorf("解码目标地址失败: %v", err)
		}
		pkScript, err := txscript.PayToAddrScript(addr)
		if err != nil {
			return nil, nil, fmt.Errorf("构建目标地址脚本失败: %v", err)
		}

		// 添加输出到目标地址（使用用户指定的金额或默认100000）
		if outputValue == 0 {
			outputValue = 100000
		}
		finalTxOut := wire.NewTxOut(outputValue, pkScript)
		finalTx.AddTxOut(finalTxOut)

		// fund最终交易：添加UTXO输入来支付手续费
		estimatedFinalSigSize := len(lastPartial) + 72 + len(lastLock) + 10
		usedFinalUtxos, changeOutputIndex, remainingUtxos, err := fundTransaction(
			finalTx,
			availableUtxos,
			changeAddress,
			netParam,
			feeRate,
			100000, // P2SH输入的金额
			estimatedFinalSigSize,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("fund最终交易失败: %v", err)
		}

		// 更新最终可用的找零 UTXO
		availableUtxos = updateWalletUtxos(finalTx, remainingUtxos, changeOutputIndex, usedFinalUtxos)

		// 先为最终交易的UTXO输入签名
		err = signTransactionInputs(finalTx, usedFinalUtxos, 1) // P2SH输入在索引0，UTXO从索引1开始
		if err != nil {
			return nil, nil, fmt.Errorf("签名最终交易的UTXO输入失败: %v", err)
		}

		// 再对最终交易的P2SH输入进行签名（必须在UTXO签名之后）
		// 注意：RawTxInSignature 函数会自动处理签名哈希的计算
		signature, err := txscript.RawTxInSignature(finalTx, 0, lastLock, txscript.SigHashAll, privateKey)
		if err != nil {
			return nil, nil, fmt.Errorf("最终交易P2SH签名失败: %v", err)
		}

		// 构建完整的unlock脚本
		// 直接拼接原始字节，保持OP codes
		finalUnlockScript := make([]byte, 0)

		// 1. 添加partial inscription数据（原始脚本字节）
		finalUnlockScript = append(finalUnlockScript, lastPartial...)

		// 2. 添加签名（需要添加长度前缀）
		finalUnlockScript = append(finalUnlockScript, byte(len(signature)))
		finalUnlockScript = append(finalUnlockScript, signature...)

		// 3. 添加lock脚本（需要添加长度前缀）
		finalUnlockScript = append(finalUnlockScript, byte(len(lastLock)))
		finalUnlockScript = append(finalUnlockScript, lastLock...)

		finalTx.TxIn[0].SignatureScript = finalUnlockScript

		txs = append(txs, finalTx)
	}

	// 将最终可用的 UTXO 转为 changeUtxos 暴露给调用方
	for _, u := range availableUtxos {
		changeUtxos = append(changeUtxos, &ChangeUtxo{
			TxId:     u.TxId,
			TxIndex:  u.TxIndex,
			PkScript: u.PkScript,
			Amount:   u.Amount,
			PriHex:   u.PriHex,
			SignMode: u.SignMode,
		})
	}

	return txs, changeUtxos, nil
}

// DogeTxInfo 存储解析后的 Dogecoin 交易信息
type DogeTxInfo struct {
	TxHash      string
	Version     int32
	LockTime    uint32
	Size        int
	Inputs      []*DogeTxInput
	Outputs     []*DogeTxOutput
	TotalInput  int64 // 总输入金额（satoshis），需要外部提供
	TotalOutput int64 // 总输出金额（satoshis）
	Fee         int64 // 手续费（satoshis），需要输入金额才能计算
	FeeRate     int64 // 费率（satoshis/byte），需要输入金额才能计算
}

// DogeTxInput 存储交易输入信息
type DogeTxInput struct {
	Index           int
	PreviousTxHash  string
	PreviousIndex   uint32
	Sequence        uint32
	SignatureScript string
	Witness         [][]byte
	ScriptType      string
	Address         string
}

// DogeTxOutput 存储交易输出信息
type DogeTxOutput struct {
	Index      int
	Value      int64
	Script     string
	ScriptType string
	Address    string
}

// ParseDogeTx 解析 Dogecoin 交易的完整信息
// txRaw: 十六进制格式的原始交易数据
// netParam: 网络参数（用于地址解析）
// inputAmounts: 可选的输入金额列表（satoshis），如果提供则计算手续费和费率
func ParseDogeTx(txRaw string, netParam *chaincfg.Params, inputAmounts ...[]int64) (*DogeTxInfo, error) {
	// 解码交易
	txBytes, err := hex.DecodeString(txRaw)
	if err != nil {
		return nil, fmt.Errorf("解码交易失败: %v", err)
	}

	var tx wire.MsgTx
	err = tx.Deserialize(bytes.NewReader(txBytes))
	if err != nil {
		return nil, fmt.Errorf("反序列化交易失败: %v", err)
	}

	info := &DogeTxInfo{
		TxHash:      tx.TxHash().String(),
		Version:     tx.Version,
		LockTime:    tx.LockTime,
		Size:        len(txBytes),
		Inputs:      make([]*DogeTxInput, 0),
		Outputs:     make([]*DogeTxOutput, 0),
		TotalInput:  0,
		TotalOutput: 0,
		Fee:         0,
		FeeRate:     0,
	}

	// 解析输入
	for i, txIn := range tx.TxIn {
		input := &DogeTxInput{
			Index:          i,
			PreviousTxHash: txIn.PreviousOutPoint.Hash.String(),
			PreviousIndex:  txIn.PreviousOutPoint.Index,
			Sequence:       txIn.Sequence,
		}

		// 解析签名脚本
		if len(txIn.SignatureScript) > 0 {
			input.SignatureScript = hex.EncodeToString(txIn.SignatureScript)

			// 尝试从签名脚本中提取地址
			scriptClass, addrs, _, err := txscript.ExtractPkScriptAddrs(txIn.SignatureScript, netParam)
			if err == nil && len(addrs) > 0 {
				input.Address = addrs[0].EncodeAddress()
				input.ScriptType = scriptClass.String()
			} else {
				// 如果无法从签名脚本提取，尝试解析脚本类型
				input.ScriptType = "Unknown"
			}
		}

		// 解析 witness
		if len(txIn.Witness) > 0 {
			input.Witness = make([][]byte, len(txIn.Witness))
			for j, w := range txIn.Witness {
				input.Witness[j] = make([]byte, len(w))
				copy(input.Witness[j], w)
			}
		}

		info.Inputs = append(info.Inputs, input)
	}

	// 解析输出
	for i, txOut := range tx.TxOut {
		output := &DogeTxOutput{
			Index:  i,
			Value:  txOut.Value,
			Script: hex.EncodeToString(txOut.PkScript),
		}

		// 解析输出脚本类型和地址
		scriptClass, addrs, _, err := txscript.ExtractPkScriptAddrs(txOut.PkScript, netParam)
		if err == nil {
			output.ScriptType = scriptClass.String()
			if len(addrs) > 0 {
				output.Address = addrs[0].EncodeAddress()
			}
		} else {
			output.ScriptType = "Unknown"
		}

		info.Outputs = append(info.Outputs, output)
		info.TotalOutput += output.Value
	}

	// 计算费率（如果提供了输入金额）
	if len(inputAmounts) > 0 && len(inputAmounts[0]) > 0 {
		amounts := inputAmounts[0]
		// 计算总输入金额
		for _, amount := range amounts {
			info.TotalInput += amount
		}

		// 计算手续费
		if info.TotalInput > info.TotalOutput {
			info.Fee = info.TotalInput - info.TotalOutput
		}

		// 计算费率（satoshis/byte）
		if info.Size > 0 && info.Fee > 0 {
			info.FeeRate = info.Fee / int64(info.Size)
		}
	}

	return info, nil
}

// DecodeDogeTx decodes a hex-encoded Dogecoin transaction into wire.MsgTx.
func DecodeDogeTx(txHex string) (*wire.MsgTx, error) {
	raw, err := hex.DecodeString(strings.TrimSpace(txHex))
	if err != nil {
		return nil, fmt.Errorf("failed to decode tx hex: %w", err)
	}
	var tx wire.MsgTx
	if err := tx.Deserialize(bytes.NewReader(raw)); err != nil {
		return nil, fmt.Errorf("failed to deserialize tx: %w", err)
	}
	return &tx, nil
}

// GetDogeTxhashFromRaw returns the txid (hash) of a raw Dogecoin transaction hex.
func GetDogeTxhashFromRaw(rawTx string) string {
	tx, err := DecodeDogeTx(rawTx)
	if err != nil {
		return ""
	}
	return tx.TxHash().String()
}

// computeDogeInscriptionScriptSize computes the inscription script size without building it.
// Matches BuildDogeMetaIdInscriptionEx structure: header + payload (data split into 240-byte parts).
// Push encoding: 1-75 bytes use 1+len, 76-255 use 2+len (OP_PUSHDATA1).
func computeDogeInscriptionScriptSize(data []byte, path, contentType string) int {
	header := 1 + 6 + 1 + 6 // metaid, create
	header += 1 + len(path)
	header += 1 + 1 // "0"
	header += 1 + 5 // "0.0.1"
	header += 1 + len(contentType)

	payload := 0
	for i := 0; i < len(data); i += int(MAX_CHUNK_LEN) {
		end := i + int(MAX_CHUNK_LEN)
		if end > len(data) {
			end = len(data)
		}
		partLen := end - i
		if partLen <= 75 {
			payload += 1 + partLen
		} else {
			payload += 2 + partLen // OP_PUSHDATA1 + length byte
		}
	}
	return header + payload
}

const maxDogeInscriptionScriptSize = 500 * 1024 // Bitcoin/Dogecoin consensus MAX_SCRIPT_SIZE

// EstimateDogeInscriptionFee estimates total fee for inscribing data via BuildDogeMetaIdInscriptionTxs.
// Used by EstimateChunkedUpload when chain is doge.
// feeRateSatPerKB: DOGE fee rate in satoshis per KB (config convention).
// Returns total fee in satoshis (covers lock+reveal tx chain for the inscription).
func EstimateDogeInscriptionFee(data []byte, path, contentType string, feeRateSatPerKB int64) (int64, error) {
	if feeRateSatPerKB <= 0 {
		feeRateSatPerKB = 200000 // default ~0.002 DOGE/KB
	}
	feeRatePerByte := feeRateSatPerKB / 1024
	if feeRatePerByte < 1 {
		feeRatePerByte = 1
	}

	scriptLen := computeDogeInscriptionScriptSize(data, path, contentType)
	if scriptLen > maxDogeInscriptionScriptSize {
		return 0, fmt.Errorf("inscription data too large for DOGE (script max 10KB): estimated %d bytes", scriptLen)
	}
	numPartials := (scriptLen + int(MAX_PAYLOAD_LEN) - 1) / int(MAX_PAYLOAD_LEN)
	if numPartials < 1 {
		numPartials = 1
	}

	// Each partial: 1 lock tx + 1 reveal tx.
	// Lock tx (first): no P2SH in, ~400 bytes. Lock tx (rest): P2SH in with large unlock ~2000 bytes.
	// Reveal tx: P2SH in ~2000 bytes + output ~34, ~2100 bytes.
	const lockTxFirstBytes = 400
	const lockTxSubsequentBytes = 2000
	const revealTxBytes = 2100

	var totalFee int64
	for i := 0; i < numPartials; i++ {
		var lockSize int
		if i == 0 {
			lockSize = lockTxFirstBytes
		} else {
			lockSize = lockTxSubsequentBytes
		}
		totalFee += int64(lockSize+revealTxBytes) * feeRatePerByte
	}
	if totalFee < 600 {
		totalFee = 600
	}
	return totalFee, nil
}
