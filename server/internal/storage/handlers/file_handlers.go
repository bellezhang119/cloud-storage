package handlers

import (
	"archive/zip"
	"context"
	"encoding/json"
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

func GetFileByIDHandler(service FileServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctxUserID, _ := middleware.GetUserID(r.Context())
		pathUserID := r.PathValue("user_id")

		if string(ctxUserID) != pathUserID {
			util.RespondWithError(w, http.StatusBadRequest, "Mismatch between token and path value: user id")
			return
		}

		fileIDStr := r.PathValue("file_id")
		if fileIDStr == "" {
			util.RespondWithError(w, http.StatusBadRequest, "Missing file_id parameter")
			return
		}

		fileID, err := uuid.Parse(fileIDStr)
		if err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid file ID format")
			return
		}

		file, err := service.GetFileByID(r.Context(), fileID, ctxUserID)
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		util.RespondWithJSON(w, http.StatusOK, file)
	}
}

func GetFileByNameInFolderHandler(service FileServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctxUserID, _ := middleware.GetUserID(r.Context())
		pathUserID := r.PathValue("user_id")

		if string(ctxUserID) != pathUserID {
			util.RespondWithError(w, http.StatusBadRequest, "Mismatch between token and path value: user id")
			return
		}

		folderID, err := util.ParseOptionalUUID(r.URL.Query().Get("folder_id"))
		if err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid folder ID")
			return
		}

		name := r.URL.Query().Get("name")
		if name == "" {
			util.RespondWithError(w, http.StatusBadRequest, "File name is required")
			return
		}

		file, err := service.GetFileByNameInFolder(r.Context(), folderID, ctxUserID, name)
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		util.RespondWithJSON(w, http.StatusOK, file)
	}
}

// ListFilesInFolderHandler make 2 routes for root folder and normal folder
func ListFilesInFolderHandler(service FileServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctxUserID, _ := middleware.GetUserID(r.Context())
		pathUserID := r.PathValue("user_id")

		if string(ctxUserID) != pathUserID {
			util.RespondWithError(w, http.StatusBadRequest, "Mismatch between token and path value: user id")
		}

		// If client is using route where no path value of folder_id is defined, folderIDStr = "",
		// and folderID = nil
		folderIDStr := r.PathValue("folder_id")

		var folderID *uuid.UUID
		if folderIDStr != "" {
			id, err := uuid.Parse(folderIDStr)
			if err != nil {
				util.RespondWithError(w, http.StatusBadRequest, "Invalid folder ID format")
				return
			}

			if id != uuid.Nil {
				folderID = &id
			}
		}

		folders, err := service.ListFilesInFolder(r.Context(), folderID, ctxUserID)
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		util.RespondWithJSON(w, http.StatusOK, folders)

	}
}

func UploadFileHandler(service FileServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract user ID from context
		ctxUserID, _ := middleware.GetUserID(r.Context())
		pathUserID := r.PathValue("user_id")

		if string(ctxUserID) != pathUserID {
			util.RespondWithError(w, http.StatusBadRequest, "Mismatch between token and path value: user id")
			return
		}

		// If client is using route where no path value of parent_id is defined, parentIDStr = "",
		// and parentID = nil
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

		// Parse multipart form (limit set to 100MB here)
		err := r.ParseMultipartForm(100 << 20)
		if err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Failed to parse multipart form")
			return
		}

		// Extract file from form data
		file, fileHeader, err := r.FormFile("file")
		if err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "File not provided")
			return
		}
		defer file.Close()

		// Get optional overwrite flag (from form or query param)
		overwrite := r.FormValue("overwrite") == "true"

		// Determine MIME type (fallback to binary if missing)
		mimeType := fileHeader.Header.Get("Content-Type")
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		// Get filename
		name := fileHeader.Filename
		if name == "" {
			util.RespondWithError(w, http.StatusBadRequest, "File name missing in upload")
			return
		}

		// Call service to handle the actual upload
		fileMeta, err := service.UploadFile(
			r.Context(),
			parentID,
			ctxUserID,
			name,
			fileHeader.Size,
			mimeType,
			file,
			overwrite,
		)
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		util.RespondWithJSON(w, http.StatusCreated, fileMeta)
	}
}

