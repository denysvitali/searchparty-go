package models

import (
	"database/sql/driver"
	"encoding/hex"
	"time"

	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/ewkb"
)

type GeomPoint geom.Point

// Value return geometry point value, implement driver.Valuer interface
func (g GeomPoint) Value() (driver.Value, error) {
	b := geom.Point(g)
	bp := &b
	ewkbPt := ewkb.Point{Point: bp.SetSRID(4326)}
	return ewkbPt.Value()
}

// Scan scan value into geom.Point, implements sql.Scanner interface
func (g *GeomPoint) Scan(value interface{}) error {
	t, err := hex.DecodeString(string(value.([]byte)))
	if err != nil {
		return err
	}
	gt, err := ewkb.Unmarshal(t)
	if err != nil {
		return err
	}
	p := GeomPoint(*gt.(*geom.Point))
	*g = p

	return nil
}

type Location struct {
	FoundAt         time.Time `gorm:"primaryKey;index:idx_found_at"`
	ReportedAt      time.Time `gorm:"index:idx_reported_at"`
	KeyID           string    `gorm:"primaryKey;index:idx_key_id"`
	OriginalContent []byte
	Geometry        *GeomPoint `gorm:"type:geometry(POINT,4326);index:idx_geometry"`
	Confidence      int        `gorm:"index:idx_confidence"`
	Status          int
	CurrentKeyID    string `gorm:"index:idx_current_key_id"`
}

type LocationResult struct {
	FoundAt    time.Time `json:"foundAt"`
	ReportedAt time.Time `json:"reportedAt"`
	KeyID      string    `json:"keyId"`
	Lat        float64   `json:"lat"`
	Lng        float64   `json:"lng"`
	Confidence int       `json:"confidence"`
	Status     int       `json:"status"`
}
