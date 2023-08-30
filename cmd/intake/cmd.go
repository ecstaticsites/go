package intake

import (
	"fmt"
	"log"

	"cbnr/util"

	"github.com/influxdata/influxdb-client-go/v2"
	"github.com/oschwald/geoip2-golang"
	"github.com/spf13/cobra"
	"gopkg.in/mcuadros/go-syslog.v2"
)

var IntakeCmd = &cobra.Command{
	Use:   "intake",
	Short: "intake - starts listenting for UDP syslog messages on a port",
	Run: func(cmd *cobra.Command, args []string) {

		log.Printf("STARTING")

		syslogPort, err := util.GetEnvConfig("SYSLOG_LISTENER_PORT")
		if err != nil {
			log.Fatalf("Unable to get syslog port from environment: %w", err)
		}

		// buffer for the messages from UDP port, no max size I think
		channel := make(syslog.LogPartsChannel)
		handler := syslog.NewChannelHandler(channel)

		server := syslog.NewServer()
		server.SetFormat(syslog.RFC5424)
		server.SetHandler(handler)
		server.ListenUDP(fmt.Sprintf("0.0.0.0:%s", syslogPort))
		server.Boot()

		log.Printf("SERVER BOOTED")

		influxUrl, err := util.GetEnvConfig("INFLUX_URL")
		if err != nil {
			log.Fatalf("Unable to get influx location from environment: %w", err)
		}

		influxDbName, err := util.GetEnvConfig("INFLUX_DB_NAME")
		if err != nil {
			log.Fatalf("Unable to get influx DB name from environment: %w", err)
		}

		// Empty value in auth parameter for an unauthenticated server
		influxClient := influxdb2.NewClient(influxUrl, "")

		// Empty value for org name (not used)
		// Second parameter is database/retention-policy (no slash => default retention)
		influxWriter := influxClient.WriteAPIBlocking("", influxDbName)

		log.Printf("INFLUX CLIENT INITTED")

		mmdbPath, err := util.GetEnvConfig("MMDB_PATH")
		if err != nil {
			log.Fatalf("Unable to get GeoIP DB path from environment: %w", err)
		}

		geoClient, err := geoip2.Open(mmdbPath)
		if err != nil {
			log.Fatalf("Could not create geoIP database: %w", err)
		}

		log.Printf("GEOLITE DATABASE OPENED")

		intaker := Intaker{channel, influxWriter, geoClient}

		go intaker.Consume()

		log.Printf("GOROUTINE STARTED")

		server.Wait()
	},
}
