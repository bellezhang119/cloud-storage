package file

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/folder"
	"github.com/bellezhang119/cloud-storage/internal/storage"
	"github.com/google/uuid"
)

type Queries interface {
	CreateFile(ctx context.Context, arg database.CreateFileParams) (database.File, error)
	GetFileByID(ctx context.Context, id uuid.UUID) (database.File, error)
	GetFileByNameInFolder(ctx context.Context, arg database.GetFileByNameInFolderParams) (database.File, error)
	ListFilesInFolder(ctx context.Context, arg database.ListFilesInFolderParams) ([]database.File, error)
	DeleteFile(ctx context.Context, arg database.DeleteFileParams) (int64, error)
	ListFilesRecursive(ctx context.Context, arg database.ListFilesRecursiveParams) ([]database.ListFilesRecursiveRow, error)
	UpdateFileMetadata(ctx context.Context, arg database.UpdateFileMetadataParams) (int64, error)
	UpdateFilePath(ctx context.Context, arg database.UpdateFilePathParams) (int64, error)
}

type FolderService interface {
	CreateFolder(ctx context.Context, userID int32, name string, parentID uuid.NullUUID) (database.Folder, error)
	GetFolderByID(ctx context.Context, id uuid.UUID) (database.Folder, error)
	ListFoldersByParent(ctx context.Context, userID int32, parentID uuid.NullUUID) ([]database.Folder, error)
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

func (s *Service) SaveFile(
	ctx context.Context,
	folderID *uuid.UUID,
	userID int32,
	name string,
	sizeBytes int64,
	mimeType string,
	content io.Reader,
) (database.File, error) {

	if name == "" {
		return database.File{}, errors.New("file name is required")
	}

	uID := sql.NullInt32{Int32: userID, Valid: true}

	// 1. Build folder path relative to user root
	var folderPath string
	var fID uuid.NullUUID
	if folderID != nil {
		f, err := s.folderService.GetFolderByID(ctx, *folderID)
		if err != nil {
			return database.File{}, fmt.Errorf("fetching folder: %w", err)
		}
		folderPath = s.buildFolderPath(ctx, f) // relative to user root
		fID = uuid.NullUUID{UUID: *folderID, Valid: true}
	}

	// 2. Prepare relative file path for storage
	var filePath string
	if folderPath != "" {
		filePath = filepath.Join(folderPath, name)
	} else {
		filePath = name
	}

	// 3. Check if file already exists
	existingFile, err := s.queries.GetFileByNameInFolder(ctx, database.GetFileByNameInFolderParams{
		FolderID: fID,
		Name:     name,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return database.File{}, fmt.Errorf("checking existing file: %w", err)
	}

	// 4. If file exists, delete old DB record and storage
	if existingFile.ID != uuid.Nil {
		_, _ = s.queries.DeleteFile(ctx, database.DeleteFileParams{
			ID:     existingFile.ID,
			UserID: uID,
		})
		_ = s.storage.DeleteFile(userID, existingFile.FilePath)
	}

	// 5. Create new DB record
	mType := sql.NullString{String: mimeType, Valid: mimeType != ""}
	fileMeta, err := s.queries.CreateFile(ctx, database.CreateFileParams{
		FolderID:  fID,
		UserID:    uID,
		Name:      name,
		FilePath:  filePath, // store relative path
		SizeBytes: sizeBytes,
		MimeType:  mType,
	})
	if err != nil {
		return database.File{}, fmt.Errorf("creating file record: %w", err)
	}

	// 6. Save content to storage (LocalStorage will prepend user folder)
	if err := s.storage.SaveFile(userID, filePath, content); err != nil {
		// rollback DB if storage fails
		_, _ = s.queries.DeleteFile(ctx, database.DeleteFileParams{
			ID:     fileMeta.ID,
			UserID: uID,
		})
		return database.File{}, fmt.Errorf("saving file: %w", err)
	}

	return fileMeta, nil
}

func (s *Service) GetFileByID(ctx context.Context, id uuid.UUID) (database.File, error) {
	return s.queries.GetFileByID(ctx, id)
}

func (s *Service) GetFileByNameInFolder(ctx context.Context, folderID uuid.UUID, name string) (database.File, error) {
	file, err := s.queries.GetFileByNameInFolder(ctx, database.GetFileByNameInFolderParams{
		FolderID: uuid.NullUUID{
			UUID:  folderID,
			Valid: true,
		},
		Name: name,
	})
	if err != nil {
		return database.File{}, err
	}
	return file, nil
}

func (s *Service) ListFilesInFolder(ctx context.Context, folderID *uuid.UUID, userID int32) ([]database.File, error) {
	var fID uuid.NullUUID
	if folderID != nil {
		fID = uuid.NullUUID{UUID: *folderID, Valid: true}
	}

	files, err := s.queries.ListFilesInFolder(ctx, database.ListFilesInFolderParams{
		FolderID: fID,
		UserID:   sql.NullInt32{Int32: userID, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

func (s *Service) ListFilesRecursive(ctx context.Context, folderID uuid.UUID, userID int32) ([]database.ListFilesRecursiveRow, error) {
	rows, err := s.queries.ListFilesRecursive(ctx, database.ListFilesRecursiveParams{
		ID: folderID,
		UserID: sql.NullInt32{
			Int32: userID,
			Valid: true,
		},
	})
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Service) GetFileForDownload(ctx context.Context, fileID uuid.UUID, userID int32) (database.File, io.ReadCloser, error) {
	// 1. Look up file in DB
	fileMeta, err := s.queries.GetFileByID(ctx, fileID)
	if err != nil {
		return database.File{}, nil, fmt.Errorf("fetching file metadata: %w", err)
	}

	// 2. Authorization check (make sure the user owns it)
	if fileMeta.UserID.Int32 != userID {
		return database.File{}, nil, fmt.Errorf("unauthorized access")
	}

	// 3. Read file from storage
	content, err := s.storage.ReadFile(userID, fileMeta.FilePath)
	if err != nil {
		return database.File{}, nil, fmt.Errorf("reading file: %w", err)
	}

	return fileMeta, content, nil
}

func (s *Service) DeleteFile(ctx context.Context, fileID uuid.UUID, userID int32) error {
	uID := sql.NullInt32{Int32: userID, Valid: true}

	// 1. Fetch file metadata first
	file, err := s.queries.GetFileByID(ctx, fileID)
	if err != nil {
		return fmt.Errorf("fetching file metadata: %w", err)
	}

	if file.UserID.Int32 != userID {
		return fmt.Errorf("unauthorized")
	}

	// 2. Delete DB record
	rows, err := s.queries.DeleteFile(ctx, database.DeleteFileParams{
		ID:     fileID,
		UserID: uID,
	})
	if err != nil {
		return fmt.Errorf("deleting file record: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("file not found or already deleted")
	}

	// 3. Delete file from storage
	if err := s.storage.DeleteFile(userID, file.FilePath); err != nil {
		// DB record gone, storage deletion failed
		return fmt.Errorf("file removed from DB but failed to delete from storage: %w", err)
	}

	return nil
}

func (s *Service) UpdateFileMetadata(ctx context.Context, fileID uuid.UUID, name string, userID int32) error {
	rows, err := s.queries.UpdateFileMetadata(ctx, database.UpdateFileMetadataParams{
		ID:     fileID,
		Name:   name,
		UserID: sql.NullInt32{Int32: userID, Valid: true},
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("file not found or metadata not changed")
	}
	return nil
}

func (s *Service) UpdateFilePath(ctx context.Context, fileID uuid.UUID, path string, userID int32) error {
	rows, err := s.queries.UpdateFilePath(ctx, database.UpdateFilePathParams{
		ID:       fileID,
		FilePath: path,
		UserID:   sql.NullInt32{Int32: userID, Valid: true},
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("file not found or path not changed")
	}
	return nil
}

func (s *Service) MoveFile(ctx context.Context, file database.File, destFolderID uuid.UUID, userID int32) error {
	// 1. Build destination folder path relative to user's root
	destFolder, err := s.folderService.GetFolderByID(ctx, destFolderID)
	if err != nil {
		return fmt.Errorf("fetching destination folder: %w", err)
	}

	destFolderPath := s.buildFolderPath(ctx, destFolder)
	relativeNewPath := filepath.Join(destFolderPath, file.Name)

	// 2. Update DB first (ensures name uniqueness + logical consistency)
	if err := s.UpdateFilePath(ctx, file.ID, relativeNewPath, userID); err != nil {
		return fmt.Errorf("updating file path in DB: %w", err)
	}

	// 3. Perform the physical file move
	if err := s.storage.MoveFile(userID, file.FilePath, relativeNewPath); err != nil {
		// rollback DB if storage fails
		rollbackErr := s.UpdateFilePath(ctx, file.ID, file.FilePath, userID)
		if rollbackErr != nil {
			return fmt.Errorf("storage move failed (%v), rollback also failed: %v", err, rollbackErr)
		}
		return fmt.Errorf("moving file on disk: %w", err)
	}

	return nil
}

func (s *Service) RenameFile(ctx context.Context, file database.File, newName string, userID int32) error {
	if newName == "" {
		return errors.New("new file name is required")
	}

	// Build new relative path
	oldPath := file.FilePath
	folderPath := filepath.Dir(file.FilePath) // folder containing the file
	newPath := filepath.Join(folderPath, newName)

	uID := sql.NullInt32{Int32: userID, Valid: true}

	// 1. Update DB first (enforces uniqueness)
	rows, err := s.queries.UpdateFileMetadata(ctx, database.UpdateFileMetadataParams{
		ID:     file.ID,
		Name:   newName,
		UserID: uID,
	})
	if err != nil {
		return fmt.Errorf("updating file name in DB: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("file not found or name not changed")
	}

	// 2. Rename file in storage
	if err := s.storage.MoveFile(userID, oldPath, newPath); err != nil {
		// rollback DB if storage fails
		_, rollbackErr := s.queries.UpdateFileMetadata(ctx, database.UpdateFileMetadataParams{
			ID:     file.ID,
			Name:   file.Name,
			UserID: uID,
		})
		if rollbackErr != nil {
			return fmt.Errorf("storage rename failed (%v), rollback DB also failed: %v", err, rollbackErr)
		}
		return fmt.Errorf("renaming file in storage: %w", err)
	}

	return nil
}

func (s *Service) buildFolderPath(ctx context.Context, folder database.Folder) string {
	if folder.ParentID.Valid {
		parent, err := s.folderService.GetFolderByID(ctx, folder.ParentID.UUID)
		if err != nil {
			return folder.Name
		}
		return filepath.Join(s.buildFolderPath(ctx, parent), folder.Name)
	}
	return folder.Name
}
