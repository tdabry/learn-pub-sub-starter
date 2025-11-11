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

	pubCh, _, err := pubsub.DeclareAndBind(rabbit, routing.ExchangePerilTopic, 
		routing.GameLogSlug, "game_logs.*", pubsub.Durable)
	if err != nil {
		log.Fatal(err)
	}
	err = pubsub.Subscribe(rabbit, routing.ExchangePerilTopic, routing.GameLogSlug,
			routing.GameLogSlug+".*", pubsub.Durable, handlerLog, pubsub.Gob_unmarshal)
	if err != nil {
		log.Fatal(err)
	}
 	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}
		word := words[0]
		if word == "pause" {
			err := pubsub.PublishJSON(pubCh, routing.ExchangePerilDirect, routing.PauseKey,
				routing.PlayingState{IsPaused: true})
			if err != nil {
				log.Print(err)
			}
		} else if word == "resume" {
			err := pubsub.PublishJSON(pubCh, routing.ExchangePerilDirect, routing.PauseKey,
				routing.PlayingState{IsPaused: false})
			if err != nil {
				log.Print(err)
			}
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

func handlerLog(lg routing.GameLog) pubsub.Acktype {
	defer fmt.Print("> ")
	err := gamelogic.WriteLog(lg)
	if err != nil {
		return pubsub.NackRequeue
	}
	return pubsub.Ack
}