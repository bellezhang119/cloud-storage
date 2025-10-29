package handlers

import (
	"archive/zip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/middleware"
	"github.com/bellezhang119/cloud-storage/internal/storage/services"
	"github.com/bellezhang119/cloud-storage/internal/util"
	"github.com/google/uuid"
)

type FileServiceInterface interface {
	GetFileByID(ctx context.Context, id uuid.UUID, userID int32) (database.File, error)
	GetFileByNameInFolder(ctx context.Context, folderID *uuid.UUID, userID int32, name string) (database.File, error)
	ListFilesInFolder(ctx context.Context, folderID *uuid.UUID, userID int32) ([]database.File, error)
	UploadFile(
		ctx context.Context,
		folderID *uuid.UUID,
		userID int32,
		name string,
		sizeBytes int64,
		mimeType string,
		content io.Reader,
		overwrite bool,
	) (database.File, error)
	DownloadFiles(ctx context.Context, fileIDs []uuid.UUID, userID int32) ([]services.FileDownload, error)
	DeleteFiles(ctx context.Context, filesIDs []uuid.UUID, userID int32) error
	MoveFiles(ctx context.Context, fileIDs []uuid.UUID, userID int32, destFolderID *uuid.UUID, overwrite bool) error
	RenameFile(ctx context.Context, fileID uuid.UUID, userID int32, newName string, overwrite bool) error
}

type DownloadFilesRequest struct {
	FileIDs []uuid.UUID `json:"file_ids"`
}

type DeleteFilesRequest struct {
	FileIDs []uuid.UUID `json:"file_ids"`
}

type MoveFilesRequest struct {
	FileIDs   []uuid.UUID `json:"file_ids"`
	Overwrite bool        `json:"overwrite"`
}

type RenameFileRequest struct {
	Name      string `json:"name"`
	Overwrite bool   `json:"overwrite"`
}

