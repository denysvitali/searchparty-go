package searchparty

import (
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"github.com/denysvitali/searchparty-go/model"
)

type StaticKey struct {
	privateKey   []byte
	advKey       []byte
	hashedAdvKey []byte
	keyID        string
}

func (s *StaticKey) KeyInfo() model.KeyInfo {
	return model.KeyInfo{}
}

func (s *StaticKey) Type() string {
	return "static"
}

func (s *StaticKey) ID() string {
	return s.keyID
}

func (s *StaticKey) GetSubKeys(from time.Time, to time.Time, lostAt time.Time) ([]model.SubKey, error) {
	return []model.SubKey{
		{
			MainKey:      s,
			AdvKey:       s.advKey,
			HashedAdvKey: s.hashedAdvKey,
			PrivateKey:   s.privateKey,
			Type:         model.Secondary, // It really doesn't matter
		},
	}, nil
}

var _ model.MainKey = &StaticKey{}

func LoadStaticKey(reader io.ReadCloser) (model.MainKey, error) {
	var s StaticKey
	defer reader.Close()
	/*
		File format:
		Private key: BASE64_ENCODED_STRING
		Advertisement key: BASE64_ENCODED_STRING
		Hashed adv key: BASE64_ENCODED_STRING
	*/
	pKey, advKey, hAdvKey := "", "", ""
	_, err := fmt.Fscanf(reader,
		"Private key: %s\nAdvertisement key: %s\nHashed adv key: %s\n",
		&pKey,
		&advKey,
		&hAdvKey,
	)
	if err != nil {
		return &s, err
	}
	s.privateKey, err = base64.StdEncoding.DecodeString(pKey)
	if err != nil {
		return &s, err
	}
	s.advKey, err = base64.StdEncoding.DecodeString(advKey)
	if err != nil {
		return &s, err
	}
	s.hashedAdvKey, err = base64.StdEncoding.DecodeString(hAdvKey)
	if err != nil {
		return &s, err
	}
	s.keyID = hAdvKey[0:7]
	return &s, nil
}
