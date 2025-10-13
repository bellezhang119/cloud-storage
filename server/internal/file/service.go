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
}

type FolderService interface {
	CreateFolder(ctx context.Context, userID int32, name string, parentID *uuid.UUID) (database.Folder, error)
	GetFolderByID(ctx context.Context, id uuid.UUID) (database.Folder, error)
	ListFoldersByParent(ctx context.Context, userID int32, parentID *uuid.UUID) ([]database.Folder, error)
	GetFolderFullPath(ctx context.Context, folderID uuid.UUID, userID int32) (string, error)
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
func (s *Service) GetFileByID(ctx context.Context, id uuid.UUID) (database.File, error) {
	return s.queries.GetFileByID(ctx, id)
}

func (s *Service) GetFileByNameInFolder(ctx context.Context, folderID *uuid.UUID, name string) (database.File, error) {
	file, err := s.queries.GetFileByNameInFolder(ctx, database.GetFileByNameInFolderParams{
		FolderID: uuid.NullUUID{UUID: *folderID, Valid: folderID != nil},
		Name:     name,
	})
	if err != nil {
		return database.File{}, err
	}
	return file, nil
}

func (s *Service) ListFilesInFolder(ctx context.Context, userID int32, folderID *uuid.UUID) ([]database.File, error) {
	files, err := s.queries.ListFilesInFolder(ctx, database.ListFilesInFolderParams{
		UserID:   sql.NullInt32{Int32: userID, Valid: true},
		FolderID: uuid.NullUUID{UUID: *folderID, Valid: folderID != nil},
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

func (s *Service) ListFilesRecursive(ctx context.Context, userID int32, folderID uuid.UUID) ([]database.ListFilesRecursiveRow, error) {
	rows, err := s.queries.ListFilesRecursive(ctx, database.ListFilesRecursiveParams{
		UserID: sql.NullInt32{
			Int32: userID,
			Valid: true,
		},
		ID: folderID,
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
) (database.File, error) {

	if name == "" {
		return database.File{}, errors.New("file name is required")
	}

	uID := sql.NullInt32{Int32: userID, Valid: true}

	// 1. Build folder path relative to user root
	var folderPath string
	var fID uuid.NullUUID
	if folderID != nil {
		var err error
		folderPath, err = s.folderService.GetFolderFullPath(ctx, *folderID, userID)
		if err != nil {
			return database.File{}, fmt.Errorf("building folder path: %w", err)
		}
		fID = uuid.NullUUID{UUID: *folderID, Valid: true}
	}

	// 2. Prepare initial relative file path
	filePath := name
	if folderPath != "" {
		filePath = filepath.Join(folderPath, name)
	}

	// 3. Auto-rename if a file with the same name exists
	base := strings.TrimSuffix(name, filepath.Ext(name))
	ext := filepath.Ext(name)
	counter := 1
	for {
		existingFile, err := s.queries.GetFileByNameInFolder(ctx, database.GetFileByNameInFolderParams{
			FolderID: fID,
			Name:     name,
		})
		if errors.Is(err, sql.ErrNoRows) || existingFile.ID == uuid.Nil {
			break // name is available
		} else if err != nil {
			return database.File{}, fmt.Errorf("checking existing file: %w", err)
		}

		// generate new name
		name = fmt.Sprintf("%s (%d)%s", base, counter, ext)
		if folderPath != "" {
			filePath = filepath.Join(folderPath, name)
		} else {
			filePath = name
		}
		counter++
	}

	// 4. Create new DB record
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
		return database.File{}, fmt.Errorf("creating file record: %w", err)
	}

	// 5. Save content to storage
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

// --------------------------------------------------------------------------------------------------------------------------

// Download file
func (s *Service) DownloadFile(ctx context.Context, fileID uuid.UUID, userID int32) (database.File, io.ReadCloser, error) {
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

// --------------------------------------------------------------------------------------------------------------------------

// Delete file
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

// --------------------------------------------------------------------------------------------------------------------------

// Update methods
func (s *Service) UpdateFileMetadata(
	ctx context.Context,
	fileID uuid.UUID,
	userID int32,
	name *string,
	filePath *string,
	sizeBytes *int64,
	mimeType *string,
) error {
	updateName := ""
	if name != nil {
		updateName = *name
	}

	updateFilePath := ""
	if filePath != nil {
		updateFilePath = *filePath
	}

	updateSize := int64(0)
	if sizeBytes != nil {
		updateSize = *sizeBytes
	}

	updateMime := ""
	if mimeType != nil {
		updateMime = *mimeType
	}

	uID := sql.NullInt32{Int32: userID, Valid: true}

	rows, err := s.queries.UpdateFileMetadata(ctx, database.UpdateFileMetadataParams{
		ID:      fileID,
		UserID:  uID,
		Column3: updateName,
		Column4: updateFilePath,
		Column5: updateSize,
		Column6: updateMime,
	})
	if err != nil {
		return fmt.Errorf("updating file metadata: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("file not found or metadata not changed")
	}

	return nil
}

func (s *Service) MoveFile(ctx context.Context, file database.File, destFolderID *uuid.UUID, userID int32) error {
	var relativeNewPath string
	if destFolderID != nil {
		destFolderPath, err := s.folderService.GetFolderFullPath(ctx, *destFolderID, userID)
		if err != nil {
			return fmt.Errorf("cannot build folder path by id: %w", err)
		}
		relativeNewPath = filepath.Join(destFolderPath, file.Name)
	} else {
		// root folder
		relativeNewPath = file.Name
	}

	// 2. Update DB first (ensures name uniqueness + logical consistency)
	if err := s.UpdateFileMetadata(ctx, file.ID, userID, nil, &relativeNewPath, nil, nil); err != nil {
		return fmt.Errorf("updating file path in DB: %w", err)
	}

	// 3. Perform the physical file move
	if err := s.storage.MoveFile(userID, file.FilePath, relativeNewPath); err != nil {
		// rollback DB if storage fails
		rollbackErr := s.UpdateFileMetadata(ctx, file.ID, userID, nil, &file.FilePath, nil, nil)
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
	var newPath string
	if folderPath == "." {
		newPath = newName
	} else {
		newPath = filepath.Join(folderPath, newName)
	}

	// 1. Update DB first (enforces uniqueness)
	err := s.UpdateFileMetadata(ctx, file.ID, userID, &newName, &newPath, nil, nil)
	if err != nil {
		return fmt.Errorf("updating file name in DB: %w", err)
	}

	// 2. Rename file in storage
	if err := s.storage.MoveFile(userID, oldPath, newPath); err != nil {
		// rollback DB if storage fails
		rollbackErr := s.UpdateFileMetadata(ctx, file.ID, userID, &file.Name, &file.FilePath, nil, nil)
		if rollbackErr != nil {
			return fmt.Errorf("storage rename failed (%v), rollback DB also failed: %v", err, rollbackErr)
		}
		return fmt.Errorf("renaming file in storage: %w", err)
	}

	return nil
}

// --------------------------------------------------------------------------------------------------------------------------
