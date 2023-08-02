package api

import (
	"net/http"

	"github.com/soulxburn/soulxbot/db"
	"github.com/soulxburn/soulxbot/twitch"
)

type Config struct {
	BasicAuth string
}

type API struct {
	config    Config
	db        *db.Database
	twitchAPI twitch.ITwitchAPI
}

func New(config Config, database *db.Database, twitchAPI twitch.ITwitchAPI) *API {
	return &API{
		config:    config,
		db:        database,
		twitchAPI: twitchAPI,
	}
}

func (api *API) InitAPIAndListen() error {
	poller := StreamPoller{
		db:        api.db,
		twitchAPI: api.twitchAPI,
	}
	poller.RestartStreamStatusPolls()

	http.HandleFunc("/register", api.handleRegisterUser)
	http.HandleFunc("/golive", poller.goliveHandler)

	return http.ListenAndServe(":8080", nil)
}
