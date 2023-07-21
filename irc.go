package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	twitchirc "github.com/gempir/go-twitch-irc/v2"
	"github.com/google/uuid"
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

	// Restart any streams that were live when the bot was last shut down
	streamsInProgress := AppCtx.DataStore.FindAllCurrentStreams()
	for _, stream := range streamsInProgress {
		user, ok := AppCtx.DataStore.FindUserByID(stream.UserId)
		if ok {
			go pollStreamStatus(&stream, user)
		}
	}
	http.HandleFunc("/golive", func(res http.ResponseWriter, req *http.Request) {
		params := req.URL.Query()
		apiKey := params.Get("key")

		user, ok := AppCtx.DataStore.FindUserByApiKey(apiKey)
		stream := AppCtx.DataStore.FindCurrentStream(user.ID)

		if ok && stream == nil {
			log.Printf("%s is now live!", user.DisplayName)
			stream = AppCtx.DataStore.InsertStream(user.ID, time.Now())
			res.WriteHeader(http.StatusAccepted)
			go pollStreamStatus(stream, user)
		} else {
			log.Println("Go live not authorized")
			res.WriteHeader(http.StatusUnauthorized)
		}
	})

	http.HandleFunc("/register", func(res http.ResponseWriter, req *http.Request) {
		authHeader := req.Header.Get("Authorization")
		split := strings.Split(authHeader, " ")
		if split[0] != "Basic" || len(split) != 2 {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		decoded, err := base64.StdEncoding.DecodeString(split[1])
		if err != nil {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		if basicAuth != string(decoded) {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		params := req.URL.Query()
		username := params.Get("username")
		user, ok := AppCtx.DataStore.FindUserByUsername(username)
		if !ok {
			res.WriteHeader(http.StatusNotFound)
			res.Write([]byte("User not found, have user chat in channel first"))
			return
		}

		guid := uuid.New().String()
		AppCtx.DataStore.UpdateAPIKeyForUser(user.ID, guid)

		res.WriteHeader(http.StatusOK)
		res.Write([]byte(guid))
		return
	})
	go http.ListenAndServe(":8080", nil)

	AppCtx.ClientIRC.OnUserNoticeMessage(func(message twitchirc.UserNoticeMessage) {
		fmt.Printf("Notice: %s\n", message.Message)
	})

	AppCtx.ClientIRC.OnPrivateMessage(func(message twitchirc.PrivateMessage) {
		messageUser := handleMessageUser(&message)
		streamUser, ok := AppCtx.DataStore.FindUserByUsername(strings.ToLower(message.Channel))
		if !ok {
			log.Printf("Unable to find stream user for twitch channel %s", message.Channel)
		}
		stream := AppCtx.DataStore.FindCurrentStream(streamUser.ID)

		if isCommand(message.Message) {
			command, input := parseCommand(message.Message)

			switch command {
			case "first":
				if stream != nil && stream.FirstUserId != nil {
					if *stream.FirstUserId != messageUser.ID {
						firstUser, _ := AppCtx.DataStore.FindUserByID(*stream.FirstUserId)
						AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf("Sorry %s, you are not first. %s was!", message.User.DisplayName, firstUser.DisplayName))
					} else {
						AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf("/timeout %[1]s 60 Yes %[1]s! We KNOW. You were first...", message.User.DisplayName))
					}
				}
			case "firstcount":
				timesFirst, err := AppCtx.DataStore.FindUserTimesFirst(stream.UserId, messageUser.ID)
				if err != nil {
					log.Println(err)
					return
				}
				AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf("%s, you have been first %d times", messageUser.DisplayName, timesFirst))
			case "firstleaders":
				leaders, _ := AppCtx.DataStore.FindFirstLeaders(stream.UserId, 3)
				for i, v := range leaders {
					AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf("%d. %s - %d", i+1, v.User.DisplayName, v.TimesFirst))
				}
			case "firstgive":
				if stream != nil && stream.UserId == messageUser.ID && len(input) > 0 {
					targetUser, found := AppCtx.DataStore.FindUserByUsername(input)
					if found {
						AppCtx.DataStore.UpdateFirstUser(stream.ID, targetUser.ID)
						AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf("%s has been set as first for this stream!", targetUser.Username))
					} else {
						AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf("That user does not exist"))
					}
				}
			case "qotd":
				questionOfTheDay(stream, &message)
			case "skipqotd":
				AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf("Question of the day skipped"))
				AppCtx.DataStore.UpdateStreamQuestion(stream.ID, nil)
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
			case "raid":
				var buff strings.Builder
				for i := 0; i < 9; i++ {
					buff.WriteString("%[1]s %[2]s %[3]s ")
				}
				AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf(buff.String(), "PowerUpL", "soulxbGASMShake", "PowerUpR"))
			case "thanos":
				thanos(&message)
			}
		}

		if stream != nil && stream.FirstUserId == nil && isEligibleForFirst(stream, messageUser) {
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

func handleMessageUser(message *twitchirc.PrivateMessage) *db.User {
	var user *db.User
	messageUserId, err := strconv.Atoi(message.User.ID)
	if err != nil {
		log.Printf("Failed to convert twitch user id=%s to integer", message.User.ID)
		return nil
	}
	user, ok := AppCtx.DataStore.FindUserByUsername(message.User.Name)
	if !ok {
		user = AppCtx.DataStore.InsertUser(messageUserId, message.User.Name, message.User.DisplayName)
		log.Printf("New User Found!: %d, %s, %s", user.ID, user.Username, user.DisplayName)
	}
	return user
}

// pollStreamStatus
func pollStreamStatus(stream *db.Stream, streamUser *db.User) {
	tick := time.NewTicker(5 * time.Minute)
	for {
		select {
		case <-tick.C:
			streamInfo, err := AppCtx.TwitchAPI.GetStream(streamUser.Username)
			if err != nil {
				log.Println("Error fetching stream info: ", err)
				continue
			}

			if stream.TWID == nil || stream.Title == nil {
				twid, err := strconv.Atoi(streamInfo.ID)
				if err == nil {
					AppCtx.DataStore.UpdateStreamInfo(stream.ID, twid, streamInfo.Title)
					stream = AppCtx.DataStore.FindStreamById(stream.ID)
				}
			}

			if streamInfo == nil {
				AppCtx.DataStore.UpdateStreamEndedAt(stream.ID, time.Now())
				tick.Stop()
				return
			}
		}
	}
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
	if !isSouLxBurN(message.User.DisplayName) {
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
