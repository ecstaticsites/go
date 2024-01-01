package api

import (
	"context"
	"fmt"
	"log"

	"github.com/carlmjohnson/requests"
)

type SupabaseClient struct {
	SupabaseUrl        string
	SupabaseAnonKey    string
	SupabaseServiceKey string
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

type AddHostnameToSiteRowBody struct {
	CustomHostname string `json:"custom_hostname"`
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
		Header("apikey", s.SupabaseAnonKey).
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
		Header("apikey", s.SupabaseAnonKey).
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

func (s SupabaseClient) AuthorizeHostname(ctx context.Context, userId, newHostname string, existingHostnames []string) bool {

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

func (s SupabaseClient) AddHostnameToSiteRow(ctx context.Context, jwt, siteId, hostname string) bool {

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
		Header("Authorization", jwt).
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
