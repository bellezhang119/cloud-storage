package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/bellezhang119/cloud-storage/internal/auth"
	"github.com/bellezhang119/cloud-storage/internal/config"
	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/server"
	"github.com/bellezhang119/cloud-storage/internal/user"
	"github.com/joho/godotenv"
)

func main() {
	db, err := config.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}

	queries := database.New(db)
	userService := user.NewService(queries)
	authService := auth.NewService(queries, userService)

	godotenv.Load(".env")

	portString := os.Getenv("PORT")

	if portString == "" {
		log.Fatal("Port is not defined")
	}

	fmt.Println("Port:", portString)

	router := server.NewRouter(authService)

	err = http.ListenAndServe(portString, router)

	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
