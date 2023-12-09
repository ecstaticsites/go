package git

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"cbnr/util"

	"github.com/spf13/cobra"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/jwtauth/v5"

	"github.com/asim/git-http-backend/server"

	"github.com/carlmjohnson/requests"
)

type Storage struct {
	Name  string `json:"storage_name"`
	Token string `json:"storage_token"`
}

func GitInitMiddleware(next http.Handler) http.Handler {
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

// gets the site ID from the URL path, then makes a query to supabase (using
// the JWT gotten from BasicAuthJwtVerifier) to make sure the user actually owns
// that site ID and can push there, then uses the storage name and pass to create
// a post-receive hook in the repo created by GitInitMiddleware
func RenderPostReceiveHookMiddleware(next http.Handler) http.Handler {
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
			URL("https://ewwccbgjnulfgcvfrsvj.supabase.co").
			Path("/rest/v1/site").
			Param("select", "storage_name,storage_token").
			Param("id", fmt.Sprintf("eq.%s", repoName)).
			Header("apikey", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImV3d2NjYmdqbnVsZmdjdmZyc3ZqIiwicm9sZSI6ImFub24iLCJpYXQiOjE2OTM1ODE2ODUsImV4cCI6MjAwOTE1NzY4NX0.gI3YdNSC5GMkda2D2QPRMvnBdaMOS2ynfFKxis5-WKs").
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

		hookPath := fmt.Sprintf("/tmp/%s/.git/hooks/post-receive", repoName)
		hookData := []byte("#!/bin/sh\necho 'got it thank you'\ntouch /tmp/got\n")
		err = os.WriteFile(hookPath, hookData, 0777)
		if err != nil {
			log.Printf("THIS DID NOT WORK: %s", err)
		}

		// user is allowed to push to this site ID, pass it through
		next.ServeHTTP(out, req)
		return
	})
}

var GitCmd = &cobra.Command{
	Use:   "git",
	Short: "git - handles administrative tasks regarding bunny and supabase",
	Run: func(cmd *cobra.Command, args []string) {

		log.Printf("STARTING")

		// set up channel to handle graceful shutdown
		done := make(chan os.Signal, 1)
		signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

		httpPort, err := util.GetEnvConfig("HTTP_LISTENER_PORT")
		if err != nil {
			log.Fatalf("Unable to get http listener port from environment: %v", err)
		}

		permissiveStr, err := util.GetEnvConfig("PERMISSIVE_MODE")
		if err != nil {
			log.Fatalf("Unable to get permissive mode status from environment: %v", err)
		}

		permissive := permissiveStr == "true"

		jwtSecretStr, err := util.GetEnvConfig("JWT_SECRET")
		if err != nil {
			log.Fatalf("Unable to get JWT secret token from environment: %v", err)
		}

		// as soon as supabase supports RS256 / asymmetric JWT encryption, get this
		// out of here and replace with the public key just for validation
		// https://github.com/orgs/supabase/discussions/4059
		jwtSecret := jwtauth.New("HS256", []byte(jwtSecretStr), nil)

		r := chi.NewRouter()

		r.Use(middleware.Recoverer)
		r.Use(middleware.Logger)
		//r.Use(middleware.AllowContentType("application/json"))
		r.Use(middleware.Timeout(time.Second))
		r.Use(util.BasicAuthJwtVerifier(jwtSecret))
		r.Use(util.CheckJwtMiddleware(permissive, true))
		r.Use(GitInitMiddleware)
		r.Use(RenderPostReceiveHookMiddleware)

		r.Get("/*", server.Handler())
		r.Post("/*", server.Handler())

		log.Printf("MIDDLEWARES SET UP, WILL LISTEN ON %v...", httpPort)

		server := http.Server{Addr: fmt.Sprintf(":%v", httpPort), Handler: r}
		go server.ListenAndServe()

		log.Printf("HTTP SERVER STARTED IN GOROUTINE, waiting to die...")

		// block here until we get some sort of interrupt or kill
		<-done

		log.Printf("GOT SIGNAL TO DIE, cleaning up...")

		ctx := context.Background()
		ctxTimeout, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()

		err = server.Shutdown(ctxTimeout)
		if err != nil {
			log.Fatalf("Could not cleanly shut down http server: %v", err)
		}

		log.Printf("ALL DONE, GOODBYE")
	},
}
