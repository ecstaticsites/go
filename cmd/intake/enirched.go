package intake

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/mileusna/useragent"
	"zgo.at/isbot"
)

// TODO, settle on EXACTLY ONE shared value for when a tag can't be determined
// right now it's KINDA "Unknown"
// maybe empty str?

// EnrichedLog is responsible for turning a BunnyLog into a point for influx
// with all the necessary tags, timestamps, etc
type EnrichedLog struct {
	PullZoneId     int
	Timestamp      int64
	BytesSent      int
	StatusCode     int
	StatusCategory string
	Host           string
	Path           string
	Referrer       string
	Device         string
	Browser        string
	Os             string
	Country        string
	FileType       string
	IsProbablyBot  bool
}

func Enrich(bunny BunnyLog) EnrichedLog {
	ua := useragent.Parse(bunny.UserAgent)
	return EnrichedLog{
		PullZoneId: bunny.PullZoneId,
		// bunny comes  in epoch ms, CH wants epoch sec
		Timestamp:      bunny.Timestamp / 1000,
		BytesSent:      bunny.BytesSent,
		StatusCode:     bunny.Status,
		StatusCategory: StatusCategory(bunny),
		Host:           bunny.Host,
		Path:           bunny.PathAndQuery,
		Referrer:       Referrer(bunny),
		Device:         Device(ua),
		Browser:        Browser(ua),
		Os:             Os(ua),
		Country:        bunny.Country,
		FileType:       FileType(bunny),
		IsProbablyBot:  IsProbablyBot(bunny),
	}
}

func Device(ua useragent.UserAgent) string {
	if ua.Mobile {
		return "Mobile"
	} else if ua.Tablet {
		return "Tablet"
	} else if ua.Desktop {
		return "Desktop"
	} else {
		return "Unknown"
	}
}

func Browser(ua useragent.UserAgent) string {
	if ua.Name == "-" {
		return "Unknown"
	}
	return ua.Name
}

func Os(ua useragent.UserAgent) string {
	if ua.OS == "" {
		return "Unknown"
	}
	return ua.OS
}

func StatusCategory(bunny BunnyLog) string {
	if bunny.Status < 100 {
		log.Printf("Can't get status category from weird code: %v", bunny.Status)
		return "Unknown"
	}
	return fmt.Sprint(bunny.Status/100) + "xx"
}

func Referrer(bunny BunnyLog) string {
	refUrl, err := url.Parse(bunny.Referer)
	if err != nil {
		log.Printf("[WARN] Unable to parse referrer URL: %v", err)
		return "Unknown"
	}
	return refUrl.Host
}

func FileType(bunny BunnyLog) string {

	slashIndex := strings.LastIndex(bunny.PathAndQuery, "/")
	filename := bunny.PathAndQuery[(slashIndex + 1):]

	if filename == "" {
		return "Page"
	}

	dotIndex := strings.LastIndex(filename, ".")

	if dotIndex == -1 {
		return "Page"
	}

	switch t := filename[(dotIndex + 1):]; t {

	case "html":
		return "Page"

	case "css":
		return "Stylesheet"

	case "js":
		return "Javascript"

	case "img", "jpg", "jpeg", "png", "ico", "gif", "svg", "heic":
		return "Image"

	case "ttf", "otf", "woff", "woff2":
		return "Font"

	case "txt", "csv", "pdf", "doc", "docx", "xls", "xlsx", "ppt", "pptx":
		return "Document"

	case "zip", "gz", "rar", "iso", "tar", "lzma", "bz2", "7z", "z", "tgz":
		return "Archive"

	case "mp3", "m4a", "wav", "ogg", "flac", "midi", "aac", "wma":
		return "Audio"

	case "mpg", "mpeg", "avi", "mp4", "flv", "h264", "mov", "mk4", "mkv", "m4v":
		return "Video"

	case "xml":
		return "RSS Feed"

	default:
		return "Unknown"
	}
}

func IsProbablyBot(bunny BunnyLog) bool {
	// similar to isbot's "Bot" implementation, but skips the "does the header
	// indicate this is a prefetch" check since we ain't got no headers
	BotNoHeader := func() isbot.Result {
		i := isbot.UserAgent(bunny.UserAgent)
		if i > 0 {
			return i
		}

		return isbot.IPRange(fmt.Sprintf("%s", bunny.RemoteIp))
	}

	res := BotNoHeader()
	return isbot.Is(res)
}
