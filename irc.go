package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	twitchirc "github.com/gempir/go-twitch-irc/v2"
	dotenv "github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/soulxburn/soulxbot/api"
	"github.com/soulxburn/soulxbot/db"
	"github.com/soulxburn/soulxbot/dice"
	"github.com/soulxburn/soulxbot/irc"
	"github.com/soulxburn/soulxbot/twitch"
)

const (
	NOT_LIVE = iota
	LIVE_NO_FIRST
	LIVE_W_FIRST
)

type AppContext struct {
	Timers    map[string]*time.Timer
	TwitchAPI twitch.ITwitchAPI
	ClientIRC *twitchirc.Client
	DataStore *db.Database
	DiceGame  *dice.DiceGame
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

	basicAuth := os.Getenv("SOULXBOT_BASICAUTH")
	env := os.Getenv("SOULXBOT_ENV")

	AppCtx.DataStore = db.InitDatabase()
	AppCtx.TwitchAPI = twitch.NewTwitchAPI(clientID, clientSecret, authToken, refreshToken)
	AppCtx.ClientIRC = twitchirc.NewClient(user, oauth)
	AppCtx.DiceGame = dice.NewDiceGame(AppCtx.ClientIRC, AppCtx.TwitchAPI)

	apiConfig := api.Config{BasicAuth: basicAuth}
	httpApi := api.New(apiConfig, AppCtx.DataStore, AppCtx.TwitchAPI, AppCtx.ClientIRC)
	go httpApi.InitAPIAndListen()

	// I don't think I've ever seen this used.
	AppCtx.ClientIRC.OnUserNoticeMessage(func(message twitchirc.UserNoticeMessage) {
		fmt.Printf("Notice: %s\n", message.Message)
	})

	questionCommands := irc.QuestionCommands{
		DataStore: AppCtx.DataStore,
		ClientIRC: AppCtx.ClientIRC,
	}
	firstCommands := irc.FirstCommands{
		DataStore: AppCtx.DataStore,
		ClientIRC: AppCtx.ClientIRC,
	}

	commands := make(map[string]irc.CommandHandler)
	cmds := append(
		questionCommands.GetCommands(),
		firstCommands.GetCommands()...,
	)

	if env != "prod" {
		dev := "-dev"
		for _, c := range cmds {
			commands[c.CmdString+dev] = c.Cmd
		}
	}

	AppCtx.ClientIRC.OnPrivateMessage(func(message twitchirc.PrivateMessage) {
		messageUser := handleMessageUser(&message)
		streamUser, err := AppCtx.DataStore.FindStreamUserByUserName(strings.ToLower(message.Channel))
		if err != nil {
			log.Printf("Unable to find stream user for twitch channel %s", message.Channel)
		}
		stream := AppCtx.DataStore.FindCurrentStream(streamUser.UserId)

		msgCtx := irc.MessageContext{
			Channel:     message.Channel,
			MessageUser: messageUser,
			StreamUser:  streamUser,
			Stream:      stream,
		}

		if !streamUser.BotDisabled && isCommand(message.Message) {
			command, input := parseCommand(message.Message)
			cmd, ok := commands[command]
			if ok {
				cmd(msgCtx, strings.TrimSuffix(command, "-dev"), input)
			} else {
				// This is all deprecated
				switch command {
				case "startroll":
					if isSouLxBurN(streamUser.Username) {
						if AppCtx.DiceGame.CanRoll {
							if err := AppCtx.DiceGame.StartRoll(message.Channel); err != nil {
								log.Println("Failed to start roll: ", err)
							}
						} else {
							AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf("%s, That command is on cooldown", message.User.DisplayName))
						}
					}
				case "raid":
					if isSouLxBurN(streamUser.Username) {
						var buff strings.Builder
						for i := 0; i < 9; i++ {
							buff.WriteString("%[1]s %[2]s %[3]s ")
						}
						AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf(buff.String(), "PowerUpL", "soulxbGASMShake", "PowerUpR"))
					}
				case "thanos":
					err := thanos(streamUser, &message)
					if err != nil {
						log.Println(err)
					}
				}
			}
		}

		if stream != nil && stream.FirstUserId == nil && irc.IsFirstEnabled(streamUser) && irc.IsEligibleForFirst(stream, messageUser) {
			AppCtx.DataStore.UpdateFirstUser(stream.ID, messageUser.ID)
			AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf("Congratulations %s! You're first!", message.User.DisplayName))
		}

		fmt.Printf("[%s]%s: %s\n", message.Channel, message.User.DisplayName, message.Message)
	})

	// Join all channels that have an api key
	registeredUsers, err := AppCtx.DataStore.FindAllApiKeyUsers()
	if err != nil {
		log.Fatal("Failed to fetch registered users", err)
	}

	var usernames []string
	for _, user := range registeredUsers {
		usernames = append(usernames, user.Username)
		AppCtx.ClientIRC.Join(user.Username)
	}

	if err := AppCtx.ClientIRC.Connect(); err != nil {
		panic(err)
	}
}

