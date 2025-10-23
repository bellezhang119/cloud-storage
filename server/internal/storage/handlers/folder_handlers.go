package handlers

import (
	"context"
	"io"
	"net/http"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/middleware"
	"github.com/bellezhang119/cloud-storage/internal/storage/services"
	"github.com/bellezhang119/cloud-storage/internal/util"
	"github.com/google/uuid"
)

type FolderServiceInterface interface {
	GetFolderByID(ctx context.Context, folderID uuid.UUID, userID int32) (database.Folder, error)
	ListFoldersByParent(ctx context.Context, userID int32, parentID *uuid.UUID) ([]database.Folder, error)
	GetFolderByNameInParent(ctx context.Context, userID int32, name string, parentID *uuid.UUID) (database.Folder, error)
	CreateFolder(ctx context.Context, userID int32, name string, parentID *uuid.UUID) (database.Folder, error)
	GetZippedFolderForDownload(ctx context.Context, folderID uuid.UUID, userID int32, w io.Writer) (database.Folder, error)
	GetZippedFoldersForDownload(ctx context.Context, folderIDs []uuid.UUID, userID int32, w io.Writer) ([]database.Folder, error)
	UploadFolder(
		ctx context.Context,
		userID int32,
		parentID *uuid.UUID,
		items []services.FolderUploadItem,
		overwrite bool,
		basePath string,
	) (services.UploadResult, error)
	DeleteFolders(ctx context.Context, folderIDs []uuid.UUID, userID int32) error
	MoveFolders(ctx context.Context, folderIDs []uuid.UUID, userID int32, newParentID *uuid.UUID, overwriteFiles bool) error
	RenameFolder(ctx context.Context, folderID uuid.UUID, newName string, userID int32, overwriteFiles bool) error
}

func GetFolderByID(service FolderServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctxUserID, _ := middleware.GetUserID(r.Context())
		pathUserID := r.PathValue("user_id")

		if string(ctxUserID) != pathUserID {
			util.RespondWithError(w, http.StatusBadRequest, "Mismatch between token and path value: user id")
			return
		}

		folderIDStr := r.PathValue("folder_id")
		if folderIDStr == "" {
			util.RespondWithError(w, http.StatusBadRequest, "Missing folder_id parameter")
			return
		}

		folderID, err := uuid.Parse(folderIDStr)
		if err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid file ID format")
			return
		}

		folder, err := service.GetFolderByID(r.Context(), folderID, ctxUserID)

		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		util.RespondWithJSON(w, http.StatusOK, folder)
	}
}
