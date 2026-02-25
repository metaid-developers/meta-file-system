package common

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"meta-file-system/node"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

func TestEstimateDogeInscriptionFee(t *testing.T) {
	// Small data: 1 partial
	fee, err := EstimateDogeInscriptionFee([]byte("hello"), "/file/_chunk", "text/plain", 200000)
	if err != nil {
		t.Fatalf("EstimateDogeInscriptionFee: %v", err)
	}
	if fee <= 0 {
		t.Errorf("expected positive fee, got %d", fee)
	}
	// ~2500 bytes * (200000/1024) sat/byte = ~488000 sats min
	if fee < 500 {
		t.Errorf("fee %d seems too low for DOGE inscription", fee)
	}
	t.Logf("EstimateDogeInscriptionFee(5 bytes): %d sats", fee)

	// Large data exceeding 10KB script limit: should return error
	largeData := make([]byte, 15000) // ~15KB raw data produces script > 10KB
	_, err = EstimateDogeInscriptionFee(largeData, "/file/index", "metafile/index;utf-8", 200000)
	if err == nil {
		t.Errorf("expected error for large data, got nil")
	}
	if !strings.Contains(err.Error(), "too large for DOGE") {
		t.Errorf("expected 'too large for DOGE' in error, got: %v", err)
	}
}

