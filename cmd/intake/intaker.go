package intake

import (
	"log"

	"github.com/DmitriyVTitov/size"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/oschwald/geoip2-golang"
	"gopkg.in/mcuadros/go-syslog.v2"
)

type Intaker struct {
	syslogChannel syslog.LogPartsChannel
	inflixWriter  api.WriteAPI
	geoClient     *geoip2.Reader
}

func (i Intaker) Consume() {

	tagger := Tagger{i.geoClient}

	// Create go proc for reading and logging errors
	go func() {
		for err := range i.inflixWriter.Errors() {
			log.Printf("Error writing to influxdb: %v\n", err)
		}
	}()

	for logParts := range i.syslogChannel {
		if message, ok := logParts["message"]; ok {

			bunny, err := stringToBunnyLog(message.(string))
			if err != nil {
				log.Printf("Parse error: %v\n", err)
				continue
			}

			point := tagger.Point(bunny)

			log.Printf("Writing point of size %v to measurement %v", size.Of(point), bunny.Url.Host)

			// we're using the nonblocking client so this never errors (see above)
			i.inflixWriter.WritePoint(point)
		}
	}
}
