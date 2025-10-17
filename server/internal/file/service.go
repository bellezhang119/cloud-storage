package file

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/folder"
	"github.com/bellezhang119/cloud-storage/internal/storage"
	"github.com/bellezhang119/cloud-storage/internal/util"
	"github.com/google/uuid"
)

type Queries interface {
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

type FileDownload struct {
	File    database.File
	Content io.ReadCloser
}

type Service struct {
	queries       Queries
	folderService FolderService
	storage       storage.Storage
}

func NewService(q Queries, fs FolderService, s storage.Storage) *Service {
	return &Service{queries: q, folderService: fs, storage: s}
}

func (s *Service) SetFolderService(fs *folder.Service) {
	s.folderService = fs
}

// Getters
func (s *Service) GetFileByID(ctx context.Context, id uuid.UUID, userID int32) (database.File, error) {
	return s.queries.GetFileByID(ctx, database.GetFileByIDParams{
		ID:     id,
		UserID: util.ToNullInt32(&userID),
	})
}

func (s *Service) GetFileByNameInFolder(ctx context.Context, folderID *uuid.UUID, name string) (database.File, error) {
	file, err := s.queries.GetFileByNameInFolder(ctx, database.GetFileByNameInFolderParams{
		FolderID: util.ToNullUUID(folderID),
		Name:     name,
	})
	if err != nil {
		return database.File{}, err
	}
	return file, nil
}

func (s *Service) ListFilesInFolder(ctx context.Context, userID int32, folderID *uuid.UUID) ([]database.File, error) {
	files, err := s.queries.ListFilesInFolder(ctx, database.ListFilesInFolderParams{
		UserID:   util.ToNullInt32(&userID),
		FolderID: util.ToNullUUID(folderID),
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

func (s *Service) ListFilesRecursive(ctx context.Context, userID int32, folderID uuid.UUID) ([]database.ListFilesRecursiveRow, error) {
	rows, err := s.queries.ListFilesRecursive(ctx, database.ListFilesRecursiveParams{
		UserID: util.ToNullInt32(&userID),
		ID:     folderID,
	})
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// --------------------------------------------------------------------------------------------------------------------------

// Upload file
func (s *Service) UploadFile(
	ctx context.Context,
	userID int32,
	folderID *uuid.UUID,
	name string,
	sizeBytes int64,
	mimeType string,
	content io.Reader,
	overwrite bool,
) (database.File, error) {
	if name == "" {
		return database.File{}, errors.New("file name is required")
	}

	// Build folder path relative to user root
	var folderPath string
	if folderID != nil {
		var err error
		folderPath, err = s.folderService.GetFolderFullPath(ctx, *folderID, userID)
		if err != nil {
			return database.File{}, fmt.Errorf("building folder path: %w", err)
		}
	}

	filePath := name
	if folderPath != "" {
		filePath = filepath.Join(folderPath, name)
	}

	// Check if a file with the same name exists
	existingFile, err := s.GetFileByNameInFolder(ctx, folderID, name)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return database.File{}, fmt.Errorf("checking existing file: %w", err)
	}

	if existingFile.ID != uuid.Nil {
		if overwrite {
			// Delete existing file
			if err := s.DeleteFiles(ctx, []uuid.UUID{existingFile.ID}, userID); err != nil {
				return database.File{}, fmt.Errorf("deleting existing file for overwrite: %w", err)
			}
		} else {
			return database.File{}, fmt.Errorf("file '%s' already exists in folder", name)
		}
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
		return database.File{}, fmt.Errorf("creating file record: %w", err)
	}

	// Save content to storage
	if err := s.storage.SaveFile(userID, filePath, content); err != nil {
		_ = s.DeleteFiles(ctx, []uuid.UUID{fileMeta.ID}, userID) // rollback DB
		return database.File{}, fmt.Errorf("saving file: %w", err)
	}

	return fileMeta, nil
}

// --------------------------------------------------------------------------------------------------------------------------

// Download file
func (s *Service) DownloadFiles(ctx context.Context, fileIDs []uuid.UUID, userID int32) ([]FileDownload, error) {
	if len(fileIDs) == 0 {
		return nil, fmt.Errorf("no file IDs provided")
	}

	var downloads []FileDownload

	for _, fileID := range fileIDs {
		// 1. Look up file in DB
		fileMeta, err := s.GetFileByID(ctx, fileID, userID)
		if err != nil {
			return nil, fmt.Errorf("fetching file metadata for %s: %w", fileID, err)
		}

		// 2. Authorization check
		if fileMeta.UserID.Int32 != userID {
			return nil, fmt.Errorf("unauthorized access to file %s", fileID)
		}

		// 3. Read file from storage
		content, err := s.storage.ReadFile(userID, fileMeta.FilePath)
		if err != nil {
			return nil, fmt.Errorf("reading file %s: %w", fileID, err)
		}

		downloads = append(downloads, FileDownload{
			File:    fileMeta,
			Content: content,
		})
	}

	return downloads, nil
}

// --------------------------------------------------------------------------------------------------------------------------

func (s *Service) DeleteFiles(ctx context.Context, filesIDs []uuid.UUID, userID int32) error {
	// 1. Fetch all files metadata first
	var files []database.File
	for _, id := range filesIDs {
		file, err := s.GetFileByID(ctx, id, userID)
		if err != nil {
			return fmt.Errorf("fetching file metadata for %s: %w", id, err)
		}
		if file.UserID.Int32 != userID {
			return fmt.Errorf("unauthorized to delete file %s", id)
		}
		files = append(files, file)
	}

	// 2. Delete DB records
	rows, err := s.queries.DeleteFiles(ctx, database.DeleteFilesParams{
		Column1: filesIDs,
		UserID:  util.ToNullInt32(&userID),
	})
	if err != nil {
		return fmt.Errorf("deleting file records: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("no files were deleted (maybe already deleted)")
	}

	// 3. Delete files from storage
	var storageErrs []string
	for _, file := range files {
		if err := s.storage.DeleteFile(userID, file.FilePath); err != nil {
			storageErrs = append(storageErrs, fmt.Sprintf("%s: %v", file.Name, err))
		}
	}

	if len(storageErrs) > 0 {
		return fmt.Errorf("some files removed from DB but failed to delete from storage: %s", strings.Join(storageErrs, "; "))
	}

	return nil
}

// --------------------------------------------------------------------------------------------------------------------------

// Update methods
func (s *Service) UpdateFileMetadata(
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

func (s *Service) UpdateFileParentAndPath(ctx context.Context, fileID uuid.UUID, userID int32, folderID *uuid.UUID, filePath string) error {
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

func (s *Service) UpdateFileNameAndPath(ctx context.Context, fileID uuid.UUID, userID int32, name string, filePath string) error {
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

func (s *Service) MoveFiles(ctx context.Context, fileIDs []uuid.UUID, destFolderID *uuid.UUID, userID int32, overwrite bool) error {
	for _, fileID := range fileIDs {
		file, err := s.GetFileByID(ctx, fileID, userID)
		if err != nil {
			return fmt.Errorf("file %v not found: %w", fileID, err)
		}

		// Build new relative path
		newPath := file.Name
		if destFolderID != nil {
			destFolderPath, err := s.folderService.GetFolderFullPath(ctx, *destFolderID, userID)
			if err != nil {
				return fmt.Errorf("cannot build folder path for file %v: %w", fileID, err)
			}
			newPath = filepath.Join(destFolderPath, file.Name)
		}

		// Check for existing file conflict
		existingFile, err := s.GetFileByNameInFolder(ctx, destFolderID, file.Name)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("checking existing file %v: %w", fileID, err)
		}

		if existingFile.ID != uuid.Nil {
			if !overwrite {
				return fmt.Errorf("file %s already exists in destination", file.Name)
			}
			// Delete existing file if overwrite is true
			if err := s.DeleteFiles(ctx, []uuid.UUID{existingFile.ID}, userID); err != nil {
				return fmt.Errorf("failed to delete existing file %v for overwrite: %w", existingFile.ID, err)
			}
		}

		// Update DB first
		if err := s.UpdateFileParentAndPath(ctx, file.ID, userID, destFolderID, newPath); err != nil {
			return fmt.Errorf("updating file path in DB for %v: %w", fileID, err)
		}

		// Move in storage
		if err := s.storage.MoveFile(userID, file.FilePath, newPath); err != nil {
			var oldFolderID *uuid.UUID
			if file.FolderID.Valid {
				oldFolderID = &file.FolderID.UUID
			}
			// rollback DB if storage move fails
			rollbackErr := s.UpdateFileParentAndPath(ctx, file.ID, userID, oldFolderID, file.FilePath)
			if rollbackErr != nil {
				return fmt.Errorf("storage move failed for %v (%v), rollback also failed: %v", fileID, err, rollbackErr)
			}
			return fmt.Errorf("moving file %v on disk: %w", fileID, err)
		}
	}

	return nil
}

// Rename file with merge + overwrite logic
func (s *Service) RenameFile(ctx context.Context, file database.File, newName string, userID int32, overwrite bool) error {
	if newName == "" {
		return errors.New("new file name is required")
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
	existingFile, err := s.GetFileByNameInFolder(ctx, &destFolderID, newName)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("checking existing file: %w", err)
	}

	if existingFile.ID != uuid.Nil {
		if overwrite {
			if err := s.DeleteFiles(ctx, []uuid.UUID{existingFile.ID}, userID); err != nil {
				return fmt.Errorf("deleting existing file for overwrite: %w", err)
			}
		} else {
			return fmt.Errorf("file '%s' already exists in folder", newName)
		}
	}

	// Update DB
	if err := s.UpdateFileNameAndPath(ctx, file.ID, userID, newName, newPath); err != nil {
		return fmt.Errorf("updating file name in DB: %w", err)
	}

	// Rename in storage
	if err := s.storage.MoveFile(userID, oldPath, newPath); err != nil {
		rollbackErr := s.UpdateFileNameAndPath(ctx, file.ID, userID, file.Name, oldPath)
		if rollbackErr != nil {
			return fmt.Errorf("storage rename failed (%v), rollback also failed: %v", err, rollbackErr)
		}
		return fmt.Errorf("renaming file in storage: %w", err)
	}

	return nil
}

// --------------------------------------------------------------------------------------------------------------------------
