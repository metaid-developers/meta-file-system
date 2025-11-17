package node

import (
	"errors"
	"fmt"

	"meta-file-system/conf"

	"github.com/tidwall/gjson"
)

type ClientController struct {
	ClientMap map[string]*Client
}

var (
	RPC_url      string
	RPC_username string
	RPC_password string
)

var (
	MyClientController *ClientController
)

func getChainRpcParams(chain string) (string, string, string) {
	return conf.RpcConfigMap[chain].Url, conf.RpcConfigMap[chain].Username, conf.RpcConfigMap[chain].Password
}

func NewClientController(chain string) *ClientController {
	if MyClientController != nil {
		if _, ok := MyClientController.ClientMap[chain]; ok {
			return MyClientController
		}
	} else {
		MyClientController = &ClientController{
			ClientMap: make(map[string]*Client),
		}
	}

	RPC_url, RPC_username, RPC_password = getChainRpcParams(chain)

	fmt.Println("*******RPC_url : [ ", RPC_url, " ]")

	accessToken := BasicAuth(RPC_username, RPC_password)
	MyClientController.ClientMap[chain] = NewClientNode(RPC_url, accessToken, false)
	fmt.Println("****** Build new Client completed ******")

	return MyClientController
}

func (c *ClientController) BroadcastTx(net, txHexStr string) (string, error) {
	request := []interface{}{
		txHexStr,
		false,
	}

	result, err := c.ClientMap[net].Call("sendrawtransaction", request)
	if err != nil {
		return "", err
	}
	return result.String(), nil
}

// BroadcastTxBatch batch broadcast transactions
// supports single transaction or transaction array
func (c *ClientController) BroadcastTxBatch(net string, txHexStrs ...string) (*SendRawTransactionsResult, error) {
	// build transaction object array
	txObjects := make([]map[string]interface{}, 0, len(txHexStrs))
	for _, txHex := range txHexStrs {
		txObj := map[string]interface{}{
			"hex": txHex,
		}
		txObjects = append(txObjects, txObj)
	}

	request := []interface{}{
		txObjects,
	}

	result, err := c.ClientMap[net].Call("sendrawtransactions", request)
	if err != nil {
		return nil, err
	}

	return NewSendRawTransactionsResult(result), nil
}

// BroadcastTxBatchWithOptions batch broadcast transactions，supports more options
func (c *ClientController) BroadcastTxBatchWithOptions(net string, options ...TxOption) (*SendRawTransactionsResult, error) {
	// build transaction object array
	txObjects := make([]map[string]interface{}, 0, len(options))
	for _, option := range options {
		txObj := map[string]interface{}{
			"hex": option.Hex,
		}

		if option.AllowHighFees {
			txObj["allowhighfees"] = true
		}
		if option.DontCheckFee {
			txObj["dontcheckfee"] = true
		}
		if option.ListUnconfirmedAncestors {
			txObj["listunconfirmedancestors"] = true
		}
		if option.Config != "" {
			txObj["config"] = option.Config
		}

		txObjects = append(txObjects, txObj)
	}

	request := []interface{}{
		txObjects,
	}

	result, err := c.ClientMap[net].Call("sendrawtransactions", request)
	if err != nil {
		return nil, err
	}

	return NewSendRawTransactionsResult(result), nil
}

func (c *ClientController) GetBlockhash(net string, height uint64) (string, error) {

	request := []interface{}{
		height,
	}

	result, err := c.ClientMap[net].Call("getblockhash", request)
	if err != nil {
		return "", err
	}

	return result.String(), nil
}

func (c *ClientController) GetBlockHeight(net string) (uint64, error) {

	result, err := c.ClientMap[net].Call("getblockcount", nil)
	if err != nil {
		return 0, err
	}

	return result.Uint(), nil
}

func (c *ClientController) GetBlock(net string, hash string, format ...uint64) (*Block, error) {

	request := []interface{}{
		hash,
	}

	if len(format) > 0 {
		request = append(request, format[0])
	}

	result, err := c.ClientMap[net].Call("getblock", request)
	if err != nil {
		return nil, err
	}

	return NewBlock(result), nil
}

func (c *ClientController) GetTxIDsInMemPool(net string) ([]string, error) {

	var (
		txids = make([]string, 0)
	)

	result, err := c.ClientMap[net].Call("getrawmempool", nil)
	if err != nil {
		return nil, err
	}

	if !result.IsArray() {
		return nil, errors.New("no query record")
	}

	for _, txid := range result.Array() {
		txids = append(txids, txid.String())
	}

	return txids, nil
}

func (c *ClientController) GetTransaction(net string, txid string) (*Transaction, error) {

	var (
		result *gjson.Result
		err    error
	)

	request := []interface{}{
		txid,
		true,
	}

	result, err = c.ClientMap[net].Call("getrawtransaction", request)
	if err != nil {

		request = []interface{}{
			txid,
			1,
		}

		result, err = c.ClientMap[net].Call("getrawtransaction", request)
		if err != nil {
			return nil, err
		}
	}
	//fmt.Printf("result:%s\n", result.String())
	return newTxByCore(result), nil
}

func (c *ClientController) GetTransactionHex(net string, txid string) (string, error) {

	var (
		result *gjson.Result
		err    error
	)

	request := []interface{}{
		txid,
		false,
	}

	result, err = c.ClientMap[net].Call("getrawtransaction", request)
	if err != nil {

		request = []interface{}{
			txid,
			0,
		}

		result, err = c.ClientMap[net].Call("getrawtransaction", request)
		if err != nil {
			return "", err
		}
	}

	return result.String(), nil
}

func (c *ClientController) GetMempool(net string) ([]string, error) {
	var (
		txIds = make([]string, 0)
	)

	result, err := c.ClientMap[net].Call("getrawmempool", nil)
	if err != nil {
		return nil, err
	}

	if !result.IsArray() {
		return nil, errors.New("no query record")
	}

	for _, txid := range result.Array() {
		txIds = append(txIds, txid.String())
	}

	return txIds, nil
}
