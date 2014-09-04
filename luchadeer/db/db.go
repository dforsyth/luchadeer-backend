package db

import (
	"appengine"
	"appengine/datastore"
	"luchadeer/giantbomb"
	"time"
)

type NotificationPreference struct {
	GCMRegistrationId string   `json:"gcm_registration_id"`
	Categories        []string `json:"categories"`
}

const KIND_NOTIFICATION_SUBSCRIPTION = "notificationpreference"
const KIND_GIANT_BOMB_VIDEO = "giantbombvideo"

func UpdateNotificationPreference(context appengine.Context, preference *NotificationPreference) error {
	key := datastore.NewKey(context, KIND_NOTIFICATION_SUBSCRIPTION, preference.GCMRegistrationId, 0, nil)
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
						context.Errorf("Push error %v", pe)
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
