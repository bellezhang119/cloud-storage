package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/middleware"
	"github.com/bellezhang119/cloud-storage/internal/storage/local"
	"github.com/bellezhang119/cloud-storage/internal/util"
	"github.com/google/uuid"
)

type FileQueries interface {
	CreateFile(ctx context.Context, arg database.CreateFileParams) (database.File, error)
	GetFileByID(ctx context.Context, arg database.GetFileByIDParams) (database.File, error)
	GetFileByNameInFolder(ctx context.Context, arg database.GetFileByNameInFolderParams) (database.File, error)
	ListFilesInFolder(ctx context.Context, arg database.ListFilesInFolderParams) ([]database.File, error)
	DeleteFiles(ctx context.Context, arg database.DeleteFilesParams) (int64, error)
	ListFilesRecursive(ctx context.Context, arg database.ListFilesRecursiveParams) ([]database.ListFilesRecursiveRow, error)
	UpdateFileMetadata(ctx context.Context, arg database.UpdateFileMetadataParams) (int64, error)
	UpdateFileNameAndPath(ctx context.Context, arg database.UpdateFileNameAndPathParams) (int64, error)
	UpdateFileParentAndPath(ctx context.Context, arg database.UpdateFileParentAndPathParams) (int64, error)
}

type FolderService interface {
	CreateFolder(ctx context.Context, userID int32, name string, parentID *uuid.UUID) (database.Folder, error)
	GetFolderByID(ctx context.Context, folderID uuid.UUID, userID int32) (database.Folder, error)
	ListFoldersByParent(ctx context.Context, userID int32, parentID *uuid.UUID) ([]database.Folder, error)
	GetFolderFullPath(ctx context.Context, folderID uuid.UUID, userID int32) (string, error)
}

type UserService interface {
	AdjustUsedStorage(ctx context.Context, userID int32, delta int64) error
}

type FileServiceImpl struct {
	queries FileQueries
	folders FolderService
	users   UserService
	local   local.Storage
}

type FileDownload struct {
	File    database.File
	Content io.ReadCloser
}

func NewFileService(q FileQueries, us UserService, local local.Storage) *FileServiceImpl {
	return &FileServiceImpl{queries: q, users: us, local: local}
}

func (s *FileServiceImpl) SetFolderService(f FolderService) {
	s.folders = f
}

func (s *FileServiceImpl) GetFileByID(ctx context.Context, id uuid.UUID, userID int32) (database.File, error) {
	return s.queries.GetFileByID(ctx, database.GetFileByIDParams{
		ID:     id,
		UserID: util.ToNullInt32(&userID),
	})
}

func (s *FileServiceImpl) GetFileByNameInFolder(ctx context.Context, folderID *uuid.UUID, userID int32, name string) (database.File, error) {
	file, err := s.queries.GetFileByNameInFolder(ctx, database.GetFileByNameInFolderParams{
		FolderID: util.ToNullUUID(folderID),
		UserID:   util.ToNullInt32(&userID),
		Name:     name,
	})
	if err != nil {
		return database.File{}, err
	}
	return file, nil
}

