package query

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/carlmjohnson/requests"
	"golang.org/x/exp/slices"
)

type InfluxClient struct {
	InfluxUrl    string
	InfluxDbName string
}

// below should be const, but golang knows better
var VALIDGROUPBYS = []string{"", "browser", "os", "device", "country", "path", "statuscode"}
var VALIDBOTS = []string{"true", "false"}

func (i InfluxClient) HandleQuery(out http.ResponseWriter, req *http.Request) {

	// todo, this is gross. there must be a better way of defining and validating API spec

	hostname := req.URL.Query().Get("hostname")
	if hostname == "" {
		http.Error(out, "Query param 'hostname' not provided, quitting", http.StatusBadRequest)
		return
	}

	unixStartStr := req.URL.Query().Get("start")
	if unixStartStr == "" {
		http.Error(out, "Query param 'start' not provided, quitting", http.StatusBadRequest)
		return
	}

	unixStart, err := strconv.Atoi(unixStartStr)
	if err != nil {
		http.Error(out, "Query param 'start' is not a valid int, quitting", http.StatusBadRequest)
		return
	}

	unixEndStr := req.URL.Query().Get("end")
	if unixEndStr == "" {
		http.Error(out, "Query param 'end' not provided, quitting", http.StatusBadRequest)
		return
	}

	unixEnd, err := strconv.Atoi(unixEndStr)
	if err != nil {
		http.Error(out, "Query param 'end' is not a valid int, quitting", http.StatusBadRequest)
		return
	}

	includeBots := req.URL.Query().Get("bots")
	if !slices.Contains(VALIDBOTS, includeBots) {
		http.Error(out, fmt.Sprintf("Invalid groupby %s (try one of %v)", includeBots, VALIDBOTS), http.StatusBadRequest)
	}

	groupby := req.URL.Query().Get("groupby")
	if !slices.Contains(VALIDGROUPBYS, groupby) {
		http.Error(out, fmt.Sprintf("Invalid groupby %s (try one of %v)", groupby, VALIDGROUPBYS), http.StatusBadRequest)
	}

	tz := req.URL.Query().Get("tz")
	if false {
		// todo, some actual validation here, hashtag sql injection
		http.Error(out, fmt.Sprintf("Invalid timezone %s", tz), http.StatusBadRequest)
	}

	// derive time bucket size based on the requested time range (they do not get to pick)
	bucketby := "1d"
	if (unixEnd - unixStart) <= 86400 {
		bucketby = "1h"
	}

	// todo, possible DDOS protection, validate auth header exists and looks correct here

	queryStr, err := i.BuildInfluxQuery(hostname, includeBots, groupby, bucketby, tz, unixStart, unixEnd)
	if err != nil {
		http.Error(out, fmt.Sprintf("Unable to create valid query for influxdb: %w", err), http.StatusBadRequest)
		return
	}

	log.Printf("Query to influxdb: %s", queryStr)

	err = requests.
		URL(i.InfluxUrl).
		Path("/query").
		Param("db", i.InfluxDbName).
		Param("q", queryStr).
		Param("epoch", "s").
		ToWriter(out).
		Fetch(req.Context())

	if err != nil {
		http.Error(out, fmt.Sprintf("Query was unsuccessful: %w", err), http.StatusInternalServerError)
		return
	}

	return
}

func (i InfluxClient) BuildInfluxQuery(hostname, includeBots, groupby, bucketby, tz string, unixStart, unixEnd int) (string, error) {

	var query strings.Builder

	query.WriteString("select sum(hits)")

	query.WriteString(" ")
	query.WriteString(fmt.Sprintf("from \"%s\"", hostname))

	query.WriteString(" ")
	query.WriteString(fmt.Sprintf("where filetype = 'page'"))

	// if includeBots is true, then we want everything -- so no filter
	// todo -- isprobablybot is a string ?? should fix that
	if (includeBots == "false") {
		query.WriteString(" ")
		query.WriteString(fmt.Sprintf("and isprobablybot = 'false'"))
	}

	query.WriteString(" ")
	query.WriteString(fmt.Sprintf("and time >= %ds and time <= %ds", unixStart, unixEnd))

	if groupby != "" || bucketby != "" {
		query.WriteString(" ")
		query.WriteString("group by")
	}

	if groupby != "" {
		query.WriteString(" ")
		query.WriteString(groupby)
	}

	if groupby != "" && bucketby != "" {
		query.WriteString(",")
	}

	if bucketby != "" {
		query.WriteString(fmt.Sprintf("time(%s)", bucketby))
		query.WriteString(" ")
		query.WriteString("fill(0)")
	}

	if tz != "" {
		query.WriteString(" ")
		query.WriteString(fmt.Sprintf("tz('%s')", tz))
	}

	return query.String(), nil
}
