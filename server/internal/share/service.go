package share

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/middleware"
	"github.com/bellezhang119/cloud-storage/internal/util"
	"github.com/google/uuid"
)

type Queries interface {
	CheckUserFileAccess(ctx context.Context, arg database.CheckUserFileAccessParams) (bool, error)
	CreateFileShare(ctx context.Context, arg database.CreateFileShareParams) (database.FileShare, error)
	DeleteFileShare(ctx context.Context, arg database.DeleteFileShareParams) (int64, error)
	GetFileShare(ctx context.Context, arg database.GetFileShareParams) (database.FileShare, error)
	ListFileShares(ctx context.Context, fileID uuid.NullUUID) ([]database.FileShare, error)
	ListFilesSharedWithUser(ctx context.Context, sharedUserID sql.NullInt32) ([]database.File, error)
	CheckUserFolderAccess(ctx context.Context, arg database.CheckUserFolderAccessParams) (bool, error)
	CreateFolderShare(ctx context.Context, arg database.CreateFolderShareParams) (database.FolderShare, error)
	DeleteFolderShare(ctx context.Context, arg database.DeleteFolderShareParams) (int64, error)
	GetFolderShare(ctx context.Context, arg database.GetFolderShareParams) (database.FolderShare, error)
	GetSharedSubfolders(ctx context.Context, parentID uuid.NullUUID) ([]database.Folder, error)
	GetFilesInSharedFolder(ctx context.Context, folderID uuid.NullUUID) ([]database.File, error)
	ListFolderShares(ctx context.Context, folderID uuid.NullUUID) ([]database.FolderShare, error)
	ListFoldersSharedWithUser(ctx context.Context, sharedUserID sql.NullInt32) ([]database.Folder, error)
}

type Service struct {
	queries Queries
}

func NewService(q Queries) *Service {
	return &Service{queries: q}
}

// Share files

func (s *Service) CheckUserFileAccess(ctx context.Context, fileID uuid.UUID) (bool, error) {
	userID, _ := middleware.GetUserID(ctx)

	access, err := s.queries.CheckUserFileAccess(ctx, database.CheckUserFileAccessParams{
		FileID:       util.ToNullUUID(&fileID),
		SharedUserID: util.ToNullInt32(&userID),
	})

	if err != nil {
		return false, fmt.Errorf("failed to check user file access: %w", err)
	}

	return access, nil
}

func (s *Service) CreateFileShare(ctx context.Context, fileID uuid.UUID) (database.FileShare, error) {
	userID, _ := middleware.GetUserID(ctx)

	fileShare, err := s.queries.CreateFileShare(ctx, database.CreateFileShareParams{
		FileID:       util.ToNullUUID(&fileID),
		SharedUserID: util.ToNullInt32(&userID),
	})

	if err != nil {
		return database.FileShare{}, fmt.Errorf("failed to create file share: %w", err)
	}

	return fileShare, nil
}

