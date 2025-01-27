package searchparty

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"

	"github.com/denysvitali/searchparty-go/model"
)

var logger = logrus.StandardLogger().WithField("pkg", "searchparty")

const fetchReportsUrl = "https://gateway.icloud.com/acsnservice/fetch"

type Client struct {
	auth        *Auth
	anisetteUrl string
}

type searchParams struct {
	StartDate int64    `json:"startDate"`
	EndDate   int64    `json:"endDate"`
	Ids       []string `json:"ids"` // A list of Hashed Advertisements (base64)
}

type FindRequest struct {
	Search []searchParams `json:"search"`
}

type Report struct {
	ID            string `json:"id"`
	DatePublished int64  `json:"datePublished"`
	Payload       string `json:"payload"`
	Description   string `json:"description"`
	StatusCode    int    `json:"statusCode"`
}

func (c Client) Find(ctx context.Context, keys []model.MainKey, hours int, lostAt time.Time) ([]Report, map[string]model.SubKey, error) {
	h, err := getAnisetteHeaders(c.anisetteUrl)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get anisette headers: %w", err)
	}

	now := time.Now()
	startTime := now.Add(-time.Duration(hours) * time.Hour)
	endTime := now

	start := startTime.Unix()
	end := endTime.Unix()

	subKeysMap := make(map[string]model.SubKey)
	for _, k := range keys {
		subKeys, err := k.GetSubKeys(startTime, endTime, lostAt)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to get subkeys: %w", err)
		}
		for _, v := range subKeys {
			subKeysMap[base64.StdEncoding.EncodeToString(v.HashedAdvKey)] = v
		}
	}

	keyIDs := maps.Keys(subKeysMap)
	jsonBytes, err := json.Marshal(FindRequest{
		Search: []searchParams{{
			StartDate: start * 1000,
			EndDate:   end * 1000,
			Ids:       keyIDs,
		}},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("unable to marshal find request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fetchReportsUrl, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create request: %w", err)
	}
	req.Header = h
	req.SetBasicAuth(c.auth.Dsid, c.auth.SearchPartyToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to make request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}
	var result FindResult
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, nil, fmt.Errorf("unable to decode JSON: %w", err)
	}
	return result.Results, subKeysMap, nil
}

func New(auth *Auth, anisetteURL string) *Client {
	return &Client{auth: auth, anisetteUrl: anisetteURL}
}
