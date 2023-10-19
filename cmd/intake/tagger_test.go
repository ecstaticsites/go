package intake

import (
	"fmt"
	"net"
	"testing"

	"github.com/oschwald/geoip2-golang"
	"github.com/stretchr/testify/assert"
)

type MockGeo struct{}

func (m MockGeo) Country(net.IP) (*geoip2.Country, error) {
	return nil, fmt.Errorf("Not implemented")
}

func TestNotBot(t *testing.T) {

	mock := MockGeo{}
	tagger := Tagger{mock}

	str := "HIT|200|1507167062421|412|390|163.172.53.229|-|https://www.example.com/favicon.ico|WA|Mozilla/5.0 (Windows NT 10.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/65.0.3325.146 Safari/537.36|322b688bd63fb63f2babe9de30a5d262|DE"

	bunny, err := stringToBunnyLog(str)
	assert.NoError(t, err)

	actual := tagger.Tags(bunny)

	expected := map[string]string{
		"":               "",
		"browser":        "Chrome",
		"country":        "Unknown",
		"device":         "Desktop",
		"filetype":       "image",
		"isprobablybot":  "false",
		"os":             "Windows",
		"path":           "/favicon.ico",
		"statuscode":     "200",
		"statuscategory": "2xx",
	}

	assert.Equal(t, expected, actual)

}

func TestBot(t *testing.T) {

	mock := MockGeo{}
	tagger := Tagger{mock}

	str := "HIT|404|1507167062421|412|390|163.172.53.229|-|https://www.example.com/favicon.ico|WA|curl/7.54.1|322b688bd63fb63f2babe9de30a5d262|DE"

	bunny, err := stringToBunnyLog(str)
	assert.NoError(t, err)

	actual := tagger.Tags(bunny)

	expected := map[string]string{
		"":               "",
		"browser":        "curl",
		"country":        "Unknown",
		"device":         "Unknown",
		"filetype":       "image",
		"isprobablybot":  "true",
		"os":             "",
		"path":           "/favicon.ico",
		"statuscode":     "404",
		"statuscategory": "4xx",
	}

	assert.Equal(t, expected, actual)

}
