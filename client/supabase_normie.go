package client

import (
	"context"
	"fmt"
	"log"

	"github.com/carlmjohnson/requests"
)

type SupabaseNormieClient struct {
	SupabaseUrl     string
	SupabaseAnonKey string
}

type SiteRow struct {
	Id             string `json:"id"`
	CreatorId      string `json:"creator_id"`
	Nickname       string `json:"nickname"`
	CreatedAt      string `json:"created_at"`
	LastUpdatedAt  string `json:"last_updated_at"`
	StorageToken   string `json:"storage_token"`
	IndexPath      string `json:"index_path"`
	GithubRepo     string `json:"github_repo"`
	CustomHostname string `json:"custom_hostname"`
	DeployedSha    string `json:"deployed_sha"`
	PullZoneId     int    `json:"pull_zone_id"`
	Hostname       string `json:"hostname"`
}

type UpdateDeployedShaBody struct {
	DeployedSha   string `json:"deployed_sha"`
	LastUpdatedAt string `json:"last_updated_at"`
}

type AddHostnameToSiteRowBody struct {
	CustomHostname string `json:"custom_hostname"`
}

func (s SupabaseNormieClient) GetSiteRow(ctx context.Context, jwt, siteId string) *SiteRow {

	log.Printf("[INFO] Attempting to fetch row for site ID %v from supabase", siteId)

	var rows []SiteRow
	var errorJson map[string]interface{}

	err := requests.
		URL(s.SupabaseUrl).
		Path("/rest/v1/site").
		Param("id", fmt.Sprintf("eq.%v", siteId)).
		Header("apikey", s.SupabaseAnonKey).
		Header("Authorization", jwt).
		ContentType("application/json").
		ToJSON(&rows).
		ErrorJSON(&errorJson).
		Fetch(ctx)

	if err != nil {
		log.Printf("[ERROR] Unable to query SITE row for ID %v: %v, response: %+v", siteId, err, errorJson)
		return nil
	}

	if len(rows) == 0 {
		log.Printf("[ERROR] No result rows from supabase for site ID %v (possibly RLS unauthorized?)", siteId)
		return nil
	}

	if len(rows) > 1 {
		log.Printf("[ERROR] Too many rows from supabase, what do I do: %v", siteId)
		return nil
	}

	log.Printf("[INFO] Successfully fetched row for site ID %v", siteId)

	return &rows[0]
}

func (s SupabaseNormieClient) UpdateDeployedSha(ctx context.Context, jwt, siteId, sha string) bool {

	body := UpdateDeployedShaBody{
		DeployedSha:   sha,
		LastUpdatedAt: "now", // special value understood by postgrest, me being lazy
	}

	log.Printf("[INFO] Updating fields of SITE row %v with request body: %+v", siteId, body)

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
		log.Printf("[ERROR] Unable to update fields of SITE row %v: %v, response: %+v", siteId, err, errorJson)
		return false
	}

	log.Printf("[INFO] Successfully updated sha and last_updated for site %v, sha %v", siteId, sha)

	return true
}

func (s SupabaseNormieClient) AddHostnameToSiteRow(ctx context.Context, jwt, siteId, hostname string) bool {

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