func TestBuildDogeMetaIdInscriptionTxs(t *testing.T) {
	// 指定的私钥和地址
	privateKeyHex := "0128fc43ba66d5561be9fb8910e74ee50c01806f097eb55e907ab540ebddbb53"
	address := "DG1oSLYL3zAtNg74bGx4fKhknwBEZvNw1x"
	changeAddress := address

	fmt.Printf("使用私钥: %s\n", privateKeyHex)
	fmt.Printf("地址: %s\n", address)

	// 测试数据
	testData := []byte("Hello doge")
	contentType := "text/plain"
	outputValue := int64(100000)
	feeRate := int64(100000)
	netParam := DogeMainNetParams

	// 构建 ins (UTXO 输入)
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		t.Fatalf("解码私钥失败: %v", err)
	}
	privateKey, _ := btcec.PrivKeyFromBytes(privateKeyBytes)

	// 生成 P2PKH 地址和 pkScript（Legacy 地址）
	pubKeyHash := btcutil.Hash160(privateKey.PubKey().SerializeCompressed())
	addr, err := btcutil.NewAddressPubKeyHash(pubKeyHash, netParam)
	if err != nil {
		t.Fatalf("生成地址失败: %v", err)
	}
	pkScript, err := txscript.PayToAddrScript(addr)
	if err != nil {
		t.Fatalf("生成 pkScript 失败: %v", err)
	}

	// 构建 UTXO 输入列表
	ins := []*TxInputUtxo{
		// {
		// 	TxId:     "b8223c4187a51640d57b639cbe1354b4c1879bb6883563ecb39765f8a72ac7c2", // 模拟的前置交易ID
		// 	TxIndex:  0,
		// 	PkScript: hex.EncodeToString(pkScript),
		// 	Amount:   500000000, // 5 DOGE
		// 	PriHex:   privateKeyHex,
		// 	SignMode: SignModeLegacy, // Dogecoin 使用 Legacy 签名模式
		// },
		{
			TxId:     "28aa76c9c4051446806c673f1fc4d5582e60b08a62c208cb8efba0af6b65769c", // 模拟的前置交易ID
			TxIndex:  1,
			PkScript: hex.EncodeToString(pkScript),
			Amount:   26900000, // 5 DOGE
			PriHex:   privateKeyHex,
			SignMode: SignModeLegacy, // Dogecoin 使用 Legacy 签名模式
		},
		{
			TxId:     "426c69a2d179fc5cbfa2caff7f0b0ee4f90c596fada8eb077ce17fffec01a433", // 模拟的前置交易ID
			TxIndex:  1,
			PkScript: hex.EncodeToString(pkScript),
			Amount:   25300000, // 5 DOGE
			PriHex:   privateKeyHex,
			SignMode: SignModeLegacy, // Dogecoin 使用 Legacy 签名模式
		},
		{
			TxId:     "3097b38ee8cb2fc99d95f0befa2b3d3d25c3b3fb1c4c6430c81934fd00408b33", // 模拟的前置交易ID
			TxIndex:  0,
			PkScript: hex.EncodeToString(pkScript),
			Amount:   100000000, // 5 DOGE
			PriHex:   privateKeyHex,
			SignMode: SignModeLegacy, // Dogecoin 使用 Legacy 签名模式
		},

		// {
		// 	TxId:     "520ab36085c146719f48add196c776183d2c26e9020ed45da5358235cbd3c496", // 模拟的前置交易ID
		// 	TxIndex:  1,
		// 	PkScript: hex.EncodeToString(pkScript),
		// 	Amount:   206300000, // 2.063 DOGE
		// 	PriHex:   privateKeyHex,
		// 	SignMode: SignModeLegacy, // Dogecoin 使用 Legacy 签名模式
		// },

		// {
		// 	TxId:     "eaecddbd596f7ae81de26d97765ccb8f3c16514b0652cfb42a0b5d14f1d32a09", // 模拟的前置交易ID
		// 	TxIndex:  1,
		// 	PkScript: hex.EncodeToString(pkScript),
		// 	Amount:   331000000, // 3.31 DOGE
		// 	PriHex:   privateKeyHex,
		// 	SignMode: SignModeLegacy, // Dogecoin 使用 Legacy 签名模式
		// },
	}

	fmt.Printf("构建的 UTXO 输入:\n")
	fmt.Printf("  TxId: %s\n", ins[0].TxId)
	fmt.Printf("  TxIndex: %d\n", ins[0].TxIndex)
	fmt.Printf("  Amount: %d satoshis\n", ins[0].Amount)
	fmt.Printf("  PkScript: %s\n", ins[0].PkScript)
	fmt.Println()

	// 构建inscription交易
	// 注意: outputAddress 设为空，因为 Dogecoin 地址无法被 Bitcoin 网络参数识别
	// 这样只会生成 commit 交易，实际使用时需要使用 Dogecoin 专用网络参数
	txs, changeUtxos, err := BuildDogeMetaIdInscriptionTxs(
		netParam,
		testData,
		contentType,
		ins,
		address,
		outputValue,
		changeAddress,
		feeRate,
		false,
		InscriptionFormatMetaID,
		"", // path empty for backward compat, contentType used as path
	)
	if err != nil {
		t.Fatalf("构建inscription交易失败: %v", err)
	}

	fmt.Printf("生成交易数量: %d, 剩余找零UTXO数量: %d\n\n", len(txs), len(changeUtxos))
	for i, cu := range changeUtxos {
		fmt.Printf("changeUtxo[%d]: TxId=%s TxIndex=%d Amount=%d\n", i, cu.TxId, cu.TxIndex, cu.Amount)
	}

	// 打印交易信息和txRaw
	for i, tx := range txs {
		fmt.Printf("========== 交易 %d ==========\n", i+1)
		fmt.Printf("TxHash: %s\n", tx.TxHash().String())

		// 序列化为txRaw
		var buf bytes.Buffer
		if err := tx.Serialize(&buf); err != nil {
			t.Fatalf("序列化交易失败: %v", err)
		}
		txRaw := hex.EncodeToString(buf.Bytes())

		fmt.Printf("TxRaw: %s\n", txRaw)
		fmt.Printf("大小: %d 字节\n\n", len(buf.Bytes()))
	}

	/*
					=== RUN   TestBuildDogeMetaIdInscriptionTxs
				使用私钥: 0128fc43ba66d5561be9fb8910e74ee50c01806f097eb55e907ab540ebddbb53
				地址: DG1oSLYL3zAtNg74bGx4fKhknwBEZvNw1x
				构建的 UTXO 输入:
				  TxId: 520ab36085c146719f48add196c776183d2c26e9020ed45da5358235cbd3c496
				  TxIndex: 1
				  Amount: 206300000 satoshis
				  PkScript: 76a9147748321f517ae351be891b6fe702563293672b4f88ac


				=== P2SH Inscription 临时密钥对 ===
				私钥: 0347de88e3f8aef6e5429956ad7cbab4a3882397145b8672f00b374f7cffee1b
				公钥: 03e9dc60453597070391c142448a01b4a772a2f92dda6f82a9aa2a509b2ee8074b
				tempTxSize:224, estimatedSigSize:0, len(usedUtxos):1, feeRate:600000, finalFee:134400000, changeAmount:71800000
				usedUtxos:1
				usedUtxo:&{TxId:520ab36085c146719f48add196c776183d2c26e9020ed45da5358235cbd3c496 TxIndex:1 PkScript:76a9147748321f517ae351be891b6fe702563293672b4f88ac Amount:206300000 PriHex:0128fc43ba66d5561be9fb8910e74ee50c01806f097eb55e907ab540ebddbb53 SignMode:legacy}
				remainingUtxos:1
				remainingUtxo:&{TxId:eaecddbd596f7ae81de26d97765ccb8f3c16514b0652cfb42a0b5d14f1d32a09 TxIndex:1 PkScript:76a9147748321f517ae351be891b6fe702563293672b4f88ac Amount:331000000 PriHex:0128fc43ba66d5561be9fb8910e74ee50c01806f097eb55e907ab540ebddbb53 SignMode:legacy}
				changeOutputIndex:1
				next availableUtxos:2
				next availableUtxo:&{TxId:eaecddbd596f7ae81de26d97765ccb8f3c16514b0652cfb42a0b5d14f1d32a09 TxIndex:1 PkScript:76a9147748321f517ae351be891b6fe702563293672b4f88ac Amount:331000000 PriHex:0128fc43ba66d5561be9fb8910e74ee50c01806f097eb55e907ab540ebddbb53 SignMode:legacy}
				next availableUtxo:&{TxId:e197ce0d1ad91ab653dcdf5a5a94a8579c6e379d065377adee007c47430af56c TxIndex:1 PkScript:76a9147748321f517ae351be891b6fe702563293672b4f88ac Amount:71800000 PriHex:0128fc43ba66d5561be9fb8910e74ee50c01806f097eb55e907ab540ebddbb53 SignMode:legacy}
				tempTxSize:436, estimatedSigSize:169, len(usedUtxos):1, feeRate:600000, finalFee:261600000, changeAmount:69400000
				usedUtxos:1
				usedUtxo:&{TxId:eaecddbd596f7ae81de26d97765ccb8f3c16514b0652cfb42a0b5d14f1d32a09 TxIndex:1 PkScript:76a9147748321f517ae351be891b6fe702563293672b4f88ac Amount:331000000 PriHex:0128fc43ba66d5561be9fb8910e74ee50c01806f097eb55e907ab540ebddbb53 SignMode:legacy}
				remainingUtxos:1
				remainingUtxo:&{TxId:e197ce0d1ad91ab653dcdf5a5a94a8579c6e379d065377adee007c47430af56c TxIndex:1 PkScript:76a9147748321f517ae351be891b6fe702563293672b4f88ac Amount:71800000 PriHex:0128fc43ba66d5561be9fb8910e74ee50c01806f097eb55e907ab540ebddbb53 SignMode:legacy}
				changeOutputIndex:1
				生成交易数量: 2

				========== 交易 1 ==========
				TxHash: e197ce0d1ad91ab653dcdf5a5a94a8579c6e379d065377adee007c47430af56c
				TxRaw: 020000000196c4d3cb358235a55dd40e02e9262c3d1876c796d1ad489f7146c18560b30a52010000006b48304502210090c3c39870c4f91550855ca3acdcf95986f8d92a5e8f064f649613af9d2e1386022014248aca3435c88c900173f4e92dde06cbe6cb8fc6d96c852a2dd7b22eb62b180121029276cc28460500aae93fa8fa619e25ca98b5689b381ba1d730e9441b04fb6ceeffffffff02a08601000000000017a9146973cd4c82bac039704428ee2b4725c9dfcdf27187c0944704000000001976a9147748321f517ae351be891b6fe702563293672b4f88ac00000000
				大小: 224 字节

				========== 交易 2 ==========
				TxHash: 98ee4ec537d485e94bd45bc8b48da752af6179ac312f6824416dc444c1c812d5
				TxRaw: 02000000026cf50a43477c00eead7753069d376e9c57a8945a5adfdc53b61ad91a0dce97e100000000a1036f7264510a746578742f706c61696e001c48656c6c6f2c20446f6765636f696e20496e736372697074696f6e214830450221009e844fd256e96053f166e720009fc0063197f6bbdd085da39540d135e6d10c4602202d76e4e1263ec4e1ce365ff585a36bfced3427e364f6fe581051a40bd101bf4701292103e9dc60453597070391c142448a01b4a772a2f92dda6f82a9aa2a509b2ee8074bad757575757551ffffffff092ad3f1145d0b2ab4cf52064b51163c8fcb5c76976de21de87a6f59bdddecea010000006a473044022065cacffd524520fe1442ab6ec614c6527eaa732785d39a47c083d66ceb61795902207b6054caca639eb51b149ea26d9c2eb31209c325a38e4fece71318636aa5dd910121029276cc28460500aae93fa8fa619e25ca98b5689b381ba1d730e9441b04fb6ceeffffffff02a0860100000000001976a9147748321f517ae351be891b6fe702563293672b4f88acc0f52204000000001976a9147748321f517ae351be891b6fe702563293672b4f88ac00000000
				大小: 427 字节

				--- PASS: TestBuildDogeMetaIdInscriptionTxs (0.01s)

				第一个交易广播成功，交易ID: e197ce0d1ad91ab653dcdf5a5a94a8579c6e379d065377adee007c47430af56c
		第二个交易广播成功，交易ID: 98ee4ec537d485e94bd45bc8b48da752af6179ac312f6824416dc444c1c812d5
	*/

	/*
						metaid


							=== RUN   TestBuildDogeMetaIdInscriptionTxs
							使用私钥: 0128fc43ba66d5561be9fb8910e74ee50c01806f097eb55e907ab540ebddbb53
							地址: DG1oSLYL3zAtNg74bGx4fKhknwBEZvNw1x
							构建的 UTXO 输入:
							  TxId: b8223c4187a51640d57b639cbe1354b4c1879bb6883563ecb39765f8a72ac7c2
							  TxIndex: 0
							  Amount: 500000000 satoshis
							  PkScript: 76a9147748321f517ae351be891b6fe702563293672b4f88ac


							=== P2SH Inscription 临时密钥对 ===
							私钥: 27779dd2c8b5e90f997e4d415d34df3a4a08c964495e8605429cc28b8968f3c0
							公钥: 0319c22b4b1e99ce8e08495b47f38cd901d851d7d3467c4dfe52b70fa6c5d2fee6
							tempTxSize:224, estimatedSigSize:0, len(usedUtxos):1, feeRate:600000, finalFee:134400000, changeAmount:365500000
							usedUtxos:1
							usedUtxo:&{TxId:b8223c4187a51640d57b639cbe1354b4c1879bb6883563ecb39765f8a72ac7c2 TxIndex:0 PkScript:76a9147748321f517ae351be891b6fe702563293672b4f88ac Amount:500000000 PriHex:0128fc43ba66d5561be9fb8910e74ee50c01806f097eb55e907ab540ebddbb53 SignMode:legacy}
							remainingUtxos:0
							changeOutputIndex:1
							next availableUtxos:1
							next availableUtxo:&{TxId:b6829b5b95f83ccad0d3a7608c915d08d348e46b87a2bbc217be18be88d2b43c TxIndex:1 PkScript:76a9147748321f517ae351be891b6fe702563293672b4f88ac Amount:365500000 PriHex:0128fc43ba66d5561be9fb8910e74ee50c01806f097eb55e907ab540ebddbb53 SignMode:legacy}
							tempTxSize:478, estimatedSigSize:211, len(usedUtxos):1, feeRate:600000, finalFee:286800000, changeAmount:78700000
							usedUtxos:1
							usedUtxo:&{TxId:b6829b5b95f83ccad0d3a7608c915d08d348e46b87a2bbc217be18be88d2b43c TxIndex:1 PkScript:76a9147748321f517ae351be891b6fe702563293672b4f88ac Amount:365500000 PriHex:0128fc43ba66d5561be9fb8910e74ee50c01806f097eb55e907ab540ebddbb53 SignMode:legacy}
							remainingUtxos:0
							changeOutputIndex:1
							生成交易数量: 2

							========== 交易 1 ==========
							TxHash: b6829b5b95f83ccad0d3a7608c915d08d348e46b87a2bbc217be18be88d2b43c
							TxRaw: 0200000001c2c72aa7f86597b3ec633588b69b87c1b45413be9c637bd54016a587413c22b8000000006b483045022100f90f3254752360f1db9b6dac5004141ac3f6be0f5305739b3a4d4fa04d7b85d402201c0f5d04322a3989d73d8ab2df5092d53523b3f7e84c3542b692a6fafd6f00c50121029276cc28460500aae93fa8fa619e25ca98b5689b381ba1d730e9441b04fb6ceeffffffff02a08601000000000017a9143840b9dfebf6bc50f554da41bf5a2966a2b93abf876016c915000000001976a9147748321f517ae351be891b6fe702563293672b4f88ac00000000
							大小: 224 字节

							========== 交易 2 ==========
							TxHash: 15e9e77536415287e5957217f80cf39cc4b01cc5b71c200c65f590fb8c19c3ba
							TxRaw: 02000000023cb4d288be18be17c2bba2876be448d3085d918c60a7d3d0ca3cf8955b9b82b600000000ca066d6574616964066372656174650a746578742f706c61696e013005302e302e31106170706c69636174696f6e2f6a736f6e2348656c6c6f2c20446f6765636f696e204d657461494420496e736372697074696f6e2147304402206e73853881351f912d62924e11eeacdcaf977bdaab5a9c8d5800d2dea936d01302201e2d47320324bae38dc19112e4f47da77fafbd85c74c1b49f87dba1995d2e079012b210319c22b4b1e99ce8e08495b47f38cd901d851d7d3467c4dfe52b70fa6c5d2fee6ad7575757575757551ffffffff3cb4d288be18be17c2bba2876be448d3085d918c60a7d3d0ca3cf8955b9b82b6010000006a47304402200dc8933188e9dc7eb712363f0ecfbe314ca7dc2f9f5a49d74c05261e584d4e9c022061b204ac203c22bef2f4cb5797eeeb7617fdee2558828ec3ffdd4711797e51d10121029276cc28460500aae93fa8fa619e25ca98b5689b381ba1d730e9441b04fb6ceeffffffff02a0860100000000001976a9147748321f517ae351be891b6fe702563293672b4f88ace0ddb004000000001976a9147748321f517ae351be891b6fe702563293672b4f88ac00000000
							大小: 468 字节

							--- PASS: TestBuildDogeMetaIdInscriptionTxs (0.01s)
				第一个交易广播成功，交易ID: b6829b5b95f83ccad0d3a7608c915d08d348e46b87a2bbc217be18be88d2b43c
				第二个交易广播成功，交易ID: 15e9e77536415287e5957217f80cf39cc4b01cc5b71c200c65f590fb8c19c3ba


			========== 交易 1 ==========
		TxHash: 28aa76c9c4051446806c673f1fc4d5582e60b08a62c208cb8efba0af6b65769c
		TxRaw: 02000000016cf50a43477c00eead7753069d376e9c57a8945a5adfdc53b61ad91a0dce97e1010000006a47304402201a71bf5196a271110eb2c00b45661bbc8473a5b015af3bc732a2efe86e71ea240220667062e77efa0f6a21dd4f27acb3678449baecdb530d2441ffedee9d5ffc33cd0121029276cc28460500aae93fa8fa619e25ca98b5689b381ba1d730e9441b04fb6ceeffffffff02a08601000000000017a914beb0d72cff11f5e69390ad60013bf8c738ff4afb8720769a01000000001976a9147748321f517ae351be891b6fe702563293672b4f88ac00000000
		大小: 223 字节

		========== 交易 2 ==========
		TxHash: 426c69a2d179fc5cbfa2caff7f0b0ee4f90c596fada8eb077ce17fffec01a433
		TxRaw: 02000000039c76656bafa0fb8ecb08c2628ab0602e58d5c41f3f676c80461405c4c976aa2800000000be066d6574616964066372656174650a746578742f706c61696e013005302e302e31106170706c69636174696f6e2f6a736f6e17446f6765206d657461696420696e736372697074696f6e47304402203f685bd7a2062f7726623381246af3f4d40ef268d571ed067d476c54250770ad022043a9d79b216cdf1b54885fcaa9b0eb8073e8eab0cf3d180f09e2361355233c9c012b2102dc3647d7dbeaf9223800276a924c9d4a07c886417e0c65d9d2c92eb080356afcad7575757575757551ffffffffd512c8c144c46d4124682f31ac7961af52a78db4c85bd44be985d437c54eee98010000006a4730440220294d502896262b31a3ed29c21a4e32f54319c858e5f610060c3b98823661d23a02203b45a72495b8108c0fdc5549f38b2eecb377c8bf8bbc7145009e545e2c09a6970121029276cc28460500aae93fa8fa619e25ca98b5689b381ba1d730e9441b04fb6ceeffffffffbac3198cfb90f5650c201cb7c51cb0c49cf30cf8177295e58752413675e7e915010000006b483045022100fecb40bfb3059d6597b93630f9a292092b7ad8331d7465aef719e6525e70ef6802205fda5e8ca7c825ef2ab6f3e710aea07c2bfe4e835a897c9c6a07b092e68ebac00121029276cc28460500aae93fa8fa619
		e25ca98b5689b381ba1d730e9441b04fb6ceeffffffff02a0860100000000001976a9147748321f517ae351be891b6fe702563293672b4f88ac200c8201000000001976a9147748321f517ae351be891b6fe702563293672b4f88ac00000000

	*/
}

