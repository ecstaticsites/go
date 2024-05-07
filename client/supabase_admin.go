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

type AddHostnameToSiteRowBody struct {
	CustomHostname string `json:"custom_hostname"`
}

type AuthorizeHostnameBody struct {
	AppMetadata map[string][]string `json:"app_metadata"`
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

func (s SupabaseAdminClient) AuthorizeHostname(ctx context.Context, userId, newHostname string, existingHostnames []string) bool {

	// mutating in place because Go makes anything else annoyingly difficult
	existingHostnames = append(existingHostnames, newHostname)

	body := AuthorizeHostnameBody{
		AppMetadata: map[string][]string{
			"hostnames": existingHostnames,
		},
	}

	log.Printf("[INFO] Authorizing user %v for hostname with request body: %+v", userId, body)

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
		log.Printf("[ERROR] Unable to create pull zone in BunnyCDN: %v, response: %+v", err, errorJson)
		return false
	}

	log.Printf("[INFO] Successfully authorized user %v for hostname %v", userId, newHostname)

	return true
}

// todo - unclear why this needs the admin client, can service role bypass no-update trigger?
func (s SupabaseAdminClient) AddHostnameToSiteRow(ctx context.Context, siteId, hostname string) bool {

	body := AddHostnameToSiteRowBody{
		CustomHostname: hostname,
	}

	log.Printf("[INFO] Adding hostname to SITE row %v with request body: %+v", siteId, body)

	var errorJson map[string]interface{}

	err := requests.
		URL(s.SupabaseUrl).
		Path("/rest/v1/site").
		Param("id", fmt.Sprintf("eq.%v", siteId)).
		Header("apikey", s.SupabaseAnonKey).
		Header("Authorization", fmt.Sprintf("Bearer %v", s.SupabaseServiceKey)).
		ContentType("application/json").
		BodyJSON(&body).
		ErrorJSON(&errorJson).
		Patch().
		Fetch(ctx)

	if err != nil {
		log.Printf("[ERROR] Unable to add hostname to SITE row %v: %v, response: %+v", siteId, err, errorJson)
		return false
	}

	log.Printf("[INFO] Successfully added new hostname for site %v, hostname %v", siteId, hostname)

	return true
}
