package model

import "time"

type MainKey interface {
	ID() string
	GetSubKeys(from time.Time, to time.Time, lostAt time.Time) ([]SubKey, error)
	KeyInfo() KeyInfo
	Type() string
}

type KeyInfo struct {
	Model            string    `json:"model"`
	PairingDate      time.Time `json:"pairingDate"`
	Identifier       string    `json:"identifier"`
	StableIdentifier string    `json:"stableIdentifier"`
}

type SubKey struct {
	MainKey      MainKey
	AdvKey       []byte
	HashedAdvKey []byte
	PrivateKey   []byte
	Type         SubKeyType
}

type SubKeyType int

const (
	Primary SubKeyType = iota
	Secondary
)
