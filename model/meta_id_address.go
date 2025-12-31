package model

// MetaIdAddress MetaID 和 Address 的映射关系
type MetaIdAddress struct {
	MetaId  string `json:"metaId"`  // MetaID (SHA256 of address)
	Address string `json:"address"` // 区块链地址
}

type GlobalMetaIdAddress struct {
	GlobalMetaId string              `json:"globalMetaId"` // Global MetaID
	Items        []MetaIdAddressItem `json:"items"`        // MetaID 和 Address 的映射关系
}

type MetaIdAddressItem struct {
	MetaId  string `json:"metaId"`  // MetaID (SHA256 of address)
	Address string `json:"address"` // 区块链地址
}
