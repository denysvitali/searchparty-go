package server

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/twpayne/go-geom"
	"golang.org/x/exp/maps"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/denysvitali/searchparty-go"
	"github.com/denysvitali/searchparty-go/model"
	"github.com/denysvitali/searchparty-go/server/models"
	"github.com/denysvitali/searchparty-go/server/responses"
)

var logger = logrus.StandardLogger().WithField("pkg", "server")

type Server struct {
	dsn            string
	db             *gorm.DB
	e              *gin.Engine
	c              *searchparty.Client
	keyMap         map[string]model.MainKey
	beaconStoreKey []byte
}

func New(auth *searchparty.Auth, anisetteURL string, dsn string, beaconStoreKey []byte) (*Server, error) {
	s := Server{
		dsn:            dsn,
		e:              gin.New(),
		c:              searchparty.New(auth, anisetteURL),
		keyMap:         map[string]model.MainKey{},
		beaconStoreKey: beaconStoreKey,
	}

	if err := s.loadKeys("./beacons/"); err != nil {
		return nil, fmt.Errorf("failed to load keys: %w", err)
	}
	if err := s.init(); err != nil {
		return nil, fmt.Errorf("unable to init server: %w", err)
	}
	return &s, nil
}

func (s *Server) Listen(addr ...string) error {
	logger.Infof("listening on %s", addr)
	return s.e.Run(addr...)
}

func (s *Server) init() error {
	var errArr []error
	errArr = append(errArr, s.initDB())
	s.e.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
	}))
	v1 := s.e.Group("/api/v1")
	v1.GET("/keys", s.getKeys)
	v1.GET("/keys/:keyId", s.getLastLocation)
	v1.GET("/keys/:keyId/refresh", s.refreshLocation)
	v1.GET("/keys/:keyId/history", s.getLocationHistory)
	return errors.Join(errArr...)
}

func (s *Server) initDB() error {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:        s.dsn,
		DriverName: "postgres",
	}), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	m := []any{
		&models.Location{},
		&models.KeyAlias{},
		&models.KeyInfo{},
	}
	for _, m := range m {
		if err := db.AutoMigrate(m); err != nil {
			return fmt.Errorf("failed to migrate model: %w", err)
		}
	}
	s.db = db
	return nil
}

func (s *Server) getKeys(c *gin.Context) {
	var keyAliases []models.KeyAlias
	keys := maps.Keys(s.keyMap)
	tx := s.db.Model(&models.KeyAlias{}).Where("key_id IN ?", keys).Find(&keyAliases)
	if tx.Error != nil {
		logger.Errorf("unable to fetch key aliases: %v", tx.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to fetch key aliases"})
		return
	}

	var keyAliasesMap = make(map[string]models.KeyAlias)
	for _, k := range keyAliases {
		keyAliasesMap[k.KeyID] = k
	}

	res := make([]responses.Key, 0)
	for _, k := range keys {
		ka, ok := keyAliasesMap[k]
		if !ok {
			ka = models.KeyAlias{KeyID: k}
		}
		lastLocation, err := s.getLastLocationByID(nil, k)
		if err != nil {
			logger.Warnf("unable to get last location: %v", err)
		}
		res = append(res, responses.Key{
			ID:           cleanedKeyID(k),
			Alias:        ka.Alias,
			Type:         ka.Type,
			KeyInfo:      s.keyMap[k].KeyInfo(),
			LastLocation: lastLocation,
		})
	}

	sort.Sort(responses.ByKeyID(res))

	c.JSON(http.StatusOK, res)
}

func cleanedKeyID(key string) string {
	// Replaces / with another non-base64 character
	return strings.ReplaceAll(key, "/", "-")
}
func dirtyKeyID(key string) string {
	// Replaces - with /
	return strings.ReplaceAll(key, "-", "/")
}

func (s *Server) getLocationHistory(c *gin.Context) {
	keyID := c.Param("keyId")
	keyID = dirtyKeyID(keyID)
	locations, err := s.getLocationBetweenInterval(c.Request.Context(), time.Time{}, time.Now(), s.keyMap[keyID])
	if err != nil {
		logger.Errorf("unable to get location history: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to get location history"})
		return
	}
	c.JSON(http.StatusOK, locations)
}

func (s *Server) getLocationBetweenInterval(ctx context.Context, startTime time.Time, endTime time.Time, key model.MainKey) ([]Location, error) {
	var locations []Location
	tx := s.db.
		WithContext(ctx).
		Find(&locations, "key_id = ? AND found_at BETWEEN ? AND ?", key.ID(), startTime, endTime)
	if tx.Error != nil {
		return nil, fmt.Errorf("unable to fetch locations: %w", tx.Error)
	}
	return locations, nil
}

