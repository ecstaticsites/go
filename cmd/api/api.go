package api

import (
	"fmt"
	"log"
	"net/http"

	"cbnr/util"

	//"github.com/go-chi/jwtauth/v5"
)

type Server struct {
	supabase SupabaseClient
	bunny    BunnyClient
}

func (s Server) CreateSite(out http.ResponseWriter, req *http.Request) {

	var err error

	// _, claims, err := jwtauth.FromContext(req.Context())
	// if err != nil {
	// 	http.Error(out, fmt.Sprintf("Unable to parse claims from JWT: %v", err), http.StatusInternalServerError)
	// 	return
	// }

	// userIdUntyped, found := claims["user_id"]
	// if !found {
	// 	http.Error(out, fmt.Sprintf("No 'user_id' field found in JWT claims"), http.StatusInternalServerError)
	// 	return
	// }

	// userId, ok := userIdUntyped.(string)
	// if !ok {
	// 	http.Error(out, "Claims 'user_id' could not be parsed as string", http.StatusInternalServerError)
	// 	return
	// }

	// no need to validate here, impossible to get this far if JWT is invalid
	jwt := req.Header.Get("Authorization")

	// needed by pretty much all the below functions, so let's gen it here
	siteId := fmt.Sprintf("%v-%v-%v", util.RandomString(3), util.RandomString(3), util.RandomString(3))
	log.Printf("Creating a new site with generated ID %v...", siteId)

	storage, err := s.bunny.CreateStorageZone(req.Context(), siteId)
	if err != nil {
		http.Error(out, fmt.Sprintf("Unable to create new storage zone: %v", err), http.StatusInternalServerError)
		return
	}

	hostname, err := s.bunny.CreatePullZone(req.Context(), siteId, storage)
	if err != nil {
		http.Error(out, fmt.Sprintf("Unable to create new pull zone: %v", err), http.StatusInternalServerError)
		return
	}

	err = s.supabase.CreateSiteRow(req.Context(), jwt, "userId", siteId, "nickname", storage)
	if err != nil {
		http.Error(out, fmt.Sprintf("Unable to create new SITE row in supabase: %v", err), http.StatusInternalServerError)
		return
	}

	err = s.supabase.CreateAliasRow(req.Context(), jwt, "userId", siteId, hostname)
	if err != nil {
		http.Error(out, fmt.Sprintf("Unable to create new ALIAS row in supabase: %v", err), http.StatusInternalServerError)
		return
	}

	err = s.supabase.AuthorizeHostname(req.Context(), jwt, "userId", hostname)
	if err != nil {
		http.Error(out, fmt.Sprintf("Unable to authorize user for new hostname: %v", err), http.StatusInternalServerError)
		return
	}

	return
}
