package searchparty

import (
	"bytes"
	"encoding/base64"
	"io"
	"time"

	"github.com/denysvitali/searchparty-keys"

	"github.com/denysvitali/searchparty-go/model"
)

const (
	primaryRotationMinutes = 15
	secondaryRotationHours = 24
)

type DynamicKey struct {
	beacon *searchpartykeys.Beacon
}

func (d *DynamicKey) ID() string {
	return base64.StdEncoding.EncodeToString(sha256Hash(d.beacon.PublicKey.Key.Data)[:8])
}

func (d *DynamicKey) Type() string {
	return "dynamic"
}

func (d *DynamicKey) KeyInfo() model.KeyInfo {
	return model.KeyInfo{
		Model:            d.beacon.Model,
		PairingDate:      d.beacon.PairingDate,
		Identifier:       d.beacon.Identifier,
		StableIdentifier: d.beacon.StableIdentifier[0],
	}
}

var _ model.MainKey = &DynamicKey{}

const (
	primaryRotation   = 15 * time.Minute
	secondaryRotation = 24 * time.Hour
)

func CalculateKeyRotation(from, to, initialTime time.Time, rotationDuration time.Duration) (int, int) {
	realFrom := from.Truncate(rotationDuration)

	elapsedFrom := realFrom.Sub(initialTime)
	// Calculate the elapsed time from the initial time to the "to" time
	elapsedTo := to.Sub(initialTime)

	// Calculate the number of amountKeys at each time
	rotationsFrom := int(elapsedFrom / rotationDuration)
	rotationsTo := int(elapsedTo / rotationDuration)

	// The number of amountKeys between "from" and "to"
	amountKeys := rotationsTo - rotationsFrom

	if amountKeys == 0 {
		amountKeys = 1
	}

	// Calculate the offset from the initial time to the "to" time
	offset := elapsedFrom / rotationDuration
	if offset < 0 {
		offset = 0
	}

	return amountKeys, int(offset)
}

func (d *DynamicKey) GetSubKeys(from time.Time, to time.Time, lostAt time.Time) (subKeys []model.SubKey, err error) {
	amountPrimary, offsetPrimary := CalculateKeyRotation(lostAt, to, d.beacon.PairingDate, primaryRotation)
	amountSecondary, offsetSecondary := CalculateKeyRotation(lostAt, to, d.beacon.PairingDate, secondaryRotation)

	logger.Debugf("Primary: %d, Secondary: %d", amountPrimary, amountSecondary)
	logger.Debugf("Primary offset: %d, Secondary offset: %d", offsetPrimary, offsetSecondary)

	primaryKeys, err := searchpartykeys.CalculateAdvertisementKeys(
		d.beacon.PrivateKey.Key.Data,
		d.beacon.SharedSecret.Key.Data,
		amountPrimary,
		offsetPrimary+2,
	)
	if err != nil {
		return nil, err
	}
	secondaryKeys, err := searchpartykeys.CalculateAdvertisementKeys(
		d.beacon.PrivateKey.Key.Data,
		d.beacon.SecondarySharedSecret.Key.Data,
		amountSecondary,
		offsetSecondary+2,
	)
	if err != nil {
		return nil, err
	}

	for _, p := range primaryKeys {
		logger.Debugf("Adding primary key %s", base64.StdEncoding.EncodeToString(p.HashedAdvKey()))
		subKeys = append(subKeys, model.SubKey{
			AdvKey:       p.AdvKeyBytes(),
			HashedAdvKey: p.HashedAdvKey(),
			PrivateKey:   p.PrivateKey(),
			Type:         model.Primary,
			MainKey:      model.MainKey(d),
		})
	}

	for _, s := range secondaryKeys {
		logger.Debugf("Adding secondary key %s", base64.StdEncoding.EncodeToString(s.HashedAdvKey()))
		subKeys = append(subKeys, model.SubKey{
			AdvKey:       s.AdvKeyBytes(),
			HashedAdvKey: s.HashedAdvKey(),
			PrivateKey:   s.PrivateKey(),
			Type:         model.Secondary,
			MainKey:      model.MainKey(d),
		})
	}
	return
}

func LoadDynamicKey(reader io.ReadSeeker, key []byte) (model.MainKey, error) {
	d, err := searchpartykeys.Decrypt(reader, key)
	if err != nil {
		return nil, err
	}

	decoded, err := searchpartykeys.Decode(bytes.NewReader(d))
	if err != nil {
		return nil, err
	}

	return &DynamicKey{
		beacon: decoded,
	}, nil
}
