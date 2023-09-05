package serve

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"cbnr/util"

	"github.com/spf13/cobra"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

var ServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "serve - starts an API server which can be queried for data",
	Run: func(cmd *cobra.Command, args []string) {

		log.Printf("STARTING")

		httpPort, err := util.GetEnvConfig("HTTP_LISTENER_PORT")
		if err != nil {
			log.Fatalf("Unable to get http listener port from environment: %w", err)
		}

		influxUrl, err := util.GetEnvConfig("INFLUX_URL")
		if err != nil {
			log.Fatalf("Unable to get influx location from environment: %w", err)
		}

		influxDbName, err := util.GetEnvConfig("INFLUX_DB_NAME")
		if err != nil {
			log.Fatalf("Unable to get influx DB name from environment: %w", err)
		}

		i := InfluxClient{influxUrl, influxDbName}

		log.Printf("INFLUX PARAMS PARSED, SET UP STRUCT")

		corsOrigin, err := util.GetEnvConfig("CORS_ALLOWED_ORIGIN")
		if err != nil {
			log.Fatalf("Unable to get CORS allowed origin from environment: %w", err)
		}

		r := chi.NewRouter()

		r.Use(middleware.Recoverer)
		r.Use(middleware.Logger)
		r.Use(middleware.AllowContentType("application/json"))
		r.Use(middleware.Timeout(time.Second))
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   []string{corsOrigin},
			AllowedMethods:   []string{"GET", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
			AllowCredentials: true,
		}))

		r.Get("/query", i.HandleQuery)

		log.Printf("MIDDLEWARES SET UP, WILL LISTEN ON %v...", httpPort)

		http.ListenAndServe(fmt.Sprintf(":%v", httpPort), r)
	},
}
