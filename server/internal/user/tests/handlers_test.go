package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/middleware"
	"github.com/bellezhang119/cloud-storage/internal/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockService struct {
	mock.Mock
}

func (m *MockService) GetUserByID(ctx context.Context, id int32) (database.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(database.User), args.Error(1)
}

func (m *MockService) GetUserByEmail(ctx context.Context, email string) (database.User, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(database.User), args.Error(1)
}

func (m *MockService) UpdateUserPassword(ctx context.Context, userID int32, newPassword string) error {
	args := m.Called(ctx, userID, newPassword)
	return args.Error(0)
}

func (m *MockService) UpdateUsedStorage(ctx context.Context, userID int32, newUsedStorage int64) error {
	args := m.Called(ctx, userID, newUsedStorage)
	return args.Error(0)
}

func (m *MockService) DeleteUser(ctx context.Context, userID int32) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func TestGetUserByIDHandler(t *testing.T) {
	mockSvc := &MockService{}
	mockUser := database.User{ID: 1, Email: "foo@bar.com"}

	mockSvc.On("GetUserByID", mock.Anything, int32(1)).Return(mockUser, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /user/{user_id}", user.GetUserByIDHandler(mockSvc))

	req := httptest.NewRequest("GET", "/user/1", nil)

	ctx := context.WithValue(req.Context(), middleware.UserIDKey, int32(1))
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var u database.User
	json.NewDecoder(rr.Body).Decode(&u)
	assert.Equal(t, "foo@bar.com", u.Email)

	mockSvc.AssertExpectations(t)
}

func TestGetUserByEmailHandler(t *testing.T) {
	mockSvc := &MockService{}
	mockUser := database.User{ID: 2, Email: "bar@foo.com"}

	mockSvc.On("GetUserByEmail", mock.Anything, "bar@foo.com").Return(mockUser, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /user/email/{user_email}", user.GetUserByEmailHandler(mockSvc))

	req := httptest.NewRequest("GET", "/user/email/bar@foo.com", nil)

	ctx := context.WithValue(req.Context(), middleware.UserEmailKey, "bar@foo.com")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var u database.User
	json.NewDecoder(rr.Body).Decode(&u)
	assert.Equal(t, "bar@foo.com", u.Email)

	mockSvc.AssertExpectations(t)
}

func TestUpdatePasswordHandler(t *testing.T) {
	mockSvc := &MockService{}
	mockSvc.On("UpdateUserPassword", mock.Anything, int32(1), "password123").Return(nil)

	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /user/{user_id}/password", user.UpdatePasswordHandler(mockSvc))

	body := `{"new_password":"password123"}`
	req := httptest.NewRequest("PATCH", "/user/1/password", bytes.NewBufferString(body))

	ctx := context.WithValue(req.Context(), middleware.UserIDKey, int32(1))
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	mockSvc.AssertExpectations(t)
}

func TestDeleteUserHandler(t *testing.T) {
	mockSvc := &MockService{}
	mockSvc.On("DeleteUser", mock.Anything, int32(1)).Return(nil)

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /users/{user_id}", user.DeleteUserHandler(mockSvc))

	req := httptest.NewRequest("DELETE", "/users/1", nil)

	ctx := context.WithValue(req.Context(), middleware.UserIDKey, int32(1))
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	mockSvc.AssertExpectations(t)
}
