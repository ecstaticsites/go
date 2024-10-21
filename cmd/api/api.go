package api

import (
	"encoding/json"
	"log"
	"net/http"

	"ecstatic/client"
	"ecstatic/util"

	"github.com/go-chi/jwtauth/v5"
)

type Server struct {
	SupaNormie client.SupabaseNormieClient
	SupaAdmin  client.SupabaseAdminClient
	BunnyAdmin client.BunnyAdminClient
}

type CreateSiteRequest struct {
	Nickname string
}

type CreateSiteResponse struct {
	Id string `json:"id"`
}

type AddHostnameRequest struct {
	SiteId   string `json:"siteid"`
	Hostname string `json:"hostname"`
}

type PostPushRequest struct {
	SiteId string `json:"siteid"`
	Sha    string `json:"sha"`
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

	// get the currently-authorized sites from the claims for appending later
	existingZoneIds, err := util.GetZoneIdsFromClaims(claims)
	if err != nil {
		log.Printf("[ERROR] Unable to get site IDs from JWT claims: %v", err)
		http.Error(out, "Invalid JWT app metadata", http.StatusUnauthorized)
		return
	}

	// get the nickname
	var body CreateSiteRequest

	err = json.NewDecoder(req.Body).Decode(&body)
	if err != nil {
		log.Printf("[ERROR] Request body did not parse as expected: %v, body %v", err, req.Body)
		http.Error(out, "Malformed input, just send JSON with a nickname field", http.StatusBadRequest)
		return
	}

	// needed by pretty much all the below functions, so let's gen it here
	siteId := util.RandomIamTriple()
	log.Printf("[INFO] Creating a new site with generated ID %v...", siteId)

	storage := s.BunnyAdmin.CreateStorageZone(req.Context(), siteId)
	if storage == nil {
		http.Error(out, "Unable to create new storage zone", http.StatusInternalServerError)
		return
	}

	pull := s.BunnyAdmin.CreatePullZone(req.Context(), siteId, storage)
	if pull == nil {
		http.Error(out, "Unable to create new pull zone", http.StatusInternalServerError)
		return
	}

	worked := s.SupaAdmin.CreateSiteRow(req.Context(), userId, siteId, body.Nickname, storage, pull)
	if !worked {
		http.Error(out, "Unable to create new SITE row", http.StatusInternalServerError)
		return
	}

	worked = s.SupaAdmin.AuthorizeZoneId(req.Context(), userId, int(pull.Id), existingZoneIds)
	if !worked {
		http.Error(out, "Unable to authorize user for new zone ID", http.StatusInternalServerError)
		return
	}

	log.Printf("[INFO] All good, site %v created, writing response body...", siteId)

	resp := CreateSiteResponse{
		Id: siteId,
	}

	// TODO, can/should probably use go-chi render for all this?
	out.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(out).Encode(resp)
	if err != nil {
		http.Error(out, "Unable to render output, SITE WAS STILL CREATED", http.StatusInternalServerError)
		return
	}

	return
}

func (s Server) AddHostname(out http.ResponseWriter, req *http.Request) {

	var err error

	// no need to validate here, impossible to get this far if JWT is invalid
	jwt := req.Header.Get("Authorization")

	var body AddHostnameRequest

	err = json.NewDecoder(req.Body).Decode(&body)
	if err != nil {
		log.Printf("[ERROR] Request body did not parse as expected: %v, body %v", err, req.Body)
		http.Error(out, "Malformed input, just send JSON with a nickname field", http.StatusBadRequest)
		return
	}

	row := s.SupaNormie.GetSiteRow(req.Context(), jwt, body.SiteId)
	if row == nil {
		http.Error(out, "Unable to query Supabase for site row", http.StatusInternalServerError)
		return
	}

	worked := s.BunnyAdmin.AddCustomHostname(req.Context(), row.PullZoneId, body.Hostname)
	if !worked {
		http.Error(out, "Unable to add new hostname", http.StatusInternalServerError)
		return
	}

	worked = s.BunnyAdmin.SetUpFreeCertificate(req.Context(), body.Hostname)
	if !worked {
		// roll back the new hostname if we can't finish the job
		worked = s.BunnyAdmin.RemoveCustomHostname(req.Context(), row.PullZoneId, body.Hostname)
		if !worked {
			log.Printf("[ERROR] VERY SAD, could not remove %v hostname %v, manual cleanup required", row.PullZoneId, body.Hostname)
			http.Error(out, "Unable to acquire SSL certificate, INTERMEDIATE STATE", http.StatusInternalServerError)
			return
		}
		http.Error(out, "Unable to acquire SSL certificate", http.StatusInternalServerError)
		return
	}

	worked = s.BunnyAdmin.ForceSsl(req.Context(), row.PullZoneId, body.Hostname)
	if !worked {
		// roll back the new hostname if we can't finish the job
		worked = s.BunnyAdmin.RemoveCustomHostname(req.Context(), row.PullZoneId, body.Hostname)
		if !worked {
			log.Printf("[ERROR] VERY SAD, could not remove %v hostname %v, manual cleanup required", row.PullZoneId, body.Hostname)
			http.Error(out, "Unable to set up SSL enforcement, INTERMEDIATE STATE", http.StatusInternalServerError)
			return
		}
		http.Error(out, "Unable to set up SSL enforcement", http.StatusInternalServerError)
		return
	}

	// todo -- rollback the new hostname if the below don't work, too? Agh

	worked = s.SupaNormie.AddHostnameToSiteRow(req.Context(), jwt, body.SiteId, body.Hostname)
	if !worked {
		http.Error(out, "Unable to add new hostname to Supabase row", http.StatusInternalServerError)
		return
	}

	log.Printf("[INFO] All good, hostname %v created, responding 2xx...", body.Hostname)

	return
}

func (s Server) PostPush(out http.ResponseWriter, req *http.Request) {

	// no need to validate here, impossible to get this far if JWT is invalid
	jwt := req.Header.Get("Authorization")

	var body PostPushRequest

	err := json.NewDecoder(req.Body).Decode(&body)
	if err != nil {
		log.Printf("[ERROR] Request body did not parse as expected: %v, body %v", err, req.Body)
		http.Error(out, "Malformed input, just send JSON with siteid and sha", http.StatusBadRequest)
		return
	}

	// todo -- validate that sha is not empty and looks like a real SHA

	worked := s.SupaNormie.UpdateDeployedSha(req.Context(), jwt, body.SiteId, body.Sha)
	if !worked {
		http.Error(out, "Unable to update site row in Supabase", http.StatusInternalServerError)
		return
	}

	row := s.SupaNormie.GetSiteRow(req.Context(), jwt, body.SiteId)
	if row == nil {
		http.Error(out, "Unable to query Supabase for site row", http.StatusInternalServerError)
		return
	}

	worked2 := s.BunnyAdmin.PurgeCache(req.Context(), row.PullZoneId)
	if !worked2 {
		http.Error(out, "Unable to purge pull zone cache", http.StatusInternalServerError)
		return
	}

	log.Printf("[INFO] All good, pull zone for site %v purged, responding 2xx...", body.SiteId)

	return
}
