package util

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/jwtauth/v5"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"golang.org/x/exp/slices"
)

type AuthOptions struct {
	Permissive      bool
	EnforceHostname bool
}

// derived from https://github.com/go-chi/jwtauth/blob/master/jwtauth.go#L161
func (a AuthOptions) Authenticator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(out http.ResponseWriter, req *http.Request) {

		if a.Permissive {

			log.Printf("Permissive mode is enabled, not validating JWT tokens! SHOULD NOT SEE IN PROD")

		} else {

			token, claims, err := jwtauth.FromContext(req.Context())

			if err != nil {
				http.Error(out, fmt.Sprintf("Unable to parse claims from JWT: %v", err), http.StatusUnauthorized)
				return
			}

			if (token == nil) || (jwt.Validate(token) != nil) {
				http.Error(out, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			// some endpoints, like /new, don't require an authorized hostname, just valid JWT
			if a.EnforceHostname {

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
		}

		// token is authenticated / we've decided we don't care, pass it through
		next.ServeHTTP(out, req)
	})
}
