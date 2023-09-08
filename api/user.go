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

	streamUser, err := api.db.FindStreamUserByUserID(user.ID)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte("Unable to register user"))
		return
	}
	if streamUser != nil && streamUser.StreamConfig.ID != 0 {
		res.WriteHeader(http.StatusBadRequest)
		res.Write([]byte("User is already registered"))
		return
	}

	_, err = api.db.CreateStreamConfig(user.ID)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte("Unable to register user"))
		return
	}

	guid := uuid.New().String()
	if err := api.db.UpdateAPIKeyForUser(user.ID, guid); err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte("Unable to register user"))
		return
	}
	res.WriteHeader(http.StatusOK)
	res.Write([]byte(guid))
	return
}
