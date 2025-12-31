package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/bitcoinsv/bsvd/bsvec"
	"github.com/bitcoinsv/bsvd/chaincfg"
	"github.com/bitcoinsv/bsvutil"
	"golang.org/x/crypto/ripemd160"

	"meta-file-system/service/common_service/idaddress"
)

// KeyPair 密钥对
type KeyPair struct {
	PrivateKeyHex string
	PublicKeyHex  string
	WIF           string
	IDAddress     string
	MVCAddress    string
	Hash160       string
}

// hash160Bytes 计算Hash160 (SHA256 + RIPEMD160)
func hash160Bytes(data []byte) []byte {
	sha := sha256.Sum256(data)
	ripemd := ripemd160.New()
	ripemd.Write(sha[:])
	return ripemd.Sum(nil)
}

// GenerateKeyPair 生成新的密钥对
func GenerateKeyPair() (*KeyPair, error) {
	// 生成32字节随机私钥
	privKeyBytes := make([]byte, 32)
	_, err := rand.Read(privKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("generate random bytes: %w", err)
	}

	// 创建私钥
	privKey, pubKey := bsvec.PrivKeyFromBytes(bsvec.S256(), privKeyBytes)

	// 压缩公钥格式
	pubKeyBytes := pubKey.SerializeCompressed()

	// 计算公钥哈希 (Hash160)
	hash160 := hash160Bytes(pubKeyBytes)

	// 生成WIF格式私钥 (MVC主网)
	wif, err := bsvutil.NewWIF(privKey, &chaincfg.MainNetParams, true)
	if err != nil {
		return nil, fmt.Errorf("create WIF: %w", err)
	}

	// 生成ID地址
	idAddr, err := idaddress.NewP2PKHAddress(pubKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("create ID address: %w", err)
	}

	// 生成MVC地址 (P2PKH)
	mvcAddr, err := bsvutil.NewAddressPubKeyHash(hash160, &chaincfg.MainNetParams)
	if err != nil {
		return nil, fmt.Errorf("create MVC address: %w", err)
	}

	return &KeyPair{
		PrivateKeyHex: hex.EncodeToString(privKeyBytes),
		PublicKeyHex:  hex.EncodeToString(pubKeyBytes),
		WIF:           wif.String(),
		IDAddress:     idAddr,
		MVCAddress:    mvcAddr.EncodeAddress(),
		Hash160:       hex.EncodeToString(hash160),
	}, nil
}

// ConvertIDToMVC 将ID地址转换为MVC地址
func ConvertIDToMVC(idAddr string) (string, error) {
	// 解码ID地址
	info, err := idaddress.DecodeIDAddress(idAddr)
	if err != nil {
		return "", fmt.Errorf("decode ID address: %w", err)
	}

	// 检查版本
	if info.Version != idaddress.VersionP2PKH {
		return "", fmt.Errorf("only P2PKH addresses supported, got version %d", info.Version)
	}

	// 创建MVC地址
	mvcAddr, err := bsvutil.NewAddressPubKeyHash(info.Data, &chaincfg.MainNetParams)
	if err != nil {
		return "", fmt.Errorf("create MVC address: %w", err)
	}

	return mvcAddr.EncodeAddress(), nil
}

// ConvertMVCToID 将MVC地址转换为ID地址
func ConvertMVCToID(mvcAddr string) (string, error) {
	// 解码MVC地址
	addr, err := bsvutil.DecodeAddress(mvcAddr, &chaincfg.MainNetParams)
	if err != nil {
		return "", fmt.Errorf("decode MVC address: %w", err)
	}

	// 获取脚本地址
	scriptAddr, ok := addr.(*bsvutil.AddressPubKeyHash)
	if !ok {
		return "", fmt.Errorf("not a P2PKH address")
	}

	// 创建ID地址
	idAddr, err := idaddress.EncodeIDAddress(idaddress.VersionP2PKH, scriptAddr.ScriptAddress())
	if err != nil {
		return "", fmt.Errorf("create ID address: %w", err)
	}

	return idAddr, nil
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "generate":
		generateKey()
	case "convert":
		convertAddress()
	case "info":
		showInfo()
	case "prepare":
		prepareTransfer()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("MVC密钥生成和地址转换工具")
	fmt.Println()
	fmt.Println("使用方法:")
	fmt.Println("  mvckey generate              - 生成新的密钥对")
	fmt.Println("  mvckey convert <address>     - 转换地址格式 (ID ↔ MVC)")
	fmt.Println("  mvckey info <privkey_hex>    - 从私钥恢复地址信息")
	fmt.Println("  mvckey prepare <to_address>  - 准备向某地址转账（支持ID地址）")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  mvckey generate")
	fmt.Println("  mvckey convert idq1w508d6qejxtdg4y5r3zarvary0c5xw7ky30xwh")
	fmt.Println("  mvckey convert 1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa")
	fmt.Println("  mvckey info abc123...")
	fmt.Println("  mvckey prepare idq19fy6p84fmlzqgghzhrmvgu2fd3yxg3qezrq6zx")
}

