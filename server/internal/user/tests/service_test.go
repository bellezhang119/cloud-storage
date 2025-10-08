package tests

import (
	"context"
	"testing"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockQueries struct {
	mock.Mock
}

func (m *MockQueries) GetUserByEmail(ctx context.Context, email string) (database.User, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(database.User), args.Error(1)
}

func (m *MockQueries) GetUserByID(ctx context.Context, userID int32) (database.User, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(database.User), args.Error(1)
}

func (m *MockQueries) UpdateUserPassword(ctx context.Context, params database.UpdateUserPasswordParams) (int64, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockQueries) UpdateUsedStorage(ctx context.Context, params database.UpdateUsedStorageParams) (int64, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockQueries) DeleteUser(ctx context.Context, userID int32) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func TestUpdatePassword(t *testing.T) {
	mockQ := new(MockQueries)
	svc := user.NewService(mockQ)
	ctx := context.Background()
	userID := int32(1)
	newPassword := "newpassword"

	mockQ.On("UpdateUserPassword", ctx, mock.MatchedBy(func(params database.UpdateUserPasswordParams) bool {
		return params.ID == userID && params.PasswordHash != ""
	})).Return(int64(1), nil)

	err := svc.UpdateUserPassword(ctx, userID, newPassword)
	assert.NoError(t, err)
	mockQ.AssertExpectations(t)
}

func TestUpdateUsedStorage(t *testing.T) {
	mockQ := new(MockQueries)
	svc := user.NewService(mockQ)
	ctx := context.Background()
	userID := int32(1)
	newStorage := int64(1024)

	mockQ.On("UpdateUsedStorage", ctx, database.UpdateUsedStorageParams{
		ID:          userID,
		UsedStorage: newStorage,
	}).Return(int64(1), nil)

	err := svc.UpdateUsedStorage(ctx, userID, newStorage)
	assert.NoError(t, err)
	mockQ.AssertExpectations(t)
}

func TestDeleteUser(t *testing.T) {
	mockQ := new(MockQueries)
	svc := user.NewService(mockQ)
	ctx := context.Background()
	userID := int32(1)

	mockQ.On("DeleteUser", ctx, userID).Return(int64(1), nil)

	err := svc.DeleteUser(ctx, userID)
	assert.NoError(t, err)
	mockQ.AssertExpectations(t)
}
