package models

import "time"

type KeyInfo struct {
	ID     string     `gorm:"primaryKey" json:"id"`
	Alias  *KeyAlias  `gorm:"foreignKey:KeyID;references:ID" json:"alias"`
	LostAt *time.Time `json:"lostAt"`
}
