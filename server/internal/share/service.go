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

func (s *Service) CheckUserFileAccess(ctx context.Context, fileID uuid.UUID, userID int32) (bool, error) {
	logger := middleware.GetLogger(ctx)
	logger.Info("checking user file access")

	access, err := s.queries.CheckUserFileAccess(ctx, database.CheckUserFileAccessParams{
		FileID:       util.ToNullUUID(&fileID),
		SharedUserID: util.ToNullInt32(&userID),
	})

	if err != nil {
		logger.Error("failed to check user file access", "error", err)
		return false, fmt.Errorf("failed to check user file access: %w", err)
	}

	logger.Info("user file access result", "file_id", fileID, "access", access)
	return access, nil
}

func (s *Service) CreateFileShare(ctx context.Context, fileID uuid.UUID, userID int32) (database.FileShare, error) {
	logger := middleware.GetLogger(ctx)
	logger.Info("creating file share")

	fileShare, err := s.queries.CreateFileShare(ctx, database.CreateFileShareParams{
		FileID:       util.ToNullUUID(&fileID),
		SharedUserID: util.ToNullInt32(&userID),
	})

	if err != nil {
		logger.Error("failed to create file share", "error", err)
		return database.FileShare{}, fmt.Errorf("failed to create file share: %w", err)
	}

	logger.Info("file share created", "file_id", fileShare.FileID)
	return fileShare, nil
}

func (s *Service) DeleteFileShare(ctx context.Context, fileID uuid.UUID, userID int32) error {
	logger := middleware.GetLogger(ctx)
	logger.Info("deleting file share")

	rows, err := s.queries.DeleteFileShare(ctx, database.DeleteFileShareParams{
		FileID:       util.ToNullUUID(&fileID),
		SharedUserID: util.ToNullInt32(&userID),
	})

	if err != nil {
		logger.Error("failed to delete file share", "error", err)
		return fmt.Errorf("failed to delete file share: %w", err)
	}

	if rows == 0 {
		logger.Warn("no file shares deleted", "file_id", fileID)
		return fmt.Errorf("no file shares deleted - check id")
	}

	logger.Info("file share deleted", "file_id", fileID)
	return nil
}

func (s *Service) GetFileShare(ctx context.Context, fileID uuid.UUID, userID int32) (database.FileShare, error) {
	logger := middleware.GetLogger(ctx)
	logger.Info("getting file share")

	fileShare, err := s.queries.GetFileShare(ctx, database.GetFileShareParams{
		FileID:       util.ToNullUUID(&fileID),
		SharedUserID: util.ToNullInt32(&userID),
	})

	if err != nil {
		logger.Error("failed to get file share", "error", err)
		return database.FileShare{}, fmt.Errorf("failed to get file share: %w", err)
	}

	logger.Info("file share retrieved", "file_id", fileShare.FileID)
	return fileShare, nil
}

func (s *Service) ListFileShares(ctx context.Context, fileID uuid.UUID) ([]database.FileShare, error) {
	logger := middleware.GetLogger(ctx)
	logger.Info("listing file shares")

	fileShares, err := s.queries.ListFileShares(ctx, util.ToNullUUID(&fileID))
	if err != nil {
		logger.Error("failed to list file shares", "error", err)
		return nil, fmt.Errorf("failed to get file shares: %w", err)
	}

	logger.Info("file shares listed", "file_id", fileID, "count", len(fileShares))
	return fileShares, nil
}

func (s *Service) ListFilesSharedWithUser(ctx context.Context, userID int32) ([]database.File, error) {
	logger := middleware.GetLogger(ctx)
	logger.Info("listing files shared with user")

	files, err := s.queries.ListFilesSharedWithUser(ctx, util.ToNullInt32(&userID))
	if err != nil {
		logger.Error("failed to list files shared with user", "error", err)
		return nil, fmt.Errorf("failed to get files shared with user: %w", err)
	}

	logger.Info("files shared with user retrieved", "count", len(files))
	return files, nil
}

// Share folders

func (s *Service) CheckUserFolderAccess(ctx context.Context, folderID uuid.UUID, userID int32) (bool, error) {
	logger := middleware.GetLogger(ctx)
	logger.Info("checking user folder access")

	access, err := s.queries.CheckUserFolderAccess(ctx, database.CheckUserFolderAccessParams{
		ID:           folderID,
		SharedUserID: util.ToNullInt32(&userID),
	})

	if err != nil {
		logger.Error("failed to check user folder access", "error", err)
		return false, fmt.Errorf("failed to check user folder access: %w", err)
	}

	logger.Info("user folder access result", "folder_id", folderID, "access", access)
	return access, nil
}

