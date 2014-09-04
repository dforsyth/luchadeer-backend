package tasks

import (
	"appengine"
	"appengine/taskqueue"
	"appengine/urlfetch"
	"luchadeer/config"
	"luchadeer/db"
	"luchadeer/gcm"
	"luchadeer/giantbomb"
	"net/http"
	"strconv"
)

const PUSH_ALERTS_FOR_VIDEO_URL = "/task/push_alerts_for_video"

func Init() {
	http.HandleFunc(PUSH_ALERTS_FOR_VIDEO_URL, pushAlertsForVideo)
}

func PushAlertsForVideo(context appengine.Context, video *giantbomb.Video) {
	task := taskqueue.NewPOSTTask(
		PUSH_ALERTS_FOR_VIDEO_URL,
		map[string][]string{
			"video_type": {video.VideoType},
			"video_name": {video.Name},
			"video_id":   {strconv.FormatInt(video.Id, 10)},
		},
	)

	if _, err := taskqueue.Add(context, task, ""); err != nil {
		context.Errorf("PushAlertsForVideo: %v", err.Error())
	}
}

func pushAlertsForVideo(w http.ResponseWriter, r *http.Request) {
	if config.GCMApiKey == "" {
		return
	}

	context := appengine.NewContext(r)

	videoType := r.FormValue("video_type")
	videoName := r.FormValue("video_name")
	videoId := r.FormValue("video_id")

	preferences, err := db.NotificationSubscriptions(context, videoType)
	if err != nil {
		context.Errorf("Couldn't fetch subscriptions for push: %v", err)
	}

	registrationIds := []string{}
	for _, preference := range preferences {
		registrationIds = append(registrationIds, preference.GCMRegistrationId)
	}

	push := gcm.NewGCM(config.GCMApiKey, urlfetch.Client(context))

	// we can only push 1000 registered ids at a time. for now, we'll just chunk in this task. we might want to chain later, though.
	for off := 0; off < len(registrationIds); off += 1000 {
		max := off + 1000
		if max > len(registrationIds) {
			max = len(registrationIds) - off
		}
		pushResult, err := push.Send(
			map[string]interface{}{"video_name": videoName, "video_id": videoId},
			registrationIds[off:max])

		if err != nil {
			context.Errorf("Push error: %v", err)
		}
		context.Infof("Push result %v", pushResult)
	}
}
