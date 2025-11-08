package pubsub 

import (
	"log"
	"context"
	amqp "github.com/rabbitmq/amqp091-go"
	"encoding/json"
)
type SimpleQueueType int
const (
	Durable SimpleQueueType = 0
	Transient SimpleQueueType = 1
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	val_json, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return ch.PublishWithContext(context.Background(), exchange, key, false, false, 
		amqp.Publishing{ContentType: "application/json", Body: val_json})

}

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error) {
	ch, err := conn.Channel()
	if err != nil {
		log.Print("error creating channel")
		return nil, amqp.Queue{}, err
	}
	dur := false
	autodel := false
	excl := false
	switch (queueType) {
	case Durable:
		dur = true
	case Transient:
		autodel = true
		excl = true
	}
	newQ, err := ch.QueueDeclare(queueName, dur, autodel, excl, false, nil)
	if err != nil {
		log.Print("error declaring queue")
		return nil, amqp.Queue{}, err
	}
	err = ch.QueueBind(queueName, key, exchange, false, nil)
	if err != nil {
		log.Printf("qName: %s, key: %s, ex: %s", queueName, key, exchange)
		log.Print("error binding queue")
		return nil, amqp.Queue{}, err
	}
	return ch, newQ, nil
}
