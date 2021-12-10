package twitch

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
)

const (
	TWITCH_HELIX_API = "https://api.twitch.tv/helix"
	TWITCH_OAUTH_API = "https://id.twitch.tv/oauth2"
	TOKEN            = "/token"
	PREDICTIONS      = "/predictions"
	STREAMS          = "/streams"
)

type ITwitchAPI interface {
	GetStream(string) (*TwitchStreamInfo, error)
	CreatePrediction(string, int, []string) (*TwitchPrediction, error)
	EndPrediction(*TwitchPrediction, string) error
	RefreshToken() error
}

type TwitchAPI struct {
	clientID     string
	clientSecret string
	authToken    string
	refreshToken string
}

func NewTwitchAPI(clientID string, clientSecret string, authToken string, refreshToken string) ITwitchAPI {
	return &TwitchAPI{
		clientID:     clientID,
		clientSecret: clientSecret,
		authToken:    authToken,
		refreshToken: refreshToken,
	}
}

// GetStream
func (a *TwitchAPI) GetStream(user string) (*TwitchStreamInfo, error) {
	req, err := http.NewRequest("GET", TWITCH_HELIX_API+STREAMS, bytes.NewReader([]byte{}))
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", a.authToken))
	req.Header.Add("Client-Id", a.clientID)

	q := req.URL.Query()
	q.Add("user_login", user)
	req.URL.RawQuery = q.Encode()

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	if response.ContentLength <= 0 {
		return nil, errors.New("Error fetching stream info, Unexpected content length")
	}

	respBody := make([]byte, response.ContentLength)
	response.Body.Read(respBody)

	data := new(TwitchDataResponse)
	if err := json.Unmarshal(respBody, data); err != nil {
		return nil, err
	}

	streams := []*TwitchStreamInfo{}
	if err := json.Unmarshal(data.Data, &streams); err != nil {
		return nil, err
	}

	if len(streams) <= 0 {
		return nil, nil
	}

	return streams[0], nil
}

// CreatePrediction
func (a *TwitchAPI) CreatePrediction(title string, window int, outcomes []string) (*TwitchPrediction, error) {
	requestBody := CreatePredictionBody{
		BroadcasterID:    "31568083",
		Title:            title,
		PredictionWindow: window,
		Outcomes:         make([]Outcome, len(outcomes)),
	}

	for i, outcome := range outcomes {
		o := Outcome{
			Title: outcome,
		}
		requestBody.Outcomes[i] = o
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		log.Println("Failed to Marshal Body")
	}

	req, err := http.NewRequest("POST", TWITCH_HELIX_API+PREDICTIONS, bytes.NewReader(body))
	req.Header.Add("content-type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", a.authToken))
	req.Header.Add("Client-Id", a.clientID)

	response, err := http.DefaultClient.Do(req)
	respBody := make([]byte, response.ContentLength)
	response.Body.Read(respBody)

	predictionResp := new(TwitchPredictionResponse)
	json.Unmarshal(respBody, predictionResp)

	fmt.Println(response)
	fmt.Println(err)

	return predictionResp.Data[0], nil
}

// EndPrediction
func (a *TwitchAPI) EndPrediction(prediction *TwitchPrediction, winningID string) error {
	requestBody := EndPredictionBody{
		ID:               prediction.ID,
		BroadcasterID:    prediction.BroadcasterID,
		Status:           "RESOLVED",
		WinningOutcomeID: &winningID,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		log.Println("Failed to Marshal Body")
	}

	req, err := http.NewRequest("PATCH", TWITCH_HELIX_API+PREDICTIONS, bytes.NewReader(body))
	req.Header.Add("content-type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", a.authToken))
	req.Header.Add("Client-Id", a.clientID)

	response, err := http.DefaultClient.Do(req)
	respBody := make([]byte, response.ContentLength)
	response.Body.Read(respBody)

	predictionResp := new(TwitchPredictionResponse)
	json.Unmarshal(respBody, predictionResp)

	fmt.Println(response)
	fmt.Println(err)

	return nil
}

// RefreshToken
func (a *TwitchAPI) RefreshToken() error {
	req, err := http.NewRequest("POST", TWITCH_OAUTH_API+TOKEN, bytes.NewReader([]byte{}))
	req.Header.Add("Client-Id", a.clientID)

	q := req.URL.Query()
	q.Add("grant_type", "refresh_token")
	q.Add("client_id", a.clientID)
	q.Add("client_secret", a.clientSecret)
	q.Add("refresh_token", a.refreshToken)
	req.URL.RawQuery = q.Encode()

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(err)
		return err
	}

	if response.ContentLength <= 0 {
		return errors.New("Error refreshing token. Unexpected content length")
	}

	respBody := make([]byte, response.ContentLength)
	response.Body.Read(respBody)

	newTokens := new(TokenResponse)
	json.Unmarshal(respBody, newTokens)

	a.authToken = newTokens.AccessToken
	a.refreshToken = newTokens.RefreshToken

	return nil
}
