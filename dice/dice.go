package dice

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

	twitchirc "github.com/gempir/go-twitch-irc/v2"
	"github.com/soulxburn/soulxbot/db"
	"github.com/soulxburn/soulxbot/twitch"
)

const (
	ROLL_COOLDOWN_RESET = 180 * time.Second
	ROLL_COOLDOWN_START = 10 * time.Second
	COUNTDOWN_TIMER     = 121 * time.Second
)

type Dice struct {
	sides int
}

type DiceGame struct {
	Dice         []*Dice
	CanRoll      bool
	rollCooldown *time.Timer
	ircClient    *twitchirc.Client
	twitchAPI    twitch.ITwitchAPI
}

// NewDice
func NewDice(sides int) *Dice {
	return &Dice{
		sides: sides,
	}
}

// Allow for many dice
func NewDiceSlice(count int, sides int) []*Dice {
	// Seeds once, rather than per-dice
	dice := make([]*Dice, 0, count)

	for i := 0; i < count; i++ {
		dice = append(dice, NewDice(sides))
	}

	return dice
}

func (dg *DiceGame) RollAll(channel string) int {
	var result int
	for i, dice := range dg.Dice {
		roll := dice.Roll()
		dg.ircClient.Say(channel, fmt.Sprintf("Dice #%d: %d", i+1, roll))
		result += roll
	}
	return result
}

// Roll
func (d *Dice) Roll() int {
	roll := (rand.Int() % d.sides) + 1
	return roll
}

// NewDiceGame
func NewDiceGame(ircClient *twitchirc.Client, twitchAPI twitch.ITwitchAPI) *DiceGame {
	diceGame := &DiceGame{
		Dice:         NewDiceSlice(2, 6),
		CanRoll:      true,
		rollCooldown: time.NewTimer(ROLL_COOLDOWN_START),
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
func (dg *DiceGame) StartRoll(user db.StreamUser, channel string) error {
	if !dg.CanRoll {
		return errors.New("Roll is already in progress")
	}

	dg.rollCooldown.Reset(ROLL_COOLDOWN_RESET)
	dg.CanRoll = false
	log.Println("Executing startroll")
	// Check if prediction is in-flight.
	// Start prediction
	prediction, err := dg.twitchAPI.CreatePrediction(user, "Dice Roll Prediction!", 120, []string{"Even", "Odd"})
	if err != nil {
		return err
	}
	// Start a countdown for rolling dice
	go func() {
		// Wait for timer
		time.Sleep(COUNTDOWN_TIMER)
		// roll dice
		total := dg.RollAll(channel)
		dg.ircClient.Say(channel, fmt.Sprintf("Total: %d", total))

		// determine winner
		winner := total % 2
		if winner == 0 {
			// Even wins
			dg.endPrediction(user, prediction, "Even")
		} else {
			// Odd wins
			dg.endPrediction(user, prediction, "Odd")
		}
	}()

	return nil
}

func (dg *DiceGame) endPrediction(user db.StreamUser, prediction *twitch.TwitchPrediction, title string) {
	for _, v := range prediction.Outcomes {
		if v.Title == title {
			dg.twitchAPI.EndPrediction(user, prediction, v.ID)
		}
	}
}
