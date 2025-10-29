package user

import (
	"context"
	"fmt"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/middleware"
	"github.com/bellezhang119/cloud-storage/internal/util"
)

type Queries interface {
	UpdateUserPassword(ctx context.Context, arg database.UpdateUserPasswordParams) (int64, error)
	UpdateUsedStorage(ctx context.Context, arg database.UpdateUsedStorageParams) (int64, error)
	GetUsedStorageByID(ctx context.Context, id int32) (int64, error)
	DeleteUser(ctx context.Context, id int32) (int64, error)
	GetUserByID(ctx context.Context, id int32) (database.User, error)
	GetUserByEmail(ctx context.Context, email string) (database.User, error)
	AdjustUsedStorage(ctx context.Context, arg database.AdjustUsedStorageParams) (int64, error)
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

func (s *Service) UpdateUserPassword(ctx context.Context, userID int32, newPassword string) error {
	logger := middleware.GetLogger(ctx)
	logger.Info("updating user password")

	hashed, err := util.HashPassword(newPassword)
	if err != nil {
		logger.Error("failed to hash password", "error", err)
		return err
	}

	rowsAffected, err := s.queries.UpdateUserPassword(ctx, database.UpdateUserPasswordParams{
		ID:           userID,
		PasswordHash: hashed,
	})
	if err != nil {
		logger.Error("failed to update user password in database", "error", err)
		return err
	}
	if rowsAffected == 0 {
		logger.Warn("no user found for password update")
		return fmt.Errorf("no user found with id %d", userID)
	}

	logger.Info("password updated successfully")
	return nil
}

func (s *Service) GetUsedStorageByID(ctx context.Context, userID int32) (int64, error) {
	logger := middleware.GetLogger(ctx)
	logger.Info("getting used storage")

	rowsAffected, err := s.queries.GetUsedStorageByID(ctx, userID)
	if err != nil {
		logger.Error("failed to fetch used storage", "error", err)
		return 0, err
	}

	if rowsAffected == 0 {
		logger.Warn("no used storage found with id %d", userID)
		return 0, fmt.Errorf("no used storage found with id %d", userID)
	}

	logger.Info("used storage fetched successfully")
	return rowsAffected, nil
}

func (s *Service) UpdateUsedStorage(ctx context.Context, userID int32, newUsedStorage int64) error {
	logger := middleware.GetLogger(ctx)
	logger.Info("updating user's used storage")

	rowsAffected, err := s.queries.UpdateUsedStorage(ctx, database.UpdateUsedStorageParams{
		ID:          userID,
		UsedStorage: newUsedStorage,
	})
	if err != nil {
		logger.Error("failed to update used storage", "error", err)
		return err
	}
	if rowsAffected == 0 {
		logger.Warn("no user found for storage update")
		return fmt.Errorf("no user found with id %d", userID)
	}

	logger.Info("used storage updated successfully")
	return nil
}

func (s *Service) AdjustUsedStorage(ctx context.Context, userID int32, delta int64) error {
	logger := middleware.GetLogger(ctx)
	logger.Info("adjusting user's used storage")

	rows, err := s.queries.AdjustUsedStorage(ctx, database.AdjustUsedStorageParams{
		ID:          userID,
		UsedStorage: delta,
	})

	if err != nil {
		logger.Error("failed to adjust used storage", "error", err)
		return err
	}

	if rows == 0 {
		logger.Warn("no user found for storage adjust")
		return fmt.Errorf("no user found for user id %d", userID)
	}

	logger.Info("used storage adjusted successfully")
	return nil
}

func (s *Service) DeleteUser(ctx context.Context, userID int32) error {
	logger := middleware.GetLogger(ctx)
	logger.Info("deleting user")

	rowsAffected, err := s.queries.DeleteUser(ctx, userID)
	if err != nil {
		logger.Error("failed to delete user", "error", err)
		return err
	}
	if rowsAffected == 0 {
		logger.Warn("no user found for deletion")
		return fmt.Errorf("no user found with id %d", userID)
	}

	logger.Info("user deleted successfully")
	return nil
}
