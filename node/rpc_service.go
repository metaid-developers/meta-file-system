package node

func CurrentBlockHeight(chain string) (uint64, error) {
	client := NewClientController(chain)
	return client.GetBlockHeight(chain)
}

func GetTxRaw(chain, txId string) (string, error) {
	client := NewClientController(chain)
	return client.GetTransactionHex(chain, txId)
}

func GetTxDetail(chain, txId string) (*Transaction, error) {
	client := NewClientController(chain)
	return client.GetTransaction(chain, txId)
}

func BroadcastTx(chain, txHex string) (string, error) {
	client := NewClientController(chain)
	txId, err := client.BroadcastTx(chain, txHex)
	return txId, err
}

func BroadcastTxBatch(chain string, txHexStrs ...string) (*SendRawTransactionsResult, error) {
	client := NewClientController(chain)
	return client.BroadcastTxBatch(chain, txHexStrs...)
}

func BroadcastTxBatchWithOptions(chain string, options ...TxOption) (*SendRawTransactionsResult, error) {
	client := NewClientController(chain)
	return client.BroadcastTxBatchWithOptions(chain, options...)
}

func GetMempool(chain string) ([]string, error) {
	client := NewClientController(chain)
	return client.GetMempool(chain)
}