func handleMessageUser(message *twitchirc.PrivateMessage) *db.User {
	var user *db.User
	messageUserId, err := strconv.Atoi(message.User.ID)
	if err != nil {
		log.Printf("Failed to convert twitch user id=%s to integer", message.User.ID)
		return nil
	}

	user, ok := AppCtx.DataStore.FindUserByID(messageUserId)
	if !ok {
		user = AppCtx.DataStore.InsertUser(messageUserId, message.User.Name, message.User.DisplayName)
		log.Printf("New User Found!: %d, %s, %s", user.ID, user.Username, user.DisplayName)
	}

	if user.Username != message.User.Name {
		if err := AppCtx.DataStore.UpdateUserName(user.ID, message.User.Name, message.User.DisplayName); err != nil {
			log.Printf("Failed to update userId=%d, to new username=%s: %v", user.ID, message.User.Name, err)
			return user
		}
		log.Printf("Updated Username Found! %d, %s to %s", user.ID, user.Username, message.User.Name)
		user, _ = AppCtx.DataStore.FindUserByID(user.ID)
	}

	return user
}

// isCommand
func isCommand(message string) bool {
	return strings.HasPrefix(message, "!")
}

// parseCommand
func parseCommand(message string) (string, string) {
	split := strings.Split(message[1:], " ")
	if len(split) >= 2 {
		return split[0], split[1]
	}
	return split[0], ""
}

// isSouLxBurN
func isSouLxBurN(username string) bool {
	return strings.ToLower(username) == "soulxburn"
}

func thanos(streamUser *db.StreamUser, message *twitchirc.PrivateMessage) error {
	if !isSouLxBurN(message.User.DisplayName) && !isSouLxBurN(message.Channel) {
		return errors.New("You are not Thanos")
	}
	usernames, err := AppCtx.ClientIRC.Userlist(message.Channel)
	if err != nil {
		log.Println("Thanos was unable to fetch the user list")
		return err
	}

	for len(usernames) > 0 {
		chunksize := len(usernames)
		if chunksize > 100 {
			chunksize = 100
		}
		chunk := usernames[:chunksize]
		usernames = usernames[chunksize:]

		userInfoList, err := AppCtx.TwitchAPI.GetUsers(chunk)
		if err != nil {
			continue
		}

		rand.Shuffle(len(userInfoList), func(i, j int) { userInfoList[i], userInfoList[j] = userInfoList[j], userInfoList[i] })
		theChosen := []*twitch.TwitchUserInfo{}

		for i, usr := range userInfoList {
			if i%2 == 0 {
				theChosen = append(theChosen, usr)
			}
		}

		for _, usr := range theChosen {
			AppCtx.TwitchAPI.TimeoutUser(fmt.Sprint(streamUser.User.ID), usr.Id, 60, "*SNAP*")
			fmt.Printf("Timed Out: %s\n", usr.DisplayName)
			time.Sleep(time.Millisecond * 200)
		}
	}
	return nil
}
