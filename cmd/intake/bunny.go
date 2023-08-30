package intake

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/mileusna/useragent"
	"github.com/oschwald/geoip2-golang"
)

type BunnyLog struct {
	CacheStatus     string
	StatusCode      string
	Timestamp       int64
	BytesSent       int64
	PullZoneId      string
	RemoteIp        net.IP
	RefererUrl      *url.URL
	Url             *url.URL
	EdgeLocation    string
	UserAgent       string
	UniqueRequestId string
	CountryCode     string
}

func stringToBunnyLog(input string) (BunnyLog, error) {

	parts := strings.Split(input, "|")
	if len(parts) != 12 {
		return BunnyLog{}, fmt.Errorf("Invalid bunny log format: %v", input)
	}

	intTimestamp, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return BunnyLog{}, fmt.Errorf("Invalid timestamp format: %v", parts[2])
	}

	intBytesSent, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return BunnyLog{}, fmt.Errorf("Invalid bytes sent format: %v", parts[3])
	}

	ipaddr := net.ParseIP(parts[5])
	if ipaddr == nil {
		return BunnyLog{}, fmt.Errorf("Invalid IP address: %v", parts[5])
	}

	refUrl, err := url.Parse(parts[6])
	if err != nil {
		return BunnyLog{}, fmt.Errorf("Invalid referrer URL: %v", parts[6])
	}

	mainUrl, err := url.Parse(parts[7])
	if err != nil {
		return BunnyLog{}, fmt.Errorf("Invalid main URL: %v", parts[7])
	}

	return BunnyLog{
		CacheStatus:     parts[0],
		StatusCode:      parts[1],
		Timestamp:       intTimestamp,
		BytesSent:       intBytesSent,
		PullZoneId:      parts[4],
		RemoteIp:        ipaddr,
		RefererUrl:      refUrl,
		Url:             mainUrl,
		EdgeLocation:    parts[8],
		UserAgent:       parts[9],
		UniqueRequestId: parts[10],
		CountryCode:     parts[11],
	}, nil
}

func tagsFromUserAgent(input string) map[string]string {

	ua := useragent.Parse(input)

	device := "Unknown"
	if ua.Mobile {
		device = "Mobile"
	} else if ua.Tablet {
		device = "Tablet"
	} else if ua.Desktop {
		device = "Desktop"
	}

	return map[string]string{
		"browser": ua.Name,
		"os":      ua.OS,
		"device":  device,
	}
}

func tagsFromIpAddress(geoClient *geoip2.Reader, ipaddr net.IP) map[string]string {
	record, err := geoClient.Country(ipaddr)
	if err != nil {
		log.Printf("Unable to get country for IP %v: %w", ipaddr, err)
		return nil
	}
	return map[string]string{
		"country": record.Country.Names["en"],
	}
}

func tagsFromBunnyLog(input BunnyLog) map[string]string {
	return map[string]string{
		"path":       input.Url.Path,
		"statuscode": input.StatusCode,
	}
}

func pointFromBunnyLog(input BunnyLog, tagMaps ...map[string]string) *write.Point {

	allTags := map[string]string{}
	for _, tagMap := range tagMaps {
		for k, v := range tagMap {
			allTags[k] = v
		}
	}

	return influxdb2.NewPoint(
		// metric name
		input.Url.Host,
		// tags
		allTags,
		// fields
		map[string]interface{}{"hits": 1},
		// ts
		time.UnixMilli(input.Timestamp),
	)
}
