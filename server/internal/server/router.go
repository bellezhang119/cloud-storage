package server

import "net/http"

func NewRouter() http.Handler {
	router := http.NewServeMux()

	router.HandleFunc("GET /ready", HandlerReadiness)
	router.HandleFunc("GET /err", HandleErr)

	return router
}
