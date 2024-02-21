package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/soulxburn/soulxbot/db"
)

func (api *API) handleRegisterUser(res http.ResponseWriter, req *http.Request) {
	claims := struct {
		IDToken struct {
			PreferredUsername *string `json:"preferred_username"`
		} `json:"id_token"`
	}{}
	claimsJson, _ := json.Marshal(claims)

	var redirect url.URL
	redirect.Scheme = "https"
	redirect.Host = "id.twitch.tv"
	redirect.Path = "/oauth2/authorize"
	query := redirect.Query()
	query.Add("client_id", api.config.ClientID)
	query.Add("redirect_uri", api.config.RedirectURI)
	query.Add("response_type", "code")
	query.Add("scope", "openid channel:read:redemptions channel:manage:predictions moderator:manage:banned_users")
	query.Add("claims", string(claimsJson))
	redirect.RawQuery = query.Encode()

	res.Header().Add("Location", redirect.String())
	res.WriteHeader(http.StatusFound)

	return
}

type JWTUser struct {
	ID                string
	PreferredUsername string
}

func parseIDToken(idToken string) (*JWTUser, error) {
	k, err := keyfunc.NewDefault([]string{"https://id.twitch.tv/oauth2/keys"})
	if err != nil {
		log.Println("Failed to create jwt keyfunc", err)
		return nil, err
	}

	token, err := jwt.Parse(idToken, k.Keyfunc)
	if err != nil {
		return nil, err
	}

	claims, _ := token.Claims.(jwt.MapClaims)
	username := claims["preferred_username"].(string)
	userId, err := claims.GetSubject()
	if err != nil {
		return nil, err
	}

	return &JWTUser{ID: userId, PreferredUsername: username}, nil
}

func (api *API) handleOAuthRegisterUser(res http.ResponseWriter, req *http.Request) {
	params := req.URL.Query()
	code := params.Get("code")

	tokenResponse, err := api.twitchAPI.GetAuthenticatedUser(code)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte("Failed to obtain authenticated user"))
		log.Println("Failed to obtain authenticated user")
		return
	}
	encryptedAuthToken, authErr := db.EncryptToken(tokenResponse.AccessToken, api.config.KeyPhrase)
	encryptedRefreshToken, refreshErr := db.EncryptToken(tokenResponse.RefreshToken, api.config.KeyPhrase)
	if authErr != nil || refreshErr != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte("Failed to encrypt oauth credentials"))
		log.Println("Error during token encryption", authErr, refreshErr)
		return
	}

	jwtUser, err := parseIDToken(*tokenResponse.IDToken)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte("Error encountered parsing response token"))
		log.Println("Failed to parse id_token", err)
		return
	}

	intId, err := strconv.Atoi(jwtUser.ID)

	streamUser, err := api.db.FindStreamUserByUserID(intId)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte("Unable to register user"))
		return
	}
	if streamUser != nil && streamUser.StreamConfig.ID != 0 {
		if err := api.db.UpdateTwitchAuth(streamUser.UserId, encryptedAuthToken, encryptedRefreshToken); err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			resp := fmt.Sprintf(
				"User is already registered. Failed to update twitch authentication. Go-live key: %s",
				streamUser.StreamConfig.APIKey)
			res.Write([]byte(resp))
			return
		}
		res.WriteHeader(http.StatusOK)
		res.Write([]byte(fmt.Sprintf("User is already registered. Go-live key: %s", streamUser.StreamConfig.APIKey)))
		log.Printf("Re-registered stream_user: %s", streamUser.Username)
		return
	}

	user, ok := api.db.FindUserByID(intId)
	if !ok {
		user = api.db.InsertUser(intId, strings.ToLower(jwtUser.PreferredUsername), jwtUser.PreferredUsername)
	}

	goliveApiKey := uuid.New().String()

	config, err := api.db.CreateStreamConfig(user.ID, goliveApiKey, encryptedAuthToken, encryptedRefreshToken)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte("Unable to register user"))
		return
	}

	res.WriteHeader(http.StatusOK)
	res.Write([]byte(fmt.Sprintf("Registration Success! Go-live key: %s", config.APIKey)))
	log.Printf("Successfully registered new stream_user: %s", user.Username)
	api.twitchIRC.Join(user.Username)
}
