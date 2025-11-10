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
	fmt.Println("Starting Peril server...")
	conn := "amqp://guest:guest@localhost:5672/"
	rabbit, err := amqp.Dial(conn)
	if err != nil {
		fmt.Println("Failed to connect")
		return
	}
	defer rabbit.Close()
	fmt.Println("Connection successful")
	gamelogic.PrintServerHelp()
	ch, err := rabbit.Channel()
	if err != nil {
		log.Println("error creating channel")
		return
	}
	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}
		word := words[0]
		if word == "pause" {
			pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey,
				routing.PlayingState{IsPaused: true})
		} else if word == "resume" {
			pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey,
				routing.PlayingState{IsPaused: false})
		} else if word == "help" {
			gamelogic.PrintServerHelp()
		} else if word == "quit" {
			log.Println("Exiting...")
			break
		} else {
			log.Printf("Unknown command: <%s>", word)
		}
	}
}
