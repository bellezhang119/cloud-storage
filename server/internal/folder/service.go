package folder

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/storage"
	"github.com/google/uuid"
)

type Queries interface {
	CreateFolder(ctx context.Context, arg database.CreateFolderParams) (database.Folder, error)
	GetFolderByID(ctx context.Context, id uuid.UUID) (database.Folder, error)
	ListFoldersByParent(ctx context.Context, arg database.ListFoldersByParentParams) ([]database.Folder, error)
	GetFolderByNameInParent(ctx context.Context, arg database.GetFolderByNameInParentParams) (database.Folder, error)
	DeleteFolder(ctx context.Context, arg database.DeleteFolderParams) (int64, error)
	ListFoldersRecursive(ctx context.Context, arg database.ListFoldersRecursiveParams) ([]database.ListFoldersRecursiveRow, error)
	UpdateFolderMetadata(ctx context.Context, arg database.UpdateFolderMetadataParams) (int64, error)
	UpdateFolderParent(ctx context.Context, arg database.UpdateFolderParentParams) (int64, error)
	GetFolderFullPath(ctx context.Context, arg database.GetFolderFullPathParams) (string, error)
}

type FileService interface {
	UploadFile(
		ctx context.Context,
		userID int32,
		folderID *uuid.UUID,
		name string,
		sizeBytes int64,
		mimeType string,
		content io.Reader,
	) (database.File, error)
	ListFilesRecursive(ctx context.Context, userID int32, folderID uuid.UUID) ([]database.ListFilesRecursiveRow, error)
	GetFileByNameInFolder(ctx context.Context, folderID *uuid.UUID, name string) (database.File, error)
	UpdateFileMetadata(
		ctx context.Context,
		fileID uuid.UUID,
		userID int32,
		name *string,
		filePath *string,
		sizeBytes *int64,
		mimeType *string,
	) error
	DeleteFile(ctx context.Context, fileID uuid.UUID, userID int32) error
}

type Service struct {
	queries     Queries
	fileService FileService
	storage     storage.Storage
}

type FolderUploadItem struct {
	Name      string    // File or folder name only (no path)
	SizeBytes int64     // File size in bytes (0 for folders)
	MimeType  string    // e.g. "image/png", "folder"
	IsFolder  bool      // True if it's a folder
	Content   io.Reader // File content (nil if IsFolder = true)
}

type UploadConflict struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	IsFolder bool   `json:"isFolder"`
}

type UploadResult struct {
	Created     []string         `json:"created"`
	Overwritten []string         `json:"overwritten"`
	Conflicts   []UploadConflict `json:"conflicts"`
	Skipped     []string         `json:"skipped"`
}

func NewService(q Queries, fs FileService, s storage.Storage) *Service {
	return &Service{queries: q, fileService: fs, storage: s}
}

// Getters
func (s *Service) GetFolderByID(ctx context.Context, id uuid.UUID) (database.Folder, error) {
	return s.queries.GetFolderByID(ctx, id)
}

func (s *Service) ListFoldersByParent(ctx context.Context, userID int32, parentID *uuid.UUID) ([]database.Folder, error) {
	return s.queries.ListFoldersByParent(ctx, database.ListFoldersByParentParams{
		UserID:   sql.NullInt32{Int32: userID, Valid: true},
		ParentID: uuid.NullUUID{UUID: *parentID, Valid: parentID != nil},
	})
}

func (s *Service) GetFolderFullPath(ctx context.Context, folderID uuid.UUID, userID int32) (string, error) {
	return s.queries.GetFolderFullPath(ctx, database.GetFolderFullPathParams{
		ID:     folderID,
		UserID: sql.NullInt32{Int32: userID, Valid: true},
	})
}

func (s *Service) GetFolderByNameInParent(ctx context.Context, userID int32, name string, parentID *uuid.UUID) (database.Folder, error) {
	return s.queries.GetFolderByNameInParent(ctx, database.GetFolderByNameInParentParams{
		UserID:   sql.NullInt32{Int32: userID, Valid: true},
		Name:     name,
		ParentID: uuid.NullUUID{UUID: *parentID, Valid: parentID != nil},
	})
}

// --------------------------------------------------------------------------------------------------------------------------

