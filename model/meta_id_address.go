package model

// MetaIdAddress MetaID 和 Address 的映射关系
type MetaIdAddress struct {
	MetaId  string `json:"metaId"`  // MetaID (SHA256 of address)
	Address string `json:"address"` // 区块链地址
}