func generateKey() {
	keyPair, err := GenerateKeyPair()
	if err != nil {
		log.Fatalf("生成密钥对失败: %v", err)
	}

	fmt.Println("========================================")
	fmt.Println("新生成的密钥对")
	fmt.Println("========================================")
	fmt.Println()
	fmt.Printf("私钥 (Hex):     %s\n", keyPair.PrivateKeyHex)
	fmt.Printf("私钥 (WIF):     %s\n", keyPair.WIF)
	fmt.Println()
	fmt.Printf("公钥 (Hex):     %s\n", keyPair.PublicKeyHex)
	fmt.Printf("公钥哈希:       %s\n", keyPair.Hash160)
	fmt.Println()
	fmt.Printf("ID地址:         %s\n", keyPair.IDAddress)
	fmt.Printf("MVC地址:        %s\n", keyPair.MVCAddress)
	fmt.Println()
	fmt.Println("⚠️  请妥善保管私钥，不要泄露给任何人！")
	fmt.Println("========================================")
	fmt.Println()
	fmt.Println("测试转账步骤:")
	fmt.Println("1. 向 MVC 地址转入测试币")
	fmt.Println("2. 使用 MVC 钱包或 RPC 接口发送交易")
	fmt.Println("3. 交易确认后可在区块浏览器查看")
}

func convertAddress() {
	if len(os.Args) < 3 {
		fmt.Println("用法: mvckey convert <address>")
		os.Exit(1)
	}

	address := os.Args[2]

	// 尝试作为ID地址解析
	if len(address) > 4 && address[:2] == "id" {
		mvcAddr, err := ConvertIDToMVC(address)
		if err != nil {
			log.Fatalf("转换失败: %v", err)
		}
		fmt.Printf("ID地址:  %s\n", address)
		fmt.Printf("MVC地址: %s\n", mvcAddr)
	} else {
		// 作为MVC地址解析
		idAddr, err := ConvertMVCToID(address)
		if err != nil {
			log.Fatalf("转换失败: %v", err)
		}
		fmt.Printf("MVC地址: %s\n", address)
		fmt.Printf("ID地址:  %s\n", idAddr)
	}
}

func showInfo() {
	if len(os.Args) < 3 {
		fmt.Println("用法: mvckey info <privkey_hex>")
		os.Exit(1)
	}

	privKeyHex := os.Args[2]

	// 解析私钥
	privKeyBytes, err := hex.DecodeString(privKeyHex)
	if err != nil {
		log.Fatalf("解析私钥失败: %v", err)
	}

	if len(privKeyBytes) != 32 {
		log.Fatalf("私钥长度必须是32字节，当前是 %d 字节", len(privKeyBytes))
	}

	// 创建私钥
	privKey, pubKey := bsvec.PrivKeyFromBytes(bsvec.S256(), privKeyBytes)
	pubKeyBytes := pubKey.SerializeCompressed()
	hash160 := hash160Bytes(pubKeyBytes)

	// 生成WIF
	wif, err := bsvutil.NewWIF(privKey, &chaincfg.MainNetParams, true)
	if err != nil {
		log.Fatalf("创建WIF失败: %v", err)
	}

	// 生成ID地址
	idAddr, err := idaddress.NewP2PKHAddress(pubKeyBytes)
	if err != nil {
		log.Fatalf("创建ID地址失败: %v", err)
	}

	// 生成MVC地址
	mvcAddr, err := bsvutil.NewAddressPubKeyHash(hash160, &chaincfg.MainNetParams)
	if err != nil {
		log.Fatalf("创建MVC地址失败: %v", err)
	}

	fmt.Println("========================================")
	fmt.Println("从私钥恢复的地址信息")
	fmt.Println("========================================")
	fmt.Println()
	fmt.Printf("私钥 (Hex):     %s\n", privKeyHex)
	fmt.Printf("私钥 (WIF):     %s\n", wif.String())
	fmt.Println()
	fmt.Printf("公钥 (Hex):     %s\n", hex.EncodeToString(pubKeyBytes))
	fmt.Printf("公钥哈希:       %s\n", hex.EncodeToString(hash160))
	fmt.Println()
	fmt.Printf("ID地址:         %s\n", idAddr)
	fmt.Printf("MVC地址:        %s\n", mvcAddr.EncodeAddress())
	fmt.Println("========================================")
}

