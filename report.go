package searchparty

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/denysvitali/searchparty-go/model"
)

const coreDataTsDiff = 978307200

type FindResult struct {
	Results []Report `json:"results"`
}

func decrypt(encData, key, iv, tag []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCMWithNonceSize(block, len(iv))
	if err != nil {
		return nil, err
	}
	return aesgcm.Open(nil, iv, append(encData, tag...), nil)
}

type TagData struct {
	Time       time.Time `json:"time"`
	Lat        float64   `json:"lat"`
	Lng        float64   `json:"lng"`
	Confidence int       `json:"confidence"`
	Status     int       `json:"status"`
}

func (t TagData) String() string {
	return fmt.Sprintf("https://maps.google.com/?q=%f,%f\tconf=%v,status=%v",
		t.Lat,
		t.Lng,
		t.Confidence,
		t.Status,
	)
}

func decodeTag(data []byte) (*TagData, error) {
	if len(data) < 10 {
		return nil, fmt.Errorf("data too short")
	}
	latitude := float64(int32(binary.BigEndian.Uint32(data[:4]))) / 10000000.0
	longitude := float64(int32(binary.BigEndian.Uint32(data[4:8]))) / 10000000.0
	confidence := int(data[8])
	status := int(data[9])
	return &TagData{
		Lat:        latitude,
		Lng:        longitude,
		Confidence: confidence,
		Status:     status,
	}, nil
}

func sha256Hash(data []byte) []byte {
	h := sha256.New()
	h.Write(data)
	return h.Sum(nil)
}

func DecodeReport(report Report, key model.SubKey) (*TagData, error) {
	payload, err := base64.StdEncoding.DecodeString(report.Payload)
	if err != nil {
		return nil, err
	}
	logger.Tracef("payload_start\t0x%s", hex.EncodeToString(payload[0:10]))
	payloadLength := len(payload)

	if payloadLength == 89 {
		// V1
		logger.Tracef("V1 - payloadLength=%d", payloadLength)
	} else {
		// V2
		logger.Tracef("V2 - payloadLength=%d", payloadLength)
		newPayload := make([]byte, payloadLength+1)
		copy(newPayload, payload[0:4])
		newPayload[4] = 0x00
		copy(newPayload[5:], payload[4:])
		payload = newPayload
	}

	timestamp := binary.BigEndian.Uint32(payload[0:4])
	foundAt := time.Unix(int64(timestamp)+coreDataTsDiff, 0)

	curveBytes := payload[6:63]

	var ephKeyX, ephKeyY *big.Int
	ephKeyX, ephKeyY = elliptic.Unmarshal(elliptic.P224(), curveBytes)

	if ephKeyX == nil || ephKeyY == nil {
		return nil, fmt.Errorf("unable to unmarshal curve bytes")
	}
	ephKey := &ecdsa.PublicKey{Curve: elliptic.P224(), X: ephKeyX, Y: ephKeyY}
	sharedKey, _ := ephKey.Curve.ScalarMult(ephKey.X, ephKey.Y, key.PrivateKey)
	sharedKeyBytes := sharedKey.Bytes()

	toHash := append(sharedKeyBytes, byte(0), byte(0), byte(0), byte(1))
	toHash = append(toHash, curveBytes...)

	symmetricKey := sha256Hash(toHash)
	decryptionKey := symmetricKey[:16]
	iv := symmetricKey[16:]

	startIdx := 6 + len(curveBytes)
	encData := payload[startIdx : startIdx+8]
	tag := payload[startIdx+8:]

	decrypted, err := decrypt(encData, decryptionKey, iv, tag)
	if err != nil {
		return nil, fmt.Errorf("unable to decrypt: %w", err)
	}
	tagData, err := decodeTag(decrypted)
	if err != nil {
		return nil, fmt.Errorf("unable to decode tag: %w", err)
	}
	tagData.Time = foundAt
	logger.Tracef("tagData\t\t\t%s", tagData)
	return tagData, nil
}
