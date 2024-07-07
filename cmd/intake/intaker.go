package intake

import (
	"log"
	"context"
	"time"

	//"github.com/DmitriyVTitov/size"
	"github.com/influxdata/influxdb-client-go/v2/api"
	ch "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type Intaker struct {
	msgChannel   chan []byte
	influxWriter api.WriteAPI
	clickConn    ch.Conn
}

func (i Intaker) Consume(ctx context.Context) {

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	batch, err := i.clickConn.PrepareBatch(ctx, "INSERT INTO accesslog")
	if err != nil {
	  log.Printf("[ERROR] Could not prepare clickhouse batch: %v\n", err)
	}

	for {
	  select {
	    case <- ticker.C:
	    	// send the batch to CH
	      err = batch.Send()
				if err != nil {
					log.Printf("[ERROR] Could not parse log as JSON: %v\n", err)
					continue
				}
				// then reset the batch
				batch, err = i.clickConn.PrepareBatch(ctx, "INSERT INTO accesslog")
				if err != nil {
				  log.Printf("[ERROR] Could not prepare clickhouse batch: %v\n", err)
				}
	    case message := <- i.msgChannel:
	    	// parse the log from Bunny
				bunny, err := stringToBunnyLog(message)
				if err != nil {
					log.Printf("[ERROR] Could not parse log as JSON: %v\n", err)
					continue
				}
				// then add it to the CH batch
	      err = addToBatch(batch, Enrich(bunny))
				if err != nil {
					log.Printf("[ERROR] Could not add access log to CH batch: %v\n", err)
					continue
				}
	  }
	}
	//log.Printf("[INFO] Writing point of size %v to measurement %v", size.Of(point), bunny.Host)
}

func addToBatch(batch ch.Batch, enriched EnrichedLog) error {
	//
	// schema:
	//
  // StatusCode      UInt16,
  // StatusCategory  FixedString(3),
  // Timestamp       DateTime,
  // BytesSent       UInt32,
  // RemoteIp        IPv4,
  // Host            LowCardinality(String),
  // Path            String,
  // Referrer        String,
  // Device          LowCardinality(String),
  // Browser         LowCardinality(String),
  // Os              LowCardinality(String),
  // Country         LowCardinality(String),
  // FileType        LowCardinality(String),
  // IsProbablyBot   Bool
  return batch.Append(
		enriched.StatusCode,
		enriched.StatusCategory,
		enriched.Timestamp,
		enriched.BytesSent,
		enriched.RemoteIp,
		enriched.Host,
		enriched.Path,
		enriched.Referrer,
		enriched.Device,
		enriched.Browser,
		enriched.Os,
		enriched.Country,
		enriched.FileType,
		enriched.IsProbablyBot,
	)
}