func (s *Server) getLocation(ctx context.Context, amountHours int, key model.MainKey) ([]searchparty.TagData, error) {
	lostAt := s.getLostAt(ctx, key)
	reports, subKeysMap, err := s.c.Find(ctx, []model.MainKey{key}, amountHours, lostAt)
	if err != nil {
		return nil, err
	}

	tagData := make([]searchparty.TagData, 0)
	for _, r := range reports {
		payloadBytes, err := base64.StdEncoding.DecodeString(r.Payload)
		if err != nil {
			logger.Errorf("unable to decode payload: %v", err)
			continue
		}
		key, ok := subKeysMap[r.ID]
		if !ok {
			logger.Errorf("unable to find key for report %s", r.ID)
			continue
		}
		td, err := searchparty.DecodeReport(r, key)
		if err != nil {
			logger.Errorf("unable to decode report: %v", err)
			continue
		}
		p, err := geom.NewPoint(geom.XY).SetSRID(4326).SetCoords(geom.Coord{td.Lng, td.Lat}) //nolint:mnd
		if err != nil {
			logger.Errorf("unable to create point: %v", err)
			continue
		}

		dbPoint := models.GeomPoint(*p)
		tx := s.db.
			WithContext(ctx).
			Clauses(clause.OnConflict{DoNothing: true}).
			Create(&models.Location{
				ReportedAt:      time.Unix(r.DatePublished/1000, 0),
				FoundAt:         td.Time,
				KeyID:           key.MainKey.ID(),
				CurrentKeyID:    base64.StdEncoding.EncodeToString(key.HashedAdvKey),
				OriginalContent: payloadBytes,
				Geometry:        &dbPoint,
				Confidence:      td.Confidence,
				Status:          td.Status,
			})
		if tx.Error != nil {
			logger.Errorf("unable to insert location: %v", tx.Error)
			continue
		}
		tagData = append(tagData, *td)
	}
	return tagData, nil
}

func (s *Server) refreshLocation(c *gin.Context) {
	keyID := c.Param("keyId")
	keyID = dirtyKeyID(keyID)

	amountHours := c.Query("amountHours")
	if amountHours == "" {
		amountHours = "12"
	}
	amountHoursInt, err := strconv.Atoi(amountHours)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amountHours must be an integer"})
		return
	}

	if amountHoursInt < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amountHours must be greater than 0"})
		return
	}

	logger.Infof("Refreshing location for %q", keyID)
	key, ok := s.keyMap[keyID]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "key not found"})
		return
	}

	tagData, err := s.getLocation(c, amountHoursInt, key)
	if err != nil {
		logger.Errorf("unable to get location: %v", err)
		c.JSON(http.StatusInternalServerError, map[string]any{"error": err})
		return
	}
	sort.Sort(byTime(tagData))
	c.JSON(http.StatusOK, map[string]any{"tag_data": tagData})
}

func (s *Server) loadKeys(dir string) error {
	keys, err := searchparty.LoadKeys(dir, s.beaconStoreKey)
	if err != nil {
		return fmt.Errorf("failed to load keys: %w", err)
	}
	for _, k := range keys {
		s.keyMap[k.ID()] = k
	}
	return nil
}

func (s *Server) getLastLocation(c *gin.Context) {
	keyID := c.Param("keyId")
	keyID = dirtyKeyID(keyID)
	_, ok := s.keyMap[keyID]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "key not found"})
		return
	}

	locationRes, err := s.getLastLocationByID(c.Request.Context(), keyID)
	if err != nil {
		logger.Errorf("unable to get last location: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to get last location"})
		return
	}
	c.JSON(http.StatusOK, locationRes)
}

func (s *Server) getLastLocationByID(ctx context.Context, keyID string) (*models.LocationResult, error) {
	var location models.Location
	tx := s.db.
		WithContext(ctx).
		Where("key_id = ?", keyID).
		Order("found_at desc").
		First(&location)
	if tx.Error != nil {
		return nil, fmt.Errorf("unable to fetch location: %w", tx.Error)
	}
	return &models.LocationResult{
		FoundAt:    location.FoundAt,
		ReportedAt: location.ReportedAt,
		KeyID:      location.KeyID,
		Lat:        location.Geometry.Coords().Y(),
		Lng:        location.Geometry.Coords().X(),
		Confidence: location.Confidence,
		Status:     location.Status,
	}, nil
}

func (s *Server) getLostAt(ctx context.Context, key model.MainKey) time.Time {
	var k models.KeyInfo
	tx := s.db.
		WithContext(ctx).
		Model(&models.KeyInfo{}).
		Where("id = ?", key.ID()).First(&k)
	if tx.Error != nil {
		logger.Errorf("unable to fetch key info: %v", tx.Error)
		return time.Now()
	}
	lostAt := k.LostAt
	if lostAt == nil {
		return time.Now()
	}
	return *lostAt
}
