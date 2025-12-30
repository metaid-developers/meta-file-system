package idaddress

import (
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcutil/base58"
)

// Base58Decode 解码Base58字符串
func Base58Decode(input string) ([]byte, error) {
	decoded := base58.Decode(input)
	if len(decoded) == 0 && len(input) > 0 {
		return nil, errors.New("invalid base58 string")
	}
	return decoded, nil
}

// Base58Encode 编码为Base58字符串
func Base58Encode(input []byte) string {
	return base58.Encode(input)
}

// Base58CheckEncode 使用Base58Check编码
func Base58CheckEncode(version byte, payload []byte) string {
	// 组合版本和载荷
	data := make([]byte, 1+len(payload))
	data[0] = version
	copy(data[1:], payload)

	// 计算校验和
	checksum := doubleSHA256(data)

	// 添加校验和
	result := make([]byte, len(data)+4)
	copy(result, data)
	copy(result[len(data):], checksum[:4])

	return Base58Encode(result)
}

// Base58CheckDecode 解码Base58Check编码
func Base58CheckDecode(input string) (version byte, payload []byte, err error) {
	decoded, err := Base58Decode(input)
	if err != nil {
		return 0, nil, err
	}

	if len(decoded) < 5 {
		return 0, nil, errors.New("decoded data too short")
	}

	// 提取数据和校验和
	data := decoded[:len(decoded)-4]
	checksum := decoded[len(decoded)-4:]

	// 验证校验和
	expectedChecksum := doubleSHA256(data)
	for i := 0; i < 4; i++ {
		if checksum[i] != expectedChecksum[i] {
			return 0, nil, errors.New("checksum mismatch")
		}
	}

	return data[0], data[1:], nil
}

// doubleSHA256 计算双重SHA256
func doubleSHA256(data []byte) []byte {
	first := sha256.Sum256(data)
	second := sha256.Sum256(first[:])
	return second[:]
}

// BitcoinAddress 比特币地址信息
type BitcoinAddress struct {
	Type    string // "P2PKH", "P2SH", "P2WPKH", "P2WSH"
	Network string // "mainnet", "testnet"
	Data    []byte
}

// ConvertFromBitcoin 从比特币地址转换为ID地址
func ConvertFromBitcoin(bitcoinAddr string) (string, error) {
	// 首先尝试Base58解码 (传统地址)
	version, payload, err := Base58CheckDecode(bitcoinAddr)
	if err == nil {
		return convertFromLegacyBitcoin(version, payload)
	}

	// 尝试Bech32解码 (SegWit地址)
	// 这里假设输入是传统地址，如果需要支持Bech32，需要添加相应的解码逻辑
	return "", fmt.Errorf("unsupported address format: %s", bitcoinAddr)
}

// convertFromLegacyBitcoin 从传统比特币地址转换
func convertFromLegacyBitcoin(version byte, payload []byte) (string, error) {
	switch version {
	case 0x00: // Bitcoin主网 P2PKH
		return EncodeIDAddress(VersionP2PKH, payload)
	case 0x05: // Bitcoin主网 P2SH
		return EncodeIDAddress(VersionP2SH, payload)
	case 0x6F: // Bitcoin测试网 P2PKH
		return EncodeIDAddress(VersionP2PKH, payload)
	case 0xC4: // Bitcoin测试网 P2SH
		return EncodeIDAddress(VersionP2SH, payload)
	case 0x1E: // Dogecoin主网 P2PKH
		return EncodeIDAddress(VersionP2PKH, payload)
	case 0x16: // Dogecoin主网 P2SH
		return EncodeIDAddress(VersionP2SH, payload)
	default:
		return "", fmt.Errorf("unsupported version byte: 0x%02x", version)
	}
}

// ConvertToBitcoin 从ID地址转换为比特币地址
func ConvertToBitcoin(idAddr string, network string) (string, error) {
	info, err := DecodeIDAddress(idAddr)
	if err != nil {
		return "", err
	}

	var version byte
	switch info.Version {
	case VersionP2PKH:
		if network == "mainnet" {
			version = 0x00
		} else {
			version = 0x6F
		}
	case VersionP2SH:
		if network == "mainnet" {
			version = 0x05
		} else {
			version = 0xC4
		}
	default:
		return "", fmt.Errorf("cannot convert version %d to Bitcoin address", info.Version)
	}

	return Base58CheckEncode(version, info.Data), nil
}

// ConvertToDogecoin 从ID地址转换为狗狗币地址
func ConvertToDogecoin(idAddr string) (string, error) {
	info, err := DecodeIDAddress(idAddr)
	if err != nil {
		return "", err
	}

	var version byte
	switch info.Version {
	case VersionP2PKH:
		version = 0x1E // Dogecoin P2PKH
	case VersionP2SH:
		version = 0x16 // Dogecoin P2SH
	default:
		return "", fmt.Errorf("cannot convert version %d to Dogecoin address", info.Version)
	}

	return Base58CheckEncode(version, info.Data), nil
}

// ParseBitcoinAddress 解析比特币地址
func ParseBitcoinAddress(addr string) (*BitcoinAddress, error) {
	version, payload, err := Base58CheckDecode(addr)
	if err != nil {
		return nil, err
	}

	result := &BitcoinAddress{
		Data: payload,
	}

	switch version {
	case 0x00:
		result.Type = "P2PKH"
		result.Network = "mainnet"
	case 0x05:
		result.Type = "P2SH"
		result.Network = "mainnet"
	case 0x6F:
		result.Type = "P2PKH"
		result.Network = "testnet"
	case 0xC4:
		result.Type = "P2SH"
		result.Network = "testnet"
	case 0x1E:
		result.Type = "P2PKH"
		result.Network = "dogecoin"
	case 0x16:
		result.Type = "P2SH"
		result.Network = "dogecoin"
	default:
		return nil, fmt.Errorf("unknown version: 0x%02x", version)
	}

	return result, nil
}

// AddressConverter 地址转换器
type AddressConverter struct {
	defaultNetwork string
}

// NewAddressConverter 创建新的地址转换器
func NewAddressConverter(defaultNetwork string) *AddressConverter {
	return &AddressConverter{
		defaultNetwork: defaultNetwork,
	}
}

// ToID 转换任意区块链地址到ID地址
func (ac *AddressConverter) ToID(addr string) (string, error) {
	return ConvertFromBitcoin(addr)
}

// FromID 转换ID地址到指定网络的地址
func (ac *AddressConverter) FromID(idAddr, network string) (string, error) {
	if network == "" {
		network = ac.defaultNetwork
	}

	switch network {
	case "bitcoin", "mainnet", "testnet":
		return ConvertToBitcoin(idAddr, network)
	case "dogecoin":
		return ConvertToDogecoin(idAddr)
	default:
		return "", fmt.Errorf("unsupported network: %s", network)
	}
}

// Batch 批量转换地址
func (ac *AddressConverter) Batch(addrs []string) ([]string, []error) {
	results := make([]string, len(addrs))
	errors := make([]error, len(addrs))

	for i, addr := range addrs {
		results[i], errors[i] = ac.ToID(addr)
	}

	return results, errors
}
