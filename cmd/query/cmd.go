package query

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

	ch "github.com/ClickHouse/clickhouse-go/v2"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	promHttpMetrics "github.com/slok/go-http-metrics/metrics/prometheus"
	promHttpMiddleware "github.com/slok/go-http-metrics/middleware"
	promHttpStd "github.com/slok/go-http-metrics/middleware/std"
)

var QueryCmd = &cobra.Command{
	Use:   "query",
	Short: "query - starts an API server which can be queried for data",
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
			"METRICS_LISTENER_PORT",
			"CLICKHOUSE_URL",
			"CLICKHOUSE_DATABASE",
			"CORS_ALLOWED_ORIGIN",
			"PERMISSIVE_MODE",
			"JWT_SECRET",
		}

		config, err := util.GetEnvConfigs(configNames)
		if err != nil {
			log.Fatalf("[ERROR] Could not parse configs from environment: %v", err)
		}

		// ------------------------------------------------------------------------

		log.Printf("[INFO] Setting up middlewares...")

		corsOptions := cors.Options{
			AllowedOrigins:   []string{config["CORS_ALLOWED_ORIGIN"]},
			AllowedMethods:   []string{"GET", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
			AllowCredentials: true,
		}

		// as soon as supabase supports RS256 / asymmetric JWT encryption, get
		// this out of here and replace with the public key just for validation
		// https://github.com/orgs/supabase/discussions/4059
		jwtSecret := jwtauth.New("HS256", []byte(config["JWT_SECRET"]), nil)

		promMiddleware := promHttpMiddleware.New(promHttpMiddleware.Config{
			Recorder: promHttpMetrics.NewRecorder(promHttpMetrics.Config{}),
		})

		// ------------------------------------------------------------------------

		log.Printf("[INFO] Creating ClickHouse DB connection and consumer...")

		clickhouseConn, err := ch.Open(&ch.Options{
			Addr: []string{config["CLICKHOUSE_URL"]},
			Auth: ch.Auth{Database: config["CLICKHOUSE_DATABASE"]},
		})
		if err != nil {
			log.Fatalf("[ERROR] Could not create clickhouse connection: %v\n", err)
		}

		q := Query{clickhouseConn}

		// ------------------------------------------------------------------------

		log.Printf("[INFO] Registering middlewares and handlers...")

		r := chi.NewRouter()

		r.Use(middleware.Recoverer)
		r.Use(middleware.Logger)
		r.Use(middleware.AllowContentType("application/json"))
		r.Use(middleware.Timeout(time.Second))
		r.Use(cors.Handler(corsOptions))
		r.Use(jwtauth.Verifier(jwtSecret))
		r.Use(util.CheckJwtMiddleware((config["PERMISSIVE_MODE"] == "true"), false))
		r.Use(util.CheckHostnameMiddleware(config["PERMISSIVE_MODE"] == "true"))
		r.Use(promHttpStd.HandlerProvider("", promMiddleware))

		r.Get("/query", q.HandleQuery)

		// ------------------------------------------------------------------------

		log.Printf("[INFO] Trying to listen %v...", config["HTTP_LISTENER_PORT"])

		primary := http.Server{
			Addr:    fmt.Sprintf(":%v", config["HTTP_LISTENER_PORT"]),
			Handler: r,
		}

		metrics := http.Server{
			Addr:    fmt.Sprintf(":%v", config["METRICS_LISTENER_PORT"]),
			Handler: promhttp.Handler(),
		}

		go func() {
			err := primary.ListenAndServe()
			if err != nil {
				log.Fatalf("[ERROR] Primary server could not start: %v", err)
			}
		}()

		go func() {
			err := metrics.ListenAndServe()
			if err != nil {
				log.Fatalf("[ERROR] Metrics server could not start: %v", err)
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

		err = metrics.Shutdown(ctxTimeout)
		if err != nil {
			log.Fatalf("[ERROR] Could not cleanly shut down metrics server: %v", err)
		}

		log.Printf("[INFO] ALL DONE, GOODBYE")
	},
}
