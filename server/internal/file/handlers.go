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

			fmt.Printf("✅ Finished reading upload stream: %d bytes from user %d\n", total, userID)
		}()

		// Now pass the pipe reader to the service
		fileMeta, err := service.SaveFile(
			r.Context(),
			folderID,
			int32(userID),
			name,
			r.ContentLength,
			mimeType,
			pr, // stream from our pipe
		)
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		util.RespondWithJSON(w, http.StatusCreated, fileMeta)
	}
}

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

		// Set headers before writing body
		w.Header().Set("Content-Disposition", `attachment; filename="`+fileMeta.Name+`"`)
		w.Header().Set("Content-Type", fileMeta.MimeType.String)
		w.Header().Set("Content-Length", strconv.FormatInt(fileMeta.SizeBytes, 10))
		w.WriteHeader(http.StatusOK)

		// Custom chunk size — can adjust as needed
		const chunkSize = 64 * 1024 // 64 KB
		buffer := make([]byte, chunkSize)

		var totalSent int64
		for {
			n, readErr := reader.Read(buffer)
			if n > 0 {
				// Handle partial writes properly
				written, writeErr := w.Write(buffer[:n])
				if writeErr != nil {
					fmt.Printf("Write error: %v\n", writeErr)
					break
				}
				totalSent += int64(written)

				// Optional: flush to client (important for large files or slow clients)
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
			}

			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				fmt.Printf("Read error: %v\n", readErr)
				break
			}
		}

		fmt.Printf("✅ Successfully streamed %d bytes for user %d\n", totalSent, userID)
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
