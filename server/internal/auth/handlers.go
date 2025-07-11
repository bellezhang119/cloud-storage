package auth

import (
	"encoding/json"
	"net/http"

	"github.com/bellezhang119/cloud-storage/internal/util"
)

type RegisteRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type SendVerificationEmail struct {
	Email string `json:"email"`
}

func RegisterHandler(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RegisteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.Email == "" || req.Password == "" {
			util.RespondWithError(w, http.StatusBadRequest, "Email and password are required")
			return
		}

		hashedPassword, err := util.HashPassword(req.Password)
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "Failed to hash password")
			return
		}

		_, err = service.CreateUser(r.Context(), req.Email, hashedPassword)
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "Failed to create user")
			return
		}

		// TODO: Send verification email

		util.RespondWithJSON(w, http.StatusCreated, map[string]string{
			"message": "User created, please check your email to verify your account",
		})
	}
}

func VerifyEmailHandler(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			util.RespondWithError(w, http.StatusBadRequest, "Missing verification token")
			return
		}

		if err := service.VerifyUserByToken(r.Context(), token); err != nil {
			util.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		util.RespondWithJSON(w, http.StatusOK, map[string]string{
			"message": "Email verified",
		})
	}
}

// TODO: func SendVerificationEmailHandler
