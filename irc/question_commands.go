package irc

import (
	"fmt"

	twitchirc "github.com/gempir/go-twitch-irc/v2"
	"github.com/soulxburn/soulxbot/db"
)

type CommandHandler = func(MessageContext, string, string)

type Command struct {
	CmdString string
	Cmd       CommandHandler
}

type MessageContext struct {
	Channel     string
	MessageUser *db.User
	StreamUser  *db.StreamUser
	Stream      *db.Stream
}

type QuestionCommands struct {
	DataStore *db.Database
	ClientIRC *twitchirc.Client
}

func (q *QuestionCommands) GetCommands() []Command {
	commands := []Command{
		{"qotd", q.qotd},
		{"skipqotd", q.skipqotd},
	}
	return commands
}

func (q *QuestionCommands) qotd(msgCtx MessageContext, command string, input string) {
	if msgCtx.Stream != nil && msgCtx.StreamUser != nil && msgCtx.StreamUser.QotdEnabled {
		question := questionOfTheDay(q.DataStore, msgCtx.Stream)
		q.ClientIRC.Say(msgCtx.Channel, fmt.Sprintf("%s", question.Text))
	}
}

func (q *QuestionCommands) skipqotd(msgCtx MessageContext, command string, input string) {
	if msgCtx.Stream != nil &&
		msgCtx.Stream.UserId == msgCtx.MessageUser.ID &&
		msgCtx.Stream.QOTDId != nil &&
		msgCtx.StreamUser != nil &&
		msgCtx.StreamUser.QotdEnabled {

		if skipCount, _ := q.DataStore.IncrementQuestionSkip(*msgCtx.Stream.QOTDId); skipCount > 2 {
			q.DataStore.DisableQuestion(*msgCtx.Stream.QOTDId)
		}

		q.DataStore.UpdateStreamQuestion(msgCtx.Stream.ID, nil)
		q.ClientIRC.Say(msgCtx.Channel, fmt.Sprintf("Question of the day skipped, enter !qotd to get a new question"))
	}
}

func questionOfTheDay(dataStore *db.Database, stream *db.Stream) *db.Question {
	var question *db.Question
	if stream == nil {
		return nil
	}
	if stream.QOTDId != nil {
		question, _ = dataStore.FindQuestionByID(*stream.QOTDId)
	} else {
		question, _ = dataStore.FindRandomQuestion(stream.UserId)
		dataStore.UpdateStreamQuestion(stream.ID, &question.ID)
	}
	return question
}
