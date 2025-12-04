package indexer

// BlockEvent represents a block scanning event with all necessary data
type BlockEvent struct {
	ChainName string      // Chain name: btc, mvc, etc.
	Height    int64       // Block height
	Timestamp int64       // Block timestamp in milliseconds
	Block     interface{} // *wire.MsgBlock (MVC) or *btcwire.MsgBlock (BTC)
	TxCount   int         // Number of transactions in the block
}

// BlockEventWithTx represents a block event with parsed MetaID transactions
type BlockEventWithTx struct {
	Event     *BlockEvent
	MetaIDTxs []*MetaIDDataTx // Parsed MetaID transactions
}
