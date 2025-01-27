package models

type KeyAlias struct {
	KeyID string `json:"key_id" gorm:"primaryKey"`
	Alias string `json:"alias"`
	Type  string `json:"type"`
}
