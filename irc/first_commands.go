package irc

import (
	"fmt"
	"log"

	twitchirc "github.com/gempir/go-twitch-irc/v2"
	"github.com/soulxburn/soulxbot/db"
)

type FirstCommands struct {
	DataStore *db.Database
	ClientIRC *twitchirc.Client
}

func (q *FirstCommands) GetCommands() []Command {
	commands := []Command{
		{"first", q.first},
		{"firstcount", q.firstcount},
		{"firstcount-all", q.firstcount},
		{"firstleaders", q.firstleaders},
		{"firstleaders-all", q.firstleaders},
		{"firstleaders-reset", q.firstleadersReset},
		{"firstgive", q.firstgive},
		{"firstexclude", q.firstexclude},
	}
	return commands
}

func (q *FirstCommands) first(msgCtx MessageContext, command string, input string) {
	if !IsFirstEnabled(msgCtx.StreamUser) {
		return
	}
	if msgCtx.Stream != nil && msgCtx.Stream.FirstUserId != nil && msgCtx.StreamUser != nil && msgCtx.StreamUser.FirstEnabled {
		if *msgCtx.Stream.FirstUserId != msgCtx.MessageUser.ID {
			firstUser, _ := q.DataStore.FindUserByID(*msgCtx.Stream.FirstUserId)
			q.ClientIRC.Say(msgCtx.Channel, fmt.Sprintf("Sorry %s, you are not first. %s was!", msgCtx.MessageUser.DisplayName, firstUser.DisplayName))
		} else {
			q.ClientIRC.Say(msgCtx.Channel, fmt.Sprintf("/timeout %[1]s 60 Yes %[1]s! We KNOW. You were first...", msgCtx.MessageUser.DisplayName))
		}
	}
}

func (q *FirstCommands) firstcount(msgCtx MessageContext, command string, input string) {
	if !IsFirstEnabled(msgCtx.StreamUser) {
		return
	}
	timesFirst, err := q.DataStore.FindUserTimesFirst(msgCtx.StreamUser.User.ID, msgCtx.MessageUser.ID, command == "firstcount-all")
	if err != nil {
		log.Println(err)
		return
	}
	q.ClientIRC.Say(msgCtx.Channel, fmt.Sprintf("%s, you have been first %d times", msgCtx.MessageUser.DisplayName, timesFirst))
}

func (q *FirstCommands) firstleaders(msgCtx MessageContext, command string, input string) {
	if !IsFirstEnabled(msgCtx.StreamUser) {
		return
	}
	leaders, _ := q.DataStore.FindFirstLeaders(msgCtx.StreamUser.User.ID, 3, command == "firstleaders-all")
	for i, v := range leaders {
		q.ClientIRC.Say(msgCtx.Channel, fmt.Sprintf("%d. %s - %d", i+1, v.User.DisplayName, v.TimesFirst))
	}
}

func (q *FirstCommands) firstleadersReset(msgCtx MessageContext, command string, input string) {
	if msgCtx.StreamUser.User.ID == msgCtx.MessageUser.ID {
		q.DataStore.ResetFirstEpoch(msgCtx.Stream.UserId)
		q.ClientIRC.Say(msgCtx.Channel, "First leaders reset")
	}
}

func (q *FirstCommands) firstgive(msgCtx MessageContext, command string, input string) {
	if msgCtx.Stream != nil && msgCtx.Stream.UserId == msgCtx.MessageUser.ID && len(input) > 0 && IsFirstEnabled(msgCtx.StreamUser) {
		targetUser, found := q.DataStore.FindUserByUsername(input)
		if found {
			q.DataStore.UpdateFirstUser(msgCtx.Stream.ID, targetUser.ID)
			q.ClientIRC.Say(msgCtx.Channel, fmt.Sprintf("%s has been set as first for this stream!", targetUser.Username))
		} else {
			q.ClientIRC.Say(msgCtx.Channel, fmt.Sprintf("That user does not exist"))
		}
	}
}

func (q *FirstCommands) firstexclude(msgCtx MessageContext, command string, input string) {
	if msgCtx.StreamUser.UserId == msgCtx.MessageUser.ID && len(input) > 0 {
		q.DataStore.InsertExcludedUser(&msgCtx.StreamUser.UserId, input)
		q.ClientIRC.Say(msgCtx.Channel, fmt.Sprintf("%s has been excluded from first", input))
	}
}

func (q *FirstCommands) OnMessage(msgCtx MessageContext) {
	if msgCtx.Stream != nil &&
		msgCtx.Stream.FirstUserId == nil &&
		IsFirstEnabled(msgCtx.StreamUser) &&
		q.IsEligibleForFirst(msgCtx.Stream, msgCtx.MessageUser) {

		q.DataStore.UpdateFirstUser(msgCtx.Stream.ID, msgCtx.MessageUser.ID)
		q.ClientIRC.Say(msgCtx.Channel, fmt.Sprintf("Congratulations %s! You're first!", msgCtx.MessageUser.DisplayName))
	}
}

// IsEligibleForFirst
func (q *FirstCommands) IsEligibleForFirst(stream *db.Stream, msgUser *db.User) bool {
	return !q.DataStore.IsUserOnExclusionList(&stream.UserId, msgUser.Username) &&
		stream.UserId != msgUser.ID
}

func IsFirstEnabled(streamUser *db.StreamUser) bool {
	return streamUser != nil && streamUser.FirstEnabled
}
