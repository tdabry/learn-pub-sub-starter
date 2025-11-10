package main

import (
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/tdabry/learn-pub-sub-starter/internal/gamelogic"
	"github.com/tdabry/learn-pub-sub-starter/internal/pubsub"
	"github.com/tdabry/learn-pub-sub-starter/internal/routing"
	"log"
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
		routing.PauseKey, "something went wrong with queue binding1")
	ch := bindQueue(rabbit, username, routing.ExchangePerilTopic,
		routing.ArmyMovesPrefix, "something went wrong with queue binding2")

	gameState := gamelogic.NewGameState(username)
	subscribe(rabbit, gameState, username, routing.ExchangePerilTopic,
		routing.PauseKey, "error subscribing to queue1", handlerPause)

	subscribe(rabbit, gameState, username, routing.ExchangePerilTopic,
		routing.ArmyMovesPrefix, "error subscribing to queue2", handlerMove)

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
			log.Println("spam command not available yet")
		} else if word == "quit" {
			log.Println("Exiting...")
			break
		} else {
			log.Printf("Unknown command: <%s>", word)
		}
	}
}

func subscribe[T any](rabbit *amqp.Connection, gameState *gamelogic.GameState,
	username, exchange, key, msg string, handler func(*gamelogic.GameState) func(T) pubsub.Acktype) {

	err := pubsub.SubscribeJSON(rabbit, exchange, key+"."+username,
		key, pubsub.Transient, handler(gameState))
	if err != nil {
		log.Fatal(msg)
	}
}

func bindQueue(rabbit *amqp.Connection, username, exchange, key, msg string) *amqp.Channel {
	route := key
	if key == "army_moves" {
		route = route + ".*"
	}

	ch, queue, err := pubsub.DeclareAndBind(rabbit, exchange,
		key+"."+username, route, pubsub.Transient)
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

func handlerMove(gs *gamelogic.GameState) func(gamelogic.ArmyMove) pubsub.Acktype {
	return func(mv gamelogic.ArmyMove) pubsub.Acktype {
		defer fmt.Print("> ")
		moveOut := gs.HandleMove(mv)
		switch(moveOut) {
		case gamelogic.MoveOutComeSafe:
			fallthrough
		case gamelogic.MoveOutcomeMakeWar:
			return pubsub.Ack
		}
		return pubsub.NackDiscard
	}
}

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.Acktype {
	return func(ps routing.PlayingState) pubsub.Acktype {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
		return pubsub.Ack
	}
}
