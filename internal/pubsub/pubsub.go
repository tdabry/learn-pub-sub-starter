package pubsub

import (
	"context"
	"encoding/json"
	"log"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/tdabry/learn-pub-sub-starter/internal/routing"
)
type SimpleQueueType int
const (
	Durable SimpleQueueType = iota
	Transient 
)

type Acktype int
const (
	Ack Acktype = iota
	NackRequeue  
	NackDiscard  
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

	newQ, err := ch.QueueDeclare(queueName, dur, autodel, excl, false, 
		amqp.Table{"x-dead-letter-exchange": routing.ExchangePerilDead})
	if err != nil {
		log.Print("error declaring queue")
		return nil, amqp.Queue{}, err
	}
	err = ch.QueueBind(queueName, key, exchange, false, nil)
	if err != nil {
		log.Printf("error binding queue\nqName: %s, key: %s, ex: %s", queueName, key, exchange)
		return nil, amqp.Queue{}, err
	}
	return ch, newQ, nil
}

func SubscribeJSON[T any](
    conn *amqp.Connection,
    exchange,
    queueName,
    key string,
    queueType SimpleQueueType, // an enum to represent "durable" or "transient"
    handler func(T) Acktype,
) error {
	ch, _, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		log.Printf("%s %s", queueName, key)
		return err
	}
	deliveryCh, err := ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	go func(){
		defer ch.Close()
		for el := range deliveryCh {
			var decoded T
			err = json.Unmarshal(el.Body, &decoded)
			if err != nil {
				log.Print("error unmarshalling")
				el.Nack(false, false)
			} else {
				ackType := handler(decoded)
				switch(ackType){
				case Ack:
					log.Print("ack")
					el.Ack(false)
				case NackRequeue:
					log.Print("nack req")
					el.Nack(false, true)
				case NackDiscard:
					log.Print("nack disc")
					el.Nack(false, false)
				}
			}
		}
	}()
	return nil
}