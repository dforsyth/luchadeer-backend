package cron

import (
	"appengine"
	"luchadeer/config"
	"luchadeer/db"
	"luchadeer/giantbomb"
	"luchadeer/tasks"
	"net/http"
)

const PULL_VIDEOS_URL = "/cron/pull_videos"

func Init() {
	http.HandleFunc(PULL_VIDEOS_URL, pullVideos)
}

func pullVideos(w http.ResponseWriter, r *http.Request) {
	context := appengine.NewContext(r)

	response, err := giantbomb.GetVideos(context, nil, 0, config.VideoPullSize)
	if err != nil {
		context.Errorf("Video pull failed: %v", err)
		return
	}

	videos := response.Results

	newVideos, err := db.PutNewVideos(context, videos)
	if err != nil {
		context.Errorf("PutNewVideos error: %v", err)
		return
	}

	context.Infof("Video pull: Pulled: %v, New: %v", len(videos), len(newVideos))
	if len(newVideos) > 0 {
		for _, video := range newVideos {
			context.Infof("New video: %v", video)
			tasks.PushAlertsForVideo(context, video)
		}

		// TODO: invalidate list caches
	}
}
