package git

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cbnr/util"

	"github.com/asim/git-http-backend/server"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/jwtauth/v5"
	"github.com/spf13/cobra"
)

var GitCmd = &cobra.Command{
	Use:   "git",
	Short: "git - pretends to be a git server, then uploads files to CDN",
	Run: func(cmd *cobra.Command, args []string) {

		log.Printf("[INFO] Starting up...")

		// ------------------------------------------------------------------------

		log.Printf("[INFO] Seeding randomness for generating IDs...")

		rand.Seed(time.Now().UnixNano())

		// ------------------------------------------------------------------------

		log.Printf("[INFO] Registering int handlers for graceful shutdown...")

		done := make(chan os.Signal, 1)
		signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

		// ------------------------------------------------------------------------

		log.Printf("[INFO] Getting configs from environment...")

		configNames := []string{
			"HTTP_LISTENER_PORT",
			"PERMISSIVE_MODE",
			"JWT_SECRET",
			"SUPABASE_URL",
			"SUPABASE_ANON_KEY",
		}

		config, err := util.GetEnvConfigs(configNames)
		if err != nil {
			log.Fatalf("[ERROR] Could not parse configs from environment: %v", err)
		}

		// ------------------------------------------------------------------------

		log.Printf("[INFO] Setting up middlewares...")

		// as soon as supabase supports RS256 / asymmetric JWT encryption, get
		// this out of here and replace with the public key just for validation
		// https://github.com/orgs/supabase/discussions/4059
		jwtSecret := jwtauth.New("HS256", []byte(config["JWT_SECRET"]), nil)

		// ------------------------------------------------------------------------

		log.Printf("[INFO] Setting up custom git-middlewarer client...")

		mid := Middlewarer{
			SupabaseUrl:     config["SUPABASE_URL"],
			SupabaseAnonKey: config["SUPABASE_ANON_KEY"],
		}

		// ------------------------------------------------------------------------

		log.Printf("[INFO] Registering middlewares and handlers...")

		r := chi.NewRouter()

		r.Use(middleware.Recoverer)
		r.Use(middleware.Logger)
		r.Use(middleware.Timeout(60 * time.Second))
		r.Use(util.BasicAuthJwtVerifier(jwtSecret))
		r.Use(util.CheckJwtMiddleware((config["PERMISSIVE_MODE"] == "true"), true))
		r.Use(mid.CreateGitInitMiddleware())
		r.Use(mid.CreateGitHookMiddleware())

		r.Get("/*", server.Handler())
		r.Post("/*", server.Handler())

		// ------------------------------------------------------------------------

		log.Printf("[INFO] Trying to listen %v...", config["HTTP_LISTENER_PORT"])

		primary := http.Server{
			Addr:    fmt.Sprintf(":%v", config["HTTP_LISTENER_PORT"]),
			Handler: r,
		}

		go func() {
			err := primary.ListenAndServe()
			if err != nil {
				log.Fatalf("[ERROR] Primary server could not start: %v", err)
			}
		}()

		// ------------------------------------------------------------------------

		log.Printf("[INFO] Listening! Main thread now waiting for interrupt...")

		<-done

		log.Printf("[INFO] Got signal to die, cleaning up...")

		ctx := context.Background()
		ctxTimeout, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()

		err = primary.Shutdown(ctxTimeout)
		if err != nil {
			log.Fatalf("[ERROR] Could not cleanly shut down primary server: %v", err)
		}

		log.Printf("[INFO] ALL DONE, GOODBYE")
	},
}
