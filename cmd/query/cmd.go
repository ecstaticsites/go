package query

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

	"github.com/prometheus/client_golang/prometheus/promhttp"
	promHttpMetrics "github.com/slok/go-http-metrics/metrics/prometheus"
	promHttpMiddleware "github.com/slok/go-http-metrics/middleware"
	promHttpStd "github.com/slok/go-http-metrics/middleware/std"
)

var QueryCmd = &cobra.Command{
	Use:   "query",
	Short: "query - starts an API server which can be queried for data",
	Run: func(cmd *cobra.Command, args []string) {

		log.Printf("STARTING")

		// set up channel to handle graceful shutdown
		done := make(chan os.Signal, 1)
		signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

		httpPort, err := util.GetEnvConfig("HTTP_LISTENER_PORT")
		if err != nil {
			log.Fatalf("Unable to get http listener port from environment: %v", err)
		}

		metricsPort, err := util.GetEnvConfig("METRICS_LISTENER_PORT")
		if err != nil {
			log.Fatalf("Unable to get metrics listener port from environment: %v", err)
		}

		influxUrl, err := util.GetEnvConfig("INFLUX_URL")
		if err != nil {
			log.Fatalf("Unable to get influx location from environment: %v", err)
		}

		influxDbName, err := util.GetEnvConfig("INFLUX_DB_NAME")
		if err != nil {
			log.Fatalf("Unable to get influx DB name from environment: %v", err)
		}

		i := InfluxClient{influxUrl, influxDbName}

		log.Printf("INFLUX PARAMS PARSED, SET UP STRUCT")

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

		// as soon as supabase supports RS256 / asymmetric JWT encryption, get this
		// out of here and replace with the public key just for validation
		// https://github.com/orgs/supabase/discussions/4059
		jwtSecret := jwtauth.New("HS256", []byte(jwtSecretStr), nil)

		authOptions := util.AuthOptions{
			Permissive:      (permissiveStr == "true"),
			EnforceHostname: true,
		}

		promMiddleware := promHttpMiddleware.New(promHttpMiddleware.Config{
			Recorder: promHttpMetrics.NewRecorder(promHttpMetrics.Config{}),
		})

		r := chi.NewRouter()

		r.Use(middleware.Recoverer)
		r.Use(middleware.Logger)
		r.Use(middleware.AllowContentType("application/json"))
		r.Use(middleware.Timeout(time.Second))
		r.Use(cors.Handler(corsOptions))
		r.Use(jwtauth.Verifier(jwtSecret))
		r.Use(authOptions.Authenticator)
		r.Use(promHttpStd.HandlerProvider("", promMiddleware))

		r.Get("/query", i.HandleQuery)

		log.Printf("MIDDLEWARES SET UP, STARTING SERVERS")

		primary := http.Server{Addr: fmt.Sprintf(":%v", httpPort), Handler: r}
		metrics := http.Server{Addr: fmt.Sprintf(":%v", metricsPort), Handler: promhttp.Handler()}

		go func() {
			log.Printf("PRIMARY SERVER LISTENING ON %v", httpPort)
			err := primary.ListenAndServe()
			if err != nil {
				log.Fatalf("Error while serving primary handler: %v", err)
			}
		}()

		go func() {
			log.Printf("METRICS SERVER LISTENING ON %v", metricsPort)
			err := metrics.ListenAndServe()
			if err != nil {
				log.Fatalf("Error while serving metrics handler: %v", err)
			}
		}()

		// block here until we get some sort of interrupt or kill
		<-done

		log.Printf("GOT SIGNAL TO DIE, cleaning up...")

		ctx := context.Background()
		ctxTimeout, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()

		err = primary.Shutdown(ctxTimeout)
		if err != nil {
			log.Fatalf("Could not cleanly shut down primary server: %v", err)
		}

		err = metrics.Shutdown(ctxTimeout)
		if err != nil {
			log.Fatalf("Could not cleanly shut down metrics server: %v", err)
		}

		log.Printf("ALL DONE, GOODBYE")
	},
}
