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

package api

import (
	"appengine"
	"appengine/memcache"
	"appengine/urlfetch"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"luchadeer/config"
	"luchadeer/db"
	"luchadeer/giantbomb"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func Init() {
	http.HandleFunc("/api/1/preferences", preferencesHandler)

	// duplicating the giantbomb api to make the client work easier
	http.Handle("/api/1/giantbomb/videos/", NewGiantBombCacheHandler(VideoListCacheConfig))
	http.Handle("/api/1/giantbomb/video/", NewGiantBombCacheHandler(VideoCacheConfig))
	http.Handle("/api/1/giantbomb/games/", NewGiantBombCacheHandler(GameListCacheConfig))
	http.Handle("/api/1/giantbomb/game/", NewGiantBombCacheHandler(GameCacheConfig))

	http.Handle("/api/1/giantbomb/video_types/", NewGiantBombCacheHandler(VideoTypesCacheConfig))

	http.Handle("/api/1/giantbomb/search/", NewGiantBombCacheHandler(SearchCacheConfig))

	http.Handle("/api/1/youtube/unarchived_videos", NewYouTubeCacheHandler(YouTubeCacheConfig))
}

// update user preferences. post only.
func preferencesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}

	context := appengine.NewContext(r)

	var preferences db.NotificationPreference

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&preferences); err != nil {
		http.Error(w, "Error decoding json", http.StatusInternalServerError)
		context.Errorf("Decode error: %v", err)
		return
	}
	defer r.Body.Close()

	if err := db.UpdateNotificationPreference(context, &preferences); err != nil {
		http.Error(w, "Error updating preferences", http.StatusInternalServerError)
		context.Infof("UpdateNotificationPreferences: %v", err)
	}
}

type CacheConfig struct {
	QueryParams map[string]func([]string) bool
	TTL         time.Duration
}

var VideoTypesCacheConfig = &CacheConfig{
	TTL: config.DefaultCacheTTL,
}

var VideoCacheConfig = &CacheConfig{
	TTL: config.VideoDetailCacheTTL,
}

var VideoListCacheConfig = &CacheConfig{
	QueryParams: map[string]func([]string) bool{
		"offset": func(values []string) bool {
			if len(values) > 1 {
				return false
			}
			num, err := strconv.Atoi(values[0])
			if err != nil {
				return false
			}
			if num%100 != 0 {
				return false
			}
			return true
		},
		"video_type": func(values []string) bool {
			if len(values) > 1 {
				return false
			}
			i, err := strconv.Atoi(values[0])
			if err != nil {
				return false
			}
			_, ok := config.ValidVideoCategories[i]
			return ok
		},
	},
	TTL: config.ListRequestCacheTTL,
}

var GameCacheConfig = &CacheConfig{
	TTL: config.GameDetailCacheTTL,
}

var GameListCacheConfig = &CacheConfig{
	QueryParams: map[string]func([]string) bool{
		"offset": func(value []string) bool {
			if len(value) > 1 {
				return false
			}
			num, err := strconv.Atoi(value[0])
			if err != nil {
				return false
			}
			if num%100 != 0 {
				return false
			}
			return true
		},
		"sort": func(values []string) bool {
			if len(values) > 1 {
				return false
			}
			return values[0] == "date_added:desc"
		},
	},
	TTL: config.ListRequestCacheTTL,
}

var SearchCacheConfig = &CacheConfig{
	QueryParams: map[string]func([]string) bool{
		"query": func(values []string) bool {
			if len(values) > 1 {
				return false
			}
			return true
		},
		"resources": func(values []string) bool {
			if len(values) > 1 {
				return false
			}
			// very client specific for now
			if values[0] != "game,video," {
				return false
			}
			return true
		},
	},
	TTL: config.DefaultCacheTTL,
}

var YouTubeCacheConfig = &CacheConfig{
	QueryParams: map[string]func([]string) bool{
		"q": func(values []string) bool {
			if len(values) > 1 {
				return false
			}
			return true
		},
		"pageToken": func(values []string) bool {
			if len(values) > 1 {
				return false
			}
			return true
		},
	},
	TTL: config.DefaultCacheTTL,
}

type CacheHandler struct {
	p ProxyHandler
}

func NewGiantBombCacheHandler(c *CacheConfig) *CacheHandler {
	return &CacheHandler{&GiantBombProxyHandler{c}}
}

func NewYouTubeCacheHandler(c *CacheConfig) *CacheHandler {
	return &CacheHandler{&YouTubeProxyHandler{c}}
}

