package server

import (
	"sort"

	"github.com/denysvitali/searchparty-go"
)

type byTime []searchparty.TagData

func (b byTime) Len() int {
	return len(b)
}

func (b byTime) Less(i, j int) bool {
	return b[i].Time.Before(b[j].Time)
}

func (b byTime) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

var _ sort.Interface = byTime{}
