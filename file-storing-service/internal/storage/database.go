package storage

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Warning: .env file not loaded, using environment variables")
	}

	connectionString := fmt.Sprintf(
		"user=%s password=%s dbname=%s sslmode=%s host=%s port=%s",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_SSLMODE"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
	)

	var errOpen error
	DB, errOpen = sql.Open("postgres", connectionString)
	if errOpen != nil {
		log.Fatalf("Failed to open DB: %v", errOpen)
	}

	if DB == nil {
		log.Fatal("DB is nil after sql.Open")
	}

	if errPing := DB.Ping(); errPing != nil {
		log.Fatalf("Failed to ping DB: %v", errPing)
	}

	log.Println("Successfully connected to DB")
}
