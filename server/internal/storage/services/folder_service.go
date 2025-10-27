package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/middleware"
	"github.com/bellezhang119/cloud-storage/internal/storage/local"
	"github.com/bellezhang119/cloud-storage/internal/util"
	"github.com/google/uuid"
)

type FolderQueries interface {
	CreateFolder(ctx context.Context, arg database.CreateFolderParams) (database.Folder, error)
	GetFolderByID(ctx context.Context, arg database.GetFolderByIDParams) (database.Folder, error)
	ListFoldersByParent(ctx context.Context, arg database.ListFoldersByParentParams) ([]database.Folder, error)
	GetFolderByNameInParent(ctx context.Context, arg database.GetFolderByNameInParentParams) (database.Folder, error)
	DeleteFolders(ctx context.Context, arg database.DeleteFoldersParams) (int64, error)
	ListFoldersRecursive(ctx context.Context, arg database.ListFoldersRecursiveParams) ([]database.ListFoldersRecursiveRow, error)
	UpdateFolderMetadata(ctx context.Context, arg database.UpdateFolderMetadataParams) (int64, error)
	UpdateFoldersParent(ctx context.Context, arg database.UpdateFoldersParentParams) (int64, error)
	GetFolderFullPath(ctx context.Context, arg database.GetFolderFullPathParams) (string, error)
}

type FileService interface {
	UploadFile(
		ctx context.Context,
		folderID *uuid.UUID,
		userID int32,
		name string,
		sizeBytes int64,
		mimeType string,
		content io.Reader,
		overwrite bool,
	) (database.File, error)
	ListFilesRecursive(ctx context.Context, folderID uuid.UUID, userID int32) ([]database.ListFilesRecursiveRow, error)
	GetFileByNameInFolder(ctx context.Context, folderID *uuid.UUID, userID int32, name string) (database.File, error)
	UpdateFileMetadata(
		ctx context.Context,
		fileID uuid.UUID,
		userID int32,
		sizeBytes int64,
		mimeType string,
	) error
	DeleteFiles(ctx context.Context, filesIDs []uuid.UUID, userID int32) error
	UpdateFileNameAndPath(ctx context.Context, fileID uuid.UUID, userID int32, name string, filePath string) error
}

type FolderServiceImpl struct {
	queries FolderQueries
	files   FileService
	local   local.Storage
}

type FolderUploadItem struct {
	Name      string             `json:"name"`      // Name of the file or folder
	IsFolder  bool               `json:"isFolder"`  // Whether this is a folder
	SizeBytes int64              `json:"sizeBytes"` // For files only
	MimeType  string             `json:"mimeType"`  // For files only
	Content   io.ReadCloser      `json:"content"`   // Optional (e.g. uploaded file bytes)
	Children  []FolderUploadItem `json:"children"`  // For folders only
}

// UploadConflict represents a conflict that occurred during upload,
// e.g., when a file or folder already exists and overwrite=false.
type UploadConflict struct {
	Name     string `json:"name"`     // Item name (file/folder)
	Path     string `json:"path"`     // Full relative path (e.g. "Documents/Photos/a.jpg")
	IsFolder bool   `json:"isFolder"` // True if conflict is a folder
}

// UploadResult represents the outcome of an entire upload operation.
type UploadResult struct {
	Created     []string         `json:"created"`     // List of successfully created paths
	Overwritten []string         `json:"overwritten"` // List of overwritten files
	Conflicts   []UploadConflict `json:"conflicts"`   // List of conflicts (skipped files/folders)
}

func NewFolderService(q FolderQueries, local local.Storage) *FolderServiceImpl {
	return &FolderServiceImpl{queries: q, local: local}
}

func (s *FolderServiceImpl) SetFileService(f FileService) {
	s.files = f
}

func (s *FolderServiceImpl) GetFolderByID(ctx context.Context, folderID uuid.UUID, userID int32) (database.Folder, error) {
	return s.queries.GetFolderByID(ctx, database.GetFolderByIDParams{
		ID:     folderID,
		UserID: util.ToNullInt32(&userID),
	})
}

func (s *FolderServiceImpl) ListFoldersByParent(ctx context.Context, userID int32, parentID *uuid.UUID) ([]database.Folder, error) {
	return s.queries.ListFoldersByParent(ctx, database.ListFoldersByParentParams{
		UserID:   util.ToNullInt32(&userID),
		ParentID: util.ToNullUUID(parentID),
	})
}

