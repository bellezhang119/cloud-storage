package folder

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/google/uuid"
)

type Queries interface {
	CreateFolder(ctx context.Context, arg database.CreateFolderParams) (database.Folder, error)
	DeleteFolder(ctx context.Context, arg database.DeleteFolderParams) (int64, error)
	GetFolderByID(ctx context.Context, id uuid.UUID) (database.Folder, error)
	ListFoldersByParent(ctx context.Context, arg database.ListFoldersByParentParams) ([]database.Folder, error)
}

type Service struct {
	queries Queries
}

func NewService(q Queries) *Service {
	return &Service{queries: q}
}

func (s *Service) CreateFolder(ctx context.Context, userID int32, name string, parentID uuid.NullUUID) (database.Folder, error) {
	return s.queries.CreateFolder(ctx, database.CreateFolderParams{
		UserID: sql.NullInt32{
			Int32: userID,
			Valid: true,
		},
		Name:     name,
		ParentID: parentID,
	})
}

func (s *Service) DeleteFolder(ctx context.Context, id uuid.UUID, userID int32) error {
	rows, err := s.queries.DeleteFolder(ctx, database.DeleteFolderParams{
		ID: id,
		UserID: sql.NullInt32{
			Int32: userID,
			Valid: true,
		},
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("folder not found or already deleted")
	}
	return nil
}

func (s *Service) GetFolderByID(ctx context.Context, id uuid.UUID) (database.Folder, error) {
	return s.queries.GetFolderByID(ctx, id)
}

func (s *Service) ListFoldersByParent(ctx context.Context, userID int32, parentID uuid.UUID) ([]database.Folder, error) {
	return s.queries.ListFoldersByParent(ctx, database.ListFoldersByParentParams{
		UserID: sql.NullInt32{
			Int32: userID,
			Valid: true,
		},
		ParentID: uuid.NullUUID{
			UUID:  parentID,
			Valid: true,
		},
	})
}
