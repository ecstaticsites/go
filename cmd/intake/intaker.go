package intake

import (
	"context"
	"log"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/DmitriyVTitov/size"
)

type Intaker struct {
	msgChannel chan []byte
	clickConn  ch.Conn
}

func (i Intaker) Consume(ctx context.Context) {

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	batch, err := i.clickConn.PrepareBatch(ctx, "INSERT INTO accesslog")
	if err != nil {
		log.Fatalf("[ERROR] Could not prepare new clickhouse batch: %v\n", err)
	}

	for {
		select {
		case <-ticker.C:
			// send the batch to CH
			rows := batch.Rows()
			if rows == 0 {
				// no rows means we'll skip this one
				continue
			}
			err = batch.Send()
			if err != nil {
				log.Printf("[ERROR] Could not send batch to clickhouse: %v", err)
				continue
			}
			// then reset the batch
			batch, err = i.clickConn.PrepareBatch(ctx, "INSERT INTO accesslog")
			if err != nil {
				log.Fatalf("[ERROR] Could not prepare new clickhouse batch: %v", err)
				continue
			}
			log.Printf("[INFO] Sent batch of %v logs to clickhouse and reset", rows)
		case message := <-i.msgChannel:
			// parse the log from Bunny
			bunny, err := stringToBunnyLog(message)
			if err != nil {
				log.Printf("[ERROR] Could not parse log as JSON: %v", err)
				continue
			}
			// do a little transformation
			enriched := Enrich(bunny)
			// then add it to the CH batch
			err = addToBatch(batch, enriched)
			if err != nil {
				log.Printf("[ERROR] Could not add log to CH batch: %v", err)
				continue
			}
			log.Printf("[INFO] Added log (size %v, measurement %v) to batch", size.Of(enriched), bunny.Host)
		}
	}
}

func addToBatch(batch ch.Batch, enriched EnrichedLog) error {
	// must match the order in the schema exactly
	return batch.Append(
		enriched.PullZoneId,
		enriched.Timestamp,
		enriched.BytesSent,
		enriched.StatusCode,
		enriched.StatusCategory,
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
