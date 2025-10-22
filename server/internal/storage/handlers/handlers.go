package storage

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/middleware"
	"github.com/bellezhang119/cloud-storage/internal/storage/services"
	"github.com/bellezhang119/cloud-storage/internal/util"
	"github.com/google/uuid"
)

type FileServiceInterface interface {
	GetFileByID(ctx context.Context, id uuid.UUID, userID int32) (database.File, error)
	GetFileByNameInFolder(ctx context.Context, folderID *uuid.UUID, name string) (database.File, error)
	ListFilesInFolder(ctx context.Context, userID int32, folderID *uuid.UUID) ([]database.File, error)
	ListFilesRecursive(ctx context.Context, userID int32, folderID uuid.UUID) ([]database.ListFilesRecursiveRow, error)
	UploadFile(
		ctx context.Context,
		userID int32,
		folderID *uuid.UUID,
		name string,
		sizeBytes int64,
		mimeType string,
		content io.Reader,
		overwrite bool,
	) (database.File, error)
	DownloadFiles(ctx context.Context, fileIDs []uuid.UUID, userID int32) ([]services.FileDownload, error)
	DeleteFiles(ctx context.Context, filesIDs []uuid.UUID, userID int32) error
	UpdateFileMetadata(
		ctx context.Context,
		fileID uuid.UUID,
		userID int32,
		sizeBytes int64,
		mimeType string,
	) error
	UpdateFileParentAndPath(ctx context.Context, fileID uuid.UUID, userID int32, folderID *uuid.UUID, filePath string) error
	UpdateFileNameAndPath(ctx context.Context, fileID uuid.UUID, userID int32, name string, filePath string) error
	MoveFiles(ctx context.Context, fileIDs []uuid.UUID, destFolderID *uuid.UUID, userID int32, overwrite bool) error
	RenameFile(ctx context.Context, file database.File, newName string, userID int32, overwrite bool) error
}

type DownloadRequest struct {
	FileIDs []uuid.UUID `json:"file_ids"`
}

// TODO: Require user id in path, also get user id from context and check if they are the same

func GetFileByIDHandler(service FileServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			util.RespondWithError(w, http.StatusUnauthorized, "Unauthorized user")
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

		file, err := service.GetFileByID(r.Context(), fileID, userID)
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		util.RespondWithJSON(w, http.StatusOK, file)
	}
}

func GetFileByNameInFolderHandler(service FileServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, ok := middleware.GetUserID(r.Context())
		if !ok {
			util.RespondWithError(w, http.StatusUnauthorized, "Unauthorized user")
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
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid user ID")
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

			fmt.Printf("Finished reading upload stream: %d bytes from user %d\n", total, userID)
		}()

		// Now pass the pipe reader to the service
		fileMeta, err := service.UploadFile(
			r.Context(),
			int32(userID),
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
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			util.RespondWithError(w, http.StatusUnauthorized, "Invalid user ID")
			return
		}

		// 2. Parse JSON body
		var req DownloadRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		if len(req.FileIDs) == 0 {
			util.RespondWithError(w, http.StatusBadRequest, "No file IDs provided")
			return
		}

		// 3. Call service
		downloads, err := service.DownloadFiles(r.Context(), req.FileIDs, userID)
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// 4. If only one file → download directly
		if len(downloads) == 1 {
			file := downloads[0]
			defer file.Content.Close()

			w.Header().Set("Content-Type", func() string {
				if file.File.MimeType.Valid {
					return file.File.MimeType.String
				}
				return "application/octet-stream"
			}())
			w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.File.Name))
			if _, err := io.Copy(w, file.Content); err != nil {
				util.RespondWithError(w, http.StatusInternalServerError, "Error streaming file")
				return
			}
			return
		}

		// 5. If multiple files → stream as ZIP archive
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
