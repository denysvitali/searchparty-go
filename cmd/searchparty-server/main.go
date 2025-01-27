package main

import (
	"encoding/hex"

	"github.com/alexflint/go-arg"
	"github.com/sirupsen/logrus"

	"github.com/denysvitali/searchparty-go"
	"github.com/denysvitali/searchparty-go/service"
)

var args struct {
	AnisetteURL         string `arg:"--anisette-url,-A" default:"http://localhost:6969" help:"Anisette URL"`
	ListenAddr          string `arg:"--listen-addr,-l" default:"127.0.0.1:8500" help:"Listen address"`
	BeaconStorePassword string `arg:"--beacon-store-password,env:BEACON_STORE_PASSWORD,required" help:"Beacon store password (in hex)"`
	Dsn                 string `arg:"--dsn" default:"host=localhost port=5438 user=searchparty password=searchparty dbname=searchparty sslmode=disable binary_parameters=yes" help:"DSN for the database"`
	LogLevel            string `arg:"--log-level" default:"info" help:"Log level"`
}
var logger = logrus.StandardLogger()

func main() {
	arg.MustParse(&args)
	setLogLevel(args.LogLevel)

	auth, err := searchparty.GetAuth("auth.json")
	if err != nil {
		logger.Fatalf("failed to get auth: %v", err)
	}
	beaconStorePwdBytes, err := hex.DecodeString(args.BeaconStorePassword)

	if err != nil {
		logger.Fatalf("failed to decode beacon store password: %v", err)
	}

	s, err := service.New(auth, args.AnisetteURL, args.Dsn, beaconStorePwdBytes)
	if err != nil {
		logger.Fatalf("failed to create server: %v", err)
	}
	logger.Infof("Listening on %s", args.ListenAddr)
	if err := s.Start("127.0.0.1:8084", args.ListenAddr); err != nil {
		logger.Fatalf("start server: %v", err)
	}
}

func setLogLevel(level string) {
	l, err := logrus.ParseLevel(level)
	if err != nil {
		logger.Fatalf("failed to parse log level: %v", err)
	}
	logger.SetLevel(l)
}
