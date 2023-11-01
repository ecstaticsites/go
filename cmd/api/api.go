package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/jwtauth/v5"
)

type Server struct {
	supabase SupabaseClient
	bunny    BunnyClient
}

func (s Server) CreateSite(out http.ResponseWriter, req *http.Request) {

	var err error

	_, claims, err := jwtauth.FromContext(req.Context())
	if err != nil {
		http.Error(out, fmt.Sprintf("Unable to parse claims from JWT: %v", err), http.StatusInternalServerError)
		return
	}

	userId, found := claims["user_id"]
	if !found {
		http.Error(out, fmt.Sprintf("No 'user_id' field found in JWT claims"), http.StatusInternalServerError)
		return
	}

	userIdStr, ok := userId.(string)
	if !ok {
		http.Error(out, "Claims 'user_id' could not be parsed as string", http.StatusInternalServerError)
		return
	}

	_, err = s.bunny.CreateStorageZone(req.Context(), "aaa")
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

	err = s.supabase.AuthorizeHostname(req.Context(), userIdStr, "b")
	if err != nil {
		http.Error(out, fmt.Sprintf("Unable to authorize user for new hostname: %v", err), http.StatusInternalServerError)
		return
	}

	return
}
