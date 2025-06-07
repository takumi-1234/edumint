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
				log.Println("Successfully connected to RabbitMQ!")
				return &Client{conn: conn, ch: ch}
			}
		}
		log.Printf("Failed to connect to RabbitMQ: %v. Retrying...", err)
		time.Sleep(3 * time.Second)
	}
	log.Fatalf("Could not connect to RabbitMQ after several retries.")
	return nil
}

// Publish sends a message to the specified queue. It declares the queue to ensure it exists.
func (c *Client) Publish(queueName string, body []byte) error {
	// Declare a queue to ensure it exists. It's idempotent.
	_, err := c.ch.QueueDeclare(
		queueName,
		true,  // Durable
		false, // Delete when unused
		false, // Exclusive
		false, // No-wait
		nil,   // Arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare a queue: %w", err)
	}

	return c.ch.Publish(
		"",        // Exchange
		queueName, // Routing key
		false,     // Mandatory
		false,     // Immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent, // Make message persistent
			ContentType:  "application/json",
			Body:         body,
		},
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
