package api

import (
	"context"
	"log"

	"github.com/carlmjohnson/requests"
	"golang.org/x/exp/slices"
)

type BunnyClient struct {
	BunnyUrl       string
	BunnyAccessKey string
}

// there is also an OriginUrl field, but we omit it, we upload directly
type CreateStorageZoneBody struct {
	Name               string   `json:"Name"`
	Region             string   `json:"Region"`
	ReplicationRegions []string `json:"ReplicationRegions"`
	ZoneTier           int      `json:"ZoneTier"`
}

// there are obviously many more fields in the response, we don't care about em
type CreateStorageZoneResponse struct {
	Id                 int64    `json:"Id"`
	Name               string   `json:"Name"`
	Password           string   `json:"Password"`
	Region             string   `json:"Region"`
	ReplicationRegions []string `json:"ReplicationRegions"`
	StorageHostname    string   `json:"StorageHostname"`
}

// aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
type CreatePullZoneBody struct {
	Name                          string `json:"Name"`
	Type                          int    `json:"Type"`
	StorageZoneId                 int64  `json:"StorageZoneId"`
	OriginType                    int    `json:"OriginType"`
	EnableGeoZoneUS               bool   `json:"EnableGeoZoneUS"`
	EnableGeoZoneEU               bool   `json:"EnableGeoZoneEU"`
	EnableGeoZoneASIA             bool   `json:"EnableGeoZoneASIA"`
	EnableGeoZoneSA               bool   `json:"EnableGeoZoneSA"`
	EnableGeoZoneAF               bool   `json:"EnableGeoZoneAF"`
	EnableLogging                 bool   `json:"EnableLogging"`
	LogFormat                     int    `json:"LogFormat"`
	LogForwardingFormat           int    `json:"LogForwardingFormat"`
	LoggingIPAnonymizationEnabled bool   `json:"LoggingIPAnonymizationEnabled"`
	LogAnonymizationType          int    `json:"LogAnonymizationType"`
	LogForwardingEnabled          bool   `json:"LogForwardingEnabled"`
	LogForwardingHostname         string `json:"LogForwardingHostname"`
	LogForwardingPort             int    `json:"LogForwardingPort"`
	LogForwardingProtocol         int    `json:"LogForwardingProtocol"`
	UseStaleWhileUpdating         bool   `json:"UseStaleWhileUpdating"`
	UseStaleWhileOffline          bool   `json:"UseStaleWhileOffline"`
	EnableAutoSSL                 bool   `json:"EnableAutoSSL"`
}

// used below
type PullZoneHostname struct {
	Id               int64  `json:"Id"`
	Value            string `json:"Value"`
	IsSystemHostname bool   `json:"IsSystemHostname"`
}

// there are obviously many more fields in the response, we don't care about em
type CreatePullZoneResponse struct {
	Id        int64              `json:"Id"`
	Name      string             `json:"Name"`
	Enabled   bool               `json:"Enabled"`
	Hostnames []PullZoneHostname `json:"Hostnames"`
}

func (b BunnyClient) CreateStorageZone(ctx context.Context, siteId string) *CreateStorageZoneResponse {

	body := CreateStorageZoneBody{
		// must be globally unique, like s3 buckets
		Name: siteId,
		// this can AFAIU only be Germany if edge storage is selected (below)
		Region: "DE",
		// terrible english-speaking bias here, will expand later if it makes sense
		ReplicationRegions: []string{"NY", "LA", "SYD"},
		// zero is "standard", one is "edge" which means SSD storage
		ZoneTier: 1,
	}

	log.Printf("[INFO] Creating bunny storage zone with request body: %+v", body)

	var resp CreateStorageZoneResponse
	var errorJson map[string]interface{}

	err := requests.
		URL(b.BunnyUrl).
		Path("/storagezone").
		Header("AccessKey", b.BunnyAccessKey).
		ContentType("application/json").
		BodyJSON(&body).
		ToJSON(&resp).
		ErrorJSON(&errorJson).
		Post().
		Fetch(ctx)

	if err != nil {
		log.Printf("[ERROR] Unable to create storage zone in BunnyCDN: %v, response: %+v", err, errorJson)
		return nil
	}

	if resp.Region != "DE" {
		log.Printf("[ERROR] Storage zone region was somehow created other than DE: %v", resp.Region)
		return nil
	}

	if !slices.Equal(resp.ReplicationRegions, []string{"NY", "LA", "SYD"}) {
		log.Printf("[ERROR] Storage zone replication regions somehow created wrong: %v", resp.ReplicationRegions)
		return nil
	}

	if resp.StorageHostname != "storage.bunnycdn.com" {
		log.Printf("[ERROR] Storage zone hostname somehow not expected: %v", resp.StorageHostname)
		return nil
	}

	log.Printf("[INFO] Bunny storage zone with ID %v successfully created", siteId)
	return &resp
}

func (b BunnyClient) CreatePullZone(ctx context.Context, siteId string, storage *CreateStorageZoneResponse) string {

	body := CreatePullZoneBody{
		Name:                          siteId,
		Type:                          0, // Premium (SSD)
		StorageZoneId:                 storage.Id,
		OriginType:                    2, // StorageZone
		EnableGeoZoneUS:               true,
		EnableGeoZoneEU:               true,
		EnableGeoZoneASIA:             true,
		EnableGeoZoneSA:               true,
		EnableGeoZoneAF:               true,
		EnableLogging:                 true,
		LogFormat:                     0, // plaintext
		LogForwardingFormat:           0, // plaintext
		LoggingIPAnonymizationEnabled: true,
		LogAnonymizationType:          0, // one-octet
		LogForwardingEnabled:          true,
		LogForwardingHostname:         "intake.cbnr.xyz",
		LogForwardingPort:             517,
		LogForwardingProtocol:         0, // UDP
		UseStaleWhileUpdating:         true,
		UseStaleWhileOffline:          true,
		EnableAutoSSL:                 true,
	}

	log.Printf("[INFO] Creating bunny pull zone with request body: %+v", body)

	var resp CreatePullZoneResponse
	var errorJson map[string]interface{}

	err := requests.
		URL(b.BunnyUrl).
		Path("/pullzone").
		Header("AccessKey", b.BunnyAccessKey).
		ContentType("application/json").
		BodyJSON(&body).
		ToJSON(&resp).
		ErrorJSON(&errorJson).
		Post().
		Fetch(ctx)

	if err != nil {
		log.Printf("[ERROR] Unable to create pull zone in BunnyCDN: %v, response: %+v", err, errorJson)
		return ""
	}

	if !resp.Enabled {
		log.Printf("[ERROR] Unexpected not enabled created pull zone: %v", resp.Id)
		return ""
	}

	if len(resp.Hostnames) != 1 {
		log.Printf("[ERROR] Incorrect # of hostnames for new pull zone (expected 1): %v", resp.Hostnames)
		return ""
	}

	if !resp.Hostnames[0].IsSystemHostname {
		log.Printf("[ERROR] Unexpected not system hostname for new pull zone: %v", resp.Hostnames[0].Id)
		return ""
	}

	log.Printf("[INFO] Bunny pull zone with ID %v successfully created", siteId)
	return resp.Hostnames[0].Value
}
