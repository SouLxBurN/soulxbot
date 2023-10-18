package twitch

import (
	"encoding/json"
	"time"
)

type CreatePredictionBody struct {
	BroadcasterID    string    `json:"broadcaster_id"`
	Title            string    `json:"title"`
	PredictionWindow int       `json:"prediction_window"`
	Outcomes         []Outcome `json:"outcomes"`
}

type NewOutcome struct {
	Title string `json:"title"`
}

type EndPredictionBody struct {
	ID               string  `json:"id"`
	BroadcasterID    string  `json:"broadcaster_id"`
	Status           string  `json:"status"`
	WinningOutcomeID *string `json:"winning_outcome_id"`
}

type TwitchPredictionResponse struct {
	Data []*TwitchPrediction `json:"data"`
}

type TwitchPrediction struct {
	ID               string    `json:"id"`
	BroadcasterID    string    `json:"broadcaster_id"`
	BroadcasterName  string    `json:"broadcaster_name"`
	BroadcasterLogin string    `json:"broadcaster_login"`
	Title            string    `json:"title"`
	WinningOutcomeID *string   `json:"winning_outcome_id"`
	Outcomes         []Outcome `json:"outcomes"`
	PredictionWindow int       `json:"prediction_window"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	EndedAt          time.Time `json:"ended_at"`
	LockedAt         time.Time `json:"locked_at"`
}

type Outcome struct {
	ID            string          `json:"id"`
	Title         string          `json:"title"`
	Users         int             `json:"users"`
	ChannelPoints int             `json:"channel_points"`
	TopPredictors json.RawMessage `json:"top_predictors"`
	Color         string          `json:"color"`
}

type TokenResponse struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	Scope        []string `json:"scope"`
	TokenType    string   `json:"token_type"`
}

type TwitchDataResponse struct {
	Data json.RawMessage `json:"data"`
}

type TwitchStreamInfo struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	UserLogin    string    `json:"user_login"`
	UserName     string    `json:"user_name"`
	GameID       string    `json:"game_id"`
	GameName     string    `json:"game_name"`
	Type         string    `json:"type"`
	Title        string    `json:"title"`
	ViewerCount  int       `json:"viewer_count"`
	StartedAt    time.Time `json:"started_at"`
	Language     string    `json:"language"`
	ThumbnailURL string    `json:"thumbnail_url"`
	TagIDs       []string  `json:"tag_ids"`
	IsMature     bool      `json:"is_mature"`
}

type TwitchUsersResponse struct {
	Data []*TwitchUserInfo `json:"data"`
}

type TwitchUserInfo struct {
	Id              string `json:"id"`
	Login           string `json:"login"`
	DisplayName     string `json:"display_name"`
	Type            string `json:"type"`
	BroadcasterType string `json:"broadcaster_type"`
	Description     string `json:"description"`
	ProfileImageUrl string `json:"profile_image_url"`
	OfflineImageUrl string `json:"offline_image_url"`
	ViewCount       int    `json:"view_count"`
	Email           string `json:"email"`
	CreatedAt       string `json:"string"`
}

type BanUserRequest struct {
	Data TwitchBanUserRequestData `json:"data"`
}

type TwitchBanUserRequestData struct {
	UserID   string `json:"user_id"`
	Duration int    `json:"duration,omitempty"`
	Reason   string `json:"reason,omitempty"`
}
