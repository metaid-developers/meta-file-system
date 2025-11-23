package model

import "time"

// FileAssistent 文件托管助手模型（用于存储托管地址私钥，帮助用户异步上链分片）
type FileAssistent struct {
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// 用户相关字段
	MetaId  string `gorm:"index;type:varchar(255);not null" json:"meta_id"` // 用户 MetaID
	Address string `gorm:"index;type:varchar(100);not null" json:"address"` // 用户地址

	// 托管地址相关字段
	AssistentAddress string `gorm:"index;type:varchar(100);not null" json:"assistent_address"` // 托管地址
	AssistentPriHex  string `gorm:"type:text;not null" json:"assistent_pri_hex"`               // 托管地址私钥（hex格式）

	// 状态字段
	Status Status `gorm:"type:varchar(20);default:'success'" json:"status"` // success/failed

	// 时间戳
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"` // 创建时间
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"` // 更新时间
}

// TableName specify table name
func (FileAssistent) TableName() string {
	return "tb_file_assistent"
}
