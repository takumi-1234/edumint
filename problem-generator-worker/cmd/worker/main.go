package main

import (
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/your-username/edumint/problem-generator-worker/internal/processor"
	"github.com/your-username/edumint/problem-generator-worker/internal/queue"
	"github.com/your-username/edumint/problem-generator-worker/internal/services/gemini"
	"github.com/your-username/edumint/problem-generator-worker/internal/storage"
)

const GENERATION_QUEUE = "problem_generation_queue"

func main() {
	// Setup Database Connection
	db := connectToDB()
	defer db.Close()

	// Setup Services
	storageService := &storage.Service{DB: db}
	geminiService, err := gemini.NewService()
	if err != nil {
		log.Fatalf("Failed to initialize Gemini service: %v", err)
	}

	// Setup Processor
	jobProcessor := &processor.Processor{
		StorageService: storageService,
		GeminiService:  geminiService,
	}

	// Setup RabbitMQ Consumer
	queueClient := queue.MustConnect()
	defer queueClient.Close()

	msgs, err := queueClient.Consume(GENERATION_QUEUE)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %s", err)
	}

	forever := make(chan bool)
	log.Printf("Worker started. Waiting for jobs on queue '%s'. To exit press CTRL+C", GENERATION_QUEUE)

	// Start a goroutine to process messages from the queue.
	go func() {
		for d := range msgs {
			log.Printf("Received a job with correlation ID: %s. Processing...", d.CorrelationId)
			jobProcessor.ProcessJob(d.Body)
			d.Ack(false) // Acknowledge the message after processing
		}
	}()

	<-forever // Block forever.
}

func connectToDB() *sql.DB {
	var db *sql.DB
	var err error
	for i := 0; i < 5; i++ {
		db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
		if err == nil && db.Ping() == nil {
			log.Println("Worker successfully connected to the database!")
			return db
		}
		if db != nil {
			db.Close()
		}
		log.Printf("Worker failed to connect to DB: %v. Retrying...", err)
		time.Sleep(3 * time.Second)
	}
	log.Fatalf("Worker could not connect to the database after several retries.")
	return nil
}
