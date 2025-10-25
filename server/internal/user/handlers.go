package user

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

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
		ctxUserID, _ := middleware.GetUserID(r.Context())
		pathUserID := r.PathValue("user_id")

		if strconv.Itoa(int(ctxUserID)) != pathUserID {
			util.RespondWithError(w, http.StatusBadRequest, "Mismatch between token and path value: user id")
			return
		}

		user, err := service.GetUserByID(r.Context(), ctxUserID)
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		util.RespondWithJSON(w, http.StatusOK, user)
	}
}

func GetUserByEmailHandler(service ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctxUserEmail, _ := middleware.GetUserEmail(r.Context())
		pathUserEmail := r.PathValue("user_email")

		if ctxUserEmail != pathUserEmail {
			util.RespondWithError(w, http.StatusBadRequest, "Mismatch between token and path value: user email")
			return
		}

		user, err := service.GetUserByEmail(r.Context(), ctxUserEmail)
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		util.RespondWithJSON(w, http.StatusOK, user)
	}
}

func UpdatePasswordHandler(service ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctxUserID, _ := middleware.GetUserID(r.Context())
		pathUserID := r.PathValue("user_id")

		if strconv.Itoa(int(ctxUserID)) != pathUserID {
			util.RespondWithError(w, http.StatusBadRequest, "Mismatch between token and path value: user id")
			return
		}

		var req UpdatePasswordRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.NewPassword == "" {
			util.RespondWithError(w, http.StatusBadRequest, "New password is required")
			return
		}

		if err := service.UpdateUserPassword(r.Context(), ctxUserID, req.NewPassword); err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		util.RespondWithJSON(w, http.StatusOK, map[string]string{
			"message": "Password updated successfully",
		})
	}
}

func DeleteUserHandler(service ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctxUserID, _ := middleware.GetUserID(r.Context())
		pathUserID := r.PathValue("user_id")

		if strconv.Itoa(int(ctxUserID)) != pathUserID {
			util.RespondWithError(w, http.StatusBadRequest, "Mismatch between token and path value: user id")
			return
		}

		if err := service.DeleteUser(r.Context(), ctxUserID); err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		util.RespondWithJSON(w, http.StatusOK, map[string]string{
			"message": "User deleted successfully",
		})
	}
}
