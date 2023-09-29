package serve

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

func (i InfluxClient) HandleQuery(out http.ResponseWriter, req *http.Request) {

	supaSite := "https://ewwccbgjnulfgcvfrsvj.supabase.co"

	// todo, this is gross. there must be a better way of defining and validating API spec

	siteId := req.URL.Query().Get("site")
	if siteId == "" {
		http.Error(out, "Query param 'site' not provided, quitting", http.StatusBadRequest)
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

	var results []map[string]string

	err = requests.
		URL(supaSite).
		Path("/rest/v1/alias").
		Param("select", "hostname").
		Param("site_id", fmt.Sprintf("eq.%s", siteId)).
		Param("is_default", fmt.Sprintf("eq.%s", "TRUE")).
		// anon key
		Header("apikey", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImV3d2NjYmdqbnVsZmdjdmZyc3ZqIiwicm9sZSI6ImFub24iLCJpYXQiOjE2OTM1ODE2ODUsImV4cCI6MjAwOTE1NzY4NX0.gI3YdNSC5GMkda2D2QPRMvnBdaMOS2ynfFKxis5-WKs").
		Header("Authorization", req.Header.Get("Authorization")).
		ToJSON(&results).
		Fetch(req.Context())

	if err != nil {
		http.Error(out, fmt.Sprintf("Supabase request failed: %v, response: %v", err, results), http.StatusBadRequest)
		return
	}

	if len(results) == 0 {
		http.Error(out, fmt.Sprintf("No result rows from supabase for site ID %v (possibly RLS unauthorized?)", siteId), http.StatusBadRequest)
	}

	if len(results) > 1 {
		http.Error(out, fmt.Sprintf("Too many rows from supabase, what do I do: %v", results), http.StatusBadRequest)
	}

	log.Printf("Results from supabase: %v", results)

	hostname := results[0]["hostname"]

	queryStr, err := i.BuildInfluxQuery(hostname, groupby, bucketby, tz, unixStart, unixEnd)
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

func (i InfluxClient) BuildInfluxQuery(hostname, groupby, bucketby, tz string, unixStart, unixEnd int) (string, error) {

	var query strings.Builder

	query.WriteString("select sum(hits)")

	query.WriteString(" ")
	query.WriteString(fmt.Sprintf("from \"%s\"", hostname))

	query.WriteString(" ")
	query.WriteString(fmt.Sprintf("where filetype = 'page' and time >= %ds and time <= %ds", unixStart, unixEnd))

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
