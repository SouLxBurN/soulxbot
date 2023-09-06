package main

import (
	"errors"
	"fmt"
	"log"
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

	AppCtx.DataStore = db.InitDatabase()
	AppCtx.TwitchAPI = twitch.NewTwitchAPI(clientID, clientSecret, authToken, refreshToken)
	AppCtx.ClientIRC = twitchirc.NewClient(user, oauth)
	AppCtx.DiceGame = dice.NewDiceGame(AppCtx.ClientIRC, AppCtx.TwitchAPI)

	apiConfig := api.Config{BasicAuth: basicAuth}
	httpApi := api.New(apiConfig, AppCtx.DataStore, AppCtx.TwitchAPI)
	go httpApi.InitAPIAndListen()

	// I don't think I've ever seen this used.
	AppCtx.ClientIRC.OnUserNoticeMessage(func(message twitchirc.UserNoticeMessage) {
		fmt.Printf("Notice: %s\n", message.Message)
	})

	AppCtx.ClientIRC.OnPrivateMessage(func(message twitchirc.PrivateMessage) {
		messageUser := handleMessageUser(&message)
		streamUser, err := AppCtx.DataStore.FindStreamUserByUserName(strings.ToLower(message.Channel))
		if err != nil {
			log.Printf("Unable to find stream user for twitch channel %s", message.Channel)
		}
		stream := AppCtx.DataStore.FindCurrentStream(streamUser.UserId)

		if !streamUser.BotDisabled && isCommand(message.Message) {
			command, input := parseCommand(message.Message)

			switch command {
			case "first":
				if !isFirstEnabled(streamUser) {
					return
				}
				if stream != nil && stream.FirstUserId != nil && streamUser != nil && streamUser.FirstEnabled {
					if *stream.FirstUserId != messageUser.ID {
						firstUser, _ := AppCtx.DataStore.FindUserByID(*stream.FirstUserId)
						AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf("Sorry %s, you are not first. %s was!", message.User.DisplayName, firstUser.DisplayName))
					} else {
						AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf("/timeout %[1]s 60 Yes %[1]s! We KNOW. You were first...", message.User.DisplayName))
					}
				}
			case "firstcount":
				if !isFirstEnabled(streamUser) {
					return
				}
				timesFirst, err := AppCtx.DataStore.FindUserTimesFirst(streamUser.User.ID, messageUser.ID)
				if err != nil {
					log.Println(err)
					return
				}
				AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf("%s, you have been first %d times", messageUser.DisplayName, timesFirst))
			case "firstleaders":
				if !isFirstEnabled(streamUser) {
					return
				}
				leaders, _ := AppCtx.DataStore.FindFirstLeaders(streamUser.User.ID, 3)
				for i, v := range leaders {
					AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf("%d. %s - %d", i+1, v.User.DisplayName, v.TimesFirst))
				}
			case "firstgive":
				if stream != nil && stream.UserId == messageUser.ID && len(input) > 0 && isFirstEnabled(streamUser) {
					targetUser, found := AppCtx.DataStore.FindUserByUsername(input)
					if found {
						AppCtx.DataStore.UpdateFirstUser(stream.ID, targetUser.ID)
						AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf("%s has been set as first for this stream!", targetUser.Username))
					} else {
						AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf("That user does not exist"))
					}
				}
			case "qotd":
				if stream != nil && streamUser != nil && streamUser.QotdEnabled {
					questionOfTheDay(stream, &message)
				}
			case "skipqotd":
				if stream != nil && stream.UserId == messageUser.ID && stream.QOTDId != nil && streamUser != nil && streamUser.QotdEnabled {
					if skipCount, _ := AppCtx.DataStore.IncrementQuestionSkip(*stream.QOTDId); skipCount > 2 {
						AppCtx.DataStore.DisableQuestion(*stream.QOTDId)
					}
					AppCtx.DataStore.UpdateStreamQuestion(stream.ID, nil)
					AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf("Question of the day skipped, enter !qotd to get a new question"))
				}
			case "printall":
				if isSouLxBurN(streamUser.Username) {
					users, err := AppCtx.DataStore.FindAllUsers()
					if err != nil {
						log.Println("Failed to find all users", err)
					}
					log.Printf("%v", users)
				}
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
				thanos(&message)
			}
		}

		if stream != nil && stream.FirstUserId == nil && isFirstEnabled(streamUser) && isEligibleForFirst(stream, messageUser) {
			AppCtx.DataStore.UpdateFirstUser(stream.ID, messageUser.ID)
			AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf("Congratulations %s! You're first!", message.User.DisplayName))
		}

		fmt.Printf("[%s]%s(ID#%s): %s\n", message.Channel, message.User.DisplayName, message.User.ID, message.Message)
	})

	// Join all channels that have an api key
	registeredUsers, err := AppCtx.DataStore.FindAllApiKeyUsers()
	if err != nil {
		log.Fatal("Failed to fetch registered users", err)
	}

	for _, user := range registeredUsers {
		AppCtx.ClientIRC.Join(user.Username)
	}

	if err := AppCtx.ClientIRC.Connect(); err != nil {
		panic(err)
	}
}

func isFirstEnabled(streamUser *db.StreamUser) bool {
	return streamUser != nil && streamUser.FirstEnabled
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

// isEligibleForFirst
// TODO Expand this to a list of users
// TODO Don't just ignore people with "bot" in their name
func isEligibleForFirst(stream *db.Stream, msgUser *db.User) bool {
	return !strings.Contains(strings.ToLower(msgUser.Username), "bot") &&
		!strings.EqualFold(msgUser.Username, "PokemonCommunityGame") &&
		stream.UserId != msgUser.ID
}

// isSouLxBurN
func isSouLxBurN(username string) bool {
	return strings.ToLower(username) == "soulxburn"
}

func questionOfTheDay(stream *db.Stream, message *twitchirc.PrivateMessage) {
	var question *db.Question
	if stream == nil {
		return
	}
	if stream.QOTDId != nil {
		question, _ = AppCtx.DataStore.FindQuestionByID(*stream.QOTDId)
	} else {
		question, _ = AppCtx.DataStore.FindRandomQuestion(stream.UserId)
		AppCtx.DataStore.UpdateStreamQuestion(stream.ID, &question.ID)
	}
	AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf("%s", question.Text))
}

func thanos(message *twitchirc.PrivateMessage) error {
	if !isSouLxBurN(message.User.DisplayName) && !isSouLxBurN(message.Channel) {
		return errors.New("You are not Thanos")
	}
	users, err := AppCtx.ClientIRC.Userlist(message.Channel)
	if err != nil {
		log.Println("Thanos was unable to fetch the user list")
		return err
	}
	theChosen := []string{}
	for i, usr := range users {
		if i%2 == 0 {
			theChosen = append(theChosen, usr)
		}
	}

	for _, usr := range theChosen {
		AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf("/timeout %s 60 *SNAP*", usr))
		time.Sleep(time.Millisecond * 500)
	}
	return nil
}
