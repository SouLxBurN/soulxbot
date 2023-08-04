package api

import (
	"net/http"

	"github.com/google/uuid"
)

func (api *API) handleRegisterUser(res http.ResponseWriter, req *http.Request) {
	authenticated := api.AuthenticateRequest(res, req)
	if !authenticated {
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
