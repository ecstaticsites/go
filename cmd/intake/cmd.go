package intake

import (
	"fmt"
	"log"
	"time"
	"os"
	"os/signal"
	"syscall"

	"cbnr/util"

	"github.com/influxdata/influxdb-client-go/v2"
	"github.com/oschwald/geoip2-golang"
	"github.com/spf13/cobra"
	"gopkg.in/mcuadros/go-syslog.v2"
)

var IntakeCmd = &cobra.Command{
	Use:   "intake",
	Short: "intake - starts listenting for TCP or UDP syslog messages on a port",
	Run: func(cmd *cobra.Command, args []string) {

		log.Printf("STARTING")

		// set up channel to handle graceful shutdown
		done := make(chan os.Signal, 1)
		signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

		syslogUdpBool := false
		_, err := util.GetEnvConfig("SYSLOG_LISTENER_UDP")
		if err == nil {
			// no error means this env var was set! So let's use UDP
			syslogUdpBool = true
		}

		syslogPort, err := util.GetEnvConfig("SYSLOG_LISTENER_PORT")
		if err != nil {
			log.Fatalf("Unable to get syslog port from environment: %v", err)
		}

		// buffer for the messages from intake port, no max size I think
		channel := make(syslog.LogPartsChannel)
		handler := syslog.NewChannelHandler(channel)

		server := syslog.NewServer()
		server.SetFormat(syslog.RFC5424)
		server.SetHandler(handler)

		if syslogUdpBool {
			server.ListenUDP(fmt.Sprintf("0.0.0.0:%s", syslogPort))
		} else {
			server.ListenTCP(fmt.Sprintf("0.0.0.0:%s", syslogPort))
		}

		server.Boot()

		log.Printf("SERVER BOOTED, LISTENING UDP? %v", syslogUdpBool)

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
		influxOpts := influxdb2.DefaultOptions().SetBatchSize(1000).SetFlushInterval(1000).SetLogLevel(2)

		// Empty value in auth parameter for an unauthenticated server
		influxClient := influxdb2.NewClientWithOptions(influxUrl, "", influxOpts)

		// Empty value for org name (not used)
		// Second parameter is database/retention-policy (no slash => default retention)
		influxWriter := influxClient.WriteAPI("", influxDbName)

		log.Printf("INFLUX CLIENT INITTED")

		mmdbPath, err := util.GetEnvConfig("MMDB_PATH")
		if err != nil {
			log.Fatalf("Unable to get GeoIP DB path from environment: %v", err)
		}

		geoClient, err := geoip2.Open(mmdbPath)
		if err != nil {
			log.Fatalf("Could not create geoIP database: %v", err)
		}

		log.Printf("GEOLITE DATABASE OPENED")

		intaker := Intaker{channel, influxWriter, geoClient}

		go intaker.Consume()

		log.Printf("PARSER GOROUTINE STARTED, waiting to die...")

		// block here until we get some sort of interrupt or kill
		<-done

		log.Printf("GOT SIGNAL TO DIE, cleaning up...")

		err = server.Kill()
		if err != nil {
			log.Fatalf("Could not kill running syslog listener: %v", err)
		}

		log.Printf("SYSLOG LISTENER KILLED, SLEEPING FOR 1 SECOND")

		// terrible? Yes, but I can figure out how to actually make sure the parser
		// channel is empty later, here 1s is more than enough
		time.Sleep(1 * time.Second)

		// Force all unwritten data to be sent
		influxWriter.Flush()
		// Ensures background processes finishes
		influxClient.Close()

		log.Printf("INFLUX WRITER FLUSHED AND CLOSED")

		geoClient.Close()

		log.Printf("GEO CLIENT CLOSED")
	},
}
