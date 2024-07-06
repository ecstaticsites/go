package intake

import (
	"log"

	"github.com/DmitriyVTitov/size"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

type Intaker struct {
	msgChannel   chan []byte
	influxWriter api.WriteAPI
}

func (i Intaker) Consume() {

	// Create go proc for reading and logging errors
	go func() {
		for err := range i.influxWriter.Errors() {
			log.Printf("[ERROR] Could not write to influxdb: %v\n", err)
		}
	}()

	for message := range i.msgChannel {
		bunny, err := stringToBunnyLog(message)
		if err != nil {
			log.Printf("[ERROR] Could not parse log as JSON: %v\n", err)
			continue
		}

		enriched := Enrich(bunny)
		point := enriched.Point()

		log.Printf("[INFO] Writing point of size %v to measurement %v", size.Of(point), bunny.Host)

		// we're using the nonblocking client so this never errors (see above)
		i.influxWriter.WritePoint(point)
	}
}
