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

type MockAuthQueries struct {
	mock.Mock
}

func (m *MockAuthQueries) CreateUser(ctx context.Context, params database.CreateUserParams) (database.User, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(database.User), args.Error(1)
}

func (m *MockAuthQueries) GetUserByVerificationToken(ctx context.Context, token sql.NullString) (database.User, error) {
	args := m.Called(ctx, token)
	return args.Get(0).(database.User), args.Error(1)
}

func (m *MockAuthQueries) MarkUserAsVerified(ctx context.Context, id int32) (int64, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAuthQueries) UpdateVerificationToken(ctx context.Context, params database.UpdateVerificationTokenParams) (int64, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAuthQueries) InsertRefreshToken(ctx context.Context, params database.InsertRefreshTokenParams) (database.RefreshToken, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(database.RefreshToken), args.Error(1)
}

func (m *MockAuthQueries) GetRefreshToken(ctx context.Context, hash string) (database.GetRefreshTokenRow, error) {
	args := m.Called(ctx, hash)
	return args.Get(0).(database.GetRefreshTokenRow), args.Error(1)
}

func (m *MockAuthQueries) RevokeRefreshToken(ctx context.Context, hash string) (int64, error) {
	args := m.Called(ctx, hash)
	return args.Get(0).(int64), args.Error(1)
}

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) GetUserByID(ctx context.Context, id int32) (database.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(database.User), args.Error(1)
}

func (m *MockUserService) GetUserByEmail(ctx context.Context, email string) (database.User, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(database.User), args.Error(1)
}

func TestVerifyUserByToken_Success(t *testing.T) {
	mockQ := new(MockAuthQueries)
	mockUserSvc := new(MockUserService)
	svc := auth.NewService(mockQ, mockUserSvc)

	ctx := context.Background()
	token := "validtoken"

	user := database.User{
		ID: 1,
		VerificationTokenExpiry: sql.NullTime{
			Time:  time.Now().Add(1 * time.Hour),
			Valid: true,
		},
	}

	mockQ.On("GetUserByVerificationToken", ctx, sql.NullString{String: token, Valid: true}).Return(user, nil)
	mockQ.On("MarkUserAsVerified", ctx, user.ID).Return(int64(1), nil)

	err := svc.VerifyUserByToken(ctx, token)
	assert.NoError(t, err)
	mockQ.AssertExpectations(t)
}

func TestUpdateVerificationToken_Success(t *testing.T) {
	mockQ := new(MockAuthQueries)
	mockUserSvc := new(MockUserService)
	svc := auth.NewService(mockQ, mockUserSvc)

	ctx := context.Background()
	user := database.User{Email: "user@example.com"}

	mockQ.On("UpdateVerificationToken", ctx, mock.Anything).Return(int64(1), nil)

	token, err := svc.UpdateVerificationToken(ctx, user)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	mockQ.AssertExpectations(t)
}

func TestVerifyUserByToken_TokenExpired(t *testing.T) {
	mockQ := new(MockAuthQueries)
	mockUserSvc := new(MockUserService)
	svc := auth.NewService(mockQ, mockUserSvc)

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
	assert.EqualError(t, err, "token has expired")
	mockQ.AssertExpectations(t)
}

func TestAuthenticateUser_InvalidPassword(t *testing.T) {
	mockQ := new(MockAuthQueries)
	mockUserSvc := new(MockUserService)
	svc := auth.NewService(mockQ, mockUserSvc)

	ctx := context.Background()
	email := "user@example.com"

	hashed, _ := util.HashPassword("rightpass")
	mockUserSvc.On("GetUserByEmail", ctx, email).Return(database.User{
		PasswordHash: hashed,
		Email:        email,
	}, nil)

	_, err := svc.AuthenticateUser(ctx, email, "wrongpass")
	assert.Error(t, err)
	mockUserSvc.AssertExpectations(t)
}

func TestGenerateJWTTokens_Success(t *testing.T) {
	mockQ := new(MockAuthQueries)
	mockUserSvc := new(MockUserService)
	svc := auth.NewService(mockQ, mockUserSvc)

	ctx := context.Background()
	user := database.User{ID: 1, Email: "user@example.com"}

	mockQ.On("InsertRefreshToken", ctx, mock.Anything).Return(database.RefreshToken{}, nil)

	access, refresh, err := svc.GenerateJWTTokens(ctx, user)
	assert.NoError(t, err)
	assert.NotEmpty(t, access)
	assert.NotEmpty(t, refresh)
	mockQ.AssertExpectations(t)
}

func TestRefreshJWTTokens_Success(t *testing.T) {
	mockQ := new(MockAuthQueries)
	mockUserSvc := new(MockUserService)
	svc := auth.NewService(mockQ, mockUserSvc)

	ctx := context.Background()
	user := database.User{ID: 1, Email: "user@example.com"}

	_, oldToken, err := util.GenerateJWTTokens(user.ID, user.Email, time.Now().Add(1*time.Hour))
	assert.NoError(t, err)

	hashedOld := util.HashToken(oldToken)

	rtRow := database.GetRefreshTokenRow{
		TokenHash: hashedOld,
		UserID:    user.ID,
		Revoked:   false,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	mockQ.On("GetRefreshToken", ctx, hashedOld).Return(rtRow, nil)
	mockUserSvc.On("GetUserByID", ctx, user.ID).Return(user, nil)
	mockQ.On("InsertRefreshToken", ctx, mock.Anything).Return(database.RefreshToken{}, nil)
	mockQ.On("RevokeRefreshToken", ctx, hashedOld).Return(int64(1), nil)

	access, refresh, err := svc.RefreshJWTTokens(ctx, oldToken)
	assert.NoError(t, err)
	assert.NotEmpty(t, access)
	assert.NotEmpty(t, refresh)

	mockQ.AssertExpectations(t)
	mockUserSvc.AssertExpectations(t)
}
