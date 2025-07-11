package server

import (
	"net/http"

	"github.com/bellezhang119/cloud-storage/internal/auth"
)

func NewRouter(authService *auth.Service) *http.ServeMux {
	mux := http.NewServeMux()

	// Auth routes
	mux.HandleFunc("POST /auth/register", auth.RegisterHandler(authService))
	mux.HandleFunc("GET /auth/verify", auth.VerifyEmailHandler(authService))

	// Health checks
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Ready"))
	})
	mux.HandleFunc("GET /err", HandleErr)

	return mux
}
