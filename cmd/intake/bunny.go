package intake

import (
	"fmt"
	"encoding/json"
)

type BunnyLog struct {
	ServerId      int     `json:"ServerId"`
	IsValid       bool    `json:"IsValid"`
	PullZoneId    int     `json:"PullZoneId"`
	IsSSL         bool    `json:"IsSSL"`
	Host          string  `json:"Host"`
	Protocol      string  `json:"Protocol"`
	Scheme        string  `json:"Scheme"`
	UserAgent     string  `json:"UserAgent"`
	RequestId     string  `json:"RequestId"`
	HeaderRange   string  `json:"HeaderRange"`
	Country       string  `json:"Country"`
	PathAndQuery  string  `json:"PathAndQuery"`
	ServerZone    string  `json:"ServerZone"`
	Status        int     `json:"Status"`
	BytesSent     int     `json:"BytesSent"`
	BodyBytesSent int     `json:"BodyBytesSent"`
	Timestamp     int64   `json:"Timestamp"`
	GzipRatio     float64 `json:"GzipRatio"`
	Cached        bool    `json:"Cached"`
	RemoteIp      string  `json:"RemoteIp"`
	Referer       string  `json:"Referer"`
}

func stringToBunnyLog(input []byte) (BunnyLog, error) {

	bunnylog := BunnyLog{}

	err := json.Unmarshal(input, &bunnylog)
	if err != nil {
	    return bunnylog, fmt.Errorf("Could not parse TCP body: %v", err)
	}

	return bunnylog, nil
}
