package twitch

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/soulxburn/soulxbot/db"
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
	GetAuthenticatedUser(string) (*TokenResponse, error)
	CreatePrediction(db.StreamUser, string, int, []string) (*TwitchPrediction, error)
	EndPrediction(db.StreamUser, *TwitchPrediction, string) error
	TimeoutUser(db.StreamUser, string, int, string) error
}

type TwitchAPI struct {
	clientID         string
	clientSecret     string
	accessToken      string
	oauthRedirectUri string
	db               *db.Database
	keyPhrase        string
}

// NewTwitchAPI
func NewTwitchAPI(clientID string, clientSecret string, db *db.Database, oauthRedirectUri string, keyPhrase string) ITwitchAPI {
	return &TwitchAPI{
		clientID:         clientID,
		clientSecret:     clientSecret,
		db:               db,
		oauthRedirectUri: oauthRedirectUri,
		keyPhrase:        keyPhrase,
	}
}

// GetStream
func (a *TwitchAPI) GetStream(user string) (*TwitchStreamInfo, error) {
	authToken, err := a.getAppAccessToken()
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
	authToken, err := a.getAppAccessToken()
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
func (a *TwitchAPI) CreatePrediction(user db.StreamUser, title string, window int, outcomes []string) (*TwitchPrediction, error) {
	requestBody := CreatePredictionBody{
		BroadcasterID:    strconv.Itoa(user.UserId),
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

	authToken, err := a.getUserAuthToken(user)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", TWITCH_HELIX_API+PREDICTIONS, bytes.NewReader(body))
	req.Header.Add("content-type", "application/json")
	req.Header.Add("Client-Id", a.clientID)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", *authToken))

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

	return predictionResp.Data[0], nil
}

// EndPrediction
// Ends a twitch prediction
func (a *TwitchAPI) EndPrediction(user db.StreamUser, prediction *TwitchPrediction, winningID string) error {
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

	authToken, err := a.getUserAuthToken(user)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PATCH", TWITCH_HELIX_API+PREDICTIONS, bytes.NewReader(body))
	req.Header.Add("content-type", "application/json")
	req.Header.Add("Client-Id", a.clientID)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", *authToken))

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

	return nil
}

func (a *TwitchAPI) TimeoutUser(user db.StreamUser, userID string, duration int, reason string) error {
	requestBody := BanUserRequest{
		Data: TwitchBanUserRequestData{userID, duration, reason},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}

	authToken, err := a.getUserAuthToken(user)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, TWITCH_HELIX_API+BANS, bytes.NewReader(body))
	req.Header.Add("content-type", "application/json")
	req.Header.Add("Client-Id", a.clientID)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", *authToken))

	q := req.URL.Query()
	q.Add("broadcaster_id", strconv.Itoa(user.UserId))
	q.Add("moderator_id", strconv.Itoa(user.UserId))
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

// getUserAuthToken
func (a *TwitchAPI) getUserAuthToken(user db.StreamUser) (*string, error) {
	if user.TwitchAuthToken == nil {
		log.Printf("User=%d is missing auth token", user.User.ID)
		return nil, errors.New("Missing Auth Token")
	}

	authToken, err := db.DecryptToken(*user.TwitchAuthToken, a.keyPhrase)
	if err != nil {
		log.Println("Failed to decrypt authToken", err)
		return nil, err
	}

	if !a.validateAuthToken(authToken) {
		newAuthToken, err := a.refresUserAuthToken(user)
		if err != nil {
			return nil, err
		}
		return newAuthToken, nil
	}

	return &authToken, nil
}

func (a *TwitchAPI) GetAuthenticatedUser(code string) (*TokenResponse, error) {
	req, err := http.NewRequest("POST", TWITCH_OAUTH_API+TOKEN, bytes.NewReader([]byte{}))

	q := req.URL.Query()
	q.Add("grant_type", "authorization_code")
	q.Add("client_id", a.clientID)
	q.Add("client_secret", a.clientSecret)
	q.Add("code", code)
	q.Add("redirect_uri", a.oauthRedirectUri)
	req.URL.RawQuery = q.Encode()

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return nil, errors.New("Critical: Failed to to get authenticated user")
	}
	if response.ContentLength <= 0 {
		return nil, errors.New("Error getting authenticated user. Unexpected content length")
	}

	respBody := make([]byte, response.ContentLength)
	response.Body.Read(respBody)

	newTokens := new(TokenResponse)
	if err := json.Unmarshal(respBody, newTokens); err != nil {
		return nil, err
	}

	return newTokens, nil
}

// refreshUserAuthToken
func (a *TwitchAPI) refresUserAuthToken(user db.StreamUser) (*string, error) {
	req, err := http.NewRequest("POST", TWITCH_OAUTH_API+TOKEN, bytes.NewReader([]byte{}))
	req.Header.Add("Client-Id", a.clientID)

	if user.TwitchRefreshToken == nil {
		log.Printf("User=%d is missing refresh token", user.User.ID)
		return nil, errors.New("Missing Refresh Token")
	}

	refreshToken, err := db.DecryptToken(*user.TwitchRefreshToken, a.keyPhrase)
	if err != nil {
		log.Println("Failed to decrypt refresh token", err)
		return nil, err
	}

	q := req.URL.Query()
	q.Add("grant_type", "refresh_token")
	q.Add("client_id", a.clientID)
	q.Add("client_secret", a.clientSecret)
	q.Add("refresh_token", refreshToken)
	req.URL.RawQuery = q.Encode()

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return nil, errors.New("Critical: Failed to refresh Token")
	}
	if response.ContentLength <= 0 {
		return nil, errors.New("Error refreshing token. Unexpected content length")
	}

	respBody := make([]byte, response.ContentLength)
	response.Body.Read(respBody)

	newTokens := new(TokenResponse)
	if err := json.Unmarshal(respBody, newTokens); err != nil {
		return nil, err
	}
	encryptedAuthToken, authErr := db.EncryptToken(newTokens.AccessToken, a.keyPhrase)
	encryptedRefreshToken, refreshErr := db.EncryptToken(newTokens.RefreshToken, a.keyPhrase)
	if authErr != nil || refreshErr != nil {
		log.Println("Error during token encryption", authErr, refreshErr)
		return nil, authErr
	}

	err = a.db.UpdateTwitchAuth(user.UserId, encryptedAuthToken, encryptedRefreshToken)
	if err != nil {
		return nil, err
	}

	return &newTokens.AccessToken, nil
}