// Create Folder
func (s *Service) CreateFolder(ctx context.Context, userID int32, name string, parentID *uuid.UUID) (database.Folder, error) {
	// 1. Create DB record first
	folder, err := s.queries.CreateFolder(ctx, database.CreateFolderParams{
		UserID:   sql.NullInt32{Int32: userID, Valid: true},
		Name:     name,
		ParentID: uuid.NullUUID{UUID: *parentID, Valid: parentID != nil},
	})
	if err != nil {
		return database.Folder{}, fmt.Errorf("creating folder record: %w", err)
	}

	// 2. Build full path for the new folder
	path, err := s.GetFolderFullPath(ctx, folder.ID, userID)
	if err != nil {
		// rollback DB record
		_, _ = s.queries.DeleteFolder(ctx, database.DeleteFolderParams{
			ID:     folder.ID,
			UserID: sql.NullInt32{Int32: userID, Valid: true},
		})
		return database.Folder{}, fmt.Errorf("building folder path: %w", err)
	}

	// 3. Create folder on disk
	if err := s.storage.CreateDirectory(userID, path); err != nil {
		// rollback DB record
		_, _ = s.queries.DeleteFolder(ctx, database.DeleteFolderParams{
			ID:     folder.ID,
			UserID: sql.NullInt32{Int32: userID, Valid: true},
		})
		return database.Folder{}, fmt.Errorf("creating folder on disk: %w", err)
	}

	return folder, nil
}

// --------------------------------------------------------------------------------------------------------------------------

// Download folder
func (s *Service) GetZippedFolderForDownload(ctx context.Context, folderID uuid.UUID, userID int32, w io.Writer) (database.Folder, error) {
	// 1. Look up folder in DB
	folderMeta, err := s.queries.GetFolderByID(ctx, folderID)
	if err != nil {
		return database.Folder{}, fmt.Errorf("fetching folder metadata: %w", err)
	}

	// 2. Authorization check
	if folderMeta.UserID.Int32 != userID {
		return database.Folder{}, fmt.Errorf("unauthorized access")
	}

	// 3. Build full folder path
	folderPath, err := s.GetFolderFullPath(ctx, folderID, userID)
	if err != nil {
		return database.Folder{}, fmt.Errorf("building folder path: %w", err)
	}

	// 4. Stream zip into provided writer
	if err := s.storage.ZipFolder(userID, folderPath, w); err != nil {
		return database.Folder{}, fmt.Errorf("zipping folder: %w", err)
	}

	return folderMeta, nil
}

// --------------------------------------------------------------------------------------------------------------------------

func (s *Service) UploadFolderStructure(
	ctx context.Context,
	userID int32,
	parentID *uuid.UUID,
	items []FolderUploadItem,
	overwrite bool,
) (UploadResult, error) {
	result := UploadResult{}

	for _, item := range items {
		if item.IsFolder {
			// --- Handle Folder Upload ---
			existing, err := s.queries.GetFolderByNameInParent(ctx, database.GetFolderByNameInParentParams{
				ParentID: uuid.NullUUID{UUID: *parentID, Valid: parentID != nil},
				Name:     item.Name,
				UserID:   sql.NullInt32{Int32: userID, Valid: true},
			})
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return result, fmt.Errorf("checking existing folder: %w", err)
			}

			// Handle conflict
			if existing.ID != uuid.Nil {
				if overwrite {
					// delete and recreate
					if err := s.DeleteFolder(ctx, existing.ID, userID); err != nil {
						return result, fmt.Errorf("deleting existing folder before overwrite: %w", err)
					}
				} else {
					result.Conflicts = append(result.Conflicts, UploadConflict{
						Name:     item.Name,
						Path:     item.Name,
						IsFolder: true,
					})
					continue
				}
			}

			// Use existing method to create folder
			newFolder, err := s.CreateFolder(ctx, userID, item.Name, parentID)
			if err != nil {
				return result, fmt.Errorf("creating folder: %w", err)
			}

			result.Created = append(result.Created, newFolder.Name)
			continue
		}

		// --- Handle File Upload ---
		existingFile, err := s.fileService.GetFileByNameInFolder(ctx, parentID, item.Name)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return result, fmt.Errorf("checking existing file: %w", err)
		}

		if existingFile.ID != uuid.Nil {
			if overwrite {
				if err := s.fileService.DeleteFile(ctx, existingFile.ID, userID); err != nil {
					return result, fmt.Errorf("deleting existing file before overwrite: %w", err)
				}
				result.Overwritten = append(result.Overwritten, existingFile.FilePath)
			} else {
				result.Conflicts = append(result.Conflicts, UploadConflict{
					Name:     item.Name,
					Path:     existingFile.FilePath,
					IsFolder: false,
				})
				continue
			}
		}

		// Reuse UploadFile method (handles storage + DB)
		_, err = s.fileService.UploadFile(ctx, userID, parentID, item.Name, item.SizeBytes, item.MimeType, item.Content)
		if err != nil {
			return result, fmt.Errorf("uploading file: %w", err)
		}

		result.Created = append(result.Created, item.Name)
	}

	return result, nil
}

