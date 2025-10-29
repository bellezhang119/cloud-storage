package user

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/middleware"
	"github.com/bellezhang119/cloud-storage/internal/util"
)

type ServiceInterface interface {
	GetUserByID(ctx context.Context, id int32) (database.User, error)
	GetUserByEmail(ctx context.Context, email string) (database.User, error)
	UpdateUserPassword(ctx context.Context, userID int32, newPassword string) error
	UpdateUsedStorage(ctx context.Context, userID int32, newUsedStorage int64) error
	DeleteUser(ctx context.Context, userID int32) error
}

type UpdatePasswordRequest struct {
	NewPassword string `json:"new_password"`
}

type UpdateStorageRequest struct {
	NewUsedBytes int64 `json:"new_used_storage"`
}

func GetUserByIDHandler(service ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("handling GetUserByID request")

		userID, err := util.GetUserIDFromPathAndCheck(r)
		if err != nil {
			logger.Error("invalid user ID", "error", err)
			util.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		logger = logger.With("ctx_user_id", userID)

		user, err := service.GetUserByID(r.Context(), userID)
		if err != nil {
			logger.Error("failed to fetch user by ID", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch user")
			return
		}

		logger.Info("user fetched successfully")
		util.RespondWithJSON(w, http.StatusOK, user)
	}
}

func GetUserByEmailHandler(service ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("handling GetUserByEmail request")

		ctxUserEmail, _ := middleware.GetUserEmail(r.Context())
		pathUserEmail := r.PathValue("user_email")
		logger = logger.With("ctx_user_email", ctxUserEmail)

		if ctxUserEmail != pathUserEmail {
			logger.Warn("user email mismatch between token and path")
			util.RespondWithError(w, http.StatusBadRequest, "Mismatch between token and path value: user email")
			return
		}

		user, err := service.GetUserByEmail(r.Context(), ctxUserEmail)
		if err != nil {
			logger.Error("failed to fetch user by email", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch user")
			return
		}

		logger.Info("user fetched successfully by email")
		util.RespondWithJSON(w, http.StatusOK, user)
	}
}

func UpdatePasswordHandler(service ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("handling UpdatePassword request")

		userID, err := util.GetUserIDFromPathAndCheck(r)
		if err != nil {
			logger.Error("invalid user ID", "error", err)
			util.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		logger = logger.With("ctx_user_id", userID)

		var req UpdatePasswordRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("invalid request body", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.NewPassword == "" {
			logger.Warn("missing new password in request")
			util.RespondWithError(w, http.StatusBadRequest, "New password is required")
			return
		}

		if err := service.UpdateUserPassword(r.Context(), userID, req.NewPassword); err != nil {
			logger.Error("failed to update user password", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, "Failed to update password")
			return
		}

		logger.Info("password updated successfully", "user_id", userID)
		util.RespondWithJSON(w, http.StatusOK, map[string]string{
			"message": "Password updated successfully",
		})
	}
}

func DeleteUserHandler(service ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("handling DeleteUser request")

		userID, err := util.GetUserIDFromPathAndCheck(r)
		if err != nil {
			logger.Error("invalid user ID", "error", err)
			util.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		logger = logger.With("ctx_user_id", userID)

		if err := service.DeleteUser(r.Context(), userID); err != nil {
			logger.Error("failed to delete user", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, "Failed to delete user")
			return
		}

		logger.Info("user deleted successfully", "user_id", userID)
		util.RespondWithJSON(w, http.StatusOK, map[string]string{
			"message": "User deleted successfully",
		})
	}
}