func (s *FileServiceImpl) ListFilesInFolder(ctx context.Context, folderID *uuid.UUID, userID int32) ([]database.File, error) {
	files, err := s.queries.ListFilesInFolder(ctx, database.ListFilesInFolderParams{
		UserID:   util.ToNullInt32(&userID),
		FolderID: util.ToNullUUID(folderID),
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

func (s *FileServiceImpl) ListFilesRecursive(ctx context.Context, folderID uuid.UUID, userID int32) ([]database.ListFilesRecursiveRow, error) {
	rows, err := s.queries.ListFilesRecursive(ctx, database.ListFilesRecursiveParams{
		UserID: util.ToNullInt32(&userID),
		ID:     folderID,
	})
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *FileServiceImpl) UploadFile(
	ctx context.Context,
	folderID *uuid.UUID,
	userID int32,
	name string,
	sizeBytes int64,
	mimeType string,
	content io.Reader,
	overwrite bool,
) (database.File, error) {
	logger := middleware.GetLogger(ctx)
	logger.Info("upload started",
		"name", name,
		"size_bytes", sizeBytes,
		"mime_type", mimeType,
		"overwrite", overwrite,
	)

	if name == "" {
		err := errors.New("file name is required")
		logger.Error("upload failed: missing file name", "error", err)
		return database.File{}, err
	}

	if strings.Contains(name, "..") || strings.Contains(name, "/") {
		err := errors.New("invalid file name")
		logger.Error("upload failed: invalid file name", "name", name, "error", err)
		return database.File{}, err
	}
	if sizeBytes <= 0 {
		err := errors.New("file size must be positive")
		logger.Error("upload failed: invalid size", "size_bytes", sizeBytes, "error", err)
		return database.File{}, err
	}

	// Build folder path
	var folderPath string
	if folderID != nil {
		var err error
		folderPath, err = s.folders.GetFolderFullPath(ctx, *folderID, userID)
		if err != nil {
			logger.Error("failed to build folder path", "folder_id", folderID, "error", err)
			return database.File{}, fmt.Errorf("building folder path: %w", err)
		}
	}

	filePath := name
	if folderPath != "" {
		filePath = filepath.Join(folderPath, name)
	}

	// Check if a file with the same name exists
	existingFile, err := s.GetFileByNameInFolder(ctx, folderID, userID, name)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		logger.Error("failed to check existing file", "file", name, "error", err)
		return database.File{}, fmt.Errorf("checking existing file: %w", err)
	}

	var sizeDelta int64 = sizeBytes
	if existingFile.ID != uuid.Nil {
		if overwrite {
			logger.Info("overwriting existing file", "existing_file_id", existingFile.ID)
			if err := s.DeleteFiles(ctx, []uuid.UUID{existingFile.ID}, userID); err != nil {
				logger.Error("failed to delete existing file for overwrite", "file_id", existingFile.ID, "error", err)
				return database.File{}, fmt.Errorf("deleting existing file for overwrite: %w", err)
			}
			// subtract size of deleted file
			sizeDelta -= existingFile.SizeBytes
		} else {
			err := fmt.Errorf("file '%s' already exists in folder", name)
			logger.Warn("upload aborted: file already exists", "name", name)
			return database.File{}, err
		}
	}

	// Update user's used storage
	if err := s.users.AdjustUsedStorage(ctx, userID, sizeDelta); err != nil {
		logger.Error("failed to update user's used storage", "user_id", userID, "delta", sizeDelta, "error", err)
		return database.File{}, fmt.Errorf("updating used storage: %w", err)
	}

	// Create new DB record
	mType := sql.NullString{String: mimeType, Valid: mimeType != ""}
	fileMeta, err := s.queries.CreateFile(ctx, database.CreateFileParams{
		FolderID:  util.ToNullUUID(folderID),
		UserID:    util.ToNullInt32(&userID),
		Name:      name,
		FilePath:  filePath,
		SizeBytes: sizeBytes,
		MimeType:  mType,
	})
	if err != nil {
		logger.Error("failed to create file record in database", "file", name, "error", err)
		return database.File{}, fmt.Errorf("creating file record: %w", err)
	}

	// Save content to storage
	if err := s.local.SaveFile(ctx, userID, filePath, content); err != nil {
		logger.Error("failed to save file content", "path", filePath, "error", err)
		_ = s.DeleteFiles(ctx, []uuid.UUID{fileMeta.ID}, userID) // rollback DB
		_ = s.users.AdjustUsedStorage(ctx, userID, -sizeBytes)   // rollback storage
		return database.File{}, fmt.Errorf("saving file: %w", err)
	}

	logger.Info("upload successful",
		"user_id", userID,
		"file_id", fileMeta.ID,
		"path", fileMeta.FilePath,
		"size_bytes", sizeBytes,
	)
	return fileMeta, nil
}

func (s *FileServiceImpl) DownloadFiles(ctx context.Context, fileIDs []uuid.UUID, userID int32) ([]FileDownload, error) {
	logger := middleware.GetLogger(ctx)
	logger.Info("download started",
		"file_count", len(fileIDs),
	)

	if len(fileIDs) == 0 {
		err := fmt.Errorf("no file IDs provided")
		logger.Error("download failed: no file IDs", "error", err)
		return nil, err
	}

	var downloads []FileDownload
	var openedFiles []io.ReadCloser

	defer func() {
		if len(openedFiles) > 0 {
			for _, f := range openedFiles {
				f.Close()
			}
			logger.Warn("partial download cleanup: closed remaining open files",
				"user_id", userID,
				"open_file_count", len(openedFiles),
			)
		}
	}()

	for _, fileID := range fileIDs {
		logger.Info("processing file for download", "file_id", fileID)

		// 1. Look up file in DB
		fileMeta, err := s.GetFileByID(ctx, fileID, userID)
		if err != nil {
			logger.Error("failed to fetch file metadata", "file_id", fileID, "error", err)
			return nil, fmt.Errorf("fetching file metadata for %s: %w", fileID, err)
		}

		// 2. Authorization check
		if fileMeta.UserID.Int32 != userID {
			err := fmt.Errorf("unauthorized access to file %s", fileID)
			logger.Error("unauthorized file access attempt", "file_id", fileID, "user_id", userID)
			return nil, err
		}

		// 3. Read file from storage
		content, err := s.local.ReadFile(ctx, userID, fileMeta.FilePath)
		if err != nil {
			logger.Error("failed to read file from storage", "file_id", fileID, "path", fileMeta.FilePath, "error", err)
			return nil, fmt.Errorf("reading file %s: %w", fileID, err)
		}

		openedFiles = append(openedFiles, content)

		downloads = append(downloads, FileDownload{
			File:    fileMeta,
			Content: content,
		})

		logger.Info("file ready for download", "file_id", fileID, "path", fileMeta.FilePath)
	}

	logger.Info("download successful",
		"user_id", userID,
		"total_files", len(downloads),
	)
	openedFiles = nil
	return downloads, nil
}

func (s *FileServiceImpl) DeleteFiles(ctx context.Context, fileIDs []uuid.UUID, userID int32) error {
	logger := middleware.GetLogger(ctx)
	logger.Info("delete operation started",
		"file_count", len(fileIDs),
	)

	if len(fileIDs) == 0 {
		err := fmt.Errorf("no file IDs provided")
		logger.Error("delete operation failed: empty file list", "error", err)
		return err
	}

	// 1. Fetch all files metadata first
	var files []database.File
	var totalSize int64 = 0
	for _, id := range fileIDs {
		logger.Debug("fetching file metadata", "file_id", id)
		file, err := s.GetFileByID(ctx, id, userID)
		if err != nil {
			logger.Error("failed to fetch file metadata", "file_id", id, "error", err)
			return fmt.Errorf("fetching file metadata for %s: %w", id, err)
		}
		if file.UserID.Int32 != userID {
			err := fmt.Errorf("unauthorized to delete file %s", id)
			logger.Warn("unauthorized delete attempt", "file_id", id, "user_id", userID)
			return err
		}
		files = append(files, file)
		totalSize += file.SizeBytes
	}

	// 2. Delete DB records
	logger.Info("deleting file records from database", "file_count", len(files))
	rows, err := s.queries.DeleteFiles(ctx, database.DeleteFilesParams{
		Column1: fileIDs,
		UserID:  util.ToNullInt32(&userID),
	})
	if err != nil {
		logger.Error("database deletion failed", "error", err)
		return fmt.Errorf("deleting file records: %w", err)
	}
	if rows == 0 {
		err := fmt.Errorf("no files were deleted (maybe already deleted)")
		logger.Warn("no matching files deleted from database", "user_id", userID)
		return err
	}
	logger.Info("file records deleted from database", "deleted_rows", rows)

	// 3. Delete files from storage
	var failed []string
	for _, file := range files {
		logger.Debug("deleting file from storage", "file_path", file.FilePath)
		if err := s.local.DeleteFile(ctx, userID, file.FilePath); err != nil {
			logger.Error("failed to delete file from storage", "file_path", file.FilePath, "error", err)
			failed = append(failed, fmt.Sprintf("%s: %v", file.Name, err))
		}
	}

	if len(failed) > 0 {
		err := fmt.Errorf("some files removed from DB but failed to delete from storage: %s", strings.Join(failed, "; "))
		logger.Warn("partial delete: some files failed to remove from storage", "user_id", userID, "failed_files", failed)
		totalDeleted := totalSize
		for _, f := range failed {
			for _, file := range files {
				if strings.HasPrefix(f, file.Name) {
					totalDeleted -= file.SizeBytes
				}
			}
		}
		if err := s.users.AdjustUsedStorage(ctx, userID, -totalDeleted); err != nil {
			logger.Error("failed to update used storage after partial delete", "user_id", userID, "error", err)
		}
		return err
	}

	// 4. Update used storage
	if err := s.users.AdjustUsedStorage(ctx, userID, -totalSize); err != nil {
		logger.Error("failed to update user's used storage after delete", "user_id", userID, "deleted_size", totalSize, "error", err)
		return fmt.Errorf("updating used storage: %w", err)
	}

	logger.Info("delete operation completed successfully",
		"user_id", userID,
		"deleted_files", len(files),
		"freed_bytes", totalSize,
	)
	return nil
}

func (s *FileServiceImpl) UpdateFileMetadata(
	ctx context.Context,
	fileID uuid.UUID,
	userID int32,
	sizeBytes int64,
	mimeType string,
) error {
	rows, err := s.queries.UpdateFileMetadata(ctx, database.UpdateFileMetadataParams{
		ID:        fileID,
		UserID:    util.ToNullInt32(&userID),
		SizeBytes: sizeBytes,
		MimeType:  sql.NullString{String: mimeType, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("updating file metadata: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("file not found or metadata not changed")
	}

	return nil
}

func (s *FileServiceImpl) UpdateFileParentAndPath(ctx context.Context, fileID uuid.UUID, userID int32, folderID *uuid.UUID, filePath string) error {
	rows, err := s.queries.UpdateFileParentAndPath(ctx, database.UpdateFileParentAndPathParams{
		ID:       fileID,
		UserID:   util.ToNullInt32(&userID),
		FolderID: util.ToNullUUID(folderID),
		FilePath: filePath,
	})

	if err != nil {
		return fmt.Errorf("updating file parent: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("file not found or parent folder not changed")
	}

	return nil
}

func (s *FileServiceImpl) UpdateFileNameAndPath(ctx context.Context, fileID uuid.UUID, userID int32, name string, filePath string) error {
	rows, err := s.queries.UpdateFileNameAndPath(ctx, database.UpdateFileNameAndPathParams{
		ID:       fileID,
		UserID:   util.ToNullInt32(&userID),
		Name:     name,
		FilePath: filePath,
	})

	if err != nil {
		return fmt.Errorf("updating file name: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("file not found or name not changed")
	}

	return nil
}

func (s *FileServiceImpl) MoveFiles(ctx context.Context, fileIDs []uuid.UUID, userID int32, destFolderID *uuid.UUID, overwrite bool) error {
	logger := middleware.GetLogger(ctx)

	logger.Info("starting file move", "destination", destFolderID, "file_count", len(fileIDs), "overwrite", overwrite)

	// Validate folder (if not root)
	var destFolderPath string
	if destFolderID != nil {
		var err error
		destFolderPath, err = s.folders.GetFolderFullPath(ctx, *destFolderID, userID)
		if err != nil {
			logger.Error("destination folder invalid", "error", err)
			return fmt.Errorf("destination folder invalid: %w", err)
		}
	}

	for _, fileID := range fileIDs {
		file, err := s.GetFileByID(ctx, fileID, userID)
		if err != nil {
			logger.Error("file not found", "file_id", fileID, "error", err)
			return fmt.Errorf("file %v not found: %w", fileID, err)
		}
		if file.UserID.Int32 != userID {
			logger.Warn("unauthorized file move attempt", "file_id", fileID)
			return fmt.Errorf("unauthorized to move file %v", fileID)
		}

		// Build new path
		newPath := file.Name
		if destFolderID != nil {
			newPath = filepath.Join(destFolderPath, file.Name)
		}

		// Check for conflict
		existingFile, err := s.GetFileByNameInFolder(ctx, destFolderID, userID, file.Name)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			logger.Error("error checking existing file", "file_id", fileID, "error", err)
			return fmt.Errorf("checking existing file %v: %w", fileID, err)
		}
		if existingFile.ID != uuid.Nil {
			if !overwrite {
				logger.Warn("file already exists in destination", "file_name", file.Name)
				return fmt.Errorf("file %s already exists in destination", file.Name)
			}
			logger.Info("overwriting existing file", "file_id", existingFile.ID)
			if err := s.DeleteFiles(ctx, []uuid.UUID{existingFile.ID}, userID); err != nil {
				logger.Error("failed to overwrite existing file", "file_id", existingFile.ID, "error", err)
				return fmt.Errorf("failed to overwrite existing file %v: %w", existingFile.ID, err)
			}
		}

		// Update DB
		if err := s.UpdateFileParentAndPath(ctx, file.ID, userID, destFolderID, newPath); err != nil {
			logger.Error("failed updating DB for file move", "file_id", fileID, "error", err)
			return fmt.Errorf("updating DB for file %v: %w", fileID, err)
		}

		// Move storage
		if err := s.local.MoveFile(ctx, userID, file.FilePath, newPath); err != nil {
			// rollback DB
			var oldFolderID *uuid.UUID
			if file.FolderID.Valid {
				oldFolderID = &file.FolderID.UUID
			}
			if rbErr := s.UpdateFileParentAndPath(ctx, file.ID, userID, oldFolderID, file.FilePath); rbErr != nil {
				logger.Error("storage move failed and rollback failed", "file_id", fileID, "error", err, "rollback_error", rbErr)
				return fmt.Errorf("storage move failed for %v (%v), rollback also failed: %v", fileID, err, rbErr)
			}
			logger.Warn("storage move failed, rollback successful", "file_id", fileID, "error", err)
			return fmt.Errorf("moving file %v on disk: %w", fileID, err)
		}

		logger.Info("file moved successfully", "file_id", fileID, "new_path", newPath)
	}

	logger.Info("completed file move operation", "file_count", len(fileIDs))
	return nil
}

// RenameFile Rename file with merge + overwrite logic
func (s *FileServiceImpl) RenameFile(ctx context.Context, fileID uuid.UUID, userID int32, newName string, overwrite bool) error {
	logger := middleware.GetLogger(ctx)
	logger.Info("rename operation started")

	file, err := s.GetFileByID(ctx, fileID, userID)
	if err != nil {
		logger.Error("failed to fetch file", "error", err)
		return fmt.Errorf("failed to fetch file: %w", err)
	}

	oldPath := file.FilePath
	folderPath := filepath.Dir(file.FilePath)
	var newPath string
	if folderPath == "." {
		newPath = newName
	} else {
		newPath = filepath.Join(folderPath, newName)
	}

	// Check for existing file
	destFolderID := file.FolderID.UUID
	existingFile, err := s.GetFileByNameInFolder(ctx, &destFolderID, userID, newName)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		logger.Error("error checking for existing file", "error", err)
		return fmt.Errorf("checking existing file: %w", err)
	}

	if existingFile.ID != uuid.Nil {
		if overwrite {
			logger.Info("overwriting existing file", "existing_file_id", existingFile.ID)
			if err := s.DeleteFiles(ctx, []uuid.UUID{existingFile.ID}, userID); err != nil {
				logger.Error("failed to delete existing file for overwrite", "error", err)
				return fmt.Errorf("deleting existing file for overwrite: %w", err)
			}
		} else {
			logger.Warn("file already exists and overwrite not allowed")
			return fmt.Errorf("file '%s' already exists in folder", newName)
		}
	}

	// Update DB
	if err := s.UpdateFileNameAndPath(ctx, file.ID, userID, newName, newPath); err != nil {
		logger.Error("updating file name in DB failed", "error", err)
		return fmt.Errorf("updating file name in DB: %w", err)
	}

	// Rename in storage
	if err := s.local.MoveFile(ctx, userID, oldPath, newPath); err != nil {
		logger.Error("storage rename failed, attempting rollback", "error", err)
		rollbackErr := s.UpdateFileNameAndPath(ctx, file.ID, userID, file.Name, oldPath)
		if rollbackErr != nil {
			logger.Error("rollback failed after storage rename error", "rollback_error", rollbackErr)
			return fmt.Errorf("storage rename failed (%v), rollback also failed: %v", err, rollbackErr)
		}
		return fmt.Errorf("renaming file in storage: %w", err)
	}

	logger.Info("file renamed successfully", "old_path", oldPath, "new_path", newPath)
	return nil
}
