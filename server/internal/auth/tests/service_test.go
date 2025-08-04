package tests

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/bellezhang119/cloud-storage/internal/auth"
	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockQueries struct {
	mock.Mock
}

func (m *MockQueries) CreateUser(ctx context.Context, params database.CreateUserParams) (database.User, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(database.User), args.Error(1)
}

func (m *MockQueries) GetUserByVerificationToken(ctx context.Context, token sql.NullString) (database.User, error) {
	args := m.Called(ctx, token)
	return args.Get(0).(database.User), args.Error(1)
}

func (m *MockQueries) MarkUserAsVerified(ctx context.Context, id int32) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockQueries) GetUserByEmail(ctx context.Context, email string) (database.User, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(database.User), args.Error(1)
}

func (m *MockQueries) UpdateVerificationToken(ctx context.Context, params database.UpdateVerificationTokenParams) error {
	args := m.Called(ctx, params)
	return args.Error(0)
}

func (m *MockQueries) InsertRefreshToken(ctx context.Context, params database.InsertRefreshTokenParams) error {
	args := m.Called(ctx, params)
	return args.Error(0)
}

func (m *MockQueries) GetRefreshToken(ctx context.Context, hash string) (database.GetRefreshTokenRow, error) {
	args := m.Called(ctx, hash)
	return args.Get(0).(database.GetRefreshTokenRow), args.Error(1)
}

func (m *MockQueries) GetUserByID(ctx context.Context, id int32) (database.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(database.User), args.Error(1)
}

func (m *MockQueries) RevokeRefreshToken(ctx context.Context, hash string) error {
	args := m.Called(ctx, hash)
	return args.Error(0)
}

func TestCreateUser_Success(t *testing.T) {
	mockQ := new(MockQueries)
	svc := auth.NewService(mockQ)
	ctx := context.Background()
	email := "test@example.com"
	password := "password123"

	mockQ.On("CreateUser", mock.Anything, mock.Anything).Return(database.User{Email: email}, nil)

	user, err := svc.CreateUser(ctx, email, password)
	assert.NoError(t, err)
	assert.Equal(t, email, user.Email)
	mockQ.AssertExpectations(t)
}

func TestVerifyUserByToken_TokenExpired(t *testing.T) {
	mockQ := new(MockQueries)
	svc := auth.NewService(mockQ)
	ctx := context.Background()
	token := "expiredtoken"

	expiredUser := database.User{
		VerificationTokenExpiry: sql.NullTime{
			Time:  time.Now().Add(-1 * time.Hour),
			Valid: true,
		},
	}

	mockQ.On("GetUserByVerificationToken", mock.Anything, mock.Anything).Return(expiredUser, nil)

	err := svc.VerifyUserByToken(ctx, token)
	assert.EqualError(t, err, "Token has expired")
	mockQ.AssertExpectations(t)
}

func TestAuthenticateUser_InvalidPassword(t *testing.T) {
	mockQ := new(MockQueries)
	svc := auth.NewService(mockQ)
	ctx := context.Background()
	email := "user@example.com"

	hashed, _ := util.HashPassword("rightpass")
	mockQ.On("GetUserByEmail", ctx, email).Return(database.User{PasswordHash: hashed}, nil)

	_, err := svc.AuthenticateUser(ctx, email, "wrongpass")
	assert.Error(t, err)
	mockQ.AssertExpectations(t)
}

func TestGenerateJWTTokens_Success(t *testing.T) {
	mockQ := new(MockQueries)
	svc := auth.NewService(mockQ)
	ctx := context.Background()
	user := database.User{ID: 1, Email: "user@example.com"}

	mockQ.On("InsertRefreshToken", mock.Anything, mock.Anything).Return(nil)

	access, refresh, err := svc.GenerateJWTTokens(ctx, user)
	assert.NoError(t, err)
	assert.NotEmpty(t, access)
	assert.NotEmpty(t, refresh)
	mockQ.AssertExpectations(t)
}
