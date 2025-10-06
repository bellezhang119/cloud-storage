package user

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/util"
)

type ServiceInterface interface {
	GetUserByID(ctx context.Context, id int32) (database.User, error)
	GetUserByEmail(ctx context.Context, email string) (database.User, error)
	UpdatePassword(ctx context.Context, userID int32, newPassword string) error
	UpdateStorage(ctx context.Context, userID int32, newUsedStorage int64) error
	Delete(ctx context.Context, userID int32) error
}

type UpdatePasswordRequest struct {
	NewPassword string `json:"new_password"`
}

type UpdateStorageRequest struct {
	NewUsedBytes int64 `json:"new_used_storage"`
}

func GetUserByIDHandler(service ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid user id", http.StatusBadRequest)
			return
		}

		user, err := service.GetUserByID(r.Context(), int32(id))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	}
}

func GetUserByEmailHandler(service ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := r.URL.Query().Get("email")
		if email == "" {
			http.Error(w, "email query parameter is required", http.StatusBadRequest)
			return
		}

		user, err := service.GetUserByEmail(r.Context(), email)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	}
}

func UpdatePasswordHandler(service ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid user ID")
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

		if err := service.UpdatePassword(r.Context(), int32(id), req.NewPassword); err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		util.RespondWithJSON(w, http.StatusOK, map[string]string{
			"message": "Password updated successfully",
		})
	}
}

func UpdateStorageHandler(service ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid user ID")
			return
		}

		var req UpdateStorageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		if err := service.UpdateStorage(r.Context(), int32(id), req.NewUsedBytes); err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		util.RespondWithJSON(w, http.StatusOK, map[string]string{
			"message": "Used storage updated successfully",
		})
	}
}

func DeleteUserHandler(service ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid user ID")
			return
		}

		if err := service.Delete(r.Context(), int32(id)); err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		util.RespondWithJSON(w, http.StatusOK, map[string]string{
			"message": "User deleted successfully",
		})
	}
}
