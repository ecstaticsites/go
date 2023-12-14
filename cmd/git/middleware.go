package git

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/carlmjohnson/requests"
)

type Middlewarer struct {
	SupabaseUrl     string
	SupabaseAnonKey string
}

type Storage struct {
	Name  string `json:"storage_name"`
	Token string `json:"storage_token"`
}

func (m Middlewarer) CreateGitInitMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(out http.ResponseWriter, req *http.Request) {

			// Only need to init the repository at the start of the "git push", which
			// always begins with this GET to /reponame/info/refs, so if this request
			// is not that, it must not be the start of a push
			if !(req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/info/refs")) {
				next.ServeHTTP(out, req)
				return
			}

			repoName := strings.Split(req.URL.Path, "/")[1]
			repoPath := "/tmp/" + repoName

			log.Printf("git init bare %s", repoPath)

			cmd := exec.Command("git", "init", repoPath)
			stdout, err := cmd.Output()

			if err != nil {
				log.Printf(err.Error())
				return
			}

			// Print the output
			log.Printf(string(stdout))

			next.ServeHTTP(out, req)
		})
	}
}

// gets the site ID from the URL path, then makes a query to supabase (using
// the JWT gotten from BasicAuthJwtVerifier) to make sure the user actually owns
// that site ID and can push there, then uses the storage name and pass to create
// a post-receive hook in the repo created by GitInitMiddleware
func (m Middlewarer) CreateGitHookMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(out http.ResponseWriter, req *http.Request) {

			// Only need to init the repository at the start of the "git push", which
			// always begins with this GET to /reponame/info/refs, so if this request
			// is not that, it must not be the start of a push
			if !(req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/info/refs")) {
				next.ServeHTTP(out, req)
				return
			}

			token := req.Header.Get("Authorization")

			repoName := strings.Split(req.URL.Path, "/")[1]

			var storage []Storage

			err := requests.
				URL(m.SupabaseUrl).
				Path("/rest/v1/site").
				Param("select", "storage_name,storage_token").
				Param("id", fmt.Sprintf("eq.%s", repoName)).
				Header("apikey", m.SupabaseAnonKey).
				Header("Authorization", token).
				ToJSON(&storage).
				Fetch(req.Context())

			if err != nil {
				http.Error(out, fmt.Sprintf("Error occurred querying sites from supabase, dying: %v", err), http.StatusInternalServerError)
				return
			}

			if len(storage) == 0 {
				http.Error(out, fmt.Sprintf("No result rows from supabase for site ID %v (possibly RLS unauthorized?)", repoName), http.StatusUnauthorized)
				return
			}

			if len(storage) > 1 {
				http.Error(out, fmt.Sprintf("Too many rows from supabase, what do I do: %v", storage), http.StatusInternalServerError)
				return
			}

			log.Printf("site ID found for repo %s, storage %s", repoName, storage[0])

			hookValues := HookValues{
				SiteId:       repoName,
				StorageHost:  "storage.bunnycdn.com",
				StorageName:  storage[0].Name,
				StorageToken: storage[0].Token,
			}

			hookPath := fmt.Sprintf("/tmp/%s/.git/hooks/post-receive", repoName)

			tpl, err := template.New("naaaaame").Parse(HookTemplate)
			if err != nil {
				http.Error(out, fmt.Sprintf("Unable to render post-receive hook template: %v", err), http.StatusInternalServerError)
				return
			}

			file, err := os.OpenFile(hookPath, os.O_RDWR|os.O_CREATE, 0777)
			if err != nil {
				http.Error(out, fmt.Sprintf("Could not create and open post-receive hook file: %v", err), http.StatusInternalServerError)
				return
			}

			defer file.Close()

			err = tpl.Execute(file, hookValues)
			if err != nil {
				http.Error(out, fmt.Sprintf("Could not render template into hook file: %v", err), http.StatusInternalServerError)
				return
			}

			next.ServeHTTP(out, req)
			return
		})
	}
}