// validateAuthToken
func (a *TwitchAPI) validateAuthToken(authToken string) bool {
	req, err := http.NewRequest(http.MethodGet, TWITCH_OAUTH_API+VALIDATE, bytes.NewReader([]byte{}))
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", authToken))

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

// getAppAccessToken
// curl -X POST 'https://id.twitch.tv/oauth2/token' \
// -H 'Content-Type: application/x-www-form-urlencoded' \
// -d 'client_id=<your client id goes here>&client_secret=<your client secret goes here>&grant_type=client_credentials'
func (a *TwitchAPI) getAppAccessToken() (string, error) {
	// Validate, and return cached token if valid
	if a.validateAuthToken(a.accessToken) {
		return a.accessToken, nil
	}

	// Otherwise, get new token
	data := url.Values{}
	data.Set("client_id", a.clientID)
	data.Set("client_secret", a.clientSecret)
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequest(http.MethodPost, TWITCH_OAUTH_API+TOKEN, strings.NewReader(data.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return "", errors.New("Critical: Failed to get app access token")
	}
	if response.ContentLength <= 0 {
		return "", errors.New("Error getting app access token. Unexpected content length")
	}

	respBody := make([]byte, response.ContentLength)
	response.Body.Read(respBody)

	newAccessToken := new(TokenResponse)
	if err := json.Unmarshal(respBody, newAccessToken); err != nil {
		return "", err
	}

	a.accessToken = newAccessToken.AccessToken
	return a.accessToken, nil
}
