package pubsub 

import (
	"context"
	amqp "github.com/rabbitmq/amqp091-go"
	"encoding/json"
)
func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	val_json, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return ch.PublishWithContext(context.Background(), exchange, key, false, false, 
		amqp.Publishing{ContentType: "application/json", Body: val_json})

}