func TestGetDogeTxDetail(t *testing.T) {
	txId := "723fed7dfe007c184b67b72f7e25ddd3ea7d46ce2d851809e243bab932789fa0"
	txDetail, err := node.GetTxDetail("doge", txId)
	if err != nil {
		t.Fatalf("获取交易详情失败: %v", err)
	}
	fmt.Println(txDetail)
}

func TestGetDogeTxRaw(t *testing.T) {
	// txId := "1943d6dbb12a1a6b099c3cd9793bd0cf3c675c0934298d7e742ea7021e8ef656"
	// txId := "19aa11771ece3294c1bed6340557c9563a319120f31e2dbc547e660c148d5c2c"
	txId := "1ce0985845b187f230d4cf0b6cdf9e36098997db119118f4e8c377e1c79e2b6b"
	txRaw, err := node.GetTxRaw("doge", txId)
	if err != nil {
		t.Fatalf("获取交易详情失败: %v", err)
	}
	fmt.Println(txRaw)
}

func TestParseInscriptionFromTx(t *testing.T) {
	// MetaID格式的交易
	// metaidTxRaw := "02000000023cb4d288be18be17c2bba2876be448d3085d918c60a7d3d0ca3cf8955b9b82b600000000ca066d6574616964066372656174650a746578742f706c61696e013005302e302e31106170706c69636174696f6e2f6a736f6e2348656c6c6f2c20446f6765636f696e204d657461494420496e736372697074696f6e2147304402206e73853881351f912d62924e11eeacdcaf977bdaab5a9c8d5800d2dea936d01302201e2d47320324bae38dc19112e4f47da77fafbd85c74c1b49f87dba1995d2e079012b210319c22b4b1e99ce8e08495b47f38cd901d851d7d3467c4dfe52b70fa6c5d2fee6ad7575757575757551ffffffff3cb4d288be18be17c2bba2876be448d3085d918c60a7d3d0ca3cf8955b9b82b6010000006a47304402200dc8933188e9dc7eb712363f0ecfbe314ca7dc2f9f5a49d74c05261e584d4e9c022061b204ac203c22bef2f4cb5797eeeb7617fdee2558828ec3ffdd4711797e51d10121029276cc28460500aae93fa8fa619e25ca98b5689b381ba1d730e9441b04fb6ceeffffffff02a0860100000000001976a9147748321f517ae351be891b6fe702563293672b4f88ace0ddb004000000001976a9147748321f517ae351be891b6fe702563293672b4f88ac00000000"

	// metaidTxRaw := "020000000190c7a2ee5334e4cc5d324707e1e98fa12e18e08ce925db939357871e73d6b70800000000ab066d6574616964066372656174650a2f696e666f2f6e616d65013005312e302e300a746578742f706c61696e09646f67652d74657374483045022100dd75c985b851dbff670c37630b6f0c4c1352b73682a972c12cdd690f4fcd698f02202bffc00a39943f3ccc3aa1cf6f80310bbfb36e8f4c576fb3382743c8a7c9d0f2012b210221936b09e43ba311c67444bfaf5b99e3a0ed3f7ae2445355e4f22241deb9de01ad7575757575757551ffffffff0140420f00000000001976a914ac57ee387e607e042d3963acc06bac3ec61b8fb688ac00000000"
	metaidTxRaw := "0200000001d124ac55aeee1f47d94a0af3a999de219c859bb253991246175fde86bc38d71d00000000a3066d657461696406637265617465092f696e666f2f62696f013005312e302e300a746578742f706c61696e03646f6747304402207b0f40e2f2e03404789baf27ea8d6242d59c33a6bebf49082c68a32d0255b851022061d231444a1b77acb227e0391d0c1977574902e6a791488997e54f608198685d012b21039f26eb2bb5370ada635d09e7be342a531e4fc19e85512315691f8d83b4fbe915ad7575757575757551ffffffff0140420f00000000001976a914ac57ee387e607e042d3963acc06bac3ec61b8fb688ac00000000"

	// Doginal格式的交易（使用之前生成的）
	doginalTxRaw := "02000000026cf50a43477c00eead7753069d376e9c57a8945a5adfdc53b61ad91a0dce97e100000000a1036f7264510a746578742f706c61696e001c48656c6c6f2c20446f6765636f696e20496e736372697074696f6e214830450221009e844fd256e96053f166e720009fc0063197f6bbdd085da39540d135e6d10c4602202d76e4e1263ec4e1ce365ff585a36bfced3427e364f6fe581051a40bd101bf4701292103e9dc60453597070391c142448a01b4a772a2f92dda6f82a9aa2a509b2ee8074bad757575757551ffffffff092ad3f1145d0b2ab4cf52064b51163c8fcb5c76976de21de87a6f59bdddecea010000006a473044022065cacffd524520fe1442ab6ec614c6527eaa732785d39a47c083d66ceb61795902207b6054caca639eb51b149ea26d9c2eb31209c325a38e4fece71318636aa5dd910121029276cc28460500aae93fa8fa619e25ca98b5689b381ba1d730e9441b04fb6ceeffffffff02a0860100000000001976a9147748321f517ae351be891b6fe702563293672b4f88acc0f52204000000001976a9147748321f517ae351be891b6fe702563293672b4f88ac00000000"

	fmt.Println("=== 测试解析 MetaID 格式 ===")
	fmt.Println("\n【链上原始数据排序】")
	printRawScriptChunks(metaidTxRaw)

	metaidData, err := ParseInscriptionFromTx(metaidTxRaw, InscriptionFormatMetaID)
	if err != nil {
		t.Fatalf("解析MetaID交易失败: %v", err)
	}

	fmt.Printf("\n【解析结果】\n")
	fmt.Printf("格式: MetaID\n")
	fmt.Printf("操作: %s\n", metaidData.Operation)
	fmt.Printf("路径: %s\n", metaidData.Path)
	fmt.Printf("加密: %s\n", metaidData.Encryption)
	fmt.Printf("版本: %s\n", metaidData.Version)
	fmt.Printf("内容类型: %s\n", metaidData.ContentType)
	fmt.Printf("数据: %s\n", string(metaidData.Data))
	fmt.Printf("数据长度: %d 字节\n", len(metaidData.Data))

	fmt.Println("\n\n=== 测试解析 Doginal 格式 ===")
	fmt.Println("\n【链上原始数据排序】")
	printRawScriptChunks(doginalTxRaw)

	doginalData, err := ParseInscriptionFromTx(doginalTxRaw, InscriptionFormatDoginal)
	if err != nil {
		t.Fatalf("解析Doginal交易失败: %v", err)
	}

	fmt.Printf("\n【解析结果】\n")
	fmt.Printf("格式: Doginal\n")
	fmt.Printf("数据块数量: %d\n", doginalData.PartsCount)
	fmt.Printf("索引: %d\n", doginalData.Index)
	fmt.Printf("内容类型: %s\n", doginalData.ContentType)
	fmt.Printf("数据: %s\n", string(doginalData.Data))
	fmt.Printf("数据长度: %d 字节\n", len(doginalData.Data))
}

// printRawScriptChunks 打印链上原始脚本chunks的顺序和内容
func printRawScriptChunks(txRaw string) {
	txBytes, err := hex.DecodeString(txRaw)
	if err != nil {
		fmt.Printf("解码交易失败: %v\n", err)
		return
	}

	var tx wire.MsgTx
	err = tx.Deserialize(bytes.NewReader(txBytes))
	if err != nil {
		fmt.Printf("反序列化交易失败: %v\n", err)
		return
	}

	if len(tx.TxIn) == 0 {
		fmt.Println("交易没有输入")
		return
	}

	sigScript := tx.TxIn[0].SignatureScript
	tokenizer := txscript.MakeScriptTokenizer(0, sigScript)
	idx := 0

	for tokenizer.Next() {
		data := tokenizer.Data()
		opcode := tokenizer.Opcode()

		if len(data) > 0 {
			// 数据chunk
			if len(data) <= 50 {
				// 短数据，直接显示
				if isPrintable(data) {
					fmt.Printf("  Chunk %d: [DATA] opcode=0x%02x len=%d data='%s'\n",
						idx, opcode, len(data), string(data))
				} else {
					fmt.Printf("  Chunk %d: [DATA] opcode=0x%02x len=%d data=%x\n",
						idx, opcode, len(data), data)
				}
			} else {
				// 长数据，显示前20字节
				if isPrintable(data[:20]) {
					fmt.Printf("  Chunk %d: [DATA] opcode=0x%02x len=%d data='%s...' (hex=%x...)\n",
						idx, opcode, len(data), string(data[:20]), data[:20])
				} else {
					fmt.Printf("  Chunk %d: [DATA] opcode=0x%02x len=%d data=%x...\n",
						idx, opcode, len(data), data[:20])
				}
			}
		} else {
			// OP code
			opcodeStr := getOpcodeString(opcode)
			fmt.Printf("  Chunk %d: [OP] opcode=0x%02x %s\n", idx, opcode, opcodeStr)
		}
		idx++
	}

	if err := tokenizer.Err(); err != nil {
		fmt.Printf("解析脚本失败: %v\n", err)
	}
}

