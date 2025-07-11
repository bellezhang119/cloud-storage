package config

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func ConnectDB() (*sql.DB, error) {
	godotenv.Load(".env")
	dbUrl := os.Getenv("DB_URL")

	var err error
	db, err := sql.Open("postgres", dbUrl)

	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	fmt.Println("Successfully connected to the database")

	return db, nil
}
