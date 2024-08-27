# Setup RabbitMQ connection for the CLI game

The starter code was copied from Boot.dev's [Learn Pub/Sub](https://learn.boot.dev/learn-pub-sub) course.


## Pub/Sub, RabbitMQ Theory

Pub/Sub is a messaging pattern where senders of messages (publishers) do not send messages directly to receivers (subscribers). Instead, they just publish to a single broker. The publisher doesn't need to worry about who all the subscribers are. The broker is responsible for delivering a copy of the message to any interested subscribers.

Pub/Sub systems are often used to enable "event-driven design", or "event-driven architecture". An event-driven architecture uses events to trigger and communicate between decoupled systems.

RabbitMQ is a popular open-source message broker written in Erlang that implements the AMQP protocol. It's flexible, powerful, and has a great built-in management UI.
A message broker is a middleman that allows different parts of the system to communicate without knowing about each other.

### Conection [package link](https://github.com/rabbitmq/amqp091-go)
```go
    import amqp "github.com/rabbitmq/amqp091-go"

    conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
    defer conn.Close()

    publishCh, err := conn.Channel()
```

### [Exchange](https://www.rabbitmq.com/tutorials/amqp-concepts#exchanges)
Messages are not published directly to a queue. Instead, the producer sends messages to an exchange. Exchanges are message routing agents, defined by the virtual host within RabbitMQ. An exchange is responsible for routing the messages to different queues with the help of header attributes, bindings, and routing keys.
  - [Exchange](https://www.rabbitmq.com/tutorials/amqp-concepts#exchanges): A routing agent that sends messages to queues.
  - [Binding](https://www.rabbitmq.com/tutorials/amqp-concepts#bindings): A link between an exchange and a queue that uses a [routing key](https://www.rabbitmq.com/tutorials/tutorial-five-python#topic-exchange) to decide which messages go to the queue.
  - [Queue](https://www.rabbitmq.com/tutorials/amqp-concepts#queues): A buffer in the RabbitMQ server that holds messages until they are consumed.
  - [Channel](https://www.rabbitmq.com/tutorials/amqp-concepts#amqp-channels): A virtual connection inside a connection that allows you to create queues, exchanges, and publish messages.
  - [Connection](https://www.rabbitmq.com/tutorials/amqp-concepts#amqp-connections): A TCP connection to the RabbitMQ server.

***Types***:
- **Direct**: Messages are routed to the queues based on the message routing key exactly matching the binding key of the queue.
- **Topic**: Messages are routed to queues based on wildcard matches between the routing key and the routing pattern specified in the binding.
- **Fanout**: It routes messages to all of the queues bound to it, ignoring the routing key.
- **Headers**: Routes based on header values instead of the routing key. It's similar to topic but uses message header attributes for routing.

### Queues

[Durable queues](https://www.rabbitmq.com/docs/queues#durability) survive a RabbitMQ server restart, while **transient** queues do not. We can also set the auto-delete and exclusive properties of our queues:

- [Exclusive](https://www.rabbitmq.com/docs/queues#exclusive-queues): The queue can only be used by the connection that created it.
- [Auto-delete](https://www.rabbitmq.com/docs/queues#temporary-queues): The queue will be automatically deleted when its last connection is closed.

```go
	queue, err := ch.QueueDeclare(
		queueName,    // name
		isDurable,    // durable
		deleteUnused, // delete when unused
		isExclusive,  // exclusive
		false,        // no-wait
		amqp.Table{
			"x-dead-letter-exchange": "peril_dlx",
		}, // args
	)

	err = ch.QueueBind(
		queue.Name, // queue name
		key,        // routing key
		exchange,   // exchange
		false,      // no-wait
		nil,        // args
	)
```

### [Consumer](https://www.rabbitmq.com/docs/consumers#basics)

```go
	msgs, err := ch.Consume(
		queue.Name, // queue
		"",         // consumer
		false,      // auto-ack
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // args
	)

  	go func() {
		defer ch.Close()

		for msg := range msgs {
			target, err := unmarshaller(msg.Body)
			if err != nil {
				fmt.Printf("could not unmarshal message: %v\n", err)
				continue
			}
			switch handler(target) {
			case Ack:
				msg.Ack(false)
			case NackDiscard:
				msg.Nack(false, false)
			case NackRequeue:
				msg.Nack(false, true)
			}
		}
	}()
```


### Ack and Nack

"Ack" is short for "acknowledge", and "Nack" is short for "negative acknowledge". There are really 3 options for acknowledging a message:

- **Acknowledge**: Processed successfully.
- **Nack and requeue**: Not processed successfully, but should be requeued on the same queue to be processed again (retry).
- **Nack and discard**: Not processed successfully, and should be discarded (to a dead-letter queue if configured or just deleted entirely).

### Prefetch

When you run a consumer, you may have assumed this process for message consumption:

- Fetch a message from the queue (across the network, which can be slow)
- Process the message
- Acknowledge the message
- Repeat

But that would slow everything down to a crawl due to the full network round trip for every message. Instead, RabbitMQ allows you to [prefetch](https://www.rabbitmq.com/docs/consumer-prefetch) messages. When you prefetch messages, RabbitMQ will send you a batch of messages at once, the client library will store them in memory, and you can process them one by one. Much faster. 

Control prefetch count:
```go
	/*
		Qos controls how many messages or how many bytes the server will try to keep on
		the network for consumers before receiving delivery acks.  The intent of Qos is
		to make sure the network buffers stay full between the server and client.
	*/
	err := ch.Qos(
		10,    // prefetchCount
		0,     // prefetchSize
		false, // global
	)
	if err != nil {
		return fmt.Errorf("could not set qos: %v", err)
	}
```

### Quorum Queues

Generally speaking, there are 2 queue types to worry about:

- [Classic queues](https://www.rabbitmq.com/docs/classic-queues)
- [Quorum queues](https://www.rabbitmq.com/docs/quorum-queues)

Classic queues are the default and are great for most use cases. They are fast and simple. However, they have a single point of failure: the node that the queue is on. If that node goes down, the queue is lost.

Quorum queues are designed to be more resilient. They are stored on multiple nodes, so if one node goes down, the queue is still available. The tradeoff is that because quorum queues are stored on multiple nodes, they are slower than classic queues.

As a general rule, I use classic queues for my ephemeral queues (transient, auto-delete, etc). I use quorum queues for most of my durable queues.