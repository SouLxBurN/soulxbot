package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
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
	StreamInfo  *db.Stream
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
	goliveKey := os.Getenv("SOULXBOT_GOLIVEKEY")

	AppCtx.StreamState = NOT_LIVE
	AppCtx.DataStore = db.InitDatabase()
	AppCtx.TwitchAPI = twitch.NewTwitchAPI(clientID, clientSecret, authToken, refreshToken)
	AppCtx.ClientIRC = twitchirc.NewClient(user, oauth)
	AppCtx.DiceGame = dice.NewDiceGame(AppCtx.ClientIRC, AppCtx.TwitchAPI)

	http.HandleFunc("/golive", func(res http.ResponseWriter, req *http.Request) {
		params := req.URL.Query()
		api_key := params.Get("key")

		if api_key == goliveKey && AppCtx.StreamState == NOT_LIVE {
			fmt.Println("SouLxBurN is now live!")
			AppCtx.StreamState = LIVE_NO_FIRST
			res.WriteHeader(http.StatusAccepted)
		} else {
			fmt.Println("Go live not authorized")
			res.WriteHeader(http.StatusUnauthorized)
		}
		go pollStreamStatus()
	})
	go http.ListenAndServe(":8080", nil)

	AppCtx.ClientIRC.OnUserNoticeMessage(func(message twitchirc.UserNoticeMessage) {
		fmt.Printf("Notice: %s\n", message.Message)
	})

	AppCtx.ClientIRC.OnPrivateMessage(func(message twitchirc.PrivateMessage) {
		user, ok := AppCtx.DataStore.FindUserByUsername(message.User.DisplayName)
		if !ok {
			user = AppCtx.DataStore.InsertUser(message.User.DisplayName)
		}

		if isCommand(message.Message) {
			command, input := parseCommand(message.Message)

			switch command {
			case "first":
				if AppCtx.StreamState == LIVE_W_FIRST {
					if AppCtx.StreamInfo.FirstUser.Username != user.Username {
						AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf("Sorry %s, you are not first. %s was!", message.User.DisplayName, AppCtx.StreamInfo.FirstUser.Username))
					} else {
						AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf("/timeout %[1]s 60 Yes %[1]s! We KNOW. You were first...", message.User.DisplayName))
					}
				}
			case "firstcount":
				AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf("%s, you have been first %d times", user.Username, user.TimesFirst))
			case "firstleaders":
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
			case "firstgive":
				if isSouLxBurN(message.User.DisplayName) && len(input) > 0 {
					targetUser, found := AppCtx.DataStore.FindUserByUsername(input)
					if found {
						AppCtx.DataStore.IncrementTimesFirst(targetUser.ID)
						AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf("%s has gained a first point!", targetUser.Username))
					} else {
						AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf("That user does not exist"))
					}
				}
			case "firsttake":
				if isSouLxBurN(message.User.DisplayName) && len(input) > 0 {
					targetUser, found := AppCtx.DataStore.FindUserByUsername(input)
					if found {
						AppCtx.DataStore.DecrementTimesFirst(targetUser.ID)
						AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf("%s has lost a first point BibleThump", targetUser.Username))
					} else {
						AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf("That user does not exist"))
					}
				}
			case "raid":
				var buff strings.Builder
				for i := 0; i < 9; i++ {
					buff.WriteString("%[1]s %[2]s %[3]s ")
				}
				AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf(buff.String(), "PowerUpL", "soulxbGASMShake", "PowerUpR"))
			case "thanos":
				if isSouLxBurN(message.User.DisplayName) {
					users, err := AppCtx.ClientIRC.Userlist(message.Channel)
					if err != nil {
						log.Println("Unable to fetch user list")
					} else {
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
					}
				}
			}
		}

		if AppCtx.StreamState == LIVE_NO_FIRST && isEligibleForFirst(message.User.DisplayName) {
			AppCtx.StreamState = LIVE_W_FIRST
			AppCtx.DataStore.IncrementTimesFirst(user.ID)
			AppCtx.DataStore.UpdateFirstUser(0, user.ID)
			AppCtx.StreamInfo = &db.Stream{
				FirstUser: user,
			}
			AppCtx.ClientIRC.Say(message.Channel, fmt.Sprintf("Congratulations %s! You're first!", message.User.DisplayName))
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
	tick := time.NewTicker(5 * time.Minute)
	for {
		select {
		case <-tick.C:
			streamInfo, err := AppCtx.TwitchAPI.GetStream("SouLxBurN")
			if err != nil {
				log.Println("Error fetching stream info: ", err)
				continue
			}

			if streamInfo == nil {
				AppCtx.StreamState = NOT_LIVE
				AppCtx.StreamInfo = nil
				tick.Stop()
				return
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

// parseCommand
func parseCommand(message string) (string, string) {
	split := strings.Split(message[1:], " ")
	if len(split) >= 2 {
		return split[0], split[1]
	}
	return split[0], ""
}

// isEligibleForFirst
func isEligibleForFirst(username string) bool {
	return !strings.Contains(strings.ToLower(username), "bot") &&
		!strings.EqualFold(username, "PokemonCommunityGame") &&
		!isSouLxBurN(username)
}

// isSouLxBurN
func isSouLxBurN(username string) bool {
	return strings.ToLower(username) == "soulxburn"
}
