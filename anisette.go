package searchparty

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

const MdRinfo = "17106176" // Either 17106176 or 50660608

type AnisetteResponse struct {
	XAppleIClientTime time.Time `json:"X-Apple-I-Client-Time"`
	XAppleIMD         string    `json:"X-Apple-I-MD"`
	XAppleIMDLU       string    `json:"X-Apple-I-MD-LU"`
	XAppleIMDM        string    `json:"X-Apple-I-MD-M"`
	XAppleIMDRINFO    string    `json:"X-Apple-I-MD-RINFO"`
	XAppleISRLNO      string    `json:"X-Apple-I-SRL-NO"`
	XAppleITimeZone   string    `json:"X-Apple-I-TimeZone"`
	XAppleLocale      string    `json:"X-Apple-Locale"`
	XMMeClientInfo    string    `json:"X-MMe-Client-Info"`
	XMmeDeviceId      string    `json:"X-Mme-Device-Id"`
}

func getAnisetteHeaders(anisetteUrl string) (http.Header, error) {
	req, err := http.NewRequest("GET", anisetteUrl, nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	var response AnisetteResponse
	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return nil, err
	}
	userId := uuid.NewString()
	userId = strings.Replace(userId, "-", "", -1)
	userId = strings.ToUpper(userId)

	deviceId := strings.ToUpper(uuid.NewString())

	headers := http.Header{
		"X-Apple-I-MD":          []string{response.XAppleIMD},
		"X-Apple-I-MD-M":        []string{response.XAppleIMDM},
		"X-Apple-I-Client-Time": []string{time.Now().UTC().Format(time.RFC3339)},
		"X-Apple-I-TimeZone":    []string{time.Now().Location().String()},
		"loc":                   []string{"en_US"},
		"X-Apple-Locale":        []string{"en_US"},
		"X-Apple-I-MD-RINFO":    []string{MdRinfo},
		"X-Apple-I-MD-LU":       []string{userId},
		"X-Mme-Device-Id":       []string{deviceId},
		"X-Apple-I-SRL-NO":      []string{"0"},
	}

	return headers, nil
}