func (s *Service) CreateFolderShare(ctx context.Context, folderID uuid.UUID, userID int32) (database.FolderShare, error) {
	logger := middleware.GetLogger(ctx)
	logger.Info("creating folder share")

	folderShare, err := s.queries.CreateFolderShare(ctx, database.CreateFolderShareParams{
		FolderID:     util.ToNullUUID(&folderID),
		SharedUserID: util.ToNullInt32(&userID),
	})

	if err != nil {
		logger.Error("failed to create folder share", "error", err)
		return database.FolderShare{}, fmt.Errorf("failed to create folder share: %w", err)
	}

	logger.Info("folder share created", "folder_id", folderShare.FolderID)
	return folderShare, nil
}

func (s *Service) DeleteFolderShare(ctx context.Context, folderID uuid.UUID, userID int32) error {
	logger := middleware.GetLogger(ctx)
	logger.Info("deleting folder share")

	rows, err := s.queries.DeleteFolderShare(ctx, database.DeleteFolderShareParams{
		FolderID:     util.ToNullUUID(&folderID),
		SharedUserID: util.ToNullInt32(&userID),
	})

	if err != nil {
		logger.Error("failed to delete folder share", "error", err)
		return fmt.Errorf("failed to delete folder share: %w", err)
	}

	if rows == 0 {
		logger.Warn("no folder shares deleted", "folder_id", folderID)
		return fmt.Errorf("no folder shares deleted - check id")
	}

	logger.Info("folder share deleted", "folder_id", folderID)
	return nil
}

func (s *Service) GetFolderShare(ctx context.Context, folderID uuid.UUID, userID int32) (database.FolderShare, error) {
	logger := middleware.GetLogger(ctx)
	logger.Info("getting folder share")

	folderShare, err := s.queries.GetFolderShare(ctx, database.GetFolderShareParams{
		FolderID:     util.ToNullUUID(&folderID),
		SharedUserID: util.ToNullInt32(&userID),
	})

	if err != nil {
		logger.Error("failed to get folder share", "error", err)
		return database.FolderShare{}, fmt.Errorf("failed to get folder share: %w", err)
	}

	logger.Info("folder share retrieved", "folder_id", folderShare.FolderID)
	return folderShare, nil
}

func (s *Service) GetSharedFolderContent(ctx context.Context, folderID uuid.UUID, userID int32) ([]database.Folder, []database.File, error) {
	logger := middleware.GetLogger(ctx)
	logger.Info("getting shared folder content")

	access, err := s.CheckUserFolderAccess(ctx, folderID, userID)
	if err != nil {
		logger.Error("failed to check folder access", "error", err)
		return nil, nil, fmt.Errorf("failed to check access: %w", err)
	}

	if !access {
		logger.Warn("unauthorized access to folder", "folder_id", folderID)
		return nil, nil, fmt.Errorf("unauthorized access")
	}

	folders, err := s.queries.GetSharedSubfolders(ctx, util.ToNullUUID(&folderID))
	if err != nil {
		logger.Error("failed to get shared subfolders", "error", err)
		return nil, nil, fmt.Errorf("failed to get shared sub folders: %w", err)
	}

	files, err := s.queries.GetFilesInSharedFolder(ctx, util.ToNullUUID(&folderID))
	if err != nil {
		logger.Error("failed to get files in shared folder", "error", err)
		return nil, nil, fmt.Errorf("failed to get files in shared folder: %w", err)
	}

	logger.Info("retrieved shared folder content", "folder_id", folderID, "folder_count", len(folders), "file_count", len(files))
	return folders, files, nil
}

func (s *Service) ListFolderShares(ctx context.Context, folderID uuid.UUID) ([]database.FolderShare, error) {
	logger := middleware.GetLogger(ctx)
	logger.Info("listing folder shares")

	folderShares, err := s.queries.ListFolderShares(ctx, util.ToNullUUID(&folderID))
	if err != nil {
		logger.Error("failed to list folder shares", "error", err)
		return nil, fmt.Errorf("failed to get folder shares: %w", err)
	}

	logger.Info("folder shares listed", "folder_id", folderID, "count", len(folderShares))
	return folderShares, nil
}

func (s *Service) ListFoldersSharedWithUser(ctx context.Context, userID int32) ([]database.Folder, error) {
	logger := middleware.GetLogger(ctx)
	logger.Info("listing folders shared with user")

	folders, err := s.queries.ListFoldersSharedWithUser(ctx, util.ToNullInt32(&userID))
	if err != nil {
		logger.Error("failed to list folders shared with user", "error", err)
		return nil, fmt.Errorf("failed to get folders shared with user: %w", err)
	}

	logger.Info("folders shared with user retrieved", "count", len(folders))
	return folders, nil
}
