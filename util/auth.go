package util

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/jwtauth/v5"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"golang.org/x/exp/slices"
	"github.com/carlmjohnson/requests"
)

type Body struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// a custom way of getting a JWT from an http.Request (this "custom way" being
// to log in to supabase using the basic auth creds included in the req). For
// use below in BasicAuthJwtVerifier, plugs into jwtauth.Verify
func GetTokenFromBasicAuth(req *http.Request) string {

	email, password, ok := req.BasicAuth()
	if !ok {
		log.Printf("Unable to parse basic auth credentials")
		return ""
	}

	body := Body{
		Email:    email,
		Password: password,
	}

	var user map[string]interface{}

	err := requests.
		URL("https://ewwccbgjnulfgcvfrsvj.supabase.co").
		Path("/auth/v1/token").
		Param("grant_type", "password").
		Header("apikey", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImV3d2NjYmdqbnVsZmdjdmZyc3ZqIiwicm9sZSI6ImFub24iLCJpYXQiOjE2OTM1ODE2ODUsImV4cCI6MjAwOTE1NzY4NX0.gI3YdNSC5GMkda2D2QPRMvnBdaMOS2ynfFKxis5-WKs").
		// no authorization header since this is the anon / signin request
		BodyJSON(&body).
		ToJSON(&user).
		Fetch(req.Context())

	if err != nil {
		log.Printf("Error authing with supabase: %v", err)
		return ""
	}

	// this is a hack! It's the only way of keeping this token (the encoded string,
	// not the Token object) around so we can use it in supabase calls later
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", user["access_token"].(string)))

	return user["access_token"].(string)
}

// derived from https://github.com/go-chi/jwtauth/blob/master/jwtauth.go
//
// like jwtauth.Verifier, but gets the token from logging in to Supabase with
// basic auth credentials instead of scanning headers/cookies for "Bearer $token"
func BasicAuthJwtVerifier(ja *jwtauth.JWTAuth) func(http.Handler) http.Handler {
	return jwtauth.Verify(ja, GetTokenFromBasicAuth)
}

// derived from https://github.com/go-chi/jwtauth/blob/master/jwtauth.go
//
// permissive -- don't check to see if the JWT "Verifier" middleware worked or errored
// basic -- on error, additionally sets a header which requests HTTP basic auth to be set
func CheckJwtMiddleware(permissive, basic bool) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(out http.ResponseWriter, req *http.Request) {

			if permissive {

				log.Printf("Permissive mode is enabled, not validating JWT tokens! SHOULD NOT SEE IN PROD")

			} else {

				token, _, err := jwtauth.FromContext(req.Context())

				if err != nil {
					if basic {
						out.Header().Add("WWW-Authenticate", "Basic")
					}
					http.Error(out, fmt.Sprintf("Unable to parse claims from JWT: %v", err), http.StatusUnauthorized)
					return
				}

				if (token == nil) || (jwt.Validate(token) != nil) {
					if basic {
						out.Header().Add("WWW-Authenticate", "Basic")
					}
					http.Error(out, fmt.Sprintf("Unable to validate JWT token"), http.StatusUnauthorized)
					return
				}
			}

			// token is authenticated / we've decided we don't care, pass it through
			next.ServeHTTP(out, req)
			return
		})
	}
}

// gets the desired hostname from the query params, then checks the JWT metadata
// to make sure the user is allowed to query that hostname
func CheckHostnameMiddleware(permissive bool) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(out http.ResponseWriter, req *http.Request) {

			if permissive {

				log.Printf("Permissive mode is enabled, not validating JWT tokens! SHOULD NOT SEE IN PROD")

			} else {

				// already checked for errors etc in the previous CheckJwtMiddleware
				_, claims, _ := jwtauth.FromContext(req.Context())

				hostname := req.URL.Query().Get("hostname")
				if hostname == "" {
					http.Error(out, "Query param 'hostname' not provided, quitting", http.StatusBadRequest)
					return
				}

				metadata, found1 := claims["app_metadata"]
				if !found1 {
					http.Error(out, "No 'app_metadata' found in JWT claims", http.StatusInternalServerError)
					return
				}

				metadataMap, ok1 := metadata.(map[string]interface{})
				if !ok1 {
					http.Error(out, "Claims 'app_metadata' could not be parsed as map", http.StatusInternalServerError)
					return
				}

				hostnames, found2 := metadataMap["hostnames"]
				if !found2 {
					http.Error(out, "No 'hostnames' field in JWT claims 'app_metadata'", http.StatusInternalServerError)
					return
				}

				hostnamesArray, ok2 := hostnames.([]interface{})
				if !ok2 {
					http.Error(out, "Metadata 'hostnames' field could not be parsed as array", http.StatusInternalServerError)
					return
				}

				hostnamesStringArray := []string{}
				for _, name := range hostnamesArray {
					nameString, ok3 := name.(string)
					if !ok3 {
						http.Error(out, "Item in metadata 'hostnames' array could not be parsed as string", http.StatusInternalServerError)
						return
					}
					hostnamesStringArray = append(hostnamesStringArray, nameString)
				}

				if !slices.Contains(hostnamesStringArray, hostname) {
					http.Error(out, fmt.Sprintf("User not authorized to query hostname %v", hostname), http.StatusUnauthorized)
					return
				}
			}

			// user is allowed to query this hostname, pass it through
			next.ServeHTTP(out, req)
			return
		})
	}
}