// --------------------------------------------------------------------------------------------------------------------------

// Delete folder
func (s *Service) DeleteFolder(ctx context.Context, folderID uuid.UUID, userID int32) error {
	uID := sql.NullInt32{Int32: userID, Valid: true}

	// 1. Delete folder row from DB first (cascades handle child folders/files)
	rows, err := s.queries.DeleteFolder(ctx, database.DeleteFolderParams{
		ID:     folderID,
		UserID: uID,
	})
	if err != nil {
		return fmt.Errorf("deleting folder row from DB: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("folder not found or already deleted")
	}

	// 2. Delete folder contents from storage
	path, err := s.GetFolderFullPath(ctx, folderID, userID)
	if err != nil {
		// storage may not exist, just log
		return fmt.Errorf("building folder path after DB deletion: %w", err)
	}

	if err := s.storage.DeleteDirectory(userID, path); err != nil {
		// folder row already deleted, cannot rollback DB
		return fmt.Errorf("folder deleted in DB but failed to delete from storage: %w", err)
	}

	return nil
}

// --------------------------------------------------------------------------------------------------------------------------

// Update methods
func (s *Service) UpdateFolderMetadata(ctx context.Context, folderID uuid.UUID, userID int32, name string) error {
	rows, err := s.queries.UpdateFolderMetadata(ctx, database.UpdateFolderMetadataParams{
		ID:     folderID,
		UserID: sql.NullInt32{Int32: userID, Valid: true},
		Name:   name,
	})

	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("folder not found or metadata not changed")
	}
	return nil
}

