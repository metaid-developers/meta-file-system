package indexer

// LazyBlock represents a block that is not fully loaded; transactions are fetched by txid on demand.
// Used for large blocks to avoid loading the entire block into memory.
type LazyBlock struct {
	TxIDs     []string // Transaction IDs in block order
	Timestamp int64    // Block timestamp in milliseconds
	Blockhash string   // Block hash
}

// BlockEvent represents a block scanning event with all necessary data
type BlockEvent struct {
	ChainName string      // Chain name: btc, mvc, etc.
	Height    int64       // Block height
	Timestamp int64       // Block timestamp in milliseconds
	Block     interface{} // *wire.MsgBlock (MVC), *btcwire.MsgBlock (BTC/DOGE), or *LazyBlock for large blocks
	TxCount   int         // Number of transactions in the block

	// TxFetcher is set when Block is *LazyBlock; used to fetch and deserialize a tx by id.
	// Handler should call TxFetcher(txid) for each txid in LazyBlock.TxIDs.
	TxFetcher func(txid string) (interface{}, error)
}

// BlockEventWithTx represents a block event with parsed MetaID transactions
type BlockEventWithTx struct {
	Event     *BlockEvent
	MetaIDTxs []*MetaIDDataTx // Parsed MetaID transactions
}
