package api

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

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/jwtauth/v5"
	"github.com/spf13/cobra"
)

var ApiCmd = &cobra.Command{
	Use:   "api",
	Short: "api - handles administrative tasks regarding bunny and supabase",
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
			"CORS_ALLOWED_ORIGIN",
			"PERMISSIVE_MODE",
			"JWT_SECRET",
			"SUPABASE_URL",
			"SUPABASE_ANON_KEY",
			"SUPABASE_SERVICE_KEY",
			"BUNNY_URL",
			"BUNNY_API_KEY",
		}

		config, err := util.GetEnvConfigs(configNames)
		if err != nil {
			log.Fatalf("[ERROR] Could not parse configs from environment: %v", err)
		}

		// ------------------------------------------------------------------------

		log.Printf("[INFO] Setting up middlewares...")

		corsOptions := cors.Options{
			AllowedOrigins:   []string{config["CORS_ALLOWED_ORIGIN"]},
			AllowedMethods:   []string{"POST", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
			AllowCredentials: true,
		}

		// as soon as supabase supports RS256 / asymmetric JWT encryption, get
		// this out of here and replace with the public key just for validation
		// https://github.com/orgs/supabase/discussions/4059
		jwtSecret := jwtauth.New("HS256", []byte(config["JWT_SECRET"]), nil)

		// ------------------------------------------------------------------------

		log.Printf("[INFO] Setting up bunny and supabase clients...")

		sup := SupabaseClient{
			SupabaseUrl:        config["SUPABASE_URL"],
			SupabaseAnonKey:    config["SUPABASE_ANON_KEY"],
			SupabaseServiceKey: config["SUPABASE_SERVICE_KEY"],
		}

		bun := BunnyClient{
			BunnyUrl:       config["BUNNY_URL"],
			BunnyAccessKey: config["BUNNY_API_KEY"],
		}

		s := Server{sup, bun}

		// ------------------------------------------------------------------------

		log.Printf("[INFO] Registering middlewares and handlers...")

		r := chi.NewRouter()

		r.Use(middleware.Recoverer)
		r.Use(middleware.Logger)
		r.Use(middleware.AllowContentType("application/json"))
		r.Use(middleware.Timeout(1 * time.Minute))
		r.Use(cors.Handler(corsOptions))
		r.Use(jwtauth.Verifier(jwtSecret))
		r.Use(util.CheckJwtMiddleware((config["PERMISSIVE_MODE"] == "true"), false))
		r.Use(util.CheckReadOnlyMiddleware(config["PERMISSIVE_MODE"] == "true"))

		r.Post("/site", s.CreateSite)
		r.Post("/hostname", s.AddHostname)

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