func DownloadFilesHandler(service FileServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Extract user ID from context
		ctxUserID, _ := middleware.GetUserID(r.Context())
		pathUserID := r.PathValue("user_id")

		if string(ctxUserID) != pathUserID {
			util.RespondWithError(w, http.StatusBadRequest, "Mismatch between token and path value: user id")
			return
		}

		// 2. Parse JSON body
		var req DownloadFilesRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		if len(req.FileIDs) == 0 {
			util.RespondWithError(w, http.StatusBadRequest, "No file IDs provided")
			return
		}

		// 3. Call service
		downloads, err := service.DownloadFiles(r.Context(), req.FileIDs, ctxUserID)
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// 4. If only one file - download directly
		if len(downloads) == 1 {
			file := downloads[0]
			defer file.Content.Close()

			w.Header().Set("Content-Type", func() string {
				if file.File.MimeType.Valid {
					return file.File.MimeType.String
				}
				return "application/octet-stream"
			}())

			w.Header().Set("Content-Length", fmt.Sprintf("%d", file.File.SizeBytes))
			w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.File.Name))
			if _, err := io.Copy(w, file.Content); err != nil {
				util.RespondWithError(w, http.StatusInternalServerError, "Error streaming file")
				return
			}
			return
		}

		// 5. If multiple files - stream as ZIP archive
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", "attachment; filename=files.zip")

		zipWriter := zip.NewWriter(w)
		defer zipWriter.Close()

		for _, f := range downloads {
			defer f.Content.Close()
			fw, err := zipWriter.Create(f.File.Name)
			if err != nil {
				fmt.Printf("Error creating zip entry for %s: %v\n", f.File.Name, err)
				continue
			}
			if _, err := io.Copy(fw, f.Content); err != nil {
				fmt.Printf("Error writing file %s to zip: %v\n", f.File.Name, err)
			}
		}
	}
}

func DeleteFilesHandler(service FileServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctxUserID, _ := middleware.GetUserID(r.Context())
		pathUserID := r.PathValue("user_id")

		if string(ctxUserID) != pathUserID {
			util.RespondWithError(w, http.StatusBadRequest, "Mismatch between token and path value: user id")
			return
		}

		var req DeleteFilesRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		if len(req.FileIDs) == 0 {
			util.RespondWithError(w, http.StatusBadRequest, "No file IDs provided")
			return
		}

		err := service.DeleteFiles(r.Context(), req.FileIDs, ctxUserID)

		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		util.RespondWithJSON(w, http.StatusOK, "Files deleted")
	}
}

func MoveFilesHandler(service FileServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract user ID from context
		ctxUserID, _ := middleware.GetUserID(r.Context())

		pathUserID := r.PathValue("user_id")
		if string(ctxUserID) != pathUserID {
			util.RespondWithError(w, http.StatusBadRequest, "Mismatch between token and path value: user id")
			return
		}

		folderIDStr := r.PathValue("folder_id")

		var folderID *uuid.UUID
		if folderIDStr != "" {
			id, err := uuid.Parse(folderIDStr)
			if err != nil {
				util.RespondWithError(w, http.StatusBadRequest, "Invalid folder ID format")
				return
			}

			if id != uuid.Nil {
				folderID = &id
			}
		}

		// Parse request
		var req MoveFilesRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		if len(req.FileIDs) == 0 {
			util.RespondWithError(w, http.StatusBadRequest, "No file IDs provided")
			return
		}

		// Call service
		if err := service.MoveFiles(r.Context(), req.FileIDs, ctxUserID, folderID, req.Overwrite); err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		util.RespondWithJSON(w, http.StatusOK, "Files moved")
	}
}

func RenameFileHandler(service FileServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Extract user ID from context and path
		ctxUserID, _ := middleware.GetUserID(r.Context())
		pathUserID := r.PathValue("user_id")

		if string(ctxUserID) != pathUserID {
			util.RespondWithError(w, http.StatusBadRequest, "Mismatch between token and path value: user id")
			return
		}

		// 2. Get file ID from path (assuming /api/user/{user_id}/file/{file_id}/rename)
		fileIDStr := r.PathValue("file_id")
		fileID, err := uuid.Parse(fileIDStr)
		if err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid file ID")
			return
		}

		// 3. Parse request body
		var req RenameFileRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		name := strings.TrimSpace(req.Name)
		if name == "" {
			util.RespondWithError(w, http.StatusBadRequest, "New file name cannot be empty")
			return
		}

		// 4. Check for invalid characters
		if strings.ContainsAny(name, "/\\") {
			util.RespondWithError(w, http.StatusBadRequest, "File name cannot contain path separators")
			return
		}

		validName := regexp.MustCompile(`^[\w\-. ]+$`)
		if !validName.MatchString(name) {
			util.RespondWithError(w, http.StatusBadRequest, "File name contains invalid characters")
			return
		}

		// 5. ensure the file has an extension
		if filepath.Ext(name) == "" {
			util.RespondWithError(w, http.StatusBadRequest, "File must have an extension")
			return
		}

		// 6. Call service to rename
		if err := service.RenameFile(r.Context(), fileID, ctxUserID, name, req.Overwrite); err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to rename file: %v", err))
			return
		}

		util.RespondWithJSON(w, http.StatusOK, map[string]string{
			"message":  "File renamed successfully",
			"new_name": name,
		})
	}
}
