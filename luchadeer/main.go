package luchadeer

import (
	"luchadeer/api"
	"luchadeer/config"
	"luchadeer/cron"
	"luchadeer/tasks"
	"net/http"
)

func init() {
	api.Init()
	cron.Init()
	tasks.Init()

	http.HandleFunc("/", homeHandler)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, config.ClientDownloadURL, http.StatusSeeOther)
}
