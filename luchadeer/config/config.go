package config

import (
	"time"
)

const ContentProviderHost = "www.giantbomb.com"
const ContentProviderApiPath = "/api"

// api keys

// cloud messaging api key. leave blank to not send push notifications
const GCMApiKey = ""

// giant bomb api keys

// The pull api key. use a subscriber key here to provide push notifications for subsriber content.
const PullApiKey = ""

// Api key used to proxy users to the content provider. Use a subscriber key here only if you want
// to proxy subscriber content... which you probably dont.
const ProxyApiKey = ""

// number of videos we check with each pull
const VideoPullSize = 1

// minimum client version before forcing an update (major, minor, bugfix)
var MinVersion = []int{0, 0, 0}

// redirect from /
const ClientDownloadURL = ""

var ValidVideoCategories = map[int]string{
	2:  "Reviews",
	3:  "Quick Looks",
	4:  "TANG",
	5:  "Endurance Run",
	6:  "Events",
	7:  "Trailers",
	8:  "Features",
	10: "Subscriber",
	11: "Extra Life",
	12: "Encyclopedia Bombastica",
	13: "Unfinished",
}

// enable
const ProxyRequests = true
const SearchProxyEnabled = true

const DefaultCacheTTL = time.Hour * 24

const ListRequestCacheTTL = time.Hour

const GameDetailCacheTTL = time.Hour * 24
const VideoDetailCacheTTL = time.Hour * 24 * 7

var BadRequestCacheTTL = time.Hour
