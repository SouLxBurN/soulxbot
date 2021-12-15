package main

import (
	"fmt"
	"log"
	"os"
	"time"

	twitchirc "github.com/gempir/go-twitch-irc/v2"
	dotenv "github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/soulxburn/soulxbot/db"
	"github.com/soulxburn/soulxbot/dice"
	"github.com/soulxburn/soulxbot/twitch"
)

const (
	NOT_LIVE = iota
	LIVE_NO_FIRST
	LIVE_W_FIRST
)

type AppContext struct {
	StreamState int
	Timers      map[string]*time.Timer
	TwitchAPI   twitch.ITwitchAPI
	ClientIRC   *twitchirc.Client
	DataStore   *db.Database
	DiceGame    *dice.DiceGame
}

var AppCtx AppContext

func main() {
	if err := dotenv.Load(); err != nil {
		log.Println("No .env file found")
	}
	user := os.Getenv("SOULXBOT_USER")
	oauth := os.Getenv("SOULXBOT_OAUTH")

	clientID := os.Getenv("SOULXBOT_CLIENTID")
	clientSecret := os.Getenv("SOULXBOT_CLIENTSECRET")
	authToken := os.Getenv("SOULXBOT_AUTHTOKEN")
	refreshToken := os.Getenv("SOULXBOT_REFRESHTOKEN")

	AppCtx.StreamState = NOT_LIVE
	AppCtx.DataStore = db.InitDatabase()
	AppCtx.TwitchAPI = twitch.NewTwitchAPI(clientID, clientSecret, authToken, refreshToken)
	AppCtx.ClientIRC = twitchirc.NewClient(user, oauth)
	AppCtx.DiceGame = dice.NewDiceGame(AppCtx.ClientIRC, AppCtx.TwitchAPI)

	go pollStreamStatus()

	AppCtx.ClientIRC.OnUserNoticeMessage(func(message twitchirc.UserNoticeMessage) {
		fmt.Printf("Notice: %s\n", message.Message)
	})

	AppCtx.ClientIRC.OnPrivateMessage(func(message twitchirc.PrivateMessage) {
		user, ok := AppCtx.DataStore.FindUserByUsername(message.User.DisplayName)
		if !ok {
			user = AppCtx.DataStore.InsertUser(message.User.DisplayName)
		}

		if AppCtx.StreamState == LIVE_NO_FIRST {
			AppCtx.StreamState = LIVE_W_FIRST
			AppCtx.DataStore.IncrementTimesFirst(user.ID)
			// context.ClientIRC.Say(message.Channel, fmt.Sprintf("Congratulations %s! You're first!", message.User.DisplayName))
		}

		if isCommand(message.Message) {
			switch message.Message[1:] {
			case "firstLeaders":
				leaders, _ := AppCtx.DataStore.TimesFirstLeaders(3)
				for i, v := range leaders {
					AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf("%d. %s - %d", i+1, v.Username, v.TimesFirst))
				}
			case "printall":
				AppCtx.DataStore.FindAllUsers()
			case "startroll":
				if AppCtx.DiceGame.CanRoll {
					if err := AppCtx.DiceGame.StartRoll(message.Channel); err != nil {
						log.Println("Failed to start roll: ", err)
					}
				} else {
					AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf("%s, That command is on cooldown", message.User.DisplayName))
				}
			}
		}

		fmt.Printf("%s: %s\n", message.User.DisplayName, message.Message)
	})

	AppCtx.ClientIRC.Join("SouLxBurN")

	if err := AppCtx.ClientIRC.Connect(); err != nil {
		panic(err)
	}
}

// pollStreamStatus
func pollStreamStatus() {
	tick := time.NewTicker(1 * time.Minute)
	for {
		select {
		case <-tick.C:
			streamInfo, err := AppCtx.TwitchAPI.GetStream("SouLxBurN")
			if err != nil {
				log.Println("Error fetching stream info: ", err)
			}

			if streamInfo == nil {
				AppCtx.StreamState = NOT_LIVE
			} else {
				if AppCtx.StreamState == NOT_LIVE {
					AppCtx.StreamState = LIVE_NO_FIRST
				}
			}

		}
	}
}

// isCommand
func isCommand(message string) bool {
	if message[0:1] == "!" {
		return true
	}
	return false
}
