package storage

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

var DB *sql.DB

func InitDB() {
	err := godotenv.Load(".env")
	if err != nil {
		return
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

	DB, err = sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatal(err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Successfully connected to DB")
}
