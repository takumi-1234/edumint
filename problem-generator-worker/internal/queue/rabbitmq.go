package queue

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/streadway/amqp"
)

type Client struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

// MustConnect establishes a connection to RabbitMQ, retrying if necessary.
func MustConnect() *Client {
	var conn *amqp.Connection
	var err error
	for i := 0; i < 5; i++ {
		conn, err = amqp.Dial(os.Getenv("RABBITMQ_URL"))
		if err == nil {
			ch, err := conn.Channel()
			if err == nil {
				log.Println("Worker successfully connected to RabbitMQ!")
				return &Client{conn: conn, ch: ch}
			}
		}
		log.Printf("Worker failed to connect to RabbitMQ: %v. Retrying...", err)
		time.Sleep(3 * time.Second)
	}
	log.Fatalf("Worker could not connect to RabbitMQ after several retries.")
	return nil
}

// Consume starts consuming messages from the specified queue.
func (c *Client) Consume(queueName string) (<-chan amqp.Delivery, error) {
	_, err := c.ch.QueueDeclare(
		queueName,
		true,  // Durable
		false, // Delete when unused
		false, // Exclusive
		false, // No-wait
		nil,   // Arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare a queue: %w", err)
	}

	// Set Quality of Service to prefetch only one message at a time.
	// This ensures that a busy worker doesn't hoard messages it can't process.
	err = c.ch.Qos(
		1,     // prefetchCount
		0,     // prefetchSize
		false, // global
	)
	if err != nil {
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	return c.ch.Consume(
		queueName,
		"",    // Consumer
		false, // Auto-Ack (we will manually acknowledge)
		false, // Exclusive
		false, // No-local
		false, // No-wait
		nil,   // Args
	)
}

// Close gracefully closes the channel and connection.
func (c *Client) Close() {
	if c.ch != nil {
		c.ch.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}
