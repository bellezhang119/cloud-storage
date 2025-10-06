package folder

import (
	"context"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/google/uuid"
)

type Queries interface {
	CreateFolder(ctx context.Context, arg database.CreateFolderParams) (database.Folder, error)
	DeleteFolder(ctx context.Context, arg database.DeleteFolderParams) error
	GetFolderByID(ctx context.Context, id uuid.UUID) (database.Folder, error)
	ListFoldersByParent(ctx context.Context, arg database.ListFoldersByParentParams) ([]database.Folder, error)
}

type Service struct {
	queries Queries
}

func NewService(q Queries) *Service {
	return &Service{queries: q}
}

func (s *Service) GetFolderByID(ctx context.Context, id uuid.UUID) (database.Folder, error) {
	return s.queries.GetFolderByID(ctx, id)
}