func (s *Service) DeleteFileShare(ctx context.Context, fileID uuid.UUID) error {
	userID, _ := middleware.GetUserID(ctx)

	rows, err := s.queries.DeleteFileShare(ctx, database.DeleteFileShareParams{
		FileID:       util.ToNullUUID(&fileID),
		SharedUserID: util.ToNullInt32(&userID),
	})

	if err != nil {
		return fmt.Errorf("failed to delete file share %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("no file shares deleted - check id")
	}

	return nil
}

func (s *Service) GetFileShare(ctx context.Context, fileID uuid.UUID) (database.FileShare, error) {
	userID, _ := middleware.GetUserID(ctx)

	fileShare, err := s.queries.GetFileShare(ctx, database.GetFileShareParams{
		FileID:       util.ToNullUUID(&fileID),
		SharedUserID: util.ToNullInt32(&userID),
	})

	if err != nil {
		return database.FileShare{}, fmt.Errorf("failed to get file share: %w", err)
	}

	return fileShare, nil
}

func (s *Service) ListFileShares(ctx context.Context, fileID uuid.UUID) ([]database.FileShare, error) {
	fileShares, err := s.queries.ListFileShares(ctx, util.ToNullUUID(&fileID))

	if err != nil {
		return []database.FileShare{}, fmt.Errorf("failed to get file shares: %w", err)
	}

	return fileShares, nil
}

func (s *Service) ListFilesSharedWithUser(ctx context.Context) ([]database.File, error) {
	userID, _ := middleware.GetUserID(ctx)

	files, err := s.queries.ListFilesSharedWithUser(ctx, util.ToNullInt32(&userID))

	if err != nil {
		return []database.File{}, fmt.Errorf("failed to get files shared with user: %w", err)
	}

	return files, nil
}

// Share folders

func (s *Service) CheckUserFolderAccess(ctx context.Context, folderID uuid.UUID) (bool, error) {
	userID, _ := middleware.GetUserID(ctx)

	access, err := s.queries.CheckUserFolderAccess(ctx, database.CheckUserFolderAccessParams{
		ID:           folderID,
		SharedUserID: util.ToNullInt32(&userID),
	})

	if err != nil {
		return false, fmt.Errorf("failed to check user folder access: %w", err)
	}

	return access, nil
}

func (s *Service) CreateFolderShare(ctx context.Context, folderID uuid.UUID) (database.FolderShare, error) {
	userID, _ := middleware.GetUserID(ctx)

	folderShare, err := s.queries.CreateFolderShare(ctx, database.CreateFolderShareParams{
		FolderID:     util.ToNullUUID(&folderID),
		SharedUserID: util.ToNullInt32(&userID),
	})

	if err != nil {
		return database.FolderShare{}, fmt.Errorf("failed to create folder share: %w", err)
	}

	return folderShare, nil
}

func (s *Service) DeleteFolderShare(ctx context.Context, folderID uuid.UUID) error {
	userID, _ := middleware.GetUserID(ctx)

	rows, err := s.queries.DeleteFolderShare(ctx, database.DeleteFolderShareParams{
		FolderID:     util.ToNullUUID(&folderID),
		SharedUserID: util.ToNullInt32(&userID),
	})

	if err != nil {
		return fmt.Errorf("failed to delete folder share: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("no folder shares deleted - check id")
	}

	return nil
}

func (s *Service) GetFolderShare(ctx context.Context, folderID uuid.UUID) (database.FolderShare, error) {
	userID, _ := middleware.GetUserID(ctx)

	folderShare, err := s.queries.GetFolderShare(ctx, database.GetFolderShareParams{
		FolderID:     util.ToNullUUID(&folderID),
		SharedUserID: util.ToNullInt32(&userID),
	})

	if err != nil {
		return database.FolderShare{}, fmt.Errorf("failed to get folder share: %w", err)
	}

	return folderShare, nil
}

func (s *Service) GetSharedFolderContent(ctx context.Context, folderID uuid.UUID) ([]database.Folder, []database.File, error) {
	access, err := s.CheckUserFolderAccess(ctx, folderID)

	if !access {
		return []database.Folder{}, []database.File{}, fmt.Errorf("unauthorized access: %w", err)
	}

	if err != nil {
		return []database.Folder{}, []database.File{}, fmt.Errorf("failed to check access: %w", err)
	}

	folders, err := s.queries.GetSharedSubfolders(ctx, util.ToNullUUID(&folderID))

	if err != nil {
		return []database.Folder{}, []database.File{}, fmt.Errorf("failed to get shared sub folders: %w", err)
	}

	files, err := s.queries.GetFilesInSharedFolder(ctx, util.ToNullUUID(&folderID))

	if err != nil {
		return []database.Folder{}, []database.File{}, fmt.Errorf("failed to get files in shared folder: %w", err)
	}

	return folders, files, nil
}

func (s *Service) ListFolderShares(ctx context.Context, folderID uuid.UUID) ([]database.FolderShare, error) {
	folderShares, err := s.queries.ListFolderShares(ctx, util.ToNullUUID(&folderID))

	if err != nil {
		return []database.FolderShare{}, fmt.Errorf("failed to get folder shares: %w", err)
	}

	return folderShares, nil
}

func (s *Service) ListFoldersSharedWithUser(ctx context.Context) ([]database.Folder, error) {
	userID, _ := middleware.GetUserID(ctx)

	folders, err := s.queries.ListFoldersSharedWithUser(ctx, util.ToNullInt32(&userID))

	if err != nil {
		return []database.Folder{}, fmt.Errorf("failed to get folders shared with user: %w", err)
	}

	return folders, nil
}
