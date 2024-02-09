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
    Id             string
    CreatorId      string
    Nickname       string
    CreatedAt      string
    LastUpdatedAt  string
    StorageToken   string
    IndexPath      string
    GithubRepo     string
    CustomHostname string
    DeployedSha    string
    PullZoneId     int
    Hostname       string
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
