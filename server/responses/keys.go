package responses

import (
	"sort"

	"github.com/denysvitali/searchparty-go/model"
	"github.com/denysvitali/searchparty-go/server/models"
)

type Key struct {
	ID           string                 `json:"key_id"`
	Alias        string                 `json:"alias"`
	Type         string                 `json:"type"`
	KeyInfo      model.KeyInfo          `json:"key_info"`
	LastLocation *models.LocationResult `json:"last_location,omitempty"`
}

type ByKeyID []Key

func (b ByKeyID) Len() int {
	return len(b)
}

func (b ByKeyID) Less(i, j int) bool {
	return b[i].ID < b[j].ID
}

func (b ByKeyID) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

var _ sort.Interface = ByKeyID{}
