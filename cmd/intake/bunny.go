package intake

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
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
