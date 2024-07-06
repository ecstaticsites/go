package intake

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cbnr/util"

	"github.com/influxdata/influxdb-client-go/v2"
	"github.com/spf13/cobra"
)

var IntakeCmd = &cobra.Command{
	Use:   "intake",
	Short: "intake - starts listenting for TCP syslog messages on a port",
	Run: func(cmd *cobra.Command, args []string) {

		log.Printf("STARTING")

		// set up channel to handle graceful shutdown
		done := make(chan os.Signal, 1)
		signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

		syslogPort, err := util.GetEnvConfig("SYSLOG_LISTENER_PORT")
		if err != nil {
			log.Fatalf("Unable to get intake port from environment: %v", err)
		}

		// buffer for the messages from intake port, no max size I think
		messages := make(chan []byte)

		listen := Listener{port: syslogPort, msgChan: messages}

		go listen.Listen()

		log.Printf("SERVER BOOTED, LISTENING TCP ON PORT %v", syslogPort)

		influxUrl, err := util.GetEnvConfig("INFLUX_URL")
		if err != nil {
			log.Fatalf("Unable to get influx location from environment: %v", err)
		}

		influxDbName, err := util.GetEnvConfig("INFLUX_DB_NAME")
		if err != nil {
			log.Fatalf("Unable to get influx DB name from environment: %v", err)
		}

		// using the NON-BLOCKING client
		// https://github.com/influxdata/influxdb-client-go#non-blocking-write-client
		// flushes every 1s or 1000 points, whichever comes first, and sets loglevel 2 (info)
		// default async retries is 5, we will want to tune (lower) this
		defaultOpts := influxdb2.DefaultOptions()
		influxOpts := defaultOpts.SetBatchSize(1000).SetFlushInterval(1000).SetPrecision(time.Second).SetLogLevel(2)

		// Empty value in auth parameter for an unauthenticated server
		influxClient := influxdb2.NewClientWithOptions(influxUrl, "", influxOpts)

		// Empty value for org name (not used)
		// Second parameter is database/retention-policy (no slash => default retention)
		influxWriter := influxClient.WriteAPI("", influxDbName)

		log.Printf("INFLUX CLIENT INITTED")

		intaker := Intaker{messages, influxWriter}

		go intaker.Consume()

		log.Printf("PARSER GOROUTINE STARTED, waiting to die...")

		// block here until we get some sort of interrupt or kill
		<-done

		log.Printf("GOT SIGNAL TO DIE, cleaning up...")

		// err = server.Kill()
		// if err != nil {
		// 	log.Fatalf("Could not kill running intak listener: %v", err)
		// }

		log.Printf("INTAKE LISTENER KILLED, SLEEPING FOR 1 SECOND")

		// terrible? Yes, but I can figure out how to actually make sure the parser
		// channel is empty later, here 1s is more than enough
		time.Sleep(1 * time.Second)

		// Force all unwritten data to be sent
		influxWriter.Flush()
		// Ensures background processes finishes
		influxClient.Close()

		log.Printf("INFLUX WRITER FLUSHED AND CLOSED")
	},
}
