package irc

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	twitchirc "github.com/gempir/go-twitch-irc/v2"
	"github.com/soulxburn/soulxbot/db"
	"github.com/soulxburn/soulxbot/twitch"
)

type ThanosCommand struct {
	DataStore *db.Database
	ClientIRC *twitchirc.Client
	TwitchAPI twitch.ITwitchAPI
}

func (t *ThanosCommand) GetCommands() []Command {
	commands := []Command{
		{"thanos", t.thanos},
	}
	return commands
}

func (t *ThanosCommand) thanos(msgCtx MessageContext, command string, input string) {
	if !IsSouLxBurN(msgCtx.MessageUser.DisplayName) && !IsSouLxBurN(msgCtx.Channel) {
		t.ClientIRC.Say(msgCtx.Channel, "You are not Thanos")
		return
	}
	usernames, err := t.ClientIRC.Userlist(msgCtx.Channel)
	if err != nil {
		log.Println("Thanos was unable to fetch the user list", err)
		return
	}

	for len(usernames) > 0 {
		chunksize := len(usernames)
		if chunksize > 100 {
			chunksize = 100
		}
		userChunk := usernames[:chunksize]
		usernames = usernames[chunksize:]

		userInfoList, err := t.TwitchAPI.GetUsers(userChunk)
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
			// AppCtx.TwitchAPI.TimeoutUser(fmt.Sprint(streamUser.User.ID), usr.Id, 60, "*SNAP*")
			fmt.Printf("[%s]Timed Out: %s\n", msgCtx.Channel, usr.DisplayName)
			time.Sleep(time.Millisecond * 200)
		}
	}
}

// isSouLxBurN
func IsSouLxBurN(username string) bool {
	return strings.ToLower(username) == "soulxburn"
}
