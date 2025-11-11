package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/tdabry/learn-pub-sub-starter/internal/gamelogic"
	"github.com/tdabry/learn-pub-sub-starter/internal/pubsub"
	"github.com/tdabry/learn-pub-sub-starter/internal/routing"
)

func main() {
	fmt.Println("Starting Peril client...")
	conn := "amqp://guest:guest@localhost:5672/"
	rabbit, err := amqp.Dial(conn)
	if err != nil {
		fmt.Println("Failed to connect")
		return
	}
	defer rabbit.Close()

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Print("error getting username")
		return
	}

	bindQueue(rabbit, username, routing.ExchangePerilTopic,
		routing.PauseKey, "something went wrong with queue binding1", pubsub.Transient)
	ch := bindQueue(rabbit, username, routing.ExchangePerilTopic,
		routing.ArmyMovesPrefix, "something went wrong with queue binding2", pubsub.Transient)
	bindQueue(rabbit, "", routing.ExchangePerilTopic,
		routing.WarRecognitionsPrefix, "something went wrong with queue binding3", pubsub.Durable)

	gameState := gamelogic.NewGameState(username)
	subscribe(rabbit, gameState, username, routing.ExchangePerilTopic,
		routing.PauseKey, "error subscribing to queue1", pubsub.Transient, handlerPause)

	subscribe(rabbit, gameState, username, routing.ExchangePerilTopic,
		routing.ArmyMovesPrefix, "error subscribing to queue2", pubsub.Transient, handlerMove)

	subscribe(rabbit, gameState, username, routing.ExchangePerilTopic,
		routing.WarRecognitionsPrefix, "error subscribing to queue3", pubsub.Durable, handlerWar)

	gamelogic.PrintClientHelp()
	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}
		word := words[0]
		if word == "spawn" {
			err = gameState.CommandSpawn(words)
			if skip := hasErr(err); skip {
				continue
			}
		} else if word == "move" {
			mv, err := gameState.CommandMove(words)
			if skip := hasErr(err); skip {
				continue
			}
			err = pubsub.PublishJSON(ch, routing.ExchangePerilTopic, "army_moves",
				mv)
			if err != nil {
				continue
			}
			log.Printf("move published")
		} else if word == "status" {
			gameState.CommandStatus()
		} else if word == "help" {
			gamelogic.PrintClientHelp()
		} else if word == "spam" {
			n, err := getSpamCount(words[1:])
			if err != nil {
				log.Print(err)
				continue
			}
			spamLog(ch, n, username)
		} else if word == "quit" {
			log.Println("Exiting...")
			break
		} else {
			log.Printf("Unknown command: <%s>", word)
		}
	}
}

func subscribe[T any](rabbit *amqp.Connection, gameState *gamelogic.GameState,
	username, exchange, key, msg string, qType pubsub.SimpleQueueType,
	handler func(*gamelogic.GameState, *amqp.Connection) func(T) pubsub.Acktype) {

	qName := key + "." + username
	route := key
	if key == "war" {
		route = key + "."
		qName = key
	}
	err := pubsub.SubscribeJSON(rabbit, exchange, qName,
		route, qType, handler(gameState, rabbit))
	if err != nil {
		log.Fatal(msg)
	}
}

func bindQueue(rabbit *amqp.Connection, username, exchange, key, msg string,
	qType pubsub.SimpleQueueType) *amqp.Channel {

	route := key
	qName := key + "." + username
	switch key {
	case "army_moves":
		route = route + ".*"
	case "war":
		qName = key
		route = route + ".*"
	}
	ch, queue, err := pubsub.DeclareAndBind(rabbit, exchange,
		qName, route, qType)
	if err != nil {
		log.Fatal(msg)
	}
	fmt.Printf("Queue %v declared and bound!\n", queue.Name)
	return ch
}

func hasErr(err error) bool {
	if err != nil {
		log.Printf("%s", err)
		return true
	}
	return false
}

func handlerMove(gs *gamelogic.GameState, rabbit *amqp.Connection) func(gamelogic.ArmyMove) pubsub.Acktype {
	return func(mv gamelogic.ArmyMove) pubsub.Acktype {
		defer fmt.Print("> ")
		moveOut := gs.HandleMove(mv)
		switch moveOut {
		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack
		case gamelogic.MoveOutcomeMakeWar:
			routingKey := fmt.Sprintf("%s.%s", routing.WarRecognitionsPrefix, mv.Player.Username)
			ch, err := rabbit.Channel()
			if err != nil {
				return pubsub.NackRequeue
			}
			defer ch.Close()
			if err := pubsub.PublishJSON(ch, routing.ExchangePerilTopic, routingKey, gamelogic.RecognitionOfWar{Attacker: mv.Player, Defender: gs.Player}); err != nil {
				log.Printf("handlerMove: publish error: %v", err)
				return pubsub.NackRequeue
			}

			log.Printf("handlerMove: war recognition published to %s", routingKey)
			return pubsub.Ack
		}

		return pubsub.NackDiscard
	}
}

func handlerWar(gs *gamelogic.GameState, conn *amqp.Connection) func(gamelogic.RecognitionOfWar) pubsub.Acktype {
	return func(war gamelogic.RecognitionOfWar) pubsub.Acktype {
		defer fmt.Print("> ")
		outcome, winner, loser := gs.HandleWar(war)
		logMsg := ""
		ch, err := conn.Channel()
		if err != nil {return pubsub.NackRequeue}
		defer ch.Close()
		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard
		case gamelogic.WarOutcomeOpponentWon:
			fallthrough
		case gamelogic.WarOutcomeYouWon:
			logMsg = fmt.Sprintf("%s won a war against %s", winner, loser)
			err := publishGameLog(ch, gs.GetUsername(), logMsg)
			if err != nil {return pubsub.NackRequeue}
			return pubsub.Ack
		case gamelogic.WarOutcomeDraw:
			logMsg = fmt.Sprintf("A war between %s and %s resulted in a draw", winner, loser)
			err := publishGameLog(ch, gs.GetUsername(), logMsg)
			if err != nil {return pubsub.NackRequeue}
			return pubsub.Ack
		}
		log.Print("Unknown war outcome")
		return pubsub.NackDiscard
	}
}

func handlerPause(gs *gamelogic.GameState, rabbit *amqp.Connection) func(routing.PlayingState) pubsub.Acktype {
	return func(ps routing.PlayingState) pubsub.Acktype {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
		return pubsub.Ack
	}
}
func publishGameLog(ch *amqp.Channel, username, msg string) error {
	exchange := routing.ExchangePerilTopic
	route := routing.GameLogSlug + "." + username
	logStruct := routing.GameLog{CurrentTime: time.Now(),
		Message: msg, Username: username}
	return pubsub.PublishGob(ch, exchange, route, logStruct)
}

func getSpamCount(words []string) (int64, error) {
	if len(words) < 1 {
		return 0, fmt.Errorf("usage: spam <number>")
	}
	return strconv.ParseInt(words[0], 10, 32)
} 

func spamLog(ch *amqp.Channel, times int64, username string) {
	for range times{
		logMsg := gamelogic.GetMaliciousLog()
		err := publishGameLog(ch, username, logMsg)
		if err != nil {
			log.Print("error publishing spam msg")
		}
	}
}