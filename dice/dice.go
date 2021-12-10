package dice

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	twitchirc "github.com/gempir/go-twitch-irc/v2"
	"github.com/soulxburn/soulxbot/twitch"
)

type Dice struct {
	sides int
}

// NewDice
func NewDice(sides int) *Dice {
	rand.Seed(time.Now().UnixNano())
	return &Dice{
		sides: sides,
	}
}

// Roll
func (d *Dice) Roll() int {
	roll := (rand.Int() % d.sides) + 1
	return roll
}

type DiceGame struct {
	Dice         []*Dice
	CanRoll      bool
	rollCooldown *time.Timer
	ircClient    *twitchirc.Client
	twitchAPI    twitch.ITwitchAPI
}

// NewDiceGame
func NewDiceGame(ircClient *twitchirc.Client, twitchAPI twitch.ITwitchAPI) *DiceGame {
	diceGame := &DiceGame{
		Dice:         []*Dice{NewDice(6), NewDice(6)},
		CanRoll:      true,
		rollCooldown: time.NewTimer(10 * time.Second),
		ircClient:    ircClient,
		twitchAPI:    twitchAPI,
	}
	go diceGame.startCooldownTimer()
	return diceGame
}

func (dg *DiceGame) startCooldownTimer() {
	for {
		<-dg.rollCooldown.C
		dg.CanRoll = true
	}
}

// StartRoll
func (dg *DiceGame) StartRoll(channel string) error {
	if dg.CanRoll {
		dg.rollCooldown.Reset(180 * time.Second)
		dg.CanRoll = false
		fmt.Println("Executing startroll")
		// Check if prediction is in-flight.
		// Start prediction
		prediction, _ := dg.twitchAPI.CreatePrediction("Dice Roll Prediction!", 120, []string{"Even", "Odd"})
		// Start a countdown for rolling dice
		go func() {
			// Wait for timer
			time.Sleep(121 * time.Second)
			// roll dice
			roll1 := dg.Dice[0].Roll()
			roll2 := dg.Dice[1].Roll()
			dg.ircClient.Say(channel, fmt.Sprintf("Dice #1: %d", roll1))
			dg.ircClient.Say(channel, fmt.Sprintf("Dice #2: %d", roll2))
			dg.ircClient.Say(channel, fmt.Sprintf("Total: %d", roll1+roll2))

			// determine winner
			winner := (roll1 + roll2) % 2
			if winner == 0 {
				// Even wins
				for _, v := range prediction.Outcomes {
					if v.Title == "Even" {
						dg.twitchAPI.EndPrediction(prediction, v.ID)
					}
				}
			} else {
				// Odd wins
				for _, v := range prediction.Outcomes {
					if v.Title == "Odd" {
						dg.twitchAPI.EndPrediction(prediction, v.ID)
					}
				}
			}
		}()
	} else {
		return errors.New("Roll is already in progress")
	}

	return nil
}
