package client

import (
	"context"
	"fmt"
	"log"

	"github.com/carlmjohnson/requests"
)

type SupabaseAdminClient struct {
	SupabaseUrl        string
	SupabaseAnonKey    string
	SupabaseServiceKey string
}

// rows also include fields (created, index path, etc) which we let the DB populate for us
type CreateSiteRowBody struct {
	Id           string `json:"id"`
	CreatorId    string `json:"creator_id"`
	Nickname     string `json:"nickname"`
	StorageToken string `json:"storage_token"`
	PullZoneId   int64  `json:"pull_zone_id"`
	Hostname     string `json:"hostname"`
}

type AuthorizeZoneIdBody struct {
	AppMetadata map[string][]int `json:"app_metadata"`
}

func (s SupabaseAdminClient) CreateSiteRow(ctx context.Context, userId, siteId, nickname string, storage *CreateStorageZoneResponse, pull *CreatePullZoneResponse) bool {

	body := CreateSiteRowBody{
		Id:           siteId,
		CreatorId:    userId,
		Nickname:     nickname,
		StorageToken: storage.Password,
		PullZoneId:   pull.Id,
		Hostname:     pull.Hostnames[0].Value,
	}

	log.Printf("[INFO] Creating new SITE row with request body: %+v", body)

	var errorJson map[string]interface{}

	err := requests.
		URL(s.SupabaseUrl).
		Path("/rest/v1/site").
		Header("apikey", s.SupabaseAnonKey).
		Header("Authorization", fmt.Sprintf("Bearer %v", s.SupabaseServiceKey)).
		ContentType("application/json").
		BodyJSON(&body).
		ErrorJSON(&errorJson).
		Fetch(ctx)

	if err != nil {
		log.Printf("[ERROR] Unable to create new SITE row: %v, response: %+v", err, errorJson)
		return false
	}

	log.Printf("[INFO] Successfully created new SITE for user %v, site %v", userId, siteId)

	return true
}

func (s SupabaseAdminClient) AuthorizeZoneId(ctx context.Context, userId string, newZoneId int, existingZoneIds []int) bool {

	// mutating in place because Go makes anything else annoyingly difficult
	existingZoneIds = append(existingZoneIds, newZoneId)

	body := AuthorizeZoneIdBody{
		AppMetadata: map[string][]int{
			"zones": existingZoneIds,
		},
	}

	log.Printf("[INFO] Authorizing user %v for zone with request body: %+v", userId, body)

	var errorJson map[string]interface{}

	err := requests.
		URL(s.SupabaseUrl).
		Pathf("/auth/v1/admin/users/%s", userId).
		Header("apikey", s.SupabaseAnonKey).
		Header("Authorization", fmt.Sprintf("Bearer %v", s.SupabaseServiceKey)).
		ContentType("application/json").
		BodyJSON(&body).
		ErrorJSON(&errorJson).
		Put().
		Fetch(ctx)

	if err != nil {
		log.Printf("[ERROR] Unable to authorize user %v for zone: %v, response: %+v", userId, err, errorJson)
		return false
	}

	log.Printf("[INFO] Successfully authorized user %v for zone ID %v", userId, newZoneId)

	return true
}
