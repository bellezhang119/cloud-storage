package file

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/util"
	"github.com/google/uuid"
)

type ServiceInterface interface {
	SaveFile(
		ctx context.Context,
		folderID *uuid.UUID,
		userID int32,
		name string,
		sizeBytes int64,
		mimeType string,
		content io.Reader,
	) (database.File, error)
	GetFileByID(ctx context.Context, id uuid.UUID) (database.File, error)
	GetFileByNameInFolder(ctx context.Context, folderID uuid.UUID, name string) (database.File, error)
	ListFilesInFolder(ctx context.Context, folderID *uuid.UUID, userID int32) ([]database.File, error)
	ListFilesRecursive(ctx context.Context, folderID uuid.UUID, userID int32)
	GetFileForDownload(ctx context.Context, fileID uuid.UUID, userID int32) (database.File, io.ReadCloser, error)
	DeleteFile(ctx context.Context, fileID uuid.UUID, userID int32) error
	UpdateFileMetadata(
		ctx context.Context,
		fileID uuid.UUID,
		name string,
		userID int32,
	) error
	UpdateFilePath(ctx context.Context, fileID uuid.UUID, path string, userID int32) error
	MoveFile(
		ctx context.Context,
		fileID uuid.UUID,
		oldPath, newPath string,
		userID int32,
	) error
	RenameFile(ctx context.Context, file database.File, newName string, userID int32) error
}

// UploadFileHandler handles uploading a file
func UploadFileHandler(service ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := strconv.Atoi(r.Header.Get("X-User-ID"))
		if err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid user ID")
			return
		}

		folderIDStr := r.URL.Query().Get("folder_id")
		var folderID *uuid.UUID
		if folderIDStr != "" {
			id, err := uuid.Parse(folderIDStr)
			if err != nil {
				util.RespondWithError(w, http.StatusBadRequest, "Invalid folder ID")
				return
			}
			folderID = &id
		}

		name := r.URL.Query().Get("name")
		if name == "" {
			util.RespondWithError(w, http.StatusBadRequest, "File name is required")
			return
		}

		mimeType := r.Header.Get("Content-Type")
		fileMeta, err := service.SaveFile(r.Context(), folderID, int32(userID), name, r.ContentLength, mimeType, r.Body)
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		util.RespondWithJSON(w, http.StatusCreated, fileMeta)
	}
}

// DownloadFileHandler handles downloading a file
func DownloadFileHandler(service ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := strconv.Atoi(r.Header.Get("X-User-ID"))
		if err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid user ID")
			return
		}

		fileIDStr := r.URL.Query().Get("file_id")
		fileID, err := uuid.Parse(fileIDStr)
		if err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid file ID")
			return
		}

		fileMeta, reader, err := service.GetFileForDownload(r.Context(), fileID, int32(userID))
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer reader.Close()

		w.Header().Set("Content-Disposition", `attachment; filename="`+fileMeta.Name+`"`)
		w.Header().Set("Content-Type", fileMeta.MimeType.String)
		w.Header().Set("Content-Length", strconv.FormatInt(fileMeta.SizeBytes, 10))
		w.WriteHeader(http.StatusOK)

		if _, err := io.Copy(w, reader); err != nil {
			// Log streaming error
			fmt.Printf("Error streaming file: %v\n", err)
		}
	}
}

// DeleteFileHandler handles deleting a file
func DeleteFileHandler(service ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := strconv.Atoi(r.Header.Get("X-User-ID"))
		if err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid user ID")
			return
		}

		fileIDStr := r.URL.Query().Get("file_id")
		fileID, err := uuid.Parse(fileIDStr)
		if err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid file ID")
			return
		}

		if err := service.DeleteFile(r.Context(), fileID, int32(userID)); err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		util.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "File deleted successfully"})
	}
}

// RenameFileHandler handles renaming a file
func RenameFileHandler(service ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := strconv.Atoi(r.Header.Get("X-User-ID"))
		if err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid user ID")
			return
		}

		fileIDStr := r.URL.Query().Get("file_id")
		fileID, err := uuid.Parse(fileIDStr)
		if err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid file ID")
			return
		}

		var req struct {
			NewName string `json:"new_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		if req.NewName == "" {
			util.RespondWithError(w, http.StatusBadRequest, "New file name is required")
			return
		}

		// Fetch file first
		fileMeta, err := service.GetFileByID(r.Context(), fileID)
		if err != nil {
			util.RespondWithError(w, http.StatusNotFound, err.Error())
			return
		}

		// Rename file
		if err := service.RenameFile(r.Context(), fileMeta, req.NewName, int32(userID)); err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		util.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "File renamed successfully"})
	}
}

// ListFilesInFolderHandler handles listing files in a folder
func ListFilesInFolderHandler(service ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := strconv.Atoi(r.Header.Get("X-User-ID"))
		if err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid user ID")
			return
		}

		folderIDStr := r.URL.Query().Get("folder_id")
		var folderID *uuid.UUID
		if folderIDStr != "" {
			id, err := uuid.Parse(folderIDStr)
			if err != nil {
				util.RespondWithError(w, http.StatusBadRequest, "Invalid folder ID")
				return
			}
			folderID = &id
		}

		files, err := service.ListFilesInFolder(r.Context(), folderID, int32(userID))
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		util.RespondWithJSON(w, http.StatusOK, files)
	}
}