func (s *FolderServiceImpl) GetFolderFullPath(ctx context.Context, folderID uuid.UUID, userID int32) (string, error) {
	return s.queries.GetFolderFullPath(ctx, database.GetFolderFullPathParams{
		ID:     folderID,
		UserID: util.ToNullInt32(&userID),
	})
}

func (s *FolderServiceImpl) GetFolderByNameInParent(ctx context.Context, userID int32, name string, parentID *uuid.UUID) (database.Folder, error) {
	return s.queries.GetFolderByNameInParent(ctx, database.GetFolderByNameInParentParams{
		UserID:   util.ToNullInt32(&userID),
		Name:     name,
		ParentID: util.ToNullUUID(parentID),
	})
}

func (s *FolderServiceImpl) CreateFolder(ctx context.Context, userID int32, name string, parentID *uuid.UUID) (database.Folder, error) {
	logger := middleware.GetLogger(ctx)
	logger.Info("starting folder creation")

	// 1. Create DB record first
	folder, err := s.queries.CreateFolder(ctx, database.CreateFolderParams{
		UserID:   util.ToNullInt32(&userID),
		Name:     name,
		ParentID: util.ToNullUUID(parentID),
	})
	if err != nil {
		logger.Error("failed to create folder record", "error", err)
		return database.Folder{}, fmt.Errorf("creating folder record: %w", err)
	}

	// 2. Build full path for the new folder
	path, err := s.GetFolderFullPath(ctx, folder.ID, userID)
	if err != nil {
		// rollback DB record
		if rbErr := s.DeleteFolders(ctx, []uuid.UUID{folder.ID}, userID); rbErr != nil {
			logger.Warn("rollback failed after path build error", "error", rbErr)
		}
		logger.Error("failed to build folder path", "error", err)
		return database.Folder{}, fmt.Errorf("building folder path: %w", err)
	}

	// 3. Create folder on disk
	if err := s.local.CreateDirectory(ctx, userID, path); err != nil {
		// rollback DB record
		if rbErr := s.DeleteFolders(ctx, []uuid.UUID{folder.ID}, userID); rbErr != nil {
			logger.Warn("rollback failed after disk create error", "error", rbErr)
		}
		logger.Error("failed to create folder on disk", "error", err)
		return database.Folder{}, fmt.Errorf("creating folder on disk: %w", err)
	}

	logger.Info("folder created successfully", "folder_id", folder.ID, "path", path)
	return folder, nil
}

func (s *FolderServiceImpl) GetZippedFoldersForDownload(ctx context.Context, folderIDs []uuid.UUID, userID int32, w io.Writer) ([]database.Folder, error) {
	logger := middleware.GetLogger(ctx)
	logger.Info("starting to zip folders for download")

	var folderPaths []string
	var foldersMeta []database.Folder

	for _, folderID := range folderIDs {
		meta, err := s.GetFolderByID(ctx, folderID, userID)
		if err != nil {
			logger.Error("failed to fetch folder metadata", "folder_id", folderID, "error", err)
			return nil, err
		}

		path, err := s.GetFolderFullPath(ctx, folderID, userID)
		if err != nil {
			logger.Error("failed to get folder full path", "folder_id", folderID, "error", err)
			return nil, err
		}

		folderPaths = append(folderPaths, path)
		foldersMeta = append(foldersMeta, meta)
	}

	// Stream all folders into zip
	if err := s.local.ZipMultipleFolders(ctx, userID, folderPaths, w); err != nil {
		logger.Error("failed to zip folders", "error", err)
		return nil, err
	}

	logger.Info("folders zipped successfully", "folder_count", len(foldersMeta))
	return foldersMeta, nil
}