func (s *Service) UpdateFolderParent(ctx context.Context, folderID uuid.UUID, userID int32, newParentID *uuid.UUID) error {
	// Prevent a folder from being its own parent
	if newParentID != nil && *newParentID == folderID {
		return fmt.Errorf("cannot set folder as its own parent")
	}

	// Get folder info to know its current name
	folder, err := s.queries.GetFolderByID(ctx, folderID)
	if err != nil {
		return fmt.Errorf("fetching folder: %w", err)
	}

	// Check for duplicate name in target parent
	folders, err := s.ListFoldersByParent(ctx, userID, newParentID)
	if err != nil {
		return fmt.Errorf("checking for duplicate folder: %w", err)
	}
	for _, f := range folders {
		if f.Name == folder.Name {
			return fmt.Errorf("folder with name '%s' already exists under the target parent", folder.Name)
		}
	}

	// Optional: check if newParentID exists (FK safety)
	if newParentID != nil {
		_, err := s.queries.GetFolderByID(ctx, *newParentID)
		if err != nil {
			return fmt.Errorf("target parent folder does not exist: %w", err)
		}
	}

	// Update parent
	_, err = s.queries.UpdateFolderParent(ctx, database.UpdateFolderParentParams{
		ID:       folderID,
		ParentID: uuid.NullUUID{UUID: *newParentID, Valid: newParentID != nil},
		UserID:   sql.NullInt32{Int32: userID, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("updating folder parent: %w", err)
	}

	return nil
}

func (s *Service) RenameFolder(ctx context.Context, folderID uuid.UUID, newName string, userID int32) error {
	if newName == "" {
		return fmt.Errorf("new folder name is required")
	}

	uID := sql.NullInt32{Int32: userID, Valid: true}

	// 1. Fetch current folder info
	folder, err := s.queries.GetFolderByID(ctx, folderID)
	if err != nil {
		return fmt.Errorf("fetching folder: %w", err)
	}

	oldPath, err := s.GetFolderFullPath(ctx, folderID, userID)
	if err != nil {
		return fmt.Errorf("building old folder path: %w", err)
	}

	// 2. Update DB metadata first
	rows, err := s.queries.UpdateFolderMetadata(ctx, database.UpdateFolderMetadataParams{
		ID:     folderID,
		Name:   newName,
		UserID: uID,
	})
	if err != nil {
		return fmt.Errorf("updating folder name in DB: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("folder not found or name not changed")
	}

	// 3. Build new folder path
	folder.Name = newName
	newPath, err := s.GetFolderFullPath(ctx, folderID, userID)
	if err != nil {
		// rollback DB if path building fails
		_, _ = s.queries.UpdateFolderMetadata(ctx, database.UpdateFolderMetadataParams{
			ID:     folderID,
			Name:   folder.Name,
			UserID: uID,
		})
		return fmt.Errorf("building new folder path: %w", err)
	}

	// 4. Rename folder on disk
	if err := s.storage.MoveDirectory(userID, oldPath, newPath); err != nil {
		// rollback DB if storage rename fails
		_, _ = s.queries.UpdateFolderMetadata(ctx, database.UpdateFolderMetadataParams{
			ID:     folderID,
			Name:   folder.Name,
			UserID: uID,
		})
		return fmt.Errorf("renaming folder on disk: %w", err)
	}

	// 5. Update all child files’ paths in DB
	files, err := s.fileService.ListFilesRecursive(ctx, userID, folderID)
	if err != nil {
		return fmt.Errorf("listing files in folder: %w", err)
	}

	for _, f := range files {
		relPath, err := filepath.Rel(oldPath, f.FilePath)
		if err != nil {
			return fmt.Errorf("calculating relative path: %w", err)
		}
		newFilePath := filepath.Join(newPath, relPath)
		if err := s.fileService.UpdateFileMetadata(ctx, f.FileID, userID, nil, &newFilePath, nil, nil); err != nil {
			return fmt.Errorf("updating file path in DB: %w", err)
		}
	}

	return nil
}

// --------------------------------------------------------------------------------------------------------------------------

// Move folder
func (s *Service) MoveFolder(ctx context.Context, folderID uuid.UUID, newParentID *uuid.UUID, userID int32) error {
	// 1. Fetch current folder info
	folder, err := s.queries.GetFolderByID(ctx, folderID)
	if err != nil {
		return fmt.Errorf("fetching folder: %w", err)
	}

	oldPath, err := s.GetFolderFullPath(ctx, folderID, userID)
	if err != nil {
		return fmt.Errorf("building old folder path: %w", err)
	}

	// 2. Update parent_id in DB first
	err = s.UpdateFolderParent(ctx, folderID, userID, newParentID)
	if err != nil {
		return fmt.Errorf("updating folder parent in DB: %w", err)
	}

	// 3. Build new folder path
	newPath := folder.Name
	if newParentID != nil {
		parentPath, err := s.GetFolderFullPath(ctx, *newParentID, userID)
		if err != nil {
			// rollback DB
			var oldParentID *uuid.UUID
			if folder.ParentID.Valid {
				oldParentID = &folder.ParentID.UUID
			}
			_ = s.UpdateFolderParent(ctx, folderID, userID, oldParentID)
			return fmt.Errorf("building new folder path: %w", err)
		}
		newPath = filepath.Join(parentPath, folder.Name)
	}

	// 4. Move folder on disk (including all children)
	if err := s.storage.MoveDirectory(userID, oldPath, newPath); err != nil {
		// rollback DB
		var oldParentID *uuid.UUID
		if folder.ParentID.Valid {
			oldParentID = &folder.ParentID.UUID
		}
		_ = s.UpdateFolderParent(ctx, folderID, userID, oldParentID)
		return fmt.Errorf("moving folder on disk: %w", err)
	}

	// 5. Update all child files’ paths in DB
	files, err := s.fileService.ListFilesRecursive(ctx, userID, folderID)
	if err != nil {
		return fmt.Errorf("listing files in folder: %w", err)
	}

	for _, f := range files {
		relPath, err := filepath.Rel(oldPath, f.FilePath)
		if err != nil {
			return fmt.Errorf("calculating relative path: %w", err)
		}
		newFilePath := filepath.Join(newPath, relPath)
		if err := s.fileService.UpdateFileMetadata(ctx, f.FileID, userID, nil, &newFilePath, nil, nil); err != nil {
			return fmt.Errorf("updating file path in DB: %w", err)
		}
	}

	return nil
}

// --------------------------------------------------------------------------------------------------------------------------
