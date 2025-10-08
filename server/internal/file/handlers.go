package file

import (
	"context"
	"io"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/google/uuid"
)

type ServiceInterface interface {
	CreateFile(
		ctx context.Context,
		folderID *uuid.UUID,
		userID int32,
		name string,
		sizeBytes int64,
		mimeType string,
		content io.Reader,
	) (database.File, error)
	GetFileByID(ctx context.Context, id uuid.UUID) (database.File, error)
	GetFileByNameInFolder(ctx context.Context, folderID uuid.UUID, name string) (database.File, error)
	ListFilesInFolder(ctx context.Context, folderID *uuid.UUID) ([]database.File, error)
	PermanentlyDeleteFile(ctx context.Context, fileID uuid.UUID, userID int32) error
	RestoreFile(ctx context.Context, fileID uuid.UUID, userID int32) error
	TrashFile(ctx context.Context, fileID uuid.UUID, userID int32) error
	UpdateFileMetadata(
		ctx context.Context,
		fileID uuid.UUID,
		name string,
		userID int32,
	) error
	MoveFile(
		ctx context.Context,
		fileID uuid.UUID,
		oldPath, newPath string,
		userID int32,
	) error
}