func (s *FolderServiceImpl) UploadFolder(
	ctx context.Context,
	userID int32,
	parentID *uuid.UUID,
	items []FolderUploadItem,
	overwrite bool,
	basePath string,
) (UploadResult, error) {
	logger := middleware.GetLogger(ctx)
	logger.Info("starting folder upload", "item_count", len(items))

	result := UploadResult{}

	for _, item := range items {
		currentPath := filepath.Join(basePath, item.Name)

		if item.IsFolder {
			// --- Always merge folders ---
			folder, err := s.GetFolderByNameInParent(ctx, userID, item.Name, parentID)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				logger.Error("failed to check existing folder", "folder_name", item.Name, "error", err)
				return result, fmt.Errorf("checking existing folder: %w", err)
			}

			var folderID uuid.UUID
			if folder.ID != uuid.Nil {
				folderID = folder.ID
			} else {
				newFolder, err := s.CreateFolder(ctx, userID, item.Name, parentID)
				if err != nil {
					logger.Error("failed to create folder", "folder_name", item.Name, "error", err)
					return result, fmt.Errorf("creating folder: %w", err)
				}
				folderID = newFolder.ID
				result.Created = append(result.Created, currentPath)
				logger.Info("folder created", "path", currentPath, "folder_id", folderID)
			}

			// Recursively upload contents
			if len(item.Children) > 0 {
				subResult, err := s.UploadFolder(ctx, userID, &folderID, item.Children, overwrite, currentPath)
				if err != nil {
					logger.Error("failed to upload subfolder", "subfolder_path", currentPath, "error", err)
					return result, fmt.Errorf("uploading subfolder %s: %w", currentPath, err)
				}
				result.Created = append(result.Created, subResult.Created...)
				result.Overwritten = append(result.Overwritten, subResult.Overwritten...)
			}
			continue
		}

		// --- Handle files ---
		file, err := s.files.GetFileByNameInFolder(ctx, parentID, userID, item.Name)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			logger.Error("failed to check existing file", "file_name", item.Name, "error", err)
			return result, fmt.Errorf("checking existing file: %w", err)
		}

		if file.ID != uuid.Nil && overwrite {
			if err := s.files.DeleteFiles(ctx, []uuid.UUID{file.ID}, userID); err != nil {
				logger.Error("failed to delete existing file for overwrite", "file_name", item.Name, "file_id", file.ID, "error", err)
				return result, fmt.Errorf("deleting existing file before overwrite: %w", err)
			}
			result.Overwritten = append(result.Overwritten, currentPath)
			logger.Info("file overwritten", "path", currentPath, "file_id", file.ID)
		} else if file.ID != uuid.Nil && !overwrite {
			logger.Info("file skipped (exists and overwrite=false)", "path", currentPath, "file_id", file.ID)
			continue
		}

		// Upload file
		_, err = s.files.UploadFile(ctx, parentID, userID, item.Name, item.SizeBytes, item.MimeType, item.Content, overwrite)
		if err != nil {
			logger.Error("failed to upload file", "file_name", item.Name, "error", err)
			return result, fmt.Errorf("uploading file: %w", err)
		}
		result.Created = append(result.Created, currentPath)
		logger.Info("file uploaded", "path", currentPath)
	}

	logger.Info("folder upload completed", "created_count", len(result.Created), "overwritten_count", len(result.Overwritten))
	return result, nil
}

func (s *FolderServiceImpl) DeleteFolders(ctx context.Context, folderIDs []uuid.UUID, userID int32) error {
	logger := middleware.GetLogger(ctx)
	logger.Info("starting folder deletion", "folder_count", len(folderIDs))

	if len(folderIDs) == 0 {
		logger.Warn("no folders specified for deletion")
		return fmt.Errorf("no folders specified for deletion")
	}

	// 1. Pre-fetch all folder paths before deletion
	type folderPath struct {
		ID   uuid.UUID
		Path string
	}
	var paths []folderPath
	for _, id := range folderIDs {
		path, err := s.GetFolderFullPath(ctx, id, userID)
		if err != nil {
			logger.Error("failed to fetch folder path", "folder_id", id, "error", err)
			return fmt.Errorf("fetching path for folder %s: %w", id, err)
		}
		paths = append(paths, folderPath{ID: id, Path: path})
	}

	// 2. Delete all folder records (DB cascades handle children)
	rows, err := s.queries.DeleteFolders(ctx, database.DeleteFoldersParams{
		Column1: folderIDs,
		UserID:  util.ToNullInt32(&userID),
	})
	if err != nil {
		logger.Error("failed to delete folders from DB", "error", err)
		return fmt.Errorf("deleting folders from DB: %w", err)
	}
	if rows == 0 {
		logger.Warn("no folders deleted — check ownership or IDs")
		return fmt.Errorf("no folders deleted — check ownership or IDs")
	}
	logger.Info("folders deleted from DB", "deleted_count", rows)

	// 3. Delete directories from storage (best effort)
	for _, p := range paths {
		if err := s.local.DeleteDirectory(ctx, userID, p.Path); err != nil {
			logger.Warn("failed to remove folder from storage", "folder_path", p.Path, "error", err)
		} else {
			logger.Info("folder removed from storage", "folder_path", p.Path)
		}
	}

	logger.Info("folder deletion completed")
	return nil
}

