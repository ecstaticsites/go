package intake

import (
	"context"
	"log"

	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/oschwald/geoip2-golang"
	"gopkg.in/mcuadros/go-syslog.v2"
)

type Intaker struct {
	syslogChannel syslog.LogPartsChannel
	influxClient  api.WriteAPIBlocking
	geoClient     *geoip2.Reader
}

func (i Intaker) Consume() {

	for logParts := range i.syslogChannel {
		if message, ok := logParts["message"]; ok {

			bunny, err := stringToBunnyLog(message.(string))
			if err != nil {
				log.Printf("Parse error: %w\n", err)
				continue
			}

			uaTags := tagsFromUserAgent(bunny.UserAgent)
			ipTags := tagsFromIpAddress(i.geoClient, bunny.RemoteIp)
			logTags := tagsFromBunnyLog(bunny)

			point := pointFromBunnyLog(bunny, uaTags, ipTags, logTags)

			err = i.influxClient.WritePoint(context.Background(), point)
			if err != nil {
				log.Printf("Write error: %w\n", err)
				continue
			}

			// replace with statsd point written, contains URL host, nothing else
			log.Printf("point written yo\n")
		}
	}
}
