package intake

import (
	"context"
	"log"

	"github.com/DmitriyVTitov/size"
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

	tagger := Tagger{i.geoClient}

	for logParts := range i.syslogChannel {
		if message, ok := logParts["message"]; ok {

			bunny, err := stringToBunnyLog(message.(string))
			if err != nil {
				log.Printf("Parse error: %w\n", err)
				continue
			}

			point := tagger.Point(bunny)

			log.Printf("Writing point of size %v to measurement %v", size.Of(point), bunny.Url.Host)

			err = i.influxClient.WritePoint(context.Background(), point)
			if err != nil {
				log.Printf("Write error: %w\n", err)
				continue
			}
		}
	}
}
