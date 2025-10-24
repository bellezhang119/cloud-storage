package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/middleware"
	"github.com/bellezhang119/cloud-storage/internal/storage/services"
	"github.com/bellezhang119/cloud-storage/internal/util"
	"github.com/google/uuid"
)

type FolderServiceInterface interface {
	GetFolderByID(ctx context.Context, folderID uuid.UUID, userID int32) (database.Folder, error)
	ListFoldersByParent(ctx context.Context, userID int32, parentID *uuid.UUID) ([]database.Folder, error)
	GetFolderFullPath(ctx context.Context, folderID uuid.UUID, userID int32) (string, error)
	GetFolderByNameInParent(ctx context.Context, userID int32, name string, parentID *uuid.UUID) (database.Folder, error)
	CreateFolder(ctx context.Context, userID int32, name string, parentID *uuid.UUID) (database.Folder, error)
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

type CreateFolderRequest struct {
	Name string `json:"name"` // folder name, required
}

type DownloadFoldersRequest struct {
	FolderIDs []uuid.UUID `json:"file_ids"`
}

type FolderUploadItemRequest struct {
	Path       string                `json:"path"`      // e.g., "folder1/folder2/file.txt"
	IsFolder   bool                  `json:"is_folder"` // true for folder, false for file
	File       multipart.File        `json:"-"`         // actual file content for files
	FileHeader *multipart.FileHeader `json:"-"`         // header info for size, name, etc.
}

func validateFolderName(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("folder name is required")
	}
	if len(name) > 255 {
		return errors.New("folder name is too long")
	}
	if strings.ContainsAny(name, `/\:*?"<>|`) {
		return errors.New("folder name contains invalid characters")
	}
	return nil
}

func GetFolderByIDHandler(service FolderServiceInterface) http.HandlerFunc {
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
			util.RespondWithError(w, http.StatusBadRequest, "Invalid folder ID format")
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

// ListFoldersByParentHandler make 2 routes and root folder and normal folder
func ListFoldersByParentHandler(service FolderServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctxUserID, _ := middleware.GetUserID(r.Context())
		pathUserID := r.PathValue("user_id")

		if string(ctxUserID) != pathUserID {
			util.RespondWithError(w, http.StatusBadRequest, "Mismatch between token and path value: user id")
		}

		parentIDStr := r.PathValue("parent_id")

		var parentID *uuid.UUID
		if parentIDStr != "" {
			id, err := uuid.Parse(parentIDStr)
			if err != nil {
				util.RespondWithError(w, http.StatusBadRequest, "Invalid parent ID format")
				return
			}

			if id != uuid.Nil {
				parentID = &id
			}
		}

		folders, err := service.ListFoldersByParent(r.Context(), ctxUserID, parentID)
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		util.RespondWithJSON(w, http.StatusOK, folders)
	}
}

func GetFolderFullPath(service FolderServiceInterface) http.HandlerFunc {
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
			util.RespondWithError(w, http.StatusBadRequest, "Invalid folder ID format")
		}

		path, err := service.GetFolderFullPath(r.Context(), folderID, ctxUserID)

		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		util.RespondWithJSON(w, http.StatusOK, path)
	}
}

func GetFolderByNameInParentHandler(service FolderServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctxUserID, _ := middleware.GetUserID(r.Context())
		pathUserID := r.PathValue("user_id")

		if string(ctxUserID) != pathUserID {
			util.RespondWithError(w, http.StatusBadRequest, "Mismatch between token and path value: user id")
			return
		}

		name := r.URL.Query().Get("name")
		if name == "" {
			util.RespondWithError(w, http.StatusBadRequest, "File name is required")
			return
		}

		parentIDStr := r.PathValue("parent_id")

		var parentID *uuid.UUID
		if parentIDStr != "" {
			id, err := uuid.Parse(parentIDStr)
			if err != nil {
				util.RespondWithError(w, http.StatusBadRequest, "Invalid parent ID format")
				return
			}

			if id != uuid.Nil {
				parentID = &id
			}
		}

		folder, err := service.GetFolderByNameInParent(r.Context(), ctxUserID, name, parentID)
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		util.RespondWithJSON(w, http.StatusOK, folder)
	}
}

