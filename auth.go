package searchparty

import (
	"encoding/json"
	"os"
)

type Auth struct {
	Dsid             string `json:"dsid"`
	SearchPartyToken string `json:"searchPartyToken"`
}

func GetAuth(authFile string) (*Auth, error) {
	f, err := os.Open(authFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var auth Auth
	err = json.NewDecoder(f).Decode(&auth)
	if err != nil {
		return nil, err
	}
	return &auth, nil
}
