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
	"github.com/bellezhang119/cloud-storage/internal/storage"
	"github.com/bellezhang119/cloud-storage/internal/storage/local"
	"github.com/bellezhang119/cloud-storage/internal/user"
	"github.com/joho/godotenv"
)

// TODO: share handlers, search function

func main() {
	db, err := config.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}

	err = godotenv.Load(".env")
	if err != nil {
		return
	}

	basePath := os.Getenv("BASE_PATH")

	queries := database.New(db)
	localStorage := local.NewLocalStorage(basePath)
	userService := user.NewService(queries)
	authService := auth.NewService(queries, userService)
	storageService := storage.NewService(queries, userService, localStorage)

	portString := os.Getenv("PORT")

	if portString == "" {
		log.Fatal("Port is not defined")
	}

	fmt.Println("Port:", portString)

	router := server.NewRouter(authService, userService, storageService)

	err = http.ListenAndServe(portString, router)

	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