// isPrintable 检查数据是否可打印
func isPrintable(data []byte) bool {
	for _, b := range data {
		if b < 32 || b > 126 {
			return false
		}
	}
	return true
}

// getOpcodeString 获取opcode的字符串表示
func getOpcodeString(opcode byte) string {
	switch opcode {
	case 0x00:
		return "OP_0"
	case 0x51:
		return "OP_1"
	case 0x52:
		return "OP_2"
	case 0x53:
		return "OP_3"
	case 0x54:
		return "OP_4"
	case 0x55:
		return "OP_5"
	case 0x56:
		return "OP_6"
	case 0x57:
		return "OP_7"
	case 0x58:
		return "OP_8"
	case 0x59:
		return "OP_9"
	case 0x5a:
		return "OP_10"
	case 0x5b:
		return "OP_11"
	case 0x5c:
		return "OP_12"
	case 0x5d:
		return "OP_13"
	case 0x5e:
		return "OP_14"
	case 0x5f:
		return "OP_15"
	case 0x60:
		return "OP_16"
	case 0x75:
		return "OP_DROP"
	case 0xad:
		return "OP_CHECKSIGVERIFY"
	default:
		if opcode >= 1 && opcode <= 75 {
			return fmt.Sprintf("OP_DATA_%d", opcode)
		}
		return fmt.Sprintf("OP_UNKNOWN(0x%02x)", opcode)
	}
}

