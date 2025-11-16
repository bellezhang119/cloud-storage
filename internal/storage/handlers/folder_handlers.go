package handlers

import (
	"context"
	"encoding/json"
	"errors"
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

type DeleteFoldersRequest struct {
	FolderIDs []uuid.UUID `json:"file_ids"`
}

type MoveFoldersRequest struct {
	FolderIDs []uuid.UUID `json:"folder_ids"`
}

type RenameFolderRequest struct {
	Name string `json:"name"`
}

// validateFolderName internal helper
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

func GetFolderByIDHandler(s FolderServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("getting folder by ID")

		userID, err := util.GetUserIDFromPathAndCheck(r)
		if err != nil {
			logger.Error("invalid user ID", "error", err)
			util.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		logger = logger.With("ctx_user_id", userID)

		folderID, err := util.GetFolderIDFromPath(r)
		if err != nil {
			logger.Error("invalid folder ID", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		logger = logger.With("folder_id", folderID)

		logger.Info("retrieving folder by ID")
		folder, err := s.GetFolderByID(r.Context(), *folderID, userID)
		if err != nil {
			logger.Error("failed to retrieve folder", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logger.Info("folder retrieved successfully")
		util.RespondWithJSON(w, http.StatusOK, folder)
	}
}

func ListFoldersByParentHandler(s FolderServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("listing folders by parent")

		userID, err := util.GetUserIDFromPathAndCheck(r)
		if err != nil {
			logger.Error("invalid user ID", "error", err)
			util.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		logger = logger.With("ctx_user_id", userID)

		parentID, err := util.GetParentIDFromPath(r)
		if err != nil {
			logger.Error("invalid parent ID", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		logger = logger.With("parent_id", parentID)

		logger.Info("listing folders by parent")
		folders, err := s.ListFoldersByParent(r.Context(), userID, parentID)
		if err != nil {
			logger.Error("failed to list folders", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logger.Info("folders retrieved successfully", "folder_count", len(folders))
		util.RespondWithJSON(w, http.StatusOK, folders)
	}
}

func GetFolderFullPathHandler(s FolderServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("getting folder full path")

		userID, err := util.GetUserIDFromPathAndCheck(r)
		if err != nil {
			logger.Error("invalid user ID", "error", err)
			util.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		logger = logger.With("ctx_user_id", userID)

		folderID, err := util.GetFolderIDFromPath(r)
		if err != nil {
			logger.Error("invalid folder ID", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		logger = logger.With("folder_id", folderID)

		logger.Info("retrieving full folder path")
		path, err := s.GetFolderFullPath(r.Context(), *folderID, userID)
		if err != nil {
			logger.Error("failed to get folder full path", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logger.Info("folder full path retrieved successfully")
		util.RespondWithJSON(w, http.StatusOK, path)
	}
}

func GetFolderByNameInParentHandler(s FolderServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("getting folder by name in parent")

		userID, err := util.GetUserIDFromPathAndCheck(r)
		if err != nil {
			logger.Error("invalid user ID", "error", err)
			util.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		logger = logger.With("ctx_user_id", userID)

		parentID, err := util.GetParentIDFromPath(r)
		if err != nil {
			logger.Error("invalid parent ID", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		logger = logger.With("parent_id", parentID)

		// Get folder name
		name := r.URL.Query().Get("name")
		if name == "" {
			logger.Warn("missing folder name in query")
			util.RespondWithError(w, http.StatusBadRequest, "Folder name is required")
			return
		}
		logger = logger.With("folder_name", name)

		// Call service
		logger.Info("retrieving folder by name in parent")
		folder, err := s.GetFolderByNameInParent(r.Context(), userID, name, parentID)
		if err != nil {
			logger.Error("failed to retrieve folder", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logger.Info("folder retrieved successfully")
		util.RespondWithJSON(w, http.StatusOK, folder)
	}
}

func CreateFolderHandler(s FolderServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("creating folder")

		userID, err := util.GetUserIDFromPathAndCheck(r)
		if err != nil {
			logger.Error("invalid user ID", "error", err)
			util.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		logger = logger.With("ctx_user_id", userID)

		parentID, err := util.GetParentIDFromPath(r)
		if err != nil {
			logger.Error("invalid parent ID", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		logger = logger.With("parent_id", parentID)

		// Parse request body
		var req CreateFolderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Warn("failed to decode request body", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		logger = logger.With("folder_name", req.Name)

		// Validate folder name
		if err := validateFolderName(req.Name); err != nil {
			logger.Warn("invalid folder name", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, "Invalid folder name")
			return
		}

		// Call service
		logger.Info("creating folder")
		folder, err := s.CreateFolder(r.Context(), userID, req.Name, parentID)
		if err != nil {
			logger.Error("failed to create folder", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logger.Info("folder created successfully", "folder_id", folder.ID)
		util.RespondWithJSON(w, http.StatusCreated, folder)
	}
}

func DownloadFoldersHandler(s FolderServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())

		userID, err := util.GetUserIDFromPathAndCheck(r)
		if err != nil {
			logger.Error("invalid user ID", "error", err)
			util.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		logger = logger.With("ctx_user_id", userID)

		// Parse request body
		var req DownloadFoldersRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Warn("failed to decode request body", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		logger = logger.With("folder_ids", req.FolderIDs)

		if len(req.FolderIDs) == 0 {
			logger.Warn("no folder IDs provided")
			util.RespondWithError(w, http.StatusBadRequest, "No folder IDs provided")
			return
		}

		// Set headers for zip download
		w.Header().Set("Content-Disposition", `attachment; filename="folders.zip"`)
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)

		// Stream zip to response
		logger.Info("starting folder download")
		foldersMeta, err := s.GetZippedFoldersForDownload(r.Context(), req.FolderIDs, userID, w)
		if err != nil {
			// Cannot change status if streaming started, log error
			logger.Error("error zipping folders", "error", err)
			return
		}

		logger.Info("folders downloaded successfully", "folders_meta", foldersMeta)
	}
}

func UploadFolderHandler(s FolderServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())

		userID, err := util.GetUserIDFromPathAndCheck(r)
		if err != nil {
			logger.Error("invalid user ID", "error", err)
			util.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		logger = logger.With("ctx_user_id", userID)

		parentID, err := util.GetParentIDFromPath(r)
		if err != nil {
			logger.Error("invalid parent ID", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		logger = logger.With("parent_id", parentID)

		overwrite := r.URL.Query().Get("overwrite") == "true"
		logger = logger.With("overwrite", overwrite)

		// Parse multipart form (limit 100MB)
		if err := r.ParseMultipartForm(100 << 20); err != nil {
			logger.Warn("failed to parse multipart form", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, "Invalid multipart form")
			return
		}

		form := r.MultipartForm
		var items []services.FolderUploadItem

		for _, headers := range form.File {
			for _, fh := range headers {
				file, err := fh.Open()
				if err != nil {
					logger.Error("failed to open uploaded file", "file", fh.Filename, "error", err)
					util.RespondWithError(w, http.StatusInternalServerError, "Failed to open uploaded file")
					return
				}

				// Read content into items; close after processing
				items = append(items, services.FolderUploadItem{
					Name:      fh.Filename,
					IsFolder:  false,
					SizeBytes: fh.Size,
					MimeType:  fh.Header.Get("Content-Type"),
					Content:   file,
				})
			}
		}

		logger.Info("uploading folder", "item_count", len(items))
		result, err := s.UploadFolder(r.Context(), userID, parentID, items, overwrite, "")
		if err != nil {
			logger.Error("failed to upload folder", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logger.Info("folder uploaded successfully")
		util.RespondWithJSON(w, http.StatusCreated, result)
	}
}

func DeleteFoldersHandler(s FolderServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("starting folder delete")

		userID, err := util.GetUserIDFromPathAndCheck(r)
		if err != nil {
			logger.Error("invalid user ID", "error", err)
			util.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		logger = logger.With("ctx_user_id", userID)

		// Parse request body
		var req DeleteFoldersRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Warn("invalid request body", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if len(req.FolderIDs) == 0 {
			logger.Warn("no folder IDs provided")
			util.RespondWithError(w, http.StatusBadRequest, "No folder IDs provided")
			return
		}

		logger.Info("deleting folders", "folder_count", len(req.FolderIDs))
		if err := s.DeleteFolders(r.Context(), req.FolderIDs, userID); err != nil {
			logger.Error("failed to delete folders", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logger.Info("folders deleted successfully")
		util.RespondWithJSON(w, http.StatusOK, "Folder deleted successfully")
	}
}

func MoveFoldersHandler(s FolderServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("starting folder move")

		// Extract user ID
		ctxUserID, _ := middleware.GetUserID(r.Context())
		pathUserID := r.PathValue("user_id")
		logger = logger.With("ctx_user_id", ctxUserID, "path_user_id", pathUserID)

		if strconv.Itoa(int(ctxUserID)) != pathUserID {
			logger.Warn("token user ID does not match path user ID")
			util.RespondWithError(w, http.StatusBadRequest, "Mismatch between token and path value: user id")
			return
		}

		// Parse parent ID from path
		parentIDStr := r.PathValue("parent_id")
		var parentID *uuid.UUID
		if parentIDStr != "" {
			id, err := uuid.Parse(parentIDStr)
			if err != nil {
				logger.Warn("invalid parent ID format", "error", err)
				util.RespondWithError(w, http.StatusBadRequest, "Invalid parent ID format")
				return
			}
			if id != uuid.Nil {
				parentID = &id
			}
		}

		// Parse request body
		var req MoveFoldersRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Warn("invalid request body", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if len(req.FolderIDs) == 0 {
			logger.Warn("no folder IDs provided")
			util.RespondWithError(w, http.StatusBadRequest, "No folder IDs provided")
			return
		}

		overwrite := r.URL.Query().Get("overwrite") == "true"
		logger.Info("moving folders", "folder_count", len(req.FolderIDs), "overwrite", overwrite)

		if err := s.MoveFolders(r.Context(), req.FolderIDs, ctxUserID, parentID, overwrite); err != nil {
			logger.Error("failed to move folders", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logger.Info("folders moved successfully")
		util.RespondWithJSON(w, http.StatusOK, "Folders moved successfully")
	}
}

func RenameFolderHandler(s FolderServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("starting folder rename")

		// Extract user ID
		ctxUserID, _ := middleware.GetUserID(r.Context())
		pathUserID := r.PathValue("user_id")
		logger = logger.With("ctx_user_id", ctxUserID, "path_user_id", pathUserID)

		if strconv.Itoa(int(ctxUserID)) != pathUserID {
			logger.Warn("token user ID does not match path user ID")
			util.RespondWithError(w, http.StatusBadRequest, "Mismatch between token and path value: user id")
			return
		}

		// Parse folder ID
		folderIDStr := r.PathValue("folder_id")
		if folderIDStr == "" {
			logger.Warn("missing folder_id parameter")
			util.RespondWithError(w, http.StatusBadRequest, "Missing folder_id parameter")
			return
		}

		folderID, err := uuid.Parse(folderIDStr)
		if err != nil {
			logger.Warn("invalid folder ID format", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, "Invalid folder ID format")
			return
		}

		// Parse request body
		var req RenameFolderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Warn("invalid request body", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if err := validateFolderName(req.Name); err != nil {
			logger.Warn("invalid folder name", "name", req.Name)
			util.RespondWithError(w, http.StatusBadRequest, "Invalid folder name")
			return
		}

		overwrite := r.URL.Query().Get("overwrite") == "true"
		logger.Info("renaming folder", "folder_id", folderID, "new_name", req.Name, "overwrite", overwrite)

		if err := s.RenameFolder(r.Context(), folderID, req.Name, ctxUserID, overwrite); err != nil {
			logger.Error("failed to rename folder", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logger.Info("folder renamed successfully", "folder_id", folderID)
		util.RespondWithJSON(w, http.StatusOK, "Folder renamed successfully")
	}
}
