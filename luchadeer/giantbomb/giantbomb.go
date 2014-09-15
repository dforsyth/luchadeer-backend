/*
 * Copyright (c) 2014, David Forsythe
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 *
 *  Redistributions of source code must retain the above copyright notice, this
 *   list of conditions and the following disclaimer.
 *
 *  Redistributions in binary form must reproduce the above copyright notice,
 *   this list of conditions and the following disclaimer in the documentation
 *   and/or other materials provided with the distribution.
 *
 *  Neither the name of Luchadeer nor the names of its
 *   contributors may be used to endorse or promote products derived from
 *   this software without specific prior written permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
 * AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 * DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
 * FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
 * DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
 * SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
 * CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
 * OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
 * OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

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
const BeerIconMarkup = "<h4><i class=\"icon icon-beer\"></i>Live!</h4>"
const Header3OpenMarkup = "<h3>"
const Header3CloseMarkup = "</h3>"

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

func tryFindPremiumChat(doc string) (string, error) {
	// premium live show
	liveIcon := strings.Index(doc, BeerIconMarkup)
	if liveIcon < 0 {
		// not a premium live show
		return "", errors.New("Cant find beer icon markup")
	}

	searchable := doc[liveIcon+len(BeerIconMarkup):]

	p := strings.Index(searchable, Header3OpenMarkup)
	q := strings.Index(searchable, Header3CloseMarkup)

	if p < 0 || q < 0 {
		return "", errors.New("Cant premium title find title")
	}

	title := html.UnescapeString(searchable[p+len(Header3OpenMarkup) : q])

	return title, nil
}

func tryFindNonPremiumChat(doc string) (string, error) {
	// non-premium live show
	liveTitle := strings.Index(doc, LiveTitleMarkup)
	joinButton := strings.Index(doc, JoinButtonMarkup)

	if liveTitle < 0 || joinButton < 0 {
		return "", errors.New("Couldn't find indicator tags in " + GiantBombURL)
	}

	if liveTitle > joinButton {
		return "", errors.New("Page doesn't look as we expect")
	}

	searchable := doc[liveTitle+len(LiveTitleMarkup) : joinButton]
	o := strings.Index(searchable, Header2OpenMarkup)
	c := strings.Index(searchable, Header2CloseMarkup)

	if o < 0 || c < 0 {
		return "", errors.New("Cant find header tags")
	}

	if c < o {
		return "", errors.New("Title close came before title open in chat page")
	}

	title := html.UnescapeString(searchable[o+len(Header2OpenMarkup) : c])

	return title, nil
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

	if title, err := tryFindPremiumChat(cast); err == nil {
		context.Infof("Detected premium chat title: %v", title)
		return title, nil
	} else {
		context.Infof("no premium chat found: %v", err)
	}

	if title, err := tryFindNonPremiumChat(cast); err == nil {
		context.Infof("Detected non-premium chat title: %v", title)
		return title, nil
	} else {
		context.Infof("no free chat found: %v", err)
	}

	return "", errors.New("No chat detected")
}
