package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"cbnr/util"

	"github.com/go-chi/jwtauth/v5"
)

type Server struct {
	supabase SupabaseClient
	bunny    BunnyClient
}

type CreateSiteRequest struct {
	Nickname string
}

func (s Server) CreateSite(out http.ResponseWriter, req *http.Request) {

	var err error

	_, claims, err := jwtauth.FromContext(req.Context())
	if err != nil {
		log.Printf("[ERROR] Unable to parse claims from JWT: %v", err)
		http.Error(out, "Unable to parse claims from JWT", http.StatusUnauthorized)
		return
	}

	// what is "sub"? If nothing else, seems to be the JWT's slang for user ID
	userIdUntyped, found := claims["sub"]
	if !found {
		log.Printf("[ERROR] No 'user_id' field found in JWT claims: %v", claims)
		http.Error(out, "Unable to parse claims from JWT", http.StatusUnauthorized)
		return
	}

	userId, ok := userIdUntyped.(string)
	if !ok {
		log.Printf("[ERROR] Claims 'user_id' could not be parsed as string")
		http.Error(out, "Unable to parse claims from JWT", http.StatusUnauthorized)
		return
	}

	// get the currently-authorized hostnames from the claims for appending later
	existingHostnames, err := util.GetHostnamesFromClaims(claims)
	if err != nil {
		log.Printf("[ERROR] Unable to get hostnames from JWT claims: %v", err)
		http.Error(out, "Invalid JWT app metadata", http.StatusUnauthorized)
		return
	}

	// no need to validate here, impossible to get this far if JWT is invalid
	jwt := req.Header.Get("Authorization")

	// get the nickname
	var body CreateSiteRequest

	err = json.NewDecoder(req.Body).Decode(&body)
	if err != nil {
		log.Printf("[ERROR] Request body did not parse as expected: %v, body %v", err, req.Body)
		http.Error(out, "Malformed input, just send JSON with a nickname field", http.StatusBadRequest)
		return
	}

	// needed by pretty much all the below functions, so let's gen it here
	siteId := fmt.Sprintf("%v-%v-%v", util.RandomString(3), util.RandomString(3), util.RandomString(3))
	log.Printf("[INFO] Creating a new site with generated ID %v...", siteId)

	storage := s.bunny.CreateStorageZone(req.Context(), siteId)
	if storage == nil {
		http.Error(out, "Unable to create new storage zone", http.StatusInternalServerError)
		return
	}

	newHostname := s.bunny.CreatePullZone(req.Context(), siteId, storage)
	if newHostname == "" {
		http.Error(out, "Unable to create new pull zone", http.StatusInternalServerError)
		return
	}

	worked := s.supabase.CreateSiteRow(req.Context(), jwt, userId, siteId, body.Nickname, storage)
	if !worked {
		http.Error(out, "Unable to create new SITE row", http.StatusInternalServerError)
		return
	}

	worked = s.supabase.CreateAliasRow(req.Context(), jwt, userId, siteId, newHostname)
	if !worked {
		http.Error(out, "Unable to create new ALIAS row", http.StatusInternalServerError)
		return
	}

	worked = s.supabase.AuthorizeHostname(req.Context(), userId, newHostname, existingHostnames)
	if !worked {
		http.Error(out, "Unable to authorize user for new hostname", http.StatusInternalServerError)
		return
	}

	return
}
