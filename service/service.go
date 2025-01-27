package service

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/denysvitali/searchparty-go"
	gw "github.com/denysvitali/searchparty-go/gen/proto"
	"github.com/denysvitali/searchparty-go/model"
)

var log = logrus.StandardLogger().WithField("pkg", "service")

type Service struct {
	anisetteURL    string
	auth           *searchparty.Auth
	beaconStoreKey []byte
	db             *gorm.DB
	keyMap         map[string]model.MainKey

	gw.UnimplementedSearchPartyServer
}

func (s *Service) GetDevices(ctx context.Context, request *gw.GetDevicesRequest) (*gw.GetDevicesResponse, error) {
	return &gw.GetDevicesResponse{
		Devices: toDevices(s.keyMap),
	}, nil
}

func toDevices(keyMap map[string]model.MainKey) []*gw.Device {
	devices := make([]*gw.Device, 0, len(keyMap))
	for _, key := range keyMap {
		devices = append(devices, &gw.Device{
			Id:               key.ID(),
			Name:             "???",
			Model:            key.KeyInfo().Model,
			PairingDate:      timestamppb.New(key.KeyInfo().PairingDate),
			Identifier:       key.KeyInfo().Identifier,
			StableIdentifier: key.KeyInfo().StableIdentifier,
		})
	}
	return devices
}

func (s *Service) GetDeviceLocation(ctx context.Context, request *gw.GetDeviceLocationRequest) (*gw.GetDeviceLocationResponse, error) {
	//TODO implement me
	panic("implement me")
}

var _ gw.SearchPartyServer = (*Service)(nil)

func New(auth *searchparty.Auth, anisetteURL string, dsn string, beaconStoreKey []byte) (*Service, error) {
	db, err := gorm.Open(
		postgres.New(postgres.Config{
			DSN:        dsn,
			DriverName: "postgres",
		}),
		&gorm.Config{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	s := Service{
		db:             db,
		auth:           auth,
		anisetteURL:    anisetteURL,
		beaconStoreKey: beaconStoreKey,
		keyMap:         map[string]model.MainKey{},
	}
	err = s.init()
	return &s, err
}

func (s *Service) init() error {
	keys, err := searchparty.LoadKeys("./beacons/", s.beaconStoreKey)
	if err != nil {
		return fmt.Errorf("failed to load keys: %w", err)
	}
	for _, k := range keys {
		s.keyMap[k.ID()] = k
	}
	return nil
}

func (s *Service) Start(grpcListenAddr string, httpListenAddr string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mux := runtime.NewServeMux()
	grpcServer := grpc.NewServer()
	grpcServer.RegisterService(&gw.SearchParty_ServiceDesc, s)
	listener, err := net.Listen("tcp", grpcListenAddr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	go func() {
		err = grpcServer.Serve(listener)
		if err != nil {
			log.Fatalf("failed to serve grpc: %v", err)
		}
	}()

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	if err := gw.RegisterSearchPartyHandlerFromEndpoint(ctx, mux, grpcListenAddr, opts); err != nil {
		return fmt.Errorf("register gateway: %w", err)
	}
	httpServer := &http.Server{
		Addr:              httpListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
	}
	return httpServer.ListenAndServe()
}