func TestParseDogeTx(t *testing.T) {
	// 手动传入 raw 交易数据（十六进制字符串）
	// 可以从测试中的交易复制，或者从链上获取
	txRaw := "020000000239341b81ae32ace1da6c9da7662dad81ca9ead54d619e9a2919bd676234c2f3500000000fdeb01066d657461696406637265617465106170706c69636174696f6e2f6a736f6e013005312e302e304c596263317032306b33783263346d676c6678723577613573677467656368777374706c6438306b727532636734676d6d3475727675617171737661707875303a2f70726f746f636f6c732f73696d706c6567726f7570636861744cf07b2267726f75704944223a22653833366431623564303561353963313633623839646164363962313566653736636237363331346538316235356262623965363532366465663436323062366930222c2274696d657374616d70223a313736373533313637362c226e69636b4e616d65223a224552414e3230222c22636f6e74656e74223a223434393839323463383961616663333231366334626262623737333064643335222c22636f6e74656e7454797065223a22746578742f706c61696e222c22656e6372797074696f6e223a22616573222c227265706c7950696e223a22222c226d656e74696f6e223a5b5d017d473044022079068fb9bf4fe2d8fdbcb7422803d0ea3aab9c596cc58f7cb8b1323ed0906b9a0220582ceb01080ae62f08c3e2ec71cc2e2789433161d096bfb9cc5bb2ecfbe97acb012c2102e8622a195aba141cb15659d2f05f15d1c2d0f55bcfba6bfb3367786d32d72fe4ad757575757575757551ffffffff39341b81ae32ace1da6c9da7662dad81ca9ead54d619e9a2919bd676234c2f35010000006b48304502210088ff48057a72d055f216bd9e4e7047d3fda6ae4d9f25e29c6cabf3291db5273702201a82417e3db3ed82c32c749a23eb24c69ff901b8a0a5b5753f826104ad5c99c30121028645862c2ddc0224fe3b28bc861a5b383babbaf82573904ea1a92fbba751ef6affffffff02a0860100000000001976a914541a9019cb940c3bb519d41818b1b04ba4832bad88accf54e740000000001976a914541a9019cb940c3bb519d41818b1b04ba4832bad88ac00000000"

	netParam := DogeMainNetParams

	fmt.Println("=== 解析 Dogecoin 交易 ===")
	fmt.Printf("TxRaw: %s\n\n", txRaw)

	// 解析交易（如果需要计算费率，可以传入输入金额）
	// 示例：假设输入金额为 100000000 satoshis (1 DOGE)
	inputAmounts := []int64{1090000000}
	txInfo, err := ParseDogeTx(txRaw, netParam, inputAmounts)
	// txInfo, err := ParseDogeTx(txRaw, netParam)
	if err != nil {
		t.Fatalf("解析交易失败: %v", err)
	}

	// 打印交易基本信息
	fmt.Println("【交易基本信息】")
	fmt.Printf("交易哈希: %s\n", txInfo.TxHash)
	fmt.Printf("版本: %d\n", txInfo.Version)
	fmt.Printf("锁定时间: %d\n", txInfo.LockTime)
	fmt.Printf("交易大小: %d 字节\n", txInfo.Size)
	fmt.Printf("输入数量: %d\n", len(txInfo.Inputs))
	fmt.Printf("输出数量: %d\n", len(txInfo.Outputs))
	fmt.Printf("总输出金额: %d satoshis (%.8f DOGE)\n", txInfo.TotalOutput, float64(txInfo.TotalOutput)/1e8)
	if txInfo.TotalInput > 0 {
		fmt.Printf("总输入金额: %d satoshis (%.8f DOGE)\n", txInfo.TotalInput, float64(txInfo.TotalInput)/1e8)
		fmt.Printf("手续费: %d satoshis (%.8f DOGE)\n", txInfo.Fee, float64(txInfo.Fee)/1e8)
		fmt.Printf("费率: %d satoshis/byte (%.2f satoshis/byte)\n", txInfo.FeeRate, float64(txInfo.Fee)/float64(txInfo.Size))
	} else {
		fmt.Printf("提示: 如需计算费率，请传入输入金额参数\n")
	}
	fmt.Println()

	// 打印输入信息
	fmt.Println("【输入信息】")
	for i, input := range txInfo.Inputs {
		fmt.Printf("--- 输入 %d ---\n", i)
		fmt.Printf("  前序交易哈希: %s\n", input.PreviousTxHash)
		fmt.Printf("  前序输出索引: %d\n", input.PreviousIndex)
		fmt.Printf("  序列号: %d\n", input.Sequence)
		fmt.Printf("  脚本类型: %s\n", input.ScriptType)
		if input.Address != "" {
			fmt.Printf("  地址: %s\n", input.Address)
		}
		if input.SignatureScript != "" {
			fmt.Printf("  签名脚本长度: %d 字节\n", len(input.SignatureScript)/2)
			fmt.Printf("  签名脚本 (hex): %s\n", input.SignatureScript)

			// 解析并打印签名脚本的 chunks
			fmt.Printf("  【签名脚本解析】\n")
			printScriptChunks(input.SignatureScript)
		}
		if len(input.Witness) > 0 {
			fmt.Printf("  Witness 数量: %d\n", len(input.Witness))
			for j, w := range input.Witness {
				if len(w) <= 50 {
					fmt.Printf("    Witness[%d]: %x\n", j, w)
				} else {
					fmt.Printf("    Witness[%d]: %x... (长度: %d)\n", j, w[:20], len(w))
				}
			}
		}
		fmt.Println()
	}

	// 打印输出信息
	fmt.Println("【输出信息】")
	for i, output := range txInfo.Outputs {
		fmt.Printf("--- 输出 %d ---\n", i)
		fmt.Printf("  金额: %d satoshis (%.8f DOGE)\n", output.Value, float64(output.Value)/1e8)
		fmt.Printf("  脚本类型: %s\n", output.ScriptType)
		if output.Address != "" {
			fmt.Printf("  地址: %s\n", output.Address)
		}
		fmt.Printf("  脚本 (hex): %s\n", output.Script)

		// 解析并打印输出脚本的 chunks
		fmt.Printf("  【输出脚本解析】\n")
		printScriptChunks(output.Script)
		fmt.Println()
	}
}

