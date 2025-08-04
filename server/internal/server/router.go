package server

import (
	"net/http"

	"github.com/bellezhang119/cloud-storage/internal/auth"
	"github.com/bellezhang119/cloud-storage/internal/util"
)

func NewRouter(authService *auth.Service) *http.ServeMux {
	mux := http.NewServeMux()

	//protectedHandler := auth.AuthMiddleware(util.VerifyAccessToken)

	// Auth routes
	mux.HandleFunc("POST /auth/register", auth.RegisterHandler(authService, util.SendEmail))
	mux.HandleFunc("GET /auth/verify", auth.VerifyEmailHandler(authService))
	mux.HandleFunc("POST /auth/resend-verification", auth.SendVerificationEmailHandler(authService, util.SendEmail))
	mux.HandleFunc("POST /auth/login", auth.LoginHandler(authService))
	mux.HandleFunc("POST /auth/refresh", auth.RefreshTokenHandler(authService))

	// Health checks
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Ready"))
	})
	mux.HandleFunc("GET /err", HandleErr)

	return mux
}
