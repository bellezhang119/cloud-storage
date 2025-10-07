package user

import (
	"context"
	"fmt"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/util"
)

type Queries interface {
	UpdateUserPassword(ctx context.Context, arg database.UpdateUserPasswordParams) (int64, error)
	UpdateUsedStorage(ctx context.Context, arg database.UpdateUsedStorageParams) (int64, error)
	DeleteUser(ctx context.Context, id int32) (int64, error)
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

	rowsAffected, err := s.queries.UpdateUserPassword(ctx, database.UpdateUserPasswordParams{
		ID:           userID,
		PasswordHash: hashed,
	})
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no user found with id %d", userID)
	}
	return nil
}

func (s *Service) UpdateStorage(ctx context.Context, userID int32, newUsedStorage int64) error {
	rowsAffected, err := s.queries.UpdateUsedStorage(ctx, database.UpdateUsedStorageParams{
		ID:          userID,
		UsedStorage: newUsedStorage,
	})
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no user found with id %d", userID)
	}
	return nil
}

func (s *Service) Delete(ctx context.Context, userID int32) error {
	rowsAffected, err := s.queries.DeleteUser(ctx, userID)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no user found with id %d", userID)
	}
	return nil
}
