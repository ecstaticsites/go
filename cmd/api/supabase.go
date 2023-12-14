package api

import (
	"context"
	"fmt"
	"log"

	"github.com/carlmjohnson/requests"
)

type SupabaseClient struct {
	SupabaseUrl        string
	SupabaseAdminToken string
}

// rows also include 2 fields (created, updated) which we let the DB populate for us
type CreateSiteRowBody struct {
	Id           string `json:"id"`
	Nickname     string `json:"nickname"`
	UserId       string `json:"creator_id"`
	StorageName  string `json:"storage_name"`
	StorageToken string `json:"storage_token"`
}

// rows also include 2 fields (id, created) which we let the DB populate for us
type CreateAliasRowBody struct {
	UserId   string `json:"creator_id"`
	SiteId   string `json:"site_id"`
	Hostname string `json:"hostname"`
	Default  bool   `json:"is_default"`
}

type AuthorizeHostnameBody struct {
	AppMetadata map[string][]string `json:"app_metadata"`
}

func (s SupabaseClient) CreateSiteRow(ctx context.Context, jwt, userId, siteId, nickname string, storage *CreateStorageZoneResponse) bool {

	body := CreateSiteRowBody{
		Id:           siteId,
		Nickname:     nickname,
		UserId:       userId,
		StorageName:  storage.Name,
		StorageToken: storage.Password,
	}

	log.Printf("[INFO] Creating new SITE row with request body: %+v", body)

	var errorJson map[string]interface{}

	err := requests.
		URL(s.SupabaseUrl).
		Path("/rest/v1/site").
		Header("apikey", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImV3d2NjYmdqbnVsZmdjdmZyc3ZqIiwicm9sZSI6ImFub24iLCJpYXQiOjE2OTM1ODE2ODUsImV4cCI6MjAwOTE1NzY4NX0.gI3YdNSC5GMkda2D2QPRMvnBdaMOS2ynfFKxis5-WKs").
		Header("Authorization", jwt).
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

func (s SupabaseClient) CreateAliasRow(ctx context.Context, jwt, userId, siteId, hostname string) bool {

	body := CreateAliasRowBody{
		UserId:   userId,
		SiteId:   siteId,
		Hostname: hostname,
		Default:  true,
	}

	log.Printf("[INFO] Creating new ALIAS row with request body: %+v", body)

	var errorJson map[string]interface{}

	err := requests.
		URL(s.SupabaseUrl).
		Path("/rest/v1/alias").
		Header("apikey", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImV3d2NjYmdqbnVsZmdjdmZyc3ZqIiwicm9sZSI6ImFub24iLCJpYXQiOjE2OTM1ODE2ODUsImV4cCI6MjAwOTE1NzY4NX0.gI3YdNSC5GMkda2D2QPRMvnBdaMOS2ynfFKxis5-WKs").
		Header("Authorization", jwt).
		ContentType("application/json").
		BodyJSON(&body).
		ErrorJSON(&errorJson).
		Fetch(ctx)

	if err != nil {
		log.Printf("[ERROR] Unable to create new ALIAS row: %v, response: %+v", err, errorJson)
		return false
	}

	log.Printf("[INFO] Successfully created new ALIAS for user %v, hostname %v", userId, hostname)

	return true
}

func (s SupabaseClient) AuthorizeHostname(ctx context.Context, userId, hostname string) bool {

	// this is not overwriting the
	body := AuthorizeHostnameBody{
		AppMetadata: map[string][]string{
			"hostnames": {hostname},
		},
	}

	log.Printf("[INFO] Authorizing user %v for hostname with request body: %+v", userId, body)

	var errorJson map[string]interface{}

	err := requests.
		URL(s.SupabaseUrl).
		Pathf("/auth/v1/admin/users/%s", userId).
		Header("apikey", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImV3d2NjYmdqbnVsZmdjdmZyc3ZqIiwicm9sZSI6ImFub24iLCJpYXQiOjE2OTM1ODE2ODUsImV4cCI6MjAwOTE1NzY4NX0.gI3YdNSC5GMkda2D2QPRMvnBdaMOS2ynfFKxis5-WKs").
		Header("Authorization", fmt.Sprintf("Bearer %v", s.SupabaseAdminToken)).
		ContentType("application/json").
		BodyJSON(&body).
		ErrorJSON(&errorJson).
		Put().
		Fetch(ctx)

	if err != nil {
		log.Printf("[ERROR] Unable to create pull zone in BunnyCDN: %v, response: %+v", err, errorJson)
		return false
	}

	log.Printf("[INFO] Successfully authorized user %v for hostname %v", userId, hostname)

	return true
}
