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
	GetFileByID(ctx context.Context, id uuid.UUID) (database.File, error)
	GetFileByNameInFolder(ctx context.Context, folderID *uuid.UUID, name string) (database.File, error)
	ListFilesInFolder(ctx context.Context, folderID *uuid.UUID) ([]database.File, error)
	ListFilesRecursive(ctx context.Context, folderID uuid.UUID) ([]database.ListFilesRecursiveRow, error)
	UploadFile(
		ctx context.Context,
		folderID *uuid.UUID,
		name string,
		sizeBytes int64,
		mimeType string,
		content io.Reader,
		overwrite bool,
	) (database.File, error)
	DownloadFiles(ctx context.Context, fileIDs []uuid.UUID) ([]services.FileDownload, error)
	DeleteFiles(ctx context.Context, filesIDs []uuid.UUID) error
	MoveFiles(ctx context.Context, fileIDs []uuid.UUID, destFolderID *uuid.UUID, overwrite bool) error
	RenameFile(ctx context.Context, fileID uuid.UUID, newName string, overwrite bool) error
}

type DownloadFilesRequest struct {
	FileIDs []uuid.UUID `json:"file_ids"`
}

type DeleteFilesRequest struct {
	FileIDs []uuid.UUID `json:"file_ids"`
}

type MoveFilesRequest struct {
	FileIDs   []uuid.UUID `json:"file_ids"`
	FolderID  uuid.UUID   `json:"folder_id"`
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

		file, err := service.GetFileByID(r.Context(), fileID)
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

		folderID, err := util.ParseOptionalUUID((r.URL.Query().Get("folder_id")))
		if err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid folder ID")
			return
		}

		name := r.URL.Query().Get("name")
		if name == "" {
			util.RespondWithError(w, http.StatusBadRequest, "File name is required")
			return
		}

		file, err := service.GetFileByNameInFolder(r.Context(), folderID, name)
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		util.RespondWithJSON(w, http.StatusOK, file)
	}
}

func UploadFileHandler(service FileServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctxUserID, _ := middleware.GetUserID(r.Context())
		pathUserID := r.PathValue("user_id")

		if string(ctxUserID) != pathUserID {
			util.RespondWithError(w, http.StatusBadRequest, "Mismatch between token and path value: user id")
			return
		}

		folderID, err := util.ParseOptionalUUID((r.URL.Query().Get("folder_id")))
		if err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid folder ID")
			return
		}

		name := r.URL.Query().Get("name")
		if name == "" {
			util.RespondWithError(w, http.StatusBadRequest, "File name is required")
			return
		}

		mimeType := r.Header.Get("Content-Type")
		overwrite := r.URL.Query().Get("overwrite") == "true"

		// Create a pipe to stream chunks into service.SaveFile
		pr, pw := io.Pipe()

		// Launch a goroutine to read from the request in chunks and write to the pipe
		go func() {
			defer pw.Close()

			const chunkSize = 64 * 1024 // 64 KB
			buf := make([]byte, chunkSize)
			var total int64

			for {
				n, err := r.Body.Read(buf)
				if n > 0 {
					total += int64(n)
					if _, writeErr := pw.Write(buf[:n]); writeErr != nil {
						fmt.Printf("Upload pipe write error: %v\n", writeErr)
						return
					}
				}
				if err == io.EOF {
					break
				}
				if err != nil {
					fmt.Printf("Upload read error: %v\n", err)
					return
				}
			}

			fmt.Printf("Finished reading upload stream: %d bytes from user %d\n", total, ctxUserID)
		}()

		// Now pass the pipe reader to the service
		fileMeta, err := service.UploadFile(
			r.Context(),
			folderID,
			name,
			r.ContentLength,
			mimeType,
			pr,
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
		downloads, err := service.DownloadFiles(r.Context(), req.FileIDs)
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

		err := service.DeleteFiles(r.Context(), req.FileIDs)

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
		var destFolderID *uuid.UUID
		if req.FolderID != uuid.Nil {
			destFolderID = &req.FolderID
		}

		if err := service.MoveFiles(r.Context(), req.FileIDs, destFolderID, req.Overwrite); err != nil {
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
		if err := service.RenameFile(r.Context(), fileID, name, req.Overwrite); err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to rename file: %v", err))
			return
		}

		util.RespondWithJSON(w, http.StatusOK, map[string]string{
			"message":  "File renamed successfully",
			"new_name": name,
		})
	}
}