func (s *FolderServiceImpl) UpdateFolderMetadata(ctx context.Context, folderID uuid.UUID, userID int32, name string) error {
	rows, err := s.queries.UpdateFolderMetadata(ctx, database.UpdateFolderMetadataParams{
		ID:     folderID,
		UserID: util.ToNullInt32(&userID),
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

func (s *FolderServiceImpl) UpdateFoldersParent(ctx context.Context, folders []database.Folder, userID int32, newParentID *uuid.UUID) error {
	if len(folders) == 0 {
		return fmt.Errorf("no folders provided")
	}

	ids := make([]uuid.UUID, 0, len(folders))

	// Validate new parent exists
	if newParentID != nil {
		if _, err := s.GetFolderByID(ctx, *newParentID, userID); err != nil {
			return fmt.Errorf("target parent folder does not exist: %w", err)
		}
	}

	for _, f := range folders {
		if newParentID != nil && *newParentID == f.ID {
			return fmt.Errorf("cannot move folder %s under itself", f.ID)
		}
		ids = append(ids, f.ID)
	}

	// Perform bulk update
	if rows, err := s.queries.UpdateFoldersParent(ctx, database.UpdateFoldersParentParams{
		Column1:  ids,
		UserID:   util.ToNullInt32(&userID),
		ParentID: util.ToNullUUID(newParentID),
	}); err != nil {
		return fmt.Errorf("updating folder parents: %w", err)
	} else if rows == 0 {
		return fmt.Errorf("no folders updated (possibly not found or not owned by user)")
	}

	return nil
}

func (s *FolderServiceImpl) MoveFolders(ctx context.Context, folderIDs []uuid.UUID, userID int32, newParentID *uuid.UUID, overwriteFiles bool) error {
	logger := middleware.GetLogger(ctx)

	logger.Info("starting folder move", "folder_count", len(folderIDs))

	if len(folderIDs) == 0 {
		logger.Warn("no folder IDs provided")
		return fmt.Errorf("no folder IDs provided")
	}

	folders := make([]database.Folder, len(folderIDs))
	for i, id := range folderIDs {
		f, err := s.GetFolderByID(ctx, id, userID)
		if err != nil {
			logger.Error("failed to fetch folder", "folder_id", id, "error", err)
			return fmt.Errorf("fetching folder %s: %w", id, err)
		}
		folders[i] = f
	}

	for _, f := range folders {
		if newParentID != nil && *newParentID == f.ID {
			logger.Warn("cannot move folder under itself", "folder_id", f.ID)
			return fmt.Errorf("cannot move folder %s under itself", f.ID)
		}
	}

	existingFoldersMap := map[string]database.Folder{}
	if existingFolders, err := s.ListFoldersByParent(ctx, userID, newParentID); err != nil {
		logger.Error("failed to list folders in target parent", "error", err)
		return fmt.Errorf("fetching folders in target parent: %w", err)
	} else {
		for _, f := range existingFolders {
			existingFoldersMap[f.Name] = f
		}
	}

	var parentPath string
	if newParentID != nil {
		pp, err := s.GetFolderFullPath(ctx, *newParentID, userID)
		if err != nil {
			logger.Error("failed to get new parent path", "new_parent_id", *newParentID, "error", err)
			return fmt.Errorf("building new parent path: %w", err)
		}
		parentPath = pp
	}

	for _, f := range folders {
		oldPath, err := s.GetFolderFullPath(ctx, f.ID, userID)
		if err != nil {
			logger.Error("failed to get old folder path", "folder_id", f.ID, "error", err)
			return fmt.Errorf("building old folder path: %w", err)
		}

		targetFolder := f
		if existing, ok := existingFoldersMap[f.Name]; ok {
			// Merge into existing folder
			targetFolder = existing
			logger.Info("merging folder into existing", "folder_name", f.Name, "target_id", existing.ID)
		} else {
			// Update parent for folders not merging
			if err := s.UpdateFoldersParent(ctx, []database.Folder{f}, userID, newParentID); err != nil {
				logger.Error("failed to update folder parent in DB", "folder_id", f.ID, "error", err)
				return fmt.Errorf("updating folder parent in DB: %w", err)
			}
			logger.Info("updated folder parent in DB", "folder_id", f.ID, "new_parent_id", newParentID)
		}

		newPath := targetFolder.Name
		if newParentID != nil {
			newPath = filepath.Join(parentPath, targetFolder.Name)
		}

		if err := s.local.MoveDirectory(ctx, userID, oldPath, newPath, overwriteFiles); err != nil {
			_ = s.UpdateFoldersParent(ctx, []database.Folder{f}, userID, &f.ParentID.UUID)
			logger.Error("failed to move folder on disk", "folder_id", f.ID, "old_path", oldPath, "new_path", newPath, "error", err)
			return fmt.Errorf("moving folder %s on disk: %w", f.Name, err)
		}
		logger.Info("folder moved on disk", "folder_id", f.ID, "old_path", oldPath, "new_path", newPath)

		if err := s.updateAllChildFilePaths(ctx, userID, f.ID, oldPath, newPath); err != nil {
			logger.Error("failed to update child file paths", "folder_id", f.ID, "error", err)
			return fmt.Errorf("updating child file paths for folder %s: %w", f.Name, err)
		}
		logger.Info("child file paths updated", "folder_id", f.ID)
	}

	logger.Info("folder move completed")
	return nil
}

func (s *FolderServiceImpl) RenameFolder(ctx context.Context, folderID uuid.UUID, newName string, userID int32, overwriteFiles bool) error {
	logger := middleware.GetLogger(ctx)
	logger.Info("starting folder rename")

	if newName == "" {
		logger.Warn("new folder name is empty")
		return fmt.Errorf("new folder name is required")
	}

	// 1. Fetch folder and parent info
	folder, err := s.GetFolderByID(ctx, folderID, userID)
	if err != nil {
		logger.Error("failed to fetch folder", "error", err)
		return fmt.Errorf("fetching folder: %w", err)
	}

	oldPath, err := s.GetFolderFullPath(ctx, folderID, userID)
	if err != nil {
		logger.Error("failed to get old folder path", "error", err)
		return fmt.Errorf("building old folder path: %w", err)
	}

	var parentID *uuid.UUID
	if folder.ParentID.Valid {
		parentID = &folder.ParentID.UUID
	}

	// 2. Check for conflicts in parent
	existing, err := s.GetFolderByNameInParent(ctx, userID, newName, parentID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		logger.Error("failed to check existing folder", "error", err)
		return fmt.Errorf("checking existing folder: %w", err)
	}

	targetFolder := folder
	if existing.ID != uuid.Nil {
		targetFolder = existing
		logger.Info("merging folder into existing folder", "target_folder_id", existing.ID)
	} else {
		if err := s.UpdateFolderMetadata(ctx, folderID, userID, newName); err != nil {
			logger.Error("failed to update folder metadata", "error", err)
			return fmt.Errorf("updating folder name in DB: %w", err)
		}
		logger.Info("folder metadata updated")
	}

	// 3. Build new path
	newPath := targetFolder.Name
	if parentID != nil {
		parentPath, err := s.GetFolderFullPath(ctx, *parentID, userID)
		if err != nil {
			if targetFolder.ID == folder.ID {
				_ = s.UpdateFolderMetadata(ctx, folderID, userID, folder.Name)
			}
			logger.Error("failed to get parent folder path", "error", err)
			return fmt.Errorf("building parent folder path: %w", err)
		}
		newPath = filepath.Join(parentPath, targetFolder.Name)
	}

	// 4. Move folder on disk
	if err := s.local.MoveDirectory(ctx, userID, oldPath, newPath, overwriteFiles); err != nil {
		if targetFolder.ID == folder.ID {
			_ = s.UpdateFolderMetadata(ctx, folderID, userID, folder.Name)
		}
		logger.Error("failed to move folder on disk", "old_path", oldPath, "new_path", newPath, "error", err)
		return fmt.Errorf("moving folder on disk: %w", err)
	}
	logger.Info("folder moved on disk", "old_path", oldPath, "new_path", newPath)

	// 5. Update child file paths
	if err := s.updateAllChildFilePaths(ctx, userID, folderID, oldPath, newPath); err != nil {
		logger.Error("failed to update child file paths", "error", err)
		return fmt.Errorf("updating child file paths: %w", err)
	}
	logger.Info("child file paths updated")

	logger.Info("folder rename completed successfully")
	return nil
}

// updateAllChildFilePaths internal helper
func (s *FolderServiceImpl) updateAllChildFilePaths(ctx context.Context, userID int32, folderID uuid.UUID, oldPath, newPath string) error {
	files, err := s.files.ListFilesRecursive(ctx, folderID, userID)
	if err != nil {
		return fmt.Errorf("listing files in folder: %w", err)
	}

	for _, f := range files {
		relPath, err := filepath.Rel(oldPath, f.FilePath)
		if err != nil {
			return fmt.Errorf("calculating relative path for file %s: %w", f.FilePath, err)
		}
		newFilePath := filepath.Join(newPath, relPath)
		if err := s.files.UpdateFileNameAndPath(ctx, f.FileID, userID, f.Name, newFilePath); err != nil {
			return fmt.Errorf("updating file path in DB for file %s: %w", f.FileID, err)
		}
	}

	return nil
}
