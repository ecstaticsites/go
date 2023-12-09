package api

import (
	"context"
	"fmt"

	"github.com/carlmjohnson/requests"
	"golang.org/x/exp/slices"
)

type BunnyClient struct {
	BunnyUrl       string
	BunnyAccessKey string
}

// there is also an OriginUrl field, but we omit it, we upload directly
type AddStorageZoneBody struct {
	Name               string   `json:"Name"`
	Region             string   `json:"Region"`
	ReplicationRegions []string `json:"ReplicationRegions"`
	ZoneTier           int      `json:"ZoneTier"`
}

// there are obviously many more fields in the response, we don't care about em
type AddStorageZoneResponse struct {
	Id                 int      `json:"Id"`
	Password           string   `json:"Password"`
	Region             string   `json:"Region"`
	ReplicationRegions []string `json:"ReplicationRegions"`
	StorageHostname    string   `json:"StorageHostname"`
}

func (b BunnyClient) CreateStorageZone(ctx context.Context, name string) (*AddStorageZoneResponse, error) {

	body := AddStorageZoneBody{
		// must be globally unique, like s3 buckets
		Name: name,
		// this can AFAIU only be Germany if edge storage is selected (below)
		Region: "DE",
		// terrible english-speaking bias here, will expand later if it makes sense
		ReplicationRegions: []string{"DE", "NY", "LA", "SYD"},
		// zero is "standard", one is "edge" which means SSD storage
		ZoneTier: 1,
	}

	var resp AddStorageZoneResponse

	err := requests.
		URL(b.BunnyUrl).
		Path("https://api.bunny.net/storagezone").
		Header("AccessKey", b.BunnyAccessKey).
		ContentType("application/json").
		BodyJSON(&body).
		ToJSON(&resp).
		Post().
		Fetch(ctx)

	if err != nil {
		return nil, fmt.Errorf("Unable to create storage zone in BunnyCDN: %w", err)
	}

	if resp.Region != "DE" {
		return nil, fmt.Errorf("Region was somehow created other than DE: %v", resp.Region)
	}

	if !slices.Equal(resp.ReplicationRegions, []string{"DE", "NY", "LA", "SYD"}) {
		return nil, fmt.Errorf("Replication regions somehow created wrong: %v", resp.ReplicationRegions)
	}

	if resp.StorageHostname != "storage.bunnycdn.com" {
		return nil, fmt.Errorf("Storage hostname somehow not expected: %v", resp.StorageHostname)
	}

	return &resp, nil
}

func (b BunnyClient) CreatePullZone() error {
	return nil
}

func (b BunnyClient) ConfigureLogsForPullZone() error {
	return nil
}
