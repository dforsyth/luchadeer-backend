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
const PUSH_ALERT_FOR_CHAT_URL = "/task/push_alert_for_chat"

func Init() {
	http.HandleFunc(PUSH_ALERTS_FOR_VIDEO_URL, pushAlertsForVideo)
	http.HandleFunc(PUSH_ALERT_FOR_CHAT_URL, pushAlertForChat)
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

	registrationIds := registrationIdsForVideoType(context, videoType)
	if registrationIds == nil {
		context.Infof("No registrationIds for %v", videoType)
		return
	}

	push := gcm.NewGCM(config.GCMApiKey, urlfetch.Client(context))

	pushChunked(context, push, registrationIds, map[string]interface{}{"video_name": videoName, "video_id": videoId, "video_type": videoType})
}

func registrationIdsForVideoType(context appengine.Context, videoType string) []string {
	preferences, err := db.NotificationSubscriptions(context, videoType)
	if err != nil {
		context.Errorf("Couldn't fetch subscriptions for push: %v", err)
		return nil
	}

	registrationIds := []string{}
	for _, preference := range preferences {
		registrationIds = append(registrationIds, preference.GCMRegistrationId)
	}
	return registrationIds
}

func pushChunked(context appengine.Context, push *gcm.GCM, registrationIds []string, data map[string]interface{}) {
	for off := 0; off < len(registrationIds); off += 1000 {
		max := off + 1000
		if max > len(registrationIds) {
			max = len(registrationIds) - off
		}
		pushResult, err := push.Send(data, registrationIds[off:max])
		if err != nil {
			context.Errorf("Push error (%v-%v): %v", off, max, err)
		}
		context.Infof("Push result (%v-%v): %v", off, max, pushResult)
	}
}

func PushAlertForChat(context appengine.Context, title string) {
	task := taskqueue.NewPOSTTask(
		PUSH_ALERT_FOR_CHAT_URL,
		map[string][]string{
			"title": {title},
		},
	)

	if _, err := taskqueue.Add(context, task, ""); err != nil {
		context.Errorf("PushAlertForChat: %v", err.Error())
	}
}

func pushAlertForChat(w http.ResponseWriter, r *http.Request) {
	if config.GCMApiKey == "" {
		return
	}

	context := appengine.NewContext(r)

	registrationIds := registrationIdsForVideoType(context, "live")
	if registrationIds == nil {
		context.Infof("No registrationIds for live")
		return
	}

	title := r.FormValue("title")

	push := gcm.NewGCM(config.GCMApiKey, urlfetch.Client(context))

	pushChunked(context, push, registrationIds, map[string]interface{}{"video_name": title, "video_id": 0, "video_type": "live"})
}
