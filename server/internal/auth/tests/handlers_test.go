package tests

import (
	"bytes"
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bellezhang119/cloud-storage/internal/auth"
	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockService mocks the auth.Service interface
type MockService struct {
	mock.Mock
}

func (m *MockService) CreateUser(ctx context.Context, email, password string) (database.User, error) {
	args := m.Called(ctx, email, password)
	return args.Get(0).(database.User), args.Error(1)
}

func (m *MockService) VerifyUserByToken(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockService) GetUserByEmail(ctx context.Context, email string) (database.User, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(database.User), args.Error(1)
}

func (m *MockService) UpdateVerificationToken(ctx context.Context, user database.User) (string, error) {
	args := m.Called(ctx, user)
	return args.String(0), args.Error(1)
}

func (m *MockService) AuthenticateUser(ctx context.Context, email, password string) (database.User, error) {
	args := m.Called(ctx, email, password)
	return args.Get(0).(database.User), args.Error(1)
}

func (m *MockService) GenerateJWTTokens(ctx context.Context, user database.User) (string, string, error) {
	args := m.Called(ctx, user)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockService) RefreshJWTTokens(ctx context.Context, oldRefreshToken string) (string, string, error) {
	args := m.Called(ctx, oldRefreshToken)
	return args.String(0), args.String(1), args.Error(2)
}

func TestRegisterHandler_Success(t *testing.T) {
	mockSvc := new(MockService)
	mockEmailSender := func(to, subject, body string) error {
		return nil
	}

	handler := auth.RegisterHandler(mockSvc, mockEmailSender)

	user := database.User{
		Email: "test@example.com",
		VerificationToken: sql.NullString{
			String: "token123",
			Valid:  true,
		},
	}
	mockSvc.On("CreateUser", mock.Anything, "test@example.com", "password123").Return(user, nil)

	reqBody := `{"email":"test@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(reqBody))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Body.String(), "User created")
	mockSvc.AssertExpectations(t)
}

func TestRegisterHandler_InvalidRequest(t *testing.T) {
	mockSvc := new(MockService)
	mockEmailSender := func(to, subject, body string) error {
		return nil
	}
	handler := auth.RegisterHandler(mockSvc, mockEmailSender)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString("invalid json"))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid request body")
}

// Similarly for VerifyEmailHandler:
func TestVerifyEmailHandler_Success(t *testing.T) {
	mockSvc := new(MockService)
	handler := auth.VerifyEmailHandler(mockSvc)

	mockSvc.On("VerifyUserByToken", mock.Anything, "validtoken").Return(nil)

	req := httptest.NewRequest(http.MethodGet, "/auth/verify?token=validtoken", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Email verified")
	mockSvc.AssertExpectations(t)
}

func TestVerifyEmailHandler_MissingToken(t *testing.T) {
	mockSvc := new(MockService)
	handler := auth.VerifyEmailHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/auth/verify", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Missing verification token")
}

// Test SendVerificationEmailHandler success:
func TestSendVerificationEmailHandler_Success(t *testing.T) {
	mockSvc := new(MockService)
	mockEmailSender := func(to, subject, body string) error {
		return nil
	}
	handler := auth.SendVerificationEmailHandler(mockSvc, mockEmailSender)

	user := database.User{
		Email:      "test@example.com",
		IsVerified: false,
	}
	mockSvc.On("GetUserByEmail", mock.Anything, "test@example.com").Return(user, nil)
	mockSvc.On("UpdateVerificationToken", mock.Anything, user).Return("token123", nil)

	reqBody := `{"email":"test@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/send-verification", bytes.NewBufferString(reqBody))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Verification email sent")
	mockSvc.AssertExpectations(t)
}

// Test LoginHandler success:
func TestLoginHandler_Success(t *testing.T) {
	mockSvc := new(MockService)
	handler := auth.LoginHandler(mockSvc)

	user := database.User{
		Email: "test@example.com",
	}
	mockSvc.On("AuthenticateUser", mock.Anything, "test@example.com", "password123").Return(user, nil)
	mockSvc.On("GenerateJWTTokens", mock.Anything, user).Return("accessToken", "refreshToken", nil)

	reqBody := `{"email":"test@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(reqBody))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "access_token")
	assert.Contains(t, rec.Body.String(), "refresh_token")
	mockSvc.AssertExpectations(t)
}

// Test RefreshTokenHandler success:
func TestRefreshTokenHandler_Success(t *testing.T) {
	mockSvc := new(MockService)
	handler := auth.RefreshTokenHandler(mockSvc)

	mockSvc.On("RefreshJWTTokens", mock.Anything, "oldRefreshToken").Return("newAccess", "newRefresh", nil)

	reqBody := `{"refresh_token":"oldRefreshToken"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBufferString(reqBody))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "access_token")
	assert.Contains(t, rec.Body.String(), "refresh_token")
	mockSvc.AssertExpectations(t)
}
