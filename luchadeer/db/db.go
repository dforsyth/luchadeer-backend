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

package db

import (
	"appengine"
	"appengine/datastore"
	"errors"
	"luchadeer/giantbomb"
	"time"
)

type NotificationPreference struct {
	GCMRegistrationId string   `json:"gcm_registration_id"`
	Categories        []string `json:"categories"`
	LastUpdated       time.Time
}

const KIND_NOTIFICATION_SUBSCRIPTION = "notificationpreference"
const KIND_GIANT_BOMB_VIDEO = "giantbombvideo"
const KIND_GIANT_BOMB_CHAT = "giantbombchat"

func UpdateNotificationPreference(context appengine.Context, preference *NotificationPreference) error {
	key := datastore.NewKey(context, KIND_NOTIFICATION_SUBSCRIPTION, preference.GCMRegistrationId, 0, nil)
	preference.LastUpdated = time.Now()
	_, err := datastore.Put(context, key, preference)

	return err
}

func NotificationSubscriptions(context appengine.Context, category string) ([]NotificationPreference, error) {
	var preferences []NotificationPreference
	query := datastore.NewQuery(KIND_NOTIFICATION_SUBSCRIPTION).Filter("Categories =", category)
	if _, err := query.GetAll(context, &preferences); err != nil {
		return nil, err
	}
	return preferences, nil
}

func newVideoKey(context appengine.Context, video *giantbomb.Video) *datastore.Key {
	return datastore.NewKey(context, KIND_GIANT_BOMB_VIDEO, "", video.Id, nil)
}

func PutVideo(context appengine.Context, video *giantbomb.Video) error {
	// update Retrieved time
	video.Retrieved = time.Now()
	_, err := datastore.Put(context, newVideoKey(context, video), video)

	return err
}

// put new videos into the datastore, ignore the old ones. returns all the new videos.
func PutNewVideos(context appengine.Context, videos []giantbomb.Video) ([]*giantbomb.Video, error) {
	keys := []*datastore.Key{}
	for _, video := range videos {
		keys = append(keys, newVideoKey(context, &video))
	}

	newVideos := []*giantbomb.Video{} // we generally don't expect a lot of misses from the video pull

	if err := datastore.GetMulti(context, keys, make([]giantbomb.Video, len(keys))); err != nil {
		// we've got misses
		switch et := err.(type) {
		case (appengine.MultiError):
			for i, e := range et {
				if e == datastore.ErrNoSuchEntity {
					video := videos[i]
					if pe := PutVideo(context, &video); pe != nil {
						context.Errorf("PutVideo error %v", pe)
					} else {
						newVideos = append(newVideos, &video)
					}
				}
			}
		default:
			return nil, err
		}
	}

	return newVideos, nil
}

func PutChat(context appengine.Context, title string) (*giantbomb.Chat, error) {
	key := datastore.NewKey(context, KIND_GIANT_BOMB_CHAT, title, 0, nil)

	var chat giantbomb.Chat

	if err := datastore.Get(context, key, &chat); err != nil {
		switch err {
		case (datastore.ErrNoSuchEntity):
			chat.Title = title
			chat.FirstSeen = time.Now()
			if _, pe := datastore.Put(context, key, &chat); pe != nil {
				context.Errorf("Put error: %v", pe)
			} else {
				return &chat, nil
			}
		default:
			return nil, err
		}
	} else {
		if time.Now().After(chat.FirstSeen.Add(time.Hour * 24)) {
			chat.FirstSeen = time.Now()
			if _, pe := datastore.Put(context, key, &chat); pe != nil {
				context.Errorf("Put error on update: %v", pe)
			} else {
				context.Infof("Updating existing entry for %v since it is over 24 hours old", title)
				return &chat, nil
			}
		}
	}

	return nil, errors.New("Chat is already recorded")
}
