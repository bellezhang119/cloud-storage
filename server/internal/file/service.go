package file

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
	CreateFile(ctx context.Context, arg database.CreateFileParams) (database.File, error)
	GetFileByID(ctx context.Context, id uuid.UUID) (database.File, error)
	GetFileByNameInFolder(ctx context.Context, arg database.GetFileByNameInFolderParams) (database.File, error)
	ListFilesInFolder(ctx context.Context, folderID uuid.NullUUID) ([]database.File, error)
	PermanentlyDeleteFile(ctx context.Context, arg database.PermanentlyDeleteFileParams) (int64, error)
	RestoreFile(ctx context.Context, arg database.RestoreFileParams) (int64, error)
	TrashFile(ctx context.Context, arg database.TrashFileParams) (int64, error)
	UpdateFileMetadata(ctx context.Context, arg database.UpdateFileMetadataParams) (int64, error)
	UpdateFilePath(ctx context.Context, arg database.UpdateFilePathParams) (int64, error)
}

type FolderGetter interface {
	GetFolderByID(ctx context.Context, id uuid.UUID) (database.Folder, error)
}

type Service struct {
	queries       Queries
	folderService FolderGetter
	storage       storage.Storage
}

func NewService(q Queries, fs FolderGetter, s storage.Storage) *Service {
	return &Service{queries: q, folderService: fs, storage: s}
}

func (s *Service) CreateFile(
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

	folderPath := ""
	if folderID != nil {
		f, err := s.folderService.GetFolderByID(ctx, *folderID)
		if err != nil {
			return database.File{}, fmt.Errorf("fetching folder: %w", err)
		}

		folderPath = s.buildFolderPath(ctx, f)
	}

	filePath := filepath.Join("uploads", fmt.Sprintf("user_%d", userID), folderPath, name)

	if err := s.storage.SaveFile(filePath, content); err != nil {
		return database.File{}, fmt.Errorf("saving file: %w", err)
	}

	var fID uuid.NullUUID
	if folderID != nil {
		fID = uuid.NullUUID{UUID: *folderID, Valid: true}
	}
	uID := sql.NullInt32{Int32: userID, Valid: true}
	mType := sql.NullString{String: mimeType, Valid: mimeType != ""}

	fileMeta, err := s.queries.CreateFile(ctx, database.CreateFileParams{
		FolderID:  fID,
		UserID:    uID,
		Name:      name,
		FilePath:  filePath,
		SizeBytes: sizeBytes,
		MimeType:  mType,
	})
	if err != nil {
		_ = s.storage.DeleteFile(filePath)
		return database.File{}, err
	}

	return fileMeta, nil
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

func (s *Service) ListFilesInFolder(ctx context.Context, folderID *uuid.UUID) ([]database.File, error) {
	var folderParam uuid.NullUUID
	if folderID != nil {
		folderParam = uuid.NullUUID{
			UUID:  *folderID,
			Valid: true,
		}
	} else {
		folderParam = uuid.NullUUID{Valid: false}
	}

	files, err := s.queries.ListFilesInFolder(ctx, folderParam)
	if err != nil {
		return nil, err
	}

	return files, nil
}

func (s *Service) PermanentlyDeleteFile(ctx context.Context, fileID uuid.UUID, userID int32) error {
	rows, err := s.queries.PermanentlyDeleteFile(ctx, database.PermanentlyDeleteFileParams{
		ID:     fileID,
		UserID: sql.NullInt32{Int32: userID, Valid: true},
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("file not found or already deleted")
	}
	return nil
}

func (s *Service) RestoreFile(ctx context.Context, fileID uuid.UUID, userID int32) error {
	rows, err := s.queries.RestoreFile(ctx, database.RestoreFileParams{
		ID:     fileID,
		UserID: sql.NullInt32{Int32: userID, Valid: true},
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("file not found or not trashed")
	}
	return nil
}

func (s *Service) TrashFile(ctx context.Context, fileID uuid.UUID, userID int32) error {
	rows, err := s.queries.TrashFile(ctx, database.TrashFileParams{
		ID:     fileID,
		UserID: sql.NullInt32{Int32: userID, Valid: true},
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("file not found or already trashed")
	}
	return nil
}

func (s *Service) UpdateFileMetadata(
	ctx context.Context,
	fileID uuid.UUID,
	name string,
	userID int32,
) error {
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

func (s *Service) MoveFile(
	ctx context.Context,
	fileID uuid.UUID,
	oldPath, newPath string,
	userID int32,
) error {
	if err := s.storage.MoveFile(oldPath, newPath); err != nil {
		return fmt.Errorf("moving file on disk: %w", err)
	}

	if err := s.UpdateFilePath(ctx, fileID, newPath, userID); err != nil {
		_ = s.storage.MoveFile(newPath, oldPath)
		return fmt.Errorf("updating file path in DB: %w", err)
	}

	return nil
}
