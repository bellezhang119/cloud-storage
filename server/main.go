package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load(".env")

	portString := os.Getenv("PORT")

	if portString == "" {
		log.Fatal("Port is not defined")
	}

	fmt.Println("Port:", portString)

	router := http.NewServeMux()

	router.HandleFunc("GET /ready", handlerReadiness)
	router.HandleFunc("GET /err", handleErr)
}
