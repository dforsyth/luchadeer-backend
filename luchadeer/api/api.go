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
	http.HandleFunc("/api/1/giantbomb/videos/", videosHandler)
	http.HandleFunc("/api/1/giantbomb/video/", videoHandler)
	http.HandleFunc("/api/1/giantbomb/games/", gamesHandler)
	http.HandleFunc("/api/1/giantbomb/game/", gameHandler)

	http.HandleFunc("/api/1/giantbomb/video_types/", videoTypesHandler)

	http.HandleFunc("/api/1/giantbomb/search/", searchHandler)
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

// for later
func finishGiantBombResponse(context appengine.Context, response *giantbomb.InterfaceGiantBombResponse, w http.ResponseWriter) {
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(&response); err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		context.Errorf("Encode error: %v", err)
		return
	}
}

type CacheConfig struct {
	QueryParams map[string]func([]string) bool
	TTL         time.Duration
}

// normalize request for proxy
func normalizeURLForProxy(u *url.URL, c *CacheConfig) error {
	u.Path = strings.Replace(u.Path, "/api/1/giantbomb", config.ContentProviderApiPath, 1)

	query := u.Query()

	query.Del("api_key")
	query.Del("format")

	// we never limit
	query.Del("limit")

	for param, values := range query {
		if check, ok := c.QueryParams[param]; !ok || !check(values) {
			return errors.New("Unusable query param")
		}
	}

	u.RawQuery = query.Encode()

	return nil
}

func makeProxyRequest(context appengine.Context, u *url.URL) (*http.Response, error) {
	client := urlfetch.Client(context)

	// update host
	u.Host = config.ContentProviderHost

	// add api key and format to proxy request
	q := u.Query()
	q.Add("api_key", config.ProxyApiKey)
	q.Add("format", "json")

	u.RawQuery = q.Encode()

	context.Infof("Proxy request url: %v", u.String())
	response, err := client.Get(u.String())

	return response, err
}

var DefaultNoProxyResponse = &giantbomb.InterfaceGiantBombResponse{
	BaseGiantBombResponse: giantbomb.BaseGiantBombResponse{
		StatusCode:           -1,
		Error:                "Proxy disabled!",
		Limit:                1,
		Offset:               0,
		NumberOfPageResults:  1,
		NumberOfTotalResults: 0,
	},
	Results: map[string]interface{}{},
}

// send back error message
func writeBlockedProxyResponse(w http.ResponseWriter) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(DefaultNoProxyResponse)
}

func proxyRequest(context appengine.Context, w http.ResponseWriter, r *http.Request, c *CacheConfig) error {
	if !config.ProxyRequests {
		if err := writeBlockedProxyResponse(w); err != nil {
			return err
		}
		return nil
	}

	if err := normalizeURLForProxy(r.URL, c); err != nil {
		return err
	}

	uri := r.URL.RequestURI()

	// check memcache for a similar request
	// cached, err := getCachedValue(context, kind, uri)
	cached, err := memcache.Get(context, uri)
	if err == nil {
		// write cached request to response writer
		w.Write(cached.Value)
		return nil
	}

	if err == memcache.ErrCacheMiss {
		// make request to the content provider, cache results, and serve them.
		response, err := makeProxyRequest(context, r.URL)
		if err != nil {
			// something broke during the request, log and bail
			context.Errorf("Proxy request error: %v", err)
			return err
		}
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			// we had a problem getting to the content provider.
			return fmt.Errorf("Proxy response response code not OK: %v", response.StatusCode)
		}

		// we have to parse the json to make sure we have an OK from the content provider.
		var parsed giantbomb.BaseGiantBombResponse
		var body []byte

		body, rErr := ioutil.ReadAll(response.Body)
		if rErr != nil {
			context.Errorf("Request read error: %v", rErr)
			return rErr
		}

		if dErr := json.Unmarshal(body, &parsed); dErr != nil {
			context.Errorf("Unmarshal error: %v, %v", dErr, body)
			// Should we return the busted request to user?
			return dErr
		}

		ttl := config.ListRequestCacheTTL
		if parsed.StatusCode != giantbomb.StatusOK && parsed.StatusCode != giantbomb.StatusRestrictedContent {
			// we got an error from the content provider, log it and drop the ttl.
			context.Infof("Bad status returned by content provider: %v: %v", parsed.StatusCode, parsed.Message)
			ttl = config.BadRequestCacheTTL
		}

		item := &memcache.Item{
			Key:        uri,
			Value:      body,
			Expiration: ttl,
		}

		memcache.Set(context, item)

		w.Write(body)
		return nil
	}

	return err
}

var VidoeTypesCacheConfig = &CacheConfig{
	TTL: config.DefaultCacheTTL,
}

func videoTypesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}

	context := appengine.NewContext(r)
	if err := proxyRequest(context, w, r, VidoeTypesCacheConfig); err != nil {
		context.Errorf("Proxy request error: %v", err)
	}
}

var VideoCacheConfig = &CacheConfig{
	TTL: config.VideoDetailCacheTTL,
}

func videoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}

	context := appengine.NewContext(r)

	if err := proxyRequest(context, w, r, VideoCacheConfig); err != nil {
		context.Errorf("Proxy request error: %v", err)
	}
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

func videosHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}

	context := appengine.NewContext(r)
	if err := proxyRequest(context, w, r, VideoListCacheConfig); err != nil {
		context.Errorf("Proxy request error: %v", err)
	}
}

var GameCacheConfig = &CacheConfig{
	TTL: config.GameDetailCacheTTL,
}

func gameHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}

	context := appengine.NewContext(r)
	if err := proxyRequest(context, w, r, GameCacheConfig); err != nil {
		context.Errorf("Proxy request error: %v", err)
	}
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

func gamesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}

	context := appengine.NewContext(r)
	if err := proxyRequest(context, w, r, GameListCacheConfig); err != nil {
		context.Errorf("Proxy request error: %v", err)
	}
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

func searchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}

	context := appengine.NewContext(r)
	if config.SearchProxyEnabled {
		if err := proxyRequest(context, w, r, SearchCacheConfig); err != nil {
			context.Errorf("Proxy request error: %v", err)
		}
		return
	} else {
		// we need to return something that will pop up a message to the user
		writeBlockedProxyResponse(w)
	}
}
