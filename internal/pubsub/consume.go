package pubsub

import (
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType uint8,
) (*amqp.Channel, amqp.Queue, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	isDurable := simpleQueueType == amqp.Persistent
	isTransient := simpleQueueType == amqp.Transient
	queue, err := ch.QueueDeclare(
		queueName,   // name
		isDurable,   // durable
		isTransient, // delete when unused
		isTransient, // exclusive
		false,       // no-wait
		nil,         // arguments
	)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not declare queue: %v", err)
	}

	err = ch.QueueBind(
		queue.Name, // queue name
		key,        // routing key
		exchange,   // exchange
		false,      // no-wait
		nil,        // args
	)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not bind queue: %v", err)
	}

	return ch, queue, nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType uint8,
	handler func(T),
) error {

	ch, queue, err := DeclareAndBind(
		conn,
		exchange,
		queueName,
		key,
		simpleQueueType,
	)
	if err != nil {
		return err
	}

	msgs, err := ch.Consume(
		queue.Name, // queue
		"",         // consumer
		false,      // auto-ack
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // args
	)
	if err != nil {
		return err
	}
	go func() {
		for d := range msgs {
			var val T
			err := json.Unmarshal(d.Body, &val)
			if err != nil {
				fmt.Println(err)
			}
			handler(val)
			d.Ack(false)
		}
	}()
	
	return nil
}
