package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/middleware"
	"github.com/bellezhang119/cloud-storage/internal/share"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockService struct {
	mock.Mock
}

func (m *MockService) CheckUserFileAccess(ctx context.Context, fileID uuid.UUID, userID int32) (bool, error) {
	args := m.Called(ctx, fileID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockService) CreateFileShare(ctx context.Context, fileID uuid.UUID, userID int32) (database.FileShare, error) {
	args := m.Called(ctx, fileID, userID)
	return args.Get(0).(database.FileShare), args.Error(1)
}

func (m *MockService) DeleteFileShare(ctx context.Context, fileID uuid.UUID, userID int32) error {
	args := m.Called(ctx, fileID, userID)
	return args.Error(0)
}

func (m *MockService) GetFileShare(ctx context.Context, fileID uuid.UUID, userID int32) (database.FileShare, error) {
	args := m.Called(ctx, fileID, userID)
	return args.Get(0).(database.FileShare), args.Error(1)
}

func (m *MockService) ListFileShares(ctx context.Context, fileID uuid.UUID) ([]database.FileShare, error) {
	args := m.Called(ctx, fileID)
	return args.Get(0).([]database.FileShare), args.Error(1)
}

func (m *MockService) ListFilesSharedWithUser(ctx context.Context, userID int32) ([]database.File, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]database.File), args.Error(1)
}

func (m *MockService) CheckUserFolderAccess(ctx context.Context, folderID uuid.UUID, userID int32) (bool, error) {
	args := m.Called(ctx, folderID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockService) CreateFolderShare(ctx context.Context, folderID uuid.UUID, userID int32) (database.FolderShare, error) {
	args := m.Called(ctx, folderID, userID)
	return args.Get(0).(database.FolderShare), args.Error(1)
}

func (m *MockService) DeleteFolderShare(ctx context.Context, folderID uuid.UUID, userID int32) error {
	args := m.Called(ctx, folderID, userID)
	return args.Error(0)
}

func (m *MockService) GetFolderShare(ctx context.Context, folderID uuid.UUID, userID int32) (database.FolderShare, error) {
	args := m.Called(ctx, folderID, userID)
	return args.Get(0).(database.FolderShare), args.Error(1)
}

func (m *MockService) GetSharedFolderContent(ctx context.Context, folderID uuid.UUID, userID int32) ([]database.Folder, []database.File, error) {
	args := m.Called(ctx, folderID, userID)
	return args.Get(0).([]database.Folder), args.Get(1).([]database.File), nil
}

func (m *MockService) ListFolderShares(ctx context.Context, folderID uuid.UUID) ([]database.FolderShare, error) {
	args := m.Called(ctx, folderID)
	return args.Get(0).([]database.FolderShare), args.Error(1)
}

func (m *MockService) ListFoldersSharedWithUser(ctx context.Context, userID int32) ([]database.Folder, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]database.Folder), args.Error(1)
}

