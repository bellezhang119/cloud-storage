package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/bellezhang119/cloud-storage/internal/util"
)

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type SendVerificationEmailRequest struct {
	Email string `json:"email"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func RegisterHandler(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.Email == "" || req.Password == "" {
			util.RespondWithError(w, http.StatusBadRequest, "Email and password are required")
			return
		}

		user, err := service.CreateUser(r.Context(), req.Email, req.Password)
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "Failed to create user")
			return
		}

		portString := os.Getenv("PORT")
		verificationLink := fmt.Sprintf("http://localhost%s/auth/verify?token=%s", portString, user.VerificationToken.String)
		subject := "Verify your email address at Cloud-Storage"
		body := fmt.Sprintf("Click the link to verify your email:\n\n%s", verificationLink)
		err = util.SendEmail(user.Email, subject, body)

		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

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

func SendVerificationEmailHandler(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req SendVerificationEmailRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		user, err := service.GetUserByEmail(r.Context(), req.Email)
		if err != nil {
			util.RespondWithError(w, http.StatusNotFound, "User not found")
			return
		}

		if user.IsVerified {
			util.RespondWithJSON(w, http.StatusOK, map[string]string{
				"message": "User already verified",
			})
			return
		}

		verificationToken, err := service.UpdateVerificationToken(r.Context(), user)
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		portString := os.Getenv("PORT")
		verificationLink := fmt.Sprintf("http://localhost%s/auth/verify?token=%s", portString, verificationToken)
		subject := "Verify your email address at Cloud-Storage"
		body := fmt.Sprintf("Click the link to verify your email:\n\n%s", verificationLink)
		err = util.SendEmail(user.Email, subject, body)

		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		util.RespondWithJSON(w, http.StatusOK, map[string]string{
			"message": "Verification email sent",
		})
	}
}

func LoginHandler(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.Email == "" || req.Password == "" {
			util.RespondWithError(w, http.StatusBadRequest, "Email and password are required")
			return
		}

		user, err := service.AuthenticateUser(r.Context(), req.Email, req.Password)
		if err != nil {
			util.RespondWithError(w, http.StatusUnauthorized, "Invalid email or password")
			return
		}

		accessToken, refreshToken, err := service.GenerateJWTTokens(r.Context(), user)
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "Failed to generate tokens")
			return
		}

		util.RespondWithJSON(w, http.StatusOK, map[string]string{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		})
	}
}

func RefreshTokenHandler(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RefreshTokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid request")
			return
		}

		accessToken, refreshToken, err := service.RefreshJWTTokens(r.Context(), req.RefreshToken)
		if err != nil {
			util.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}

		util.RespondWithJSON(w, http.StatusOK, map[string]string{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		})
	}
}
