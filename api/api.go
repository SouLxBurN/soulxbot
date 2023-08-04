package api

import (
	"encoding/base64"
	"net/http"
	"strings"

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
	poller := NewStreamPoller(api.db, api.twitchAPI)
	poller.RestartStreamStatusPolls()

	mux := http.NewServeMux()

	mux.HandleFunc("/question/", api.getQuestion)
	mux.HandleFunc("/question", api.createQuestion)
	mux.HandleFunc("/register", api.handleRegisterUser)
	mux.HandleFunc("/golive", poller.goliveHandler)

	return http.ListenAndServe(":8080", mux)
}

func (api *API) AuthenticateRequest(res http.ResponseWriter, req *http.Request) bool {
	authHeader := req.Header.Get("Authorization")
	split := strings.Split(authHeader, " ")
	if split[0] != "Basic" || len(split) != 2 {
		res.WriteHeader(http.StatusBadRequest)
		res.Write([]byte("Invalid Authorization Header"))
		return false
	}

	decoded, err := base64.StdEncoding.DecodeString(split[1])
	if err != nil {
		res.WriteHeader(http.StatusUnauthorized)
		res.Write([]byte("Authentication Failed"))
		return false
	}

	if api.config.BasicAuth != string(decoded) {
		res.WriteHeader(http.StatusUnauthorized)
		res.Write([]byte("Authentication Failed"))
		return false
	}

	return true
}
