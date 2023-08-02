package api

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

func (api *API) handleRegisterUser(res http.ResponseWriter, req *http.Request) {
	authHeader := req.Header.Get("Authorization")
	split := strings.Split(authHeader, " ")
	if split[0] != "Basic" || len(split) != 2 {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	decoded, err := base64.StdEncoding.DecodeString(split[1])
	if err != nil {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	if api.config.BasicAuth != string(decoded) {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	params := req.URL.Query()
	username := params.Get("username")
	user, ok := api.db.FindUserByUsername(username)
	if !ok {
		res.WriteHeader(http.StatusNotFound)
		res.Write([]byte("User not found, have user chat in channel first"))
		return
	}

	guid := uuid.New().String()
	api.db.UpdateAPIKeyForUser(user.ID, guid)

	res.WriteHeader(http.StatusOK)
	res.Write([]byte(guid))
	return
}