func (h *CacheHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}

	context := appengine.NewContext(r)

	h.p.PrepareURL(context, r.URL)
	key := h.p.URLCacheKey(context, r.URL)

	// check cache
	cached, err := memcache.Get(context, key)
	if err == nil {
		// write cached request to response writer
		context.Infof("cache hit: %v", key)
		w.Write(cached.Value)
		return
	}

	if err != memcache.ErrCacheMiss {
		context.Errorf("memcache error: %v", err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	client := urlfetch.Client(context)

	context.Infof("proxy url: %v", r.URL.String())

	response, err := client.Get(r.URL.String())
	if err != nil {
		context.Errorf("proxy request error: %v", err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()

	body, ttl, err := h.p.ProcessResponse(context, response)
	if err != nil {
		context.Errorf("process response error: %v", err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	item := &memcache.Item{
		Key:        key,
		Value:      body,
		Expiration: ttl,
	}

	memcache.Set(context, item)

	context.Infof("cached: %s", key)

	w.Write(body)
}

type ProxyHandler interface {
	PrepareURL(appengine.Context, *url.URL) error
	URLCacheKey(appengine.Context, *url.URL) string
	ProcessResponse(appengine.Context, *http.Response) ([]byte, time.Duration, error) // body, ttl
}

type GiantBombProxyHandler struct {
	c *CacheConfig
}

func (h *GiantBombProxyHandler) PrepareURL(context appengine.Context, u *url.URL) error {
	u.Path = strings.Replace(u.Path, "/api/1/giantbomb", config.ContentProviderApiPath, 1)
	u.Host = config.ContentProviderHost

	query := u.Query()

	query.Del("api_key")
	query.Del("format")
	query.Del("limit")

	for param, values := range query {
		if check, ok := h.c.QueryParams[param]; !ok || !check(values) {
			return errors.New("Unusable query param")
		}
	}

	query.Add("api_key", config.ProxyApiKey)
	query.Add("format", "json")

	u.RawQuery = query.Encode()

	return nil
}

func (h *GiantBombProxyHandler) URLCacheKey(context appengine.Context, u *url.URL) string {
	return fmt.Sprintf("giantbomb/%s", u.RequestURI())
}

func (h *GiantBombProxyHandler) ProcessResponse(context appengine.Context, response *http.Response) ([]byte, time.Duration, error) {
	// we have to parse the json to make sure we have an OK from the content provider.
	var parsed giantbomb.BaseGiantBombResponse
	var body []byte

	body, rErr := ioutil.ReadAll(response.Body)
	if rErr != nil {
		context.Errorf("Request read error: %v", rErr)
		return nil, -1, rErr
	}

	if dErr := json.Unmarshal(body, &parsed); dErr != nil {
		context.Errorf("Unmarshal error: %v, %v", dErr, body)
		// Should we return the busted request to user?
		return nil, -1, dErr
	}

	ttl := h.c.TTL
	if parsed.StatusCode != giantbomb.StatusOK && parsed.StatusCode != giantbomb.StatusRestrictedContent {
		// we got an error from the content provider, log it and drop the ttl.
		context.Infof("Bad status returned by content provider: %v: %v", parsed.StatusCode, parsed.Message)
		ttl = config.BadRequestCacheTTL
	}

	return body, ttl, nil
}

type YouTubeProxyHandler struct {
	c *CacheConfig
}

func (h *YouTubeProxyHandler) PrepareURL(context appengine.Context, u *url.URL) error {
	u.Path = strings.Replace(u.Path, "/api/1/youtube/unarchived_videos", config.YouTubeSearchPath, 1)
	u.Host = config.YouTubeApiHost

	query := u.Query()

	query.Del("part")
	query.Del("maxResults")
	query.Del("type")
	query.Del("channelId")
	query.Del("key")
	query.Del("order")

	for param, values := range query {
		if check, ok := h.c.QueryParams[param]; !ok || !check(values) {
			return errors.New("Unusable query param")
		}
	}

	query.Add("part", "snippet")
	query.Add("maxResults", "50")
	query.Add("type", "video")

	query.Add("channelId", config.UnarchivedChannelId)
	query.Add("key", config.YouTubeApiKey)

	// pQuery.Add("pageToken", pageToken)

	if query.Get("q") == "" {
		query.Del("q")
		query.Add("order", "date")
	}

	u.RawQuery = query.Encode()

	return nil
}

func (h *YouTubeProxyHandler) URLCacheKey(context appengine.Context, u *url.URL) string {
	return fmt.Sprintf("youtube/%s", u.RequestURI())
}

func (h *YouTubeProxyHandler) ProcessResponse(context appengine.Context, response *http.Response) ([]byte, time.Duration, error) {
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, -1, err
	}

	return body, h.c.TTL, nil
}