package folder

import (
	"context"
	"database/sql"
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
	DeleteFolder(ctx context.Context, arg database.DeleteFolderParams) (int64, error)
	ListFoldersRecursive(ctx context.Context, arg database.ListFoldersRecursiveParams) ([]database.ListFoldersRecursiveRow, error)
	UpdateFolderMetadata(ctx context.Context, arg database.UpdateFolderMetadataParams) (int64, error)
	UpdateFolderParent(ctx context.Context, arg database.UpdateFolderParentParams) (int64, error)
}

type FileService interface {
	ListFilesRecursive(ctx context.Context, folderID uuid.UUID, userID int32) ([]database.ListFilesRecursiveRow, error)
	UpdateFilePath(ctx context.Context, fileID uuid.UUID, path string, userID int32) error
}

type Service struct {
	queries     Queries
	fileService FileService
	storage     storage.Storage
}

func NewService(q Queries, fs FileService, s storage.Storage) *Service {
	return &Service{queries: q, fileService: fs, storage: s}
}

func (s *Service) CreateFolder(ctx context.Context, userID int32, name string, parentID uuid.NullUUID) (database.Folder, error) {
	// 1. Create DB record first
	folder, err := s.queries.CreateFolder(ctx, database.CreateFolderParams{
		UserID:   sql.NullInt32{Int32: userID, Valid: true},
		Name:     name,
		ParentID: parentID,
	})
	if err != nil {
		return database.Folder{}, fmt.Errorf("creating folder record: %w", err)
	}

	// 2. Create folder on disk
	path, err := s.buildFolderPath(ctx, folder.ID)
	if err != nil {
		// rollback DB if path build fails
		_, _ = s.queries.DeleteFolder(ctx, database.DeleteFolderParams{
			ID:     folder.ID,
			UserID: sql.NullInt32{Int32: userID, Valid: true},
		})
		return database.Folder{}, fmt.Errorf("building folder path: %w", err)
	}

	if err := s.storage.CreateDirectory(userID, path); err != nil {
		// rollback DB if storage creation fails
		_, _ = s.queries.DeleteFolder(ctx, database.DeleteFolderParams{
			ID:     folder.ID,
			UserID: sql.NullInt32{Int32: userID, Valid: true},
		})
		return database.Folder{}, fmt.Errorf("creating folder on disk: %w", err)
	}

	return folder, nil
}

func (s *Service) GetFolderByID(ctx context.Context, id uuid.UUID) (database.Folder, error) {
	return s.queries.GetFolderByID(ctx, id)
}

func (s *Service) ListFoldersByParent(ctx context.Context, userID int32, parentID uuid.NullUUID) ([]database.Folder, error) {
	return s.queries.ListFoldersByParent(ctx, database.ListFoldersByParentParams{
		ParentID: parentID,
		UserID:   sql.NullInt32{Int32: userID, Valid: true},
	})
}

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
	folderPath, err := s.buildFolderPath(ctx, folderID)
	if err != nil {
		return database.Folder{}, fmt.Errorf("building folder path: %w", err)
	}

	// 4. Stream zip into provided writer
	if err := s.storage.ZipFolder(userID, folderPath, w); err != nil {
		return database.Folder{}, fmt.Errorf("zipping folder: %w", err)
	}

	return folderMeta, nil
}

func (s *Service) buildFolderPath(ctx context.Context, folderID uuid.UUID) (string, error) {
	folder, err := s.queries.GetFolderByID(ctx, folderID)
	if err != nil {
		return "", fmt.Errorf("fetching folder: %w", err)
	}

	if folder.ParentID.Valid {
		parentPath, err := s.buildFolderPath(ctx, folder.ParentID.UUID)
		if err != nil {
			return "", err
		}
		return filepath.Join(parentPath, folder.Name), nil
	}

	return folder.Name, nil
}

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
	path, err := s.buildFolderPath(ctx, folderID)
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

