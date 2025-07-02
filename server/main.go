package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/bellezhang119/cloud-storage/internal/config"
	"github.com/bellezhang119/cloud-storage/internal/server"
	"github.com/joho/godotenv"
)

func main() {
	config.ConnectDB()

	godotenv.Load(".env")

	portString := os.Getenv("PORT")

	if portString == "" {
		log.Fatal("Port is not defined")
	}

	fmt.Println("Port:", portString)

	router := server.NewRouter()

	err := http.ListenAndServe(portString, router)

	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
