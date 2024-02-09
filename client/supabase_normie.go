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
