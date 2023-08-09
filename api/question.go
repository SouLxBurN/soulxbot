package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type QuestionRequestBody struct {
	Question string `json:"question"`
}

type QuestionPatchBody struct {
	ID       int64 `json:"id"`
	Disabled bool  `json:"disabled"`
}

func (api *API) handleQuestionWrites(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "PATCH":
		api.patchQuestion(res, req)
	case "POST":
		api.createQuestion(res, req)
	default:
		log.Println("Invalid verb encountered")
	}
}

func (api *API) getQuestion(res http.ResponseWriter, req *http.Request) {
	id, err := strconv.Atoi(strings.Split(req.URL.Path, "/")[2])
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		res.Write([]byte("Unable to parse id"))
		return
	}

	question, ok := api.db.FindQuestionByID(int64(id))
	if !ok {
		res.WriteHeader(http.StatusNotFound)
		res.Write([]byte("No question found with that id"))
	}

	json.NewEncoder(res).Encode(question)
}

func (api *API) patchQuestion(res http.ResponseWriter, req *http.Request) {
	authenticated := api.AuthenticateRequest(res, req)
	if !authenticated {
		return
	}

	var body QuestionPatchBody
	err := json.NewDecoder(req.Body).Decode(&body)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		res.Write([]byte("Invalid Request"))
		return
	}

	_, ok := api.db.FindQuestionByID(body.ID)
	if !ok {
		res.WriteHeader(http.StatusNotFound)
		res.Write([]byte("Question id does not exist"))
		return
	}

	// TODO: Allow for ENABLING a disabled question
	err = api.db.DisableQuestion(&body.ID)
	if err != nil {
		res.WriteHeader(http.StatusConflict)
		res.Write([]byte(strings.TrimSpace(err.Error())))
	}

	question, _ := api.db.FindQuestionByID(body.ID)
	json.NewEncoder(res).Encode(question)
}

func (api *API) createQuestion(res http.ResponseWriter, req *http.Request) {
	authenticated := api.AuthenticateRequest(res, req)
	if !authenticated {
		return
	}

	var body QuestionRequestBody
	err := json.NewDecoder(req.Body).Decode(&body)
	if err != nil || body.Question == "" {
		res.WriteHeader(http.StatusBadRequest)
		res.Write([]byte("Invalid Request"))
		return
	}

	question, err := api.db.CreateQuestion(body.Question)
	if err != nil {
		res.WriteHeader(http.StatusConflict)
		res.Write([]byte(strings.TrimSpace(err.Error())))
	}

	json.NewEncoder(res).Encode(question)
}