func (s *Service) UpdateFolderMetadata(ctx context.Context, folderID uuid.UUID, name string, userID int32) error {
	uID := sql.NullInt32{Int32: userID, Valid: true}

	rows, err := s.queries.UpdateFolderMetadata(ctx, database.UpdateFolderMetadataParams{
		ID:     folderID,
		Name:   name,
		UserID: uID,
	})

	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("folder not found or metadata not changed")
	}
	return nil
}

func (s *Service) UpdateFolderParent(ctx context.Context, folderID uuid.UUID, parentID uuid.UUID, userID int32) error {
	pID := uuid.NullUUID{UUID: parentID, Valid: true}
	uID := sql.NullInt32{Int32: userID, Valid: true}

	rows, err := s.queries.UpdateFolderParent(ctx, database.UpdateFolderParentParams{
		ID:       folderID,
		ParentID: pID,
		UserID:   uID,
	})

	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("folder not found or parent not changed")
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

	oldPath, err := s.buildFolderPath(ctx, folderID)
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
	newPath, err := s.buildFolderPath(ctx, folderID)
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
	files, err := s.fileService.ListFilesRecursive(ctx, folderID, userID)
	if err != nil {
		return fmt.Errorf("listing files in folder: %w", err)
	}

	for _, f := range files {
		relPath, err := filepath.Rel(oldPath, f.FilePath)
		if err != nil {
			return fmt.Errorf("calculating relative path: %w", err)
		}
		newFilePath := filepath.Join(newPath, relPath)
		if err := s.fileService.UpdateFilePath(ctx, f.FileID, newFilePath, userID); err != nil {
			return fmt.Errorf("updating file path in DB: %w", err)
		}
	}

	return nil
}

func (s *Service) MoveFolder(ctx context.Context, folderID uuid.UUID, newParentID uuid.NullUUID, userID int32) error {
	uID := sql.NullInt32{Int32: userID, Valid: true}

	// 1. Fetch current folder info
	folder, err := s.queries.GetFolderByID(ctx, folderID)
	if err != nil {
		return fmt.Errorf("fetching folder: %w", err)
	}

	oldPath, err := s.buildFolderPath(ctx, folderID)
	if err != nil {
		return fmt.Errorf("building old folder path: %w", err)
	}

	// 2. Update parent_id in DB first
	rows, err := s.queries.UpdateFolderParent(ctx, database.UpdateFolderParentParams{
		ID:       folderID,
		ParentID: newParentID,
		UserID:   uID,
	})
	if err != nil {
		return fmt.Errorf("updating folder parent in DB: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("folder not found or parent not updated")
	}

	// 3. Build new folder path
	newPath := folder.Name
	if newParentID.Valid {
		parentPath, err := s.buildFolderPath(ctx, newParentID.UUID)
		if err != nil {
			// rollback DB
			_, _ = s.queries.UpdateFolderParent(ctx, database.UpdateFolderParentParams{
				ID:       folderID,
				ParentID: folder.ParentID,
				UserID:   uID,
			})
			return fmt.Errorf("building new folder path: %w", err)
		}
		newPath = filepath.Join(parentPath, folder.Name)
	}

	// 4. Move folder on disk (including all children)
	if err := s.storage.MoveDirectory(userID, oldPath, newPath); err != nil {
		// rollback DB
		_, _ = s.queries.UpdateFolderParent(ctx, database.UpdateFolderParentParams{
			ID:       folderID,
			ParentID: folder.ParentID,
			UserID:   uID,
		})
		return fmt.Errorf("moving folder on disk: %w", err)
	}

	// 5. Update all child files’ paths in DB
	files, err := s.fileService.ListFilesRecursive(ctx, folderID, userID)
	if err != nil {
		return fmt.Errorf("listing files in folder: %w", err)
	}

	for _, f := range files {
		relPath, err := filepath.Rel(oldPath, f.FilePath)
		if err != nil {
			return fmt.Errorf("calculating relative path: %w", err)
		}
		newFilePath := filepath.Join(newPath, relPath)
		if err := s.fileService.UpdateFilePath(ctx, f.FileID, newFilePath, userID); err != nil {
			return fmt.Errorf("updating file path in DB: %w", err)
		}
	}

	return nil
}