func prepareTransfer() {
	if len(os.Args) < 3 {
		fmt.Println("用法: mvckey prepare <to_address>")
		fmt.Println()
		fmt.Println("说明: 准备向指定地址转账，自动处理ID地址转换")
		os.Exit(1)
	}

	toAddress := os.Args[2]

	fmt.Println("========================================")
	fmt.Println("转账地址准备")
	fmt.Println("========================================")
	fmt.Println()

	// 检查是否是ID地址
	if len(toAddress) > 4 && toAddress[:2] == "id" {
		fmt.Println("✓ 检测到ID地址格式")
		fmt.Printf("  原始地址: %s\n", toAddress)
		fmt.Println()

		// 转换为MVC地址
		mvcAddr, err := ConvertIDToMVC(toAddress)
		if err != nil {
			log.Fatalf("❌ 转换失败: %v", err)
		}

		fmt.Println("✓ 已转换为MVC地址")
		fmt.Printf("  转账地址: %s\n", mvcAddr)
		fmt.Println()
		fmt.Println("========================================")
		fmt.Println("💡 重要说明")
		fmt.Println("========================================")
		fmt.Println("MVC链和钱包不能直接识别ID地址格式。")
		fmt.Println("请使用上面转换后的 MVC地址 进行转账。")
		fmt.Println()
		fmt.Println("两个地址指向同一个账户，转账到MVC地址")
		fmt.Println("等同于转账到ID地址。")
		fmt.Println()
		fmt.Println("========================================")
		fmt.Println("📝 钱包转账步骤")
		fmt.Println("========================================")
		fmt.Println("1. 复制上面的 MVC地址")
		fmt.Println("2. 在MVC钱包中选择'发送'")
		fmt.Println("3. 粘贴MVC地址作为收款地址")
		fmt.Println("4. 输入转账金额")
		fmt.Println("5. 确认并发送")
		fmt.Println()
		fmt.Println("========================================")
		fmt.Println("📝 RPC转账命令")
		fmt.Println("========================================")
		fmt.Printf("mvc-cli sendfrom \"账户名\" \"%s\" 0.001\n", mvcAddr)
		fmt.Println("========================================")

	} else {
		// 已经是MVC地址
		fmt.Println("✓ 检测到MVC地址格式")
		fmt.Printf("  转账地址: %s\n", toAddress)
		fmt.Println()

		// 尝试转换为ID地址显示
		idAddr, err := ConvertMVCToID(toAddress)
		if err == nil {
			fmt.Println("ℹ️  对应的ID地址")
			fmt.Printf("  ID地址: %s\n", idAddr)
			fmt.Println()
		}

		fmt.Println("========================================")
		fmt.Println("📝 钱包转账步骤")
		fmt.Println("========================================")
		fmt.Println("1. 在MVC钱包中选择'发送'")
		fmt.Println("2. 输入收款地址（上面的MVC地址）")
		fmt.Println("3. 输入转账金额")
		fmt.Println("4. 确认并发送")
		fmt.Println()
		fmt.Println("========================================")
		fmt.Println("📝 RPC转账命令")
		fmt.Println("========================================")
		fmt.Printf("mvc-cli sendfrom \"账户名\" \"%s\" 0.001\n", toAddress)
		fmt.Println("========================================")
	}
}