func GetFileByIDHandler(s FileServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("getting file by ID")

		userID, err := util.GetUserIDFromPathAndCheck(r)
		if err != nil {
			logger.Error("invalid user ID", "error", err)
			util.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		logger = logger.With("ctx_user_id", userID)

		fileID, err := util.GetFileIDFromPath(r)
		if err != nil {
			logger.Error("invalid file ID", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		logger = logger.With("file_id", fileID)

		logger.Info("retrieving file by ID")

		file, err := s.GetFileByID(r.Context(), fileID, userID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				logger.Warn("file not found", "file_id", fileID)
				util.RespondWithError(w, http.StatusNotFound, "file not found")
				return
			}
			logger.Error("failed to get file by ID", "file_id", fileID, "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logger.Info("file retrieved successfully", "file_id", fileID)
		util.RespondWithJSON(w, http.StatusOK, file)
	}
}

func GetFileByNameInFolderHandler(s FileServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("retrieving file by name")

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

		name := r.URL.Query().Get("name")
		if name == "" {
			logger.Warn("file name is missing in query")
			util.RespondWithError(w, http.StatusBadRequest, "File name is required")
			return
		}

		file, err := s.GetFileByNameInFolder(r.Context(), folderID, userID, name)
		if err != nil {
			logger.Error("failed to get file by name in folder", "folder_id", folderID, "name", name, "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logger.Info("file retrieved successfully", "folder_id", folderID, "name", name)
		util.RespondWithJSON(w, http.StatusOK, file)
	}
}

func ListFilesInFolderHandler(s FileServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("listing files in folder")

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

		files, err := s.ListFilesInFolder(r.Context(), folderID, userID)
		if err != nil {
			logger.Error("failed to list files in folder", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logger.Info("files listed successfully", "count", len(files))
		util.RespondWithJSON(w, http.StatusOK, files)
	}
}

func UploadFileHandler(s FileServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("uploading file")

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

		// Parse multipart form
		if err := r.ParseMultipartForm(100 << 20); err != nil {
			logger.Error("failed to parse multipart form", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, "Failed to parse multipart form")
			return
		}

		// Extract file
		file, fileHeader, err := r.FormFile("file")
		if err != nil {
			logger.Warn("file not provided in upload", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, "File not provided")
			return
		}
		defer file.Close()

		overwrite := r.FormValue("overwrite") == "true"
		name := fileHeader.Filename
		if name == "" {
			logger.Warn("file name missing in upload")
			util.RespondWithError(w, http.StatusBadRequest, "File name missing in upload")
			return
		}

		mimeType := fileHeader.Header.Get("Content-Type")
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		logger = logger.With("file_name", name, "file_size", fileHeader.Size, "mime_type", mimeType, "overwrite", overwrite)
		logger.Info("starting file upload")

		// Call service
		fileMeta, err := s.UploadFile(r.Context(), parentID, userID, name, fileHeader.Size, mimeType, file, overwrite)
		if err != nil {
			logger.Error("failed to upload file", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logger.Info("file uploaded successfully", "file_id", fileMeta.ID)
		util.RespondWithJSON(w, http.StatusCreated, fileMeta)
	}
}

func DownloadFilesHandler(s FileServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("downloading files")

		// 1. Extract user ID from context
		userID, err := util.GetUserIDFromPathAndCheck(r)
		if err != nil {
			logger.Error("invalid user ID", "error", err)
			util.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		logger = logger.With("ctx_user_id", userID)

		// 2. Parse JSON body
		var req DownloadFilesRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Warn("invalid request body", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		if len(req.FileIDs) == 0 {
			logger.Warn("no file IDs provided")
			util.RespondWithError(w, http.StatusBadRequest, "No file IDs provided")
			return
		}

		logger = logger.With("file_ids", req.FileIDs)
		logger.Info("starting file download")

		// 3. Call service
		downloads, err := s.DownloadFiles(r.Context(), req.FileIDs, userID)
		if err != nil {
			logger.Error("failed to download files", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// 4. Single file
		if len(downloads) == 1 {
			file := downloads[0]
			defer file.Content.Close()

			mimeType := "application/octet-stream"
			if file.File.MimeType.Valid {
				mimeType = file.File.MimeType.String
			}

			logger.Info("streaming single file", "file_name", file.File.Name, "size", file.File.SizeBytes, "mime_type", mimeType)

			w.Header().Set("Content-Type", mimeType)
			w.Header().Set("Content-Length", fmt.Sprintf("%d", file.File.SizeBytes))
			w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.File.Name))

			if _, err := io.Copy(w, file.Content); err != nil {
				logger.Error("error streaming file", "file_name", file.File.Name, "error", err)
				util.RespondWithError(w, http.StatusInternalServerError, "Error streaming file")
				return
			}
			return
		}

		// 5. Multiple files - stream as ZIP
		logger.Info("streaming multiple files as ZIP", "file_count", len(downloads))

		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", "attachment; filename=files.zip")

		zipWriter := zip.NewWriter(w)
		defer zipWriter.Close()

		for _, f := range downloads {
			defer f.Content.Close()
			fw, err := zipWriter.Create(f.File.Name)
			if err != nil {
				logger.Error("error creating zip entry", "file_name", f.File.Name, "error", err)
				continue
			}
			if _, err := io.Copy(fw, f.Content); err != nil {
				logger.Error("error writing file to zip", "file_name", f.File.Name, "error", err)
			} else {
				logger.Info("added file to zip", "file_name", f.File.Name, "size", f.File.SizeBytes)
			}
		}
	}
}

func DeleteFilesHandler(s FileServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())

		// Extract user ID from context
		userID, err := util.GetUserIDFromPathAndCheck(r)
		if err != nil {
			logger.Error("invalid user ID", "error", err)
			util.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		logger = logger.With("ctx_user_id", userID)

		// Parse request body
		var req DeleteFilesRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Warn("invalid request body", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		if len(req.FileIDs) == 0 {
			logger.Warn("no file IDs provided")
			util.RespondWithError(w, http.StatusBadRequest, "No file IDs provided")
			return
		}

		logger = logger.With("file_ids", req.FileIDs)
		logger.Info("deleting files")

		// Call service
		err = s.DeleteFiles(r.Context(), req.FileIDs, userID)
		if err != nil {
			logger.Error("failed to delete files", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logger.Info("files successfully deleted")
		util.RespondWithJSON(w, http.StatusOK, "Files deleted successfully")
	}
}

func MoveFilesHandler(s FileServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())

		// Extract user ID from context
		userID, err := util.GetUserIDFromPathAndCheck(r)
		if err != nil {
			logger.Error("invalid user ID", "error", err)
			util.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		logger = logger.With("ctx_user_id", userID)

		// Parse folder ID from path
		folderID, err := util.GetFolderIDFromPath(r)
		if err != nil {
			logger.Error("invalid folder ID", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		logger = logger.With("target folder_id", folderID)

		// Parse request body
		var req MoveFilesRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Warn("invalid request body", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		if len(req.FileIDs) == 0 {
			logger.Warn("no file IDs provided")
			util.RespondWithError(w, http.StatusBadRequest, "No file IDs provided")
			return
		}
		logger = logger.With("file_ids", req.FileIDs, "overwrite", req.Overwrite)
		logger.Info("moving files")

		// Call service
		if err := s.MoveFiles(r.Context(), req.FileIDs, userID, folderID, req.Overwrite); err != nil {
			logger.Error("failed to move files", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logger.Info("files successfully moved")
		util.RespondWithJSON(w, http.StatusOK, "Files moved")
	}
}

func RenameFileHandler(s FileServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())

		// 1. Extract user ID from context and path
		userID, err := util.GetUserIDFromPathAndCheck(r)
		if err != nil {
			logger.Error("invalid user ID", "error", err)
			util.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		logger = logger.With("ctx_user_id", userID)

		// 2. Get file ID from path
		fileID, err := util.GetFileIDFromPath(r)
		if err != nil {
			logger.Error("invalid file ID", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		logger = logger.With("file_id", fileID)

		// 3. Parse request body
		var req RenameFileRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Warn("invalid request body", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		name := strings.TrimSpace(req.Name)
		if name == "" {
			logger.Warn("new file name is empty")
			util.RespondWithError(w, http.StatusBadRequest, "New file name cannot be empty")
			return
		}

		if strings.ContainsAny(name, "/\\") {
			logger.Warn("file name contains invalid path separators", "name", name)
			util.RespondWithError(w, http.StatusBadRequest, "File name cannot contain path separators")
			return
		}

		validName := regexp.MustCompile(`^[\w\-. ]+$`)
		if !validName.MatchString(name) {
			logger.Warn("file name contains invalid characters", "name", name)
			util.RespondWithError(w, http.StatusBadRequest, "File name contains invalid characters")
			return
		}

		if filepath.Ext(name) == "" {
			logger.Warn("file name missing extension", "name", name)
			util.RespondWithError(w, http.StatusBadRequest, "File must have an extension")
			return
		}

		logger = logger.With("new_name", name, "overwrite", req.Overwrite)
		logger.Info("renaming file")

		// 4. Call service to rename
		if err := s.RenameFile(r.Context(), fileID, userID, name, req.Overwrite); err != nil {
			logger.Error("failed to rename file", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to rename file: %v", err))
			return
		}

		logger.Info("file renamed successfully")
		util.RespondWithJSON(w, http.StatusOK, map[string]string{
			"message":  "File renamed successfully",
			"new_name": name,
		})
	}
}
