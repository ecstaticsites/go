package git

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"text/template"

	"github.com/carlmjohnson/requests"
)

type Middlewarer struct {
	SupabaseUrl     string
	SupabaseAnonKey string
}

type SiteConfig struct {
	IndexPath    string `json:"index_path"`
	StorageToken string `json:"storage_token"`
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

			jwt := req.Header.Get("Authorization")

			repoName := strings.Split(req.URL.Path, "/")[1]

			// todo, replace all the below with a supabase_normie client call, sheesh

			var config []SiteConfig

			err := requests.
				URL(m.SupabaseUrl).
				Path("/rest/v1/site").
				Param("select", "index_path,storage_token").
				Param("id", fmt.Sprintf("eq.%s", repoName)).
				Header("apikey", m.SupabaseAnonKey).
				Header("Authorization", jwt).
				ToJSON(&config).
				Fetch(req.Context())

			if err != nil {
				http.Error(out, fmt.Sprintf("Error occurred querying sites from supabase, dying: %v", err), http.StatusInternalServerError)
				return
			}

			if len(config) == 0 {
				http.Error(out, fmt.Sprintf("No result rows from supabase for site ID %v (possibly RLS unauthorized?)", repoName), http.StatusUnauthorized)
				return
			}

			if len(config) > 1 {
				http.Error(out, fmt.Sprintf("Too many rows from supabase, what do I do: %v", config), http.StatusInternalServerError)
				return
			}

			log.Printf("site ID found for repo %s, config %s", repoName, config[0])

			hookValues := HookValues{
				SiteId:        repoName,
				SiteSubDir:    path.Dir(config[0].IndexPath),
				StorageUrl:    "ftp://storage.bunnycdn.com:21/",
				StorageName:   repoName, // we work hard so storage name == pull zone name == site ID
				StorageToken:  config[0].StorageToken,
				PurgeCacheUrl: "http://api.default:8080/purge", // todo, maybe shouldn't hardcode this
				PurgeCacheJwt: jwt,
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