func CreateFolderHandler(service FolderServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctxUserID, _ := middleware.GetUserID(r.Context())
		pathUserID := r.PathValue("user_id")

		if string(ctxUserID) != pathUserID {
			util.RespondWithError(w, http.StatusBadRequest, "Mismatch between token and path value: user id")
			return
		}

		var req CreateFolderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if validateFolderName(req.Name) != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid folder name")
		}

		parentIDStr := r.PathValue("parent_id")

		var parentID *uuid.UUID
		if parentIDStr != "" {
			id, err := uuid.Parse(parentIDStr)
			if err != nil {
				util.RespondWithError(w, http.StatusBadRequest, "Invalid parent ID format")
				return
			}

			if id != uuid.Nil {
				parentID = &id
			}
		}

		folder, err := service.CreateFolder(r.Context(), ctxUserID, req.Name, parentID)

		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		util.RespondWithJSON(w, http.StatusCreated, folder)
	}
}

func DownloadFoldersHandler(service FolderServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract user ID from context
		ctxUserID, _ := middleware.GetUserID(r.Context())
		pathUserID := r.PathValue("user_id")
		if strconv.Itoa(int(ctxUserID)) != pathUserID {
			util.RespondWithError(w, http.StatusBadRequest, "Mismatch between token and path value: user id")
			return
		}

		// Parse request body for folder IDs
		var req DownloadFoldersRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if len(req.FolderIDs) == 0 {
			util.RespondWithError(w, http.StatusBadRequest, "No folder IDs provided")
			return
		}

		// Set headers for file download
		w.Header().Set("Content-Disposition", "attachment; filename=\"folders.zip\"")
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)

		// Stream zip to the response writer
		foldersMeta, err := service.GetZippedFoldersForDownload(r.Context(), req.FolderIDs, ctxUserID, w)
		if err != nil {
			// If streaming started, cannot change HTTP status code, but you can log the error
			fmt.Printf("Error zipping folders: %v\n", err)
			return
		}

		fmt.Printf("Downloaded folders: %+v\n", foldersMeta)
	}
}

func UploadFolderHandler(service FolderServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctxUserID, _ := middleware.GetUserID(r.Context())
		pathUserID := r.PathValue("user_id")
		if strconv.Itoa(int(ctxUserID)) != pathUserID {
			util.RespondWithError(w, http.StatusBadRequest, "Mismatch between token and path value: user id")
			return
		}

		parentIDStr := r.PathValue("parent_id")
		var parentID *uuid.UUID
		if parentIDStr != "" {
			id, err := uuid.Parse(parentIDStr)
			if err != nil {
				util.RespondWithError(w, http.StatusBadRequest, "Invalid parent ID format")
				return
			}
			if id != uuid.Nil {
				parentID = &id
			}
		}

		overwrite := r.URL.Query().Get("overwrite") == "true"

		// Parse multipart form
		if err := r.ParseMultipartForm(100 << 20); err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid multipart form")
			return
		}

		var items []services.FolderUploadItem
		form := r.MultipartForm

		for _, headers := range form.File {
			for _, fh := range headers {
				file, err := fh.Open()
				if err != nil {
					util.RespondWithError(w, http.StatusInternalServerError, "Failed to open uploaded file")
					return
				}
				defer file.Close()

				items = append(items, services.FolderUploadItem{
					Name:      fh.Filename,
					IsFolder:  false,
					SizeBytes: fh.Size,
					MimeType:  fh.Header.Get("Content-Type"),
					Content:   file,
				})
			}
		}

		// Call service
		result, err := service.UploadFolder(r.Context(), ctxUserID, parentID, items, overwrite, "")
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		util.RespondWithJSON(w, http.StatusCreated, result)
	}
}
