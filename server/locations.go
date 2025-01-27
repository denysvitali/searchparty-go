package server

import (
	"github.com/denysvitali/searchparty-go"
)

type Location struct {
	PublishedAt string              `json:"publishedAt"`
	TagData     searchparty.TagData `json:"tagData"`
}
