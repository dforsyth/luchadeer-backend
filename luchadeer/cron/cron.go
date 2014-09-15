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

package cron

import (
	"appengine"
	"luchadeer/config"
	"luchadeer/db"
	"luchadeer/giantbomb"
	"luchadeer/tasks"
	"net/http"
)

const PullVideosURL = "/cron/pull_videos"
const PollChatURL = "/cron/poll_chat"

func Init() {
	http.HandleFunc(PullVideosURL, pullVideos)
	http.HandleFunc(PollChatURL, pollChat)
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

func pollChat(w http.ResponseWriter, r *http.Request) {
	context := appengine.NewContext(r)

	title, err := giantbomb.GetChat(context)
	if err != nil {
		context.Infof("pollChat: %v", err)
		return
	}
	_, perr := db.PutChat(context, title)
	if perr != nil {
		context.Infof("PutChat: %v", perr)
		return
	}

	tasks.PushAlertForChat(context, title)
}
