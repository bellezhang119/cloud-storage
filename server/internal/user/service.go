package user

import (
	"context"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/util"
)

type Queries interface {
	UpdateUserPassword(ctx context.Context, params database.UpdateUserPasswordParams) error
	UpdateUsedStorage(ctx context.Context, params database.UpdateUsedStorageParams) error
	DeleteUser(ctx context.Context, id int32) error
	GetUserByID(ctx context.Context, id int32) (database.User, error)
	GetUserByEmail(ctx context.Context, email string) (database.User, error)
}

type Service struct {
	queries Queries
}

func NewService(q Queries) *Service {
	return &Service{queries: q}
}

func (s *Service) GetUserByEmail(ctx context.Context, email string) (database.User, error) {
	return s.queries.GetUserByEmail(ctx, email)
}

func (s *Service) GetUserByID(ctx context.Context, id int32) (database.User, error) {
	return s.queries.GetUserByID(ctx, id)
}

func (s *Service) UpdatePassword(ctx context.Context, userID int32, newPassword string) error {
	hashed, err := util.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.queries.UpdateUserPassword(ctx, database.UpdateUserPasswordParams{
		ID:           userID,
		PasswordHash: hashed,
	})
}

func (s *Service) UpdateStorage(ctx context.Context, userID int32, newUsedStorage int64) error {
	return s.queries.UpdateUsedStorage(ctx, database.UpdateUsedStorageParams{
		ID:          userID,
		UsedStorage: newUsedStorage,
	})
}

func (s *Service) Delete(ctx context.Context, userID int32) error {
	return s.queries.DeleteUser(ctx, userID)
}
