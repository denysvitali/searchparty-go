package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/sirupsen/logrus"

	"github.com/denysvitali/searchparty-go"
	"github.com/denysvitali/searchparty-go/model"
)

var logger = logrus.StandardLogger()

var args struct {
	AnisetteURL         string `arg:"--anisette-url,-A" default:"http://localhost:6969" help:"Anisette URL"`
	BeaconStorePassword string `arg:"--beacon-store-password,env:BEACON_STORE_PASSWORD,required" help:"Beacon store password"`
}

func main() {
	arg.MustParse(&args)

	auth, err := searchparty.GetAuth("auth.json")
	if err != nil {
		logger.Fatalf("failed to get auth: %v", err)
	}
	c := searchparty.New(auth, args.AnisetteURL)

	cwd, err := os.Getwd()
	if err != nil {
		logger.Fatalf("failed to get current working directory: %v", err)
	}

	beaconStorePwdBytes, err := hex.DecodeString(args.BeaconStorePassword)
	if err != nil {
		logger.Fatalf("failed to decode beacon store password: %v", err)
	}

	keys, err := searchparty.LoadKeys(cwd, beaconStorePwdBytes)
	if err != nil {
		logger.Fatalf("failed to load keys: %v", err)
	}

	keyMap := map[string]model.MainKey{}
	for _, k := range keys {
		keyMap[k.ID()] = k
	}

	lostAt := time.Now()

	ctx := context.Background()
	reports, subKeysMap, err := c.Find(ctx, keys, 2, lostAt)
	if err != nil {
		logger.Errorf("failed to find reports: %v", err)
	}

	tmpFile, err := os.CreateTemp(os.TempDir(), "*.json")
	if err != nil {
		logger.Fatalf("unable to create temporary file: %v", err)
	}

	if err := json.NewEncoder(tmpFile).Encode(reports); err != nil {
		logger.Fatalf("unable to encode reports: %v", err)
	}
	logger.Tracef("stored reports to %s", tmpFile.Name())

	for _, r := range reports {
		keyId := r.ID[0:7]
		key, ok := subKeysMap[r.ID]
		if !ok {
			logger.Fatalf("unable to find key for report %s", keyId)
		}
		tagData, err := searchparty.DecodeReport(r, key)
		if err != nil {
			logger.Errorf("unable to decode report: %v", err)
		} else {
			jsonText, err := json.Marshal(map[string]any{
				"tagData": tagData,
				"report":  r,
			})
			if err != nil {
				logger.Errorf("unable to encode JSON: %v", err)
				continue
			}
			fmt.Println(string(jsonText))
		}
	}
}
