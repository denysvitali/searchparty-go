package searchparty

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// TODO: Use a generic test here - possibly with real data
func TestDecodeReport(t *testing.T) {
	logrus.SetFormatter(&logrus.TextFormatter{ForceColors: true})
	var result []Report
	f, err := os.Open("./testdata/reports.json")
	if err != nil {
		t.Fatalf("failed to open testdata/results.json: %v", err)
	}
	defer f.Close()
	err = json.NewDecoder(f).Decode(&result)
	if err != nil {
		t.Fatalf("failed to decode testdata/results.json: %v", err)
	}
	keyFile, err := os.Open("./example.keys")
	if err != nil {
		t.Fatalf("failed to open key file: %v", err)
	}
	defer keyFile.Close()
	k, err := LoadStaticKey(keyFile)
	if err != nil {
		t.Fatalf("LoadKey failed: %v", err)
	}

	var failed [][]byte
	var success []*TagData

	for _, r := range result {
		rBytes, err := base64.StdEncoding.DecodeString(r.Payload)
		if err != nil {
			t.Fatalf("unable to decode payload: %v", err)
		}
		timeZero := time.Unix(0, 0)
		subKeys, err := k.GetSubKeys(timeZero, timeZero, timeZero)
		if err != nil {
			t.Fatalf("GetSubKeys failed: %v", err)
		}
		if len(subKeys) != 1 {
			t.Fatalf("expected 1 subkey, got %d", len(subKeys))
		}
		tData, err := DecodeReport(r, subKeys[0])
		if err != nil {
			failed = append(failed, rBytes)
			t.Errorf("DecodeReport failed: %v", err)
		} else {
			success = append(success, tData)
		}
	}

	fmt.Printf("Failed:\n")
	for _, f := range failed {
		fmt.Printf("%s\n", hex.EncodeToString(f[0:16]))
	}
	fmt.Printf("Success:\n")
	for _, s := range success {
		fmt.Printf("%s\n", s)
	}
}