func TestCheckUserFileAccessHandler(t *testing.T) {
	mockSvc := &MockService{}
	fileID := uuid.New()
	userID := int32(1)

	mockSvc.On("CheckUserFileAccess", mock.Anything, fileID, userID).Return(true, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /user/{user_id}/file/{file_id}/access", share.CheckUserFileAccessHandler(mockSvc))

	req := httptest.NewRequest("GET", fmt.Sprintf("/user/%d/file/%s/access", userID, fileID), nil)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var result bool
	json.NewDecoder(rr.Body).Decode(&result)
	assert.True(t, result)

	mockSvc.AssertExpectations(t)
}

func TestCreateFileShareHandler(t *testing.T) {
	mockSvc := &MockService{}
	fileID := uuid.New()
	userID := int32(1)
	shareRec := database.FileShare{FileID: uuid.NullUUID{UUID: fileID, Valid: true}}

	mockSvc.On("CreateFileShare", mock.Anything, fileID, userID).Return(shareRec, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /user/{user_id}/file/{file_id}/share", share.CreateFileShareHandler(mockSvc))

	req := httptest.NewRequest("POST", fmt.Sprintf("/user/%d/file/%s/share", userID, fileID), nil)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)

	var resp database.FileShare
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.Equal(t, fileID, resp.FileID.UUID)

	mockSvc.AssertExpectations(t)
}
func TestDeleteFileShareHandler(t *testing.T) {
	mockSvc := &MockService{}
	fileID := uuid.New()
	userID := int32(1)

	mockSvc.On("DeleteFileShare", mock.Anything, fileID, userID).Return(nil)

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /user/{user_id}/file/{file_id}/share", share.DeleteFileShareHandler(mockSvc))

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/user/%d/file/%s/share", userID, fileID), nil)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	mockSvc.AssertExpectations(t)
}

func TestGetFileShareHandler(t *testing.T) {
	mockSvc := &MockService{}
	fileID := uuid.New()
	userID := int32(1)
	expected := database.FileShare{FileID: uuid.NullUUID{UUID: fileID, Valid: true}}

	mockSvc.On("GetFileShare", mock.Anything, fileID, userID).Return(expected, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /user/{user_id}/file/{file_id}/share", share.GetFileShareHandler(mockSvc))

	req := httptest.NewRequest("GET", fmt.Sprintf("/user/%d/file/%s/share", userID, fileID), nil)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var resp database.FileShare
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.Equal(t, fileID, resp.FileID.UUID)
	mockSvc.AssertExpectations(t)
}

func TestListFileSharesHandler(t *testing.T) {
	mockSvc := &MockService{}
	fileID := uuid.New()
	expected := []database.FileShare{{FileID: uuid.NullUUID{UUID: fileID, Valid: true}}}

	mockSvc.On("ListFileShares", mock.Anything, fileID).Return(expected, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /file/{file_id}/shares", share.ListFileSharesHandler(mockSvc))

	req := httptest.NewRequest("GET", fmt.Sprintf("/file/%s/shares", fileID), nil)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, int32(1))
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var resp []database.FileShare
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.Len(t, resp, 1)
	mockSvc.AssertExpectations(t)
}

func TestListFilesSharedWithUserHandler(t *testing.T) {
	mockSvc := &MockService{}
	userID := int32(1)
	expected := []database.File{{Name: "File1"}}

	mockSvc.On("ListFilesSharedWithUser", mock.Anything, userID).Return(expected, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /user/{user_id}/files/shares", share.ListFilesSharedWithUserHandler(mockSvc))

	req := httptest.NewRequest("GET", fmt.Sprintf("/user/%d/files/shares", userID), nil)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var files []database.File
	json.NewDecoder(rr.Body).Decode(&files)
	assert.Equal(t, "File1", files[0].Name)
	mockSvc.AssertExpectations(t)
}

func TestCheckUserFolderAccessHandler(t *testing.T) {
	mockSvc := &MockService{}
	folderID := uuid.New()
	userID := int32(1)

	mockSvc.On("CheckUserFolderAccess", mock.Anything, folderID, userID).Return(true, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /user/{user_id}/folder/{folder_id}/access", share.CheckUserFolderAccessHandler(mockSvc))

	req := httptest.NewRequest("GET", fmt.Sprintf("/user/%d/folder/%s/access", userID, folderID), nil)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var access bool
	json.NewDecoder(rr.Body).Decode(&access)
	assert.True(t, access)
	mockSvc.AssertExpectations(t)
}

func TestCreateFolderShareHandler(t *testing.T) {
	mockSvc := &MockService{}
	folderID := uuid.New()
	userID := int32(1)
	expected := database.FolderShare{FolderID: uuid.NullUUID{UUID: folderID, Valid: true}}

	mockSvc.On("CreateFolderShare", mock.Anything, folderID, userID).Return(expected, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /user/{user_id}/folder/{folder_id}/share", share.CreateFolderShareHandler(mockSvc))

	req := httptest.NewRequest("POST", fmt.Sprintf("/user/%d/folder/%s/share", userID, folderID), nil)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)

	var resp database.FolderShare
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.Equal(t, folderID, resp.FolderID.UUID)
	mockSvc.AssertExpectations(t)
}

func TestDeleteFolderShareHandler(t *testing.T) {
	mockSvc := &MockService{}
	folderID := uuid.New()
	userID := int32(1)

	mockSvc.On("DeleteFolderShare", mock.Anything, folderID, userID).Return(nil)

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /user/{user_id}/folder/{folder_id}/share", share.DeleteFolderShareHandler(mockSvc))

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/user/%d/folder/%s/share", userID, folderID), nil)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	mockSvc.AssertExpectations(t)
}

func TestGetFolderShareHandler(t *testing.T) {
	mockSvc := &MockService{}
	folderID := uuid.New()
	userID := int32(1)
	expected := database.FolderShare{FolderID: uuid.NullUUID{UUID: folderID, Valid: true}}

	mockSvc.On("GetFolderShare", mock.Anything, folderID, userID).Return(expected, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /user/{user_id}/folder/{folder_id}/share", share.GetFolderShareHandler(mockSvc))

	req := httptest.NewRequest("GET", fmt.Sprintf("/user/%d/folder/%s/share", userID, folderID), nil)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var resp database.FolderShare
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.Equal(t, folderID, resp.FolderID.UUID)
	mockSvc.AssertExpectations(t)
}

func TestGetSharedFolderContentHandler(t *testing.T) {
	mockSvc := &MockService{}
	folderID := uuid.New()
	userID := int32(1)
	expectedFolders := []database.Folder{{Name: "SubFolder"}}
	expectedFiles := []database.File{{Name: "Doc.pdf"}}

	mockSvc.On("GetSharedFolderContent", mock.Anything, folderID, userID).Return(expectedFolders, expectedFiles, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /user/{user_id}/folder/{folder_id}/content", share.GetSharedFolderContentHandler(mockSvc))

	req := httptest.NewRequest("GET", fmt.Sprintf("/user/%d/folder/%s/content", userID, folderID), nil)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		Folders []database.Folder `json:"folders"`
		Files   []database.File   `json:"files"`
	}
	json.NewDecoder(rr.Body).Decode(&resp)

	assert.Equal(t, "SubFolder", resp.Folders[0].Name)
	assert.Equal(t, "Doc.pdf", resp.Files[0].Name)
	mockSvc.AssertExpectations(t)
}
func TestListFolderSharesHandler(t *testing.T) {
	mockSvc := &MockService{}
	folderID := uuid.New()
	expected := []database.FolderShare{{FolderID: uuid.NullUUID{UUID: folderID, Valid: true}}}

	mockSvc.On("ListFolderShares", mock.Anything, folderID).Return(expected, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /folder/{folder_id}/shares", share.ListFolderSharesHandler(mockSvc))

	req := httptest.NewRequest("GET", fmt.Sprintf("/folder/%s/shares", folderID), nil)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, int32(11))
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var resp []database.FolderShare
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.Len(t, resp, 1)
	mockSvc.AssertExpectations(t)
}

func TestListFoldersSharedWithUserHandler(t *testing.T) {
	mockSvc := &MockService{}
	userID := int32(12)
	expected := []database.Folder{{Name: "SharedFolder"}}

	mockSvc.On("ListFoldersSharedWithUser", mock.Anything, userID).Return(expected, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /user/{user_id}/folders/shares", share.ListFoldersSharedWithUserHandler(mockSvc))

	req := httptest.NewRequest("GET", fmt.Sprintf("/user/%d/folders/shares", userID), nil)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var folders []database.Folder
	json.NewDecoder(rr.Body).Decode(&folders)
	assert.Equal(t, "SharedFolder", folders[0].Name)
	mockSvc.AssertExpectations(t)
}