// printScriptChunks 打印脚本的 chunks（辅助函数）
func printScriptChunks(scriptHex string) {
	scriptBytes, err := hex.DecodeString(scriptHex)
	if err != nil {
		fmt.Printf("    解码脚本失败: %v\n", err)
		return
	}

	tokenizer := txscript.MakeScriptTokenizer(0, scriptBytes)
	idx := 0

	for tokenizer.Next() {
		data := tokenizer.Data()
		opcode := tokenizer.Opcode()

		if len(data) > 0 {
			// 数据chunk
			if len(data) <= 50 {
				// 短数据，直接显示
				if isPrintable(data) {
					fmt.Printf("    Chunk %d: [DATA] opcode=0x%02x len=%d data='%s'\n",
						idx, opcode, len(data), string(data))
				} else {
					fmt.Printf("    Chunk %d: [DATA] opcode=0x%02x len=%d data=%x\n",
						idx, opcode, len(data), data)
				}
			} else {
				// 长数据，显示前20字节
				if isPrintable(data[:20]) {
					fmt.Printf("    Chunk %d: [DATA] opcode=0x%02x len=%d data='%s...' (hex=%x...)\n",
						idx, opcode, len(data), string(data[:20]), data[:20])
				} else {
					fmt.Printf("    Chunk %d: [DATA] opcode=0x%02x len=%d data=%x...\n",
						idx, opcode, len(data), data[:20])
				}
			}
		} else {
			// OP code
			opcodeStr := getOpcodeString(opcode)
			fmt.Printf("    Chunk %d: [OP] opcode=0x%02x %s\n", idx, opcode, opcodeStr)
		}
		idx++
	}

	if err := tokenizer.Err(); err != nil {
		fmt.Printf("    解析脚本失败: %v\n", err)
	}
}
