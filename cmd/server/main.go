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
	publishCh, err := rabbit.Channel()
	if err != nil {
		log.Fatalf("could not create channel: %v", err)
	}

	err = pubsub.PublishJSON(
		publishCh,
		routing.ExchangePerilDirect,
		routing.PauseKey,
		routing.PlayingState{
			IsPaused: true,
		},
	)
	if err != nil {
		log.Printf("could not publish time: %v", err)
	}
	fmt.Println("Pause message sent!")
}
