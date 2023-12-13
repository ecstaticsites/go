package api

import (
	"context"
	"fmt"

	"github.com/carlmjohnson/requests"
)

type SupabaseClient struct {
	SupabaseUrl string
}

type AuthorizeHostnameBody struct {
	AppMetadata map[string][]string `json:"app_metadata"`
}

func (s SupabaseClient) CreateSiteRow(ctx context.Context, jwt, userId, siteId, nickname string, storage *CreateStorageZoneResponse) error {
	return nil
}

func (s SupabaseClient) CreateAliasRow(ctx context.Context, jwt, userId, siteId, hostname string) error {
	return nil
}

func (s SupabaseClient) AuthorizeHostname(ctx context.Context, jwt, userId, hostname string) error {

	// this is not overwriting the
	body := AuthorizeHostnameBody{
		AppMetadata: map[string][]string{
			"hostnames": {hostname},
		},
	}

	// needs to be PUT I think?
	err := requests.
		URL(s.SupabaseUrl).
		Pathf("/auth/v1/admin/users/%s", userId).
		Header("Authorization", jwt).
		ContentType("application/json").
		BodyJSON(&body).
		Fetch(ctx)

	if err != nil {
		return fmt.Errorf("Unable to update user's app_metadata in supabase: %w", err)
	}

	return nil
}
