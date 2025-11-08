package main

import (
	"log"
	"os"
	"os/signal"
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
	var qType pubsub.SimpleQueueType = pubsub.Transient
	_, queue, err := pubsub.DeclareAndBind(rabbit, routing.ExchangePerilDirect, routing.PauseKey+"."+username,
				routing.PauseKey, qType)
	if err != nil {
		log.Print("something went wrong with queue binding")
		return
	}
	
	fmt.Printf("Queue %v declared and bound!\n", queue.Name)

	// wait for ctrl+c
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
	fmt.Println("RabbitMQ connection closed.")
}
