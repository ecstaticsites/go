package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cbnr/util"

	"github.com/spf13/cobra"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/jwtauth/v5"
)

var ApiCmd = &cobra.Command{
	Use:   "api",
	Short: "api - handles administrative tasks regarding bunny and supabase",
	Run: func(cmd *cobra.Command, args []string) {

		log.Printf("STARTING")

		// set up channel to handle graceful shutdown
		done := make(chan os.Signal, 1)
		signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

		httpPort, err := util.GetEnvConfig("HTTP_LISTENER_PORT")
		if err != nil {
			log.Fatalf("Unable to get http listener port from environment: %v", err)
		}

		corsOrigin, err := util.GetEnvConfig("CORS_ALLOWED_ORIGIN")
		if err != nil {
			log.Fatalf("Unable to get CORS allowed origin from environment: %v", err)
		}

		corsOptions := cors.Options{
			AllowedOrigins:   []string{corsOrigin},
			AllowedMethods:   []string{"GET", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
			AllowCredentials: true,
		}

		permissiveStr, err := util.GetEnvConfig("PERMISSIVE_MODE")
		if err != nil {
			log.Fatalf("Unable to get permissive mode status from environment: %v", err)
		}

		jwtSecretStr, err := util.GetEnvConfig("JWT_SECRET")
		if err != nil {
			log.Fatalf("Unable to get JWT secret token from environment: %v", err)
		}

		jwtSecret := jwtauth.New("HS256", []byte(jwtSecretStr), nil)

		// as soon as supabase supports RS256 / asymmetric JWT encryption, get this
		// out of here and replace with the public key just for validation
		// https://github.com/orgs/supabase/discussions/4059
		authOptions := util.AuthOptions{
			Permissive:      (permissiveStr == "true"),
			EnforceHostname: false,
		}

		r := chi.NewRouter()

		r.Use(middleware.Recoverer)
		r.Use(middleware.Logger)
		r.Use(middleware.AllowContentType("application/json"))
		r.Use(middleware.Timeout(time.Second))
		r.Use(cors.Handler(corsOptions))
		r.Use(jwtauth.Verifier(jwtSecret))
		r.Use(authOptions.Authenticator)

		s := Server{SupabaseClient{"a"}, BunnyClient{}}

		r.Get("/new", s.CreateSite)

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
