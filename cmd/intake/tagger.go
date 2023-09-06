package intake

import (
	"log"
	"time"

	"github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/mileusna/useragent"
	"github.com/oschwald/geoip2-golang"
	//"zgo.at/isbot"
)

// Tagger is responsible for turning a BunnyLog into a point for influx
// with all the necessary tags, timestamps, etc
type Tagger struct {
	geoClient *geoip2.Reader
}

func (t Tagger) Device(bunny BunnyLog) (string, string) {
	ua := useragent.Parse(bunny.UserAgent)
	if ua.Mobile {
		return "device", "Mobile"
	} else if ua.Tablet {
		return "device", "Tablet"
	} else if ua.Desktop {
		return "device", "Desktop"
	} else {
		return "device", "Unknown"
	}
}

func (t Tagger) Browser(bunny BunnyLog) (string, string) {
	ua := useragent.Parse(bunny.UserAgent)
	return "browser", ua.Name
}

func (t Tagger) Os(bunny BunnyLog) (string, string) {
	ua := useragent.Parse(bunny.UserAgent)
	return "os", ua.OS
}

func (t Tagger) Country(bunny BunnyLog) (string, string) {
	record, err := t.geoClient.Country(bunny.RemoteIp)
	if err != nil {
		log.Printf("Unable to get country for IP %v: %w", bunny.RemoteIp, err)
		return "", ""
	}
	return "country", record.Country.Names["en"]
}

func (t Tagger) StatusCode(bunny BunnyLog) (string, string) {
	return "statuscode", bunny.StatusCode
}

func (t Tagger) Path(bunny BunnyLog) (string, string) {
	return "path", bunny.Url.Path
}

func (t Tagger) Referrer(bunny BunnyLog) (string, string) {
	return "", ""
}

func (t Tagger) FileType(bunny BunnyLog) (string, string) {
	return "", ""
}

func (t Tagger) IsProbablyBot(bunny BunnyLog) (string, string) {
	return "", ""
}

func (t Tagger) Point(bunny BunnyLog) *write.Point {

	tagFuncSlice := []func(bunny BunnyLog) (string, string){
		t.Device,
		t.Browser,
		t.Os,
		t.Country,
		t.StatusCode,
		t.Path,
		t.Referrer,
		t.FileType,
		t.IsProbablyBot,
	}

	tags := map[string]string{}
	for _, f := range tagFuncSlice {
		name, val := f(bunny)
		tags[name] = val
	}

	return influxdb2.NewPoint(
		// metric name
		bunny.Url.Host,
		// tags
		tags,
		// fields
		map[string]interface{}{"hits": 1},
		// ts
		time.UnixMilli(bunny.Timestamp),
	)
}
