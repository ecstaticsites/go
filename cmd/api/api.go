package api

import (
	"fmt"
	"net/http"
)

type Server struct {
	supabase SupabaseClient
	bunny    BunnyClient
}

func (s Server) CreateSite(out http.ResponseWriter, req *http.Request) {

	var err error

	err = s.bunny.CreateStorageZone()
	if err != nil {
		http.Error(out, fmt.Sprintf("Unable to create new storage zone: %v", err), http.StatusInternalServerError)
		return
	}

	err = s.bunny.CreatePullZone()
	if err != nil {
		http.Error(out, fmt.Sprintf("Unable to create new pull zone: %v", err), http.StatusInternalServerError)
		return
	}

	err = s.bunny.ConfigureLogsForPullZone()
	if err != nil {
		http.Error(out, fmt.Sprintf("Unable to configure log forwarding for pull zone: %v", err), http.StatusInternalServerError)
		return
	}

	err = s.supabase.CreateSiteRow()
	if err != nil {
		http.Error(out, fmt.Sprintf("Unable to create new SITE row in supabase: %v", err), http.StatusInternalServerError)
		return
	}

	err = s.supabase.CreateAliasRow()
	if err != nil {
		http.Error(out, fmt.Sprintf("Unable to create new ALIAS row in supabase: %v", err), http.StatusInternalServerError)
		return
	}

	err = s.supabase.AuthorizeHostname(req.Context(), "a", "b")
	if err != nil {
		http.Error(out, fmt.Sprintf("Unable to authorize user for new hostname: %v", err), http.StatusInternalServerError)
		return
	}

	return
}
