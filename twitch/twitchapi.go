package twitch

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
)

const (
	TWITCH_HELIX_API = "https://api.twitch.tv/helix"
	TWITCH_OAUTH_API = "https://id.twitch.tv/oauth2"
	BANS             = "/moderation/bans"
	TOKEN            = "/token"
	PREDICTIONS      = "/predictions"
	STREAMS          = "/streams"
	USERS            = "/users"
	VALIDATE         = "/validate"
)

type ITwitchAPI interface {
	GetStream(string) (*TwitchStreamInfo, error)
	GetUsers([]string) ([]*TwitchUserInfo, error)
	CreatePrediction(string, int, []string) (*TwitchPrediction, error)
	EndPrediction(*TwitchPrediction, string) error
	TimeoutUser(string, string, int, string) error
}

type TwitchAPI struct {
	clientID     string
	clientSecret string
	authToken    string
	refreshToken string
}

// NewTwitchAPI
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
	authToken, err := a.getAuthToken()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("GET", TWITCH_HELIX_API+STREAMS, bytes.NewReader([]byte{}))
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", authToken))
	req.Header.Add("Client-Id", a.clientID)

	q := req.URL.Query()
	q.Add("user_login", user)
	req.URL.RawQuery = q.Encode()

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		if response != nil {
			response.Body.Close()
		}
		return nil, err
	}
	defer response.Body.Close()
	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

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

// Get up to 100 users from twitch based on a list of twitch usernames
func (a *TwitchAPI) GetUsers(usernames []string) ([]*TwitchUserInfo, error) {
	authToken, err := a.getAuthToken()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("GET", TWITCH_HELIX_API+USERS, bytes.NewReader([]byte{}))
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", authToken))
	req.Header.Add("Client-Id", a.clientID)

	q := req.URL.Query()
	for _, u := range usernames {
		q.Add("login", u)
	}
	req.URL.RawQuery = q.Encode()

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		if response != nil {
			response.Body.Close()
		}
		return nil, err
	}
	if response.StatusCode != 200 {
		log.Printf("GetUsers returned non-200 status: %s", response.Status)
	}
	defer response.Body.Close()
	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	data := new(TwitchUsersResponse)
	if err := json.Unmarshal(respBody, data); err != nil {
		return nil, err
	}

	users := data.Data
	if len(users) <= 0 {
		return nil, nil
	}

	return users, nil
}

// CreatePrediction
// Creates a twitch prediction
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

	authToken, err := a.getAuthToken()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", TWITCH_HELIX_API+PREDICTIONS, bytes.NewReader(body))
	req.Header.Add("content-type", "application/json")
	req.Header.Add("Client-Id", a.clientID)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		if response != nil {
			response.Body.Close()
		}
		return nil, err
	}
	defer response.Body.Close()

	respBody := make([]byte, response.ContentLength)
	response.Body.Read(respBody)

	predictionResp := new(TwitchPredictionResponse)
	json.Unmarshal(respBody, predictionResp)

	log.Println(response)

	return predictionResp.Data[0], nil
}

// EndPrediction
// Ends a twitch prediction
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

	authToken, err := a.getAuthToken()
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PATCH", TWITCH_HELIX_API+PREDICTIONS, bytes.NewReader(body))
	req.Header.Add("content-type", "application/json")
	req.Header.Add("Client-Id", a.clientID)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		if response != nil {
			response.Body.Close()
		}
		return err
	}
	defer response.Body.Close()

	respBody := make([]byte, response.ContentLength)
	response.Body.Read(respBody)

	predictionResp := new(TwitchPredictionResponse)
	json.Unmarshal(respBody, predictionResp)

	log.Println(response)

	return nil
}

func (a *TwitchAPI) TimeoutUser(streamerID string, userID string, duration int, reason string) error {
	requestBody := BanUserRequest{
		Data: TwitchBanUserRequestData{userID, duration, reason},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}

	authToken, err := a.getAuthToken()
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, TWITCH_HELIX_API+BANS, bytes.NewReader(body))
	req.Header.Add("content-type", "application/json")
	req.Header.Add("Client-Id", a.clientID)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

	q := req.URL.Query()
	q.Add("broadcaster_id", streamerID)
	q.Add("moderator_id", streamerID)
	req.URL.RawQuery = q.Encode()

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		if response != nil && response.Body != nil {
			response.Body.Close()
		}
		return err
	}
	if response.StatusCode != 200 {
		body, _ := io.ReadAll(response.Body)
		log.Printf("Timeout user returned non-200 status %s | %s", response.Status, body)
	}
	defer response.Body.Close()

	return nil
}

// getAuthToken
func (a *TwitchAPI) getAuthToken() (string, error) {
	if !a.validateAuthToken() {
		if err := a.refreshAuthToken(); err != nil {
			return "", err
		}
	}
	return a.authToken, nil
}

// refreshAuthToken
func (a *TwitchAPI) refreshAuthToken() error {
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
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return errors.New("Critical: Failed to refresh Token")
	}
	if response.ContentLength <= 0 {
		return errors.New("Error refreshing token. Unexpected content length")
	}

	respBody := make([]byte, response.ContentLength)
	response.Body.Read(respBody)

	newTokens := new(TokenResponse)
	if err := json.Unmarshal(respBody, newTokens); err != nil {
		return err
	}

	a.authToken = newTokens.AccessToken
	a.refreshToken = newTokens.RefreshToken

	return nil
}

// validateAuthToken
func (a *TwitchAPI) validateAuthToken() bool {
	req, err := http.NewRequest(http.MethodGet, TWITCH_OAUTH_API+VALIDATE, bytes.NewReader([]byte{}))
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", a.authToken))

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
		return false
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK || response.ContentLength <= 0 {
		return false
	}

	respBody := make([]byte, response.ContentLength)
	response.Body.Read(respBody)
	return true
}
