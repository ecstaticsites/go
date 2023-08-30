package serve

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/exp/slices"
	"github.com/carlmjohnson/requests"
)

type InfluxClient struct {
	InfluxUrl    string
	InfluxDbName string
}

// below should be const, but golang knows better
var VALIDGROUPBYS = []string{"", "browser", "os", "device", "country", "path", "statuscode"}
var VALIDBUCKETBYS = []string{"", "1h", "1d", "1w"}

func (i InfluxClient) HandleQuery(out http.ResponseWriter, req *http.Request) {

	// todo, this will come from somewhere else (eg via resolved api key in headers) someday
	// JWT, gotrue, https://go-chi.io/#/pages/middleware?id=jwt-authentication
	site := "brandon-test-pull-zone.b-cdn.net"

	queryStr, err := i.BuildInfluxQuery(site, req.URL.Query())
	if err != nil {
		http.Error(out, fmt.Sprintf("Unable to create valid query for influxdb: %w", err),  http.StatusBadRequest)
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

func (i InfluxClient) BuildInfluxQuery(site string, queryParams url.Values) (string, error) {

	var query strings.Builder

	query.WriteString("select sum(hits)")

	query.WriteString(" ")
	query.WriteString(fmt.Sprintf("from \"%s\"", site))

	groupby := queryParams.Get("groupby")
	if !slices.Contains(VALIDGROUPBYS, groupby) {
		return "", fmt.Errorf("Invalid groupby %s (try one of %v)", groupby, VALIDGROUPBYS)
	}

	bucketby := queryParams.Get("bucketby")
	if !slices.Contains(VALIDBUCKETBYS, bucketby) {
		return "", fmt.Errorf("Invalid bucketby %s (try one of %v)", bucketby, VALIDBUCKETBYS)
	}

	tz := queryParams.Get("tz")
	if false {
		// todo, some actual validation here, hashtag sql injection
		return "", fmt.Errorf("Invalid timezone %s", tz)
	}

	if (groupby != "" || bucketby != "") {
		query.WriteString(" ")
		query.WriteString("group by")
	}

	if (groupby != "") {
		query.WriteString(" ")
		query.WriteString(groupby)
	}

	if (groupby != "" && bucketby != "") {
		query.WriteString(",")
	}

	if (bucketby != "") {
		query.WriteString(fmt.Sprintf("time(%s)", bucketby))
		query.WriteString(" ")
		query.WriteString("fill(0)")
	}

	if (tz != "") {
		query.WriteString(" ")
		query.WriteString(fmt.Sprintf("tz('%s')", tz))
	}

	return query.String(), nil
}
