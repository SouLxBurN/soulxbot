package api

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/soulxburn/soulxbot/db"
	"github.com/soulxburn/soulxbot/twitch"
)

type StreamPoller struct {
	db        *db.Database
	twitchAPI twitch.ITwitchAPI
}

func NewStreamPoller(db *db.Database, twitchAPI twitch.ITwitchAPI) StreamPoller {
	return StreamPoller{db, twitchAPI}
}

func (sp StreamPoller) goliveHandler(res http.ResponseWriter, req *http.Request) {
	params := req.URL.Query()
	apiKey := params.Get("key")

	user, ok := sp.db.FindUserByApiKey(apiKey)
	stream := sp.db.FindCurrentStream(user.ID)

	if ok && stream == nil {
		stream, err := sp.db.InsertStream(user.ID, time.Now())
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			res.Write([]byte("Go live request failed"))
			return
		}
		res.WriteHeader(http.StatusAccepted)
		go sp.PollStreamStatus(stream, user, false)
		log.Printf("%s is now live!", user.DisplayName)
	} else if ok && stream != nil {
		log.Printf("User %s, is already online", user.Username)
		res.WriteHeader(http.StatusConflict)
	} else {
		log.Println("Go live not authorized")
		res.WriteHeader(http.StatusUnauthorized)
	}
}

// PollStreamStatus
func (sp StreamPoller) PollStreamStatus(stream *db.Stream, streamUser *db.User, isRestart bool) {
	if isRestart {
		live, err := sp.isStreamLive(stream, streamUser)
		if err != nil {
			log.Println("Error fetching stream info on restart: ", streamUser.Username, err)
		} else if !live {
			return
		}
	}

	tick := time.NewTicker(5 * time.Minute)
	for {
		select {
		case <-tick.C:
			live, err := sp.isStreamLive(stream, streamUser)
			if err != nil {
				log.Println("Error fetching stream info: ", streamUser.Username, err)
				continue
			} else if !live {
				tick.Stop()
				return
			}
		}
	}
}

func (sp StreamPoller) isStreamLive(stream *db.Stream, streamUser *db.User) (bool, error) {
	streamInfo, err := sp.twitchAPI.GetStream(streamUser.Username)
	if err != nil {
		return false, err
	}

	if streamInfo == nil {
		sp.db.UpdateStreamEndedAt(stream.ID, time.Now())
		return false, nil
	}

	if stream.TWID == nil || stream.Title == nil {
		twid, err := strconv.Atoi(streamInfo.ID)
		if err == nil {
			sp.db.UpdateStreamInfo(stream.ID, twid, streamInfo.Title)
			stream = sp.db.FindStreamById(stream.ID)
		}
	}
	return true, nil
}

func (sp StreamPoller) RestartStreamStatusPolls() {
	// Restart any streams that were live when the bot was last shut down
	streamsInProgress := sp.db.FindAllCurrentStreams()
	for _, stream := range streamsInProgress {
		user, ok := sp.db.FindUserByID(stream.UserId)
		if ok {
			go sp.PollStreamStatus(stream, user, true)
		}
	}
}
