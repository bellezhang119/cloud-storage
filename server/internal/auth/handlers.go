package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/middleware"
	"github.com/bellezhang119/cloud-storage/internal/util"
)

type ServiceInterface interface {
	CreateUser(ctx context.Context, email, password string) (database.User, error)
	VerifyUserByToken(ctx context.Context, token string) error
	GetUserByEmail(ctx context.Context, email string) (database.User, error)
	UpdateVerificationToken(ctx context.Context, user database.User) (string, error)
	AuthenticateUser(ctx context.Context, email, password string) (database.User, error)
	GenerateJWTTokens(ctx context.Context, user database.User) (string, string, error)
	RefreshJWTTokens(ctx context.Context, oldRefreshToken string) (string, string, error)
}

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

type EmailSender func(to, subject, body string) error

func RegisterHandler(service ServiceInterface, sendEmail EmailSender) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("handling register request")

		var req RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("invalid request body", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		logger = logger.With("email", req.Email)

		if req.Email == "" || req.Password == "" {
			logger.Warn("missing email or password in request")
			util.RespondWithError(w, http.StatusBadRequest, "Email and password are required")
			return
		}

		user, err := service.CreateUser(r.Context(), req.Email, req.Password)
		if err != nil {
			logger.Error("failed to create user", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, "Failed to create user")
			return
		}

		logger = logger.With("user_id", user.ID)

		portString := os.Getenv("PORT")
		verificationLink := fmt.Sprintf("http://localhost%s/auth/verify?token=%s", portString, user.VerificationToken.String)
		subject := "Verify your email address at Cloud-Storage"
		body := fmt.Sprintf("Click the link to verify your email:\n\n%s", verificationLink)

		if err := sendEmail(user.Email, subject, body); err != nil {
			logger.Error("failed to send verification email", "error", err, "email", user.Email)
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logger.Info("user registered successfully", "user_id", user.ID)
		util.RespondWithJSON(w, http.StatusCreated, map[string]string{
			"message": "User created, please check your email to verify your account",
		})
	}
}

func VerifyEmailHandler(service ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("handling verify email request")

		token := r.URL.Query().Get("token")
		if token == "" {
			logger.Warn("missing verification token")
			util.RespondWithError(w, http.StatusBadRequest, "Missing verification token")
			return
		}

		if err := service.VerifyUserByToken(r.Context(), token); err != nil {
			logger.Error("failed to verify user", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		logger.Info("email verified successfully", "token", token)
		util.RespondWithJSON(w, http.StatusOK, map[string]string{
			"message": "Email verified",
		})
	}
}

func SendVerificationEmailHandler(service ServiceInterface, sendEmail EmailSender) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("handling send verification email request")

		var req SendVerificationEmailRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("invalid request body", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		logger = logger.With("email", req.Email)

		user, err := service.GetUserByEmail(r.Context(), req.Email)
		if err != nil {
			logger.Warn("user not found", "email", req.Email)
			util.RespondWithError(w, http.StatusNotFound, "User not found")
			return
		}

		logger = logger.With("user_id", user.ID)

		if user.IsVerified {
			logger.Info("user already verified", "user_id", user.ID)
			util.RespondWithJSON(w, http.StatusOK, map[string]string{
				"message": "User already verified",
			})
			return
		}

		verificationToken, err := service.UpdateVerificationToken(r.Context(), user)
		if err != nil {
			logger.Error("failed to update verification token", "error", err, "user_id", user.ID)
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		portString := os.Getenv("PORT")
		verificationLink := fmt.Sprintf("http://localhost%s/auth/verify?token=%s", portString, verificationToken)
		subject := "Verify your email address at Cloud-Storage"
		body := fmt.Sprintf("Click the link to verify your email:\n\n%s", verificationLink)

		if err := sendEmail(user.Email, subject, body); err != nil {
			logger.Error("failed to send verification email", "error", err, "email", user.Email)
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logger.Info("verification email sent", "user_id", user.ID)
		util.RespondWithJSON(w, http.StatusOK, map[string]string{
			"message": "Verification email sent",
		})
	}
}

func LoginHandler(service ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("handling login request")

		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("invalid request body", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		logger = logger.With("email", req.Email)

		if req.Email == "" || req.Password == "" {
			logger.Warn("missing email or password")
			util.RespondWithError(w, http.StatusBadRequest, "Email and password are required")
			return
		}

		user, err := service.AuthenticateUser(r.Context(), req.Email, req.Password)
		if err != nil {
			logger.Warn("authentication failed", "email", req.Email, "error", err)
			util.RespondWithError(w, http.StatusUnauthorized, "Invalid email or password")
			return
		}

		logger = logger.With("user_id", user.ID)

		accessToken, refreshToken, err := service.GenerateJWTTokens(r.Context(), user)
		if err != nil {
			logger.Error("failed to generate JWT tokens", "error", err, "user_id", user.ID)
			util.RespondWithError(w, http.StatusInternalServerError, "Failed to generate tokens")
			return
		}

		logger.Info("user logged in successfully", "user_id", user.ID)
		util.RespondWithJSON(w, http.StatusOK, map[string]string{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		})
	}
}

func RefreshTokenHandler(service ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("handling refresh token request")

		var req RefreshTokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("invalid request body", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, "Invalid request")
			return
		}

		logger = logger.With("refresh_token", req.RefreshToken)

		accessToken, refreshToken, err := service.RefreshJWTTokens(r.Context(), req.RefreshToken)
		if err != nil {
			logger.Warn("failed to refresh JWT tokens", "error", err)
			util.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}

		logger.Info("JWT tokens refreshed successfully")
		util.RespondWithJSON(w, http.StatusOK, map[string]string{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		})
	}
}
