package giantbomb

import (
	"appengine"
	"appengine/urlfetch"
	"encoding/json"
	"luchadeer/config"
	"net/url"
	"strconv"
	"time"
)

const GIANT_BOMB_API_URL = "http://www.giantbomb.com/api/"
const GIANT_BOMB_CHAT_URL = "http://www.giantbomb.com/chat/"

// TODO move these into db
type Video struct {
	Retrieved time.Time `json:"-"`

	Id            int64  `json:"id"`
	Name          string `json:"name"`
	Deck          string `json:"deck"`
	Image         Image  `json:"image"`
	VideoType     string `json:"video_type"`
	LengthSeconds int64  `json:"length_seconds"`
	PublishDate   string `json:"publish_date"` // store this as a string so we don't have to deal with formatting
	SiteDetailUrl string `json:"site_detail_url"`
}

type Image struct {
	SuperUrl string `json:"super_url"`
}

const StatusOK = 1
const StatusRestrictedContent = 105

type BaseGiantBombResponse struct {
	StatusCode           int    `json:"status_code"`
	Error                string `json:"error"`
	Message              string `json:"message"`
	Limit                int64  `json:"limit"`
	Offset               int64  `json:"offset"`
	NumberOfPageResults  int64  `json:"number_of_page_results"`
	NumberOfTotalResults int64  `json:"number_of_total_results"`
}

// anything
type InterfaceGiantBombResponse struct {
	BaseGiantBombResponse
	Results interface{} `json:"results"`
}

// Videos
type VideosGiantBombResponse struct {
	BaseGiantBombResponse
	Results []Video `json:"results"`
}

func GetVideos(context appengine.Context, videoTypes []int, offset, limit int) (*VideosGiantBombResponse, error) {
	endpoint := GIANT_BOMB_API_URL + "videos/"

	values := url.Values{}
	values.Add("api_key", config.PullApiKey)
	values.Add("format", "json")
	if offset > 0 {
		values.Add("offset", strconv.Itoa(offset))
	}
	if limit > 0 {
		values.Add("limit", strconv.Itoa(limit))
	}

	client := urlfetch.Client(context)

	response, err := client.Get(endpoint + "?" + values.Encode())
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	var decoded VideosGiantBombResponse

	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(&decoded); err != nil {
		return nil, err
	}

	return &decoded, nil
}
