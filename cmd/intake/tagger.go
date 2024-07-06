package intake

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/mileusna/useragent"
	"zgo.at/isbot"
)

// TODO, settle on EXACTLY ONE shared value for when a tag can't be determined
// right now it's KINDA "Unknown"
// maybe empty str?

// EnrichedLog is responsible for turning a BunnyLog into a point for influx
// with all the necessary tags, timestamps, etc
type EnrichedLog struct {
	bunny     BunnyLog
	userAgent useragent.UserAgent
	refUrl    *url.URL
}

func Enrich(bunny BunnyLog) EnrichedLog {
	refUrl, err := url.Parse(bunny.Referer)
	if err != nil {
		log.Printf("[WARN] Unable to parse referrer URL: %v", err)
	}
	return EnrichedLog{
		bunny:     bunny,
		userAgent: useragent.Parse(bunny.UserAgent),
		refUrl:    refUrl,
	}
}

func (e EnrichedLog) Device() (string, string) {
	if e.userAgent.Mobile {
		return "device", "Mobile"
	} else if e.userAgent.Tablet {
		return "device", "Tablet"
	} else if e.userAgent.Desktop {
		return "device", "Desktop"
	} else {
		return "device", "Unknown"
	}
}

func (e EnrichedLog) Browser() (string, string) {
	if e.userAgent.Name == "-" {
		return "browser", "Unknown"
	}
	return "browser", e.userAgent.Name
}

func (e EnrichedLog) Os() (string, string) {
	if e.userAgent.OS == "" {
		return "os", "Unknown"
	}
	return "os", e.userAgent.OS
}

func (e EnrichedLog) Country() (string, string) {
	return "country", e.bunny.Country
}

func (e EnrichedLog) StatusCode() (string, string) {
	return "statuscode", string(e.bunny.Status)
}

func (e EnrichedLog) StatusCategory() (string, string) {
	if e.bunny.Status < 100 {
		log.Printf("Can't get status category from weird code: %v", e.bunny.Status)
		return "statuscategory", "Unknown"
	}
	return "statuscategory", string(e.bunny.Status/100) + "xx"
}

func (e EnrichedLog) Path() (string, string) {
	return "path", e.bunny.PathAndQuery
}

func (e EnrichedLog) Referrer() (string, string) {
	return "referrer", e.refUrl.Host
}

func (e EnrichedLog) FileType() (string, string) {

	slashIndex := strings.LastIndex(e.bunny.PathAndQuery, "/")
	filename := e.bunny.PathAndQuery[(slashIndex + 1):]

	if filename == "" {
		return "filetype", "Page"
	}

	dotIndex := strings.LastIndex(filename, ".")

	if dotIndex == -1 {
		return "filetype", "Page"
	}

	switch t := filename[(dotIndex + 1):]; t {

	case "html":
		return "filetype", "Page"

	case "css":
		return "filetype", "Stylesheet"

	case "js":
		return "filetype", "Javascript"

	case "img", "jpg", "jpeg", "png", "ico", "gif", "svg", "heic":
		return "filetype", "Image"

	case "ttf", "otf", "woff", "woff2":
		return "filetype", "Font"

	case "txt", "csv", "pdf", "doc", "docx", "xls", "xlsx", "ppt", "pptx":
		return "filetype", "Document"

	case "zip", "gz", "rar", "iso", "tar", "lzma", "bz2", "7z", "z", "tgz":
		return "filetype", "Archive"

	case "mp3", "m4a", "wav", "ogg", "flac", "midi", "aac", "wma":
		return "filetype", "Audio"

	case "mpg", "mpeg", "avi", "mp4", "flv", "h264", "mov", "mk4", "mkv", "m4v":
		return "filetype", "Video"

	case "xml":
		return "filetype", "RSS Feed"

	default:
		return "filetype", "Unknown"
	}
}

func (e EnrichedLog) IsProbablyBot() (string, string) {
	// similar to isbot's "Bot" implementation, but skips the "does the header
	// indicate this is a prefetch" check since we ain't got no headers
	BotNoHeader := func() isbot.Result {
		i := isbot.UserAgent(e.bunny.UserAgent)
		if i > 0 {
			return i
		}

		return isbot.IPRange(fmt.Sprintf("%s", e.bunny.RemoteIp))
	}

	res := BotNoHeader()
	return "isprobablybot", fmt.Sprintf("%v", isbot.Is(res))
}

func (e EnrichedLog) Tags() map[string]string {

	tagFuncSlice := []func() (string, string){
		e.Device,
		e.Browser,
		e.Os,
		e.Country,
		e.StatusCode,
		e.StatusCategory,
		e.Path,
		e.Referrer,
		e.FileType,
		e.IsProbablyBot,
	}

	tags := map[string]string{}
	for _, f := range tagFuncSlice {
		name, val := f()
		tags[name] = val
	}

	return tags
}

func (e EnrichedLog) Point() *write.Point {

	tags := e.Tags()

	return influxdb2.NewPoint(
		// metric name
		e.bunny.Host,
		// tags
		tags,
		// fields
		map[string]interface{}{"hits": 1},
		// ts
		time.UnixMilli(e.bunny.Timestamp),
	)
}
