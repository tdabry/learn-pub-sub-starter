package main

import (
	"log"
	amqp "github.com/rabbitmq/amqp091-go"
	"fmt"
	"github.com/tdabry/learn-pub-sub-starter/internal/pubsub"
	"github.com/tdabry/learn-pub-sub-starter/internal/routing"
	"github.com/tdabry/learn-pub-sub-starter/internal/gamelogic"
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
	
	_, queue, err := pubsub.DeclareAndBind(rabbit, routing.ExchangePerilDirect, 
						routing.PauseKey+"."+username, routing.PauseKey, pubsub.Transient)
	if err != nil {
		log.Print("something went wrong with queue binding1")
		return
	}

	fmt.Printf("Queue %v declared and bound!\n", queue.Name)
	
	gamelogic.PrintClientHelp()
	gameState := gamelogic.NewGameState(username)
	err = pubsub.SubscribeJSON(rabbit, routing.ExchangePerilDirect, routing.PauseKey+"."+username,
							routing.PauseKey, pubsub.Transient, handlerPause(gameState))	
	if err != nil {
		log.Print("something went wrong with queue binding2")
		return
	}
	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}
		word := words[0]
		if word == "spawn" {
			err = gameState.CommandSpawn(words)
			if skip := hasErr(err); skip {continue}
		} else if word == "move" {
			_, err := gameState.CommandMove(words)
			if skip := hasErr(err); skip {continue}
			log.Printf("move %s %v", words[1], words[2:])
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

func hasErr(err error) bool {
	if err != nil {
		log.Printf("%s", err)
		return true
	}
	return false
}

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) {
	return func(ps routing.PlayingState){
		defer fmt.Print("> ")
		gs.HandlePause(ps)
	}
}