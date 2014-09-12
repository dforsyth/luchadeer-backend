package giantbomb

import (
	"appengine"
	"appengine/urlfetch"
	"encoding/json"
	"errors"
	"html"
	"io/ioutil"
	"luchadeer/config"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const GiantBombURL = "http://www.giantbomb.com/"
const GiantBombApiURL = GiantBombURL + "api/"

// markup for chat checks
const LiveTitleMarkup = "<h4 class=\"grad-text\">Live on Giant Bomb!</h4>"
const JoinButtonMarkup = "<p><button class=\"btn btn-primary\">Join the chat</button></p>"
const Header2OpenMarkup = "<h2>"
const Header2CloseMarkup = "</h2>"

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

type Chat struct {
	Title     string
	FirstSeen time.Time
}

func GetVideos(context appengine.Context, videoTypes []int, offset, limit int) (*VideosGiantBombResponse, error) {
	endpoint := GiantBombApiURL + "videos/"

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

func GetChat(context appengine.Context) (string, error) {
	client := urlfetch.Client(context)

	response, err := client.Get(GiantBombURL)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	// just brutalize it.
	doc, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	cast := string(doc)

	liveTitle := strings.Index(cast, LiveTitleMarkup)
	joinButton := strings.Index(cast, JoinButtonMarkup)

	if liveTitle < 0 || joinButton < 0 {
		return "", errors.New("Couldn't find indicator tags in " + GiantBombURL)
	}

	if liveTitle > joinButton {
		return "", errors.New("Page doesn't look as we expect")
	}

	searchable := cast[liveTitle+len(LiveTitleMarkup) : joinButton]
	o := strings.Index(searchable, Header2OpenMarkup)
	c := strings.Index(searchable, Header2CloseMarkup)

	if o < 0 || c < 0 {
		return "", errors.New("Cant find header tags")
	}

	if c < o {
		return "", errors.New("Title close came before title open in chat page")
	}

	title := html.UnescapeString(searchable[o+len(Header2OpenMarkup) : c])

	context.Infof("Detected chat title: %v", title)

	return title, nil
}
