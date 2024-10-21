package intake

import (
	"context"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ecstatic/util"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/spf13/cobra"
)

var IntakeCmd = &cobra.Command{
	Use:   "intake",
	Short: "intake - starts listenting for TCP syslog messages on a port",
	Run: func(cmd *cobra.Command, args []string) {

		log.Printf("[INFO] Starting up...")

		// ------------------------------------------------------------------------

		log.Printf("[INFO] Seeding randomness for generating IDs...")

		rand.Seed(time.Now().UnixNano())

		// ------------------------------------------------------------------------

		log.Printf("[INFO] Creating context...")

		ctx := context.Background()

		// ------------------------------------------------------------------------

		log.Printf("[INFO] Registering int handlers for graceful shutdown...")

		done := make(chan os.Signal, 1)
		signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

		// ------------------------------------------------------------------------

		log.Printf("[INFO] Getting configs from environment...")

		configNames := []string{
			"METRICS_LISTENER_PORT",
			"SYSLOG_LISTENER_PORT",
			"CLICKHOUSE_URL",
			"CLICKHOUSE_DATABASE",
		}

		config, err := util.GetEnvConfigs(configNames)
		if err != nil {
			log.Fatalf("[ERROR] Could not parse configs from environment: %v", err)
		}

		// ------------------------------------------------------------------------

		log.Printf("[INFO] Starting TCP on %v...", config["SYSLOG_LISTENER_PORT"])

		// buffer for the messages from intake port, no max size I think
		msgChan := make(chan []byte)

		listen := Listener{port: config["SYSLOG_LISTENER_PORT"], msgChan: msgChan}

		go listen.Listen()

		// ------------------------------------------------------------------------

		log.Printf("[INFO] Creating ClickHouse DB connection and consumer...")

		clickhouseConn, err := ch.Open(&ch.Options{
			Addr: []string{config["CLICKHOUSE_URL"]},
			Auth: ch.Auth{Database: config["CLICKHOUSE_DATABASE"]},
		})
		if err != nil {
			log.Fatalf("[ERROR] Could not create clickhouse connection: %v\n", err)
		}

		intaker := Intaker{msgChan, clickhouseConn}

		go intaker.Consume(ctx)

		// ------------------------------------------------------------------------

		log.Printf("[INFO] Listening! Main thread now waiting for interrupt...")

		<-done

		log.Printf("[INFO] Got signal to die, cleaning up...")

		// todo, proper cleanup, set deadline for TCP listener, close CH conn

		// err = server.Kill()
		// if err != nil {
		// 	log.Fatalf("Could not kill running intak listener: %v", err)
		// }

		log.Printf("[INFO] INTAKE LISTENER KILLED, SLEEPING FOR 1 SECOND")

		// terrible? Yes, but I can figure out how to actually make sure the parser
		// channel is empty later, here 1s is more than enough
		// time.Sleep(1 * time.Second)

		// // Force all unwritten data to be sent
		// influxWriter.Flush()
		// // Ensures background processes finishes
		// influxClient.Close()

		log.Printf("[INFO] INFLUX WRITER FLUSHED AND CLOSED")
	},
}
