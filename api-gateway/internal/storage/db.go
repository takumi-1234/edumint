package storage

import (
	"database/sql"
	"log"
	"os"
	"time"
)

// ConnectDB establishes a connection to the PostgreSQL database.
// It retries a few times before failing.
func ConnectDB() *sql.DB {
	var db *sql.DB
	var err error
	for i := 0; i < 5; i++ {
		db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
		if err != nil {
			log.Printf("Failed to open DB connection: %v. Retrying...", err)
			time.Sleep(3 * time.Second)
			continue
		}
		if err = db.Ping(); err == nil {
			log.Println("Successfully connected to the database!")
			return db
		}
		log.Printf("Database ping failed: %v. Retrying...", err)
		db.Close()
		time.Sleep(3 * time.Second)
	}
	log.Fatalf("Could not connect to the database after several retries.")
	return nil
}
