package model

// IndexerUserInfo 用户信息模型
type IndexerUserInfo struct {
	MetaId             string `json:"metaId"`             // 用户 MetaID
	Address            string `json:"address"`            // 用户地址
	Name               string `json:"name"`               // 用户名称
	NamePinId          string `json:"namePinId"`          // 用户名称 PIN ID
	Avatar             string `json:"avatar"`             // 头像路径
	AvatarPinId        string `json:"avatarPinId"`        // 头像 PIN ID
	ChatPublicKey      string `json:"chatPublicKey"`      // 聊天公钥
	ChatPublicKeyPinId string `json:"chatPublicKeyPinId"` // 聊天公钥 PIN ID
	ChainName          string `json:"chainName"`          // 链名称
	BlockHeight        int64  `json:"blockHeight"`        // 区块高度
	Timestamp          int64  `json:"timestamp"`          // 时间戳
}

// UserNameInfo 用户名称信息
type UserNameInfo struct {
	Name        string `json:"name"`        // 用户名称
	FirstPinID  string `json:"firstPinId"`  // 第一个 PIN ID
	FirstPath   string `json:"firstPath"`   // 第一个 PIN 的路径
	PinID       string `json:"pinId"`       // PIN ID
	ChainName   string `json:"chainName"`   // 链名称
	BlockHeight int64  `json:"blockHeight"` // 区块高度
	Timestamp   int64  `json:"timestamp"`   // 时间戳
}

// UserAvatarInfo 用户头像信息
type UserAvatarInfo struct {
	Avatar      string `json:"avatar"`      // 头像路径
	FirstPinID  string `json:"firstPinId"`  // 第一个 PIN ID
	FirstPath   string `json:"firstPath"`   // 第一个 PIN 的路径
	PinID       string `json:"pinId"`       // PIN ID
	ChainName   string `json:"chainName"`   // 链名称
	BlockHeight int64  `json:"blockHeight"` // 区块高度
	Timestamp   int64  `json:"timestamp"`   // 时间戳

	AvatarUrl     string `json:"avatarUrl"`     // 头像 URL
	ContentType   string `json:"contentType"`   // Content type (e.g., image/jpeg)
	FileSize      int64  `json:"fileSize"`      // File size (bytes)
	FileMd5       string `json:"fileMd5"`       // File MD5 hash
	FileHash      string `json:"fileHash"`      // File Hash SHA256
	FileExtension string `json:"fileExtension"` // File extension, e.g. .jpg, .png, .mp4, .mp3, .doc, .pdf, etc.
	FileType      string `json:"fileType"`      // File type (image/video/audio/document/other)
}

// UserChatPublicKeyInfo 用户聊天公钥信息
type UserChatPublicKeyInfo struct {
	ChatPublicKey string `json:"chatPublicKey"` // 聊天公钥
	FirstPinID    string `json:"firstPinId"`    // 第一个 PIN ID
	FirstPath     string `json:"firstPath"`     // 第一个 PIN 的路径
	PinID         string `json:"pinId"`         // PIN ID
	ChainName     string `json:"chainName"`     // 链名称
	BlockHeight   int64  `json:"blockHeight"`   // 区块高度
	Timestamp     int64  `json:"timestamp"`     // 时间戳
}
