package model

import "time"

// Assistant assistant model
type Assistant struct {
	ID                  int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	MetaId              string    `gorm:"uniqueIndex;type:varchar(64);not null" json:"meta_id"`          // MetaID
	Address             string    `gorm:"index;type:varchar(255);not null" json:"address"`               // Mnemonic address
	AssistantPrivateKey string    `gorm:"index;type:varchar(255);not null" json:"assistant_private_key"` // Mnemonic private key
	AssistantAddress    string    `gorm:"index;type:varchar(255);not null" json:"assistant_address"`     // Mnemonic address
	AssistantMetaId     string    `gorm:"index;type:varchar(255);not null" json:"assistant_meta_id"`     // Mnemonic MetaID
	CreatedAt           time.Time `gorm:"autoCreateTime" json:"created_at"`                              // Creation time
	UpdatedAt           time.Time `gorm:"autoUpdateTime" json:"updated_at"`                              // Update time
	State               int64     `gorm:"type:int(11)" json:"state"`                                     // State 0:EXIST,1:DELETED
}

// TableName specify table name
func (Assistant) TableName() string {
	return "tb_assistant"
}
