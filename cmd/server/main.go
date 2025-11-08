package main

import (
	amqp "github.com/rabbitmq/amqp091-go"
	"fmt"
	"os"
	"os/signal"
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
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
 	fmt.Println("Connection closed")
}
