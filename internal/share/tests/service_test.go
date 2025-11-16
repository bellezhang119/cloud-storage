package tests

import (
	"context"
	"database/sql"
	"testing"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/share"
	"github.com/bellezhang119/cloud-storage/internal/util"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockQueries struct {
	mock.Mock
}

func (m *MockQueries) CheckUserFileAccess(ctx context.Context, arg database.CheckUserFileAccessParams) (bool, error) {
	args := m.Called(ctx, arg)
	return args.Bool(0), args.Error(1)
}

func (m *MockQueries) CreateFileShare(ctx context.Context, arg database.CreateFileShareParams) (database.FileShare, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(database.FileShare), args.Error(1)
}

func (m *MockQueries) DeleteFileShare(ctx context.Context, arg database.DeleteFileShareParams) (int64, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockQueries) GetFileShare(ctx context.Context, arg database.GetFileShareParams) (database.FileShare, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(database.FileShare), args.Error(1)
}

func (m *MockQueries) ListFileShares(ctx context.Context, fileID uuid.NullUUID) ([]database.FileShare, error) {
	args := m.Called(ctx, fileID)
	return args.Get(0).([]database.FileShare), args.Error(1)
}

func (m *MockQueries) ListFilesSharedWithUser(ctx context.Context, sharedUserID sql.NullInt32) ([]database.File, error) {
	args := m.Called(ctx, sharedUserID)
	return args.Get(0).([]database.File), args.Error(1)
}

func (m *MockQueries) CheckUserFolderAccess(ctx context.Context, arg database.CheckUserFolderAccessParams) (bool, error) {
	args := m.Called(ctx, arg)
	return args.Bool(0), args.Error(1)
}

func (m *MockQueries) CreateFolderShare(ctx context.Context, arg database.CreateFolderShareParams) (database.FolderShare, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(database.FolderShare), args.Error(1)
}

func (m *MockQueries) DeleteFolderShare(ctx context.Context, arg database.DeleteFolderShareParams) (int64, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockQueries) GetFolderShare(ctx context.Context, arg database.GetFolderShareParams) (database.FolderShare, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(database.FolderShare), args.Error(1)
}

func (m *MockQueries) GetSharedSubfolders(ctx context.Context, parentID uuid.NullUUID) ([]database.Folder, error) {
	args := m.Called(ctx, parentID)
	return args.Get(0).([]database.Folder), args.Error(1)
}

func (m *MockQueries) GetFilesInSharedFolder(ctx context.Context, folderID uuid.NullUUID) ([]database.File, error) {
	args := m.Called(ctx, folderID)
	return args.Get(0).([]database.File), args.Error(1)
}

func (m *MockQueries) ListFolderShares(ctx context.Context, folderID uuid.NullUUID) ([]database.FolderShare, error) {
	args := m.Called(ctx, folderID)
	return args.Get(0).([]database.FolderShare), args.Error(1)
}

func (m *MockQueries) ListFoldersSharedWithUser(ctx context.Context, sharedUserID sql.NullInt32) ([]database.Folder, error) {
	args := m.Called(ctx, sharedUserID)
	return args.Get(0).([]database.Folder), args.Error(1)
}

func TestCheckUserFileAccess(t *testing.T) {
	mockQ := new(MockQueries)
	svc := share.NewService(mockQ)

	fileID := uuid.New()
	userID := int32(1)
	mockQ.On("CheckUserFileAccess", mock.Anything, database.CheckUserFileAccessParams{
		FileID:       util.ToNullUUID(&fileID),
		SharedUserID: util.ToNullInt32(&userID),
	}).Return(true, nil)

	res, err := svc.CheckUserFileAccess(context.Background(), fileID, 1)
	assert.NoError(t, err)
	assert.True(t, res)
	mockQ.AssertExpectations(t)
}

func TestCreateFileShare(t *testing.T) {
	mockQ := new(MockQueries)
	svc := share.NewService(mockQ)

	fileID := uuid.New()
	userID := int32(1)
	expected := database.FileShare{FileID: util.ToNullUUID(&fileID)}

	mockQ.On("CreateFileShare", mock.Anything, database.CreateFileShareParams{
		FileID:       util.ToNullUUID(&fileID),
		SharedUserID: util.ToNullInt32(&userID),
	}).Return(expected, nil)

	res, err := svc.CreateFileShare(context.Background(), fileID, 1)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
	mockQ.AssertExpectations(t)
}

func TestDeleteFileShare(t *testing.T) {
	mockQ := new(MockQueries)
	svc := share.NewService(mockQ)

	fileID := uuid.New()
	userID := int32(1)
	mockQ.On("DeleteFileShare", mock.Anything, database.DeleteFileShareParams{
		FileID:       util.ToNullUUID(&fileID),
		SharedUserID: util.ToNullInt32(&userID),
	}).Return(int64(1), nil)

	err := svc.DeleteFileShare(context.Background(), fileID, 1)
	assert.NoError(t, err)
	mockQ.AssertExpectations(t)
}

func TestGetFileShare(t *testing.T) {
	mockQ := new(MockQueries)
	svc := share.NewService(mockQ)

	fileID := uuid.New()
	userID := int32(1)
	expected := database.FileShare{FileID: util.ToNullUUID(&fileID)}

	mockQ.On("GetFileShare", mock.Anything, database.GetFileShareParams{
		FileID:       util.ToNullUUID(&fileID),
		SharedUserID: util.ToNullInt32(&userID),
	}).Return(expected, nil)

	res, err := svc.GetFileShare(context.Background(), fileID, 1)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
	mockQ.AssertExpectations(t)
}

func TestListFileShares(t *testing.T) {
	mockQ := new(MockQueries)
	svc := share.NewService(mockQ)

	fileID := uuid.New()
	expected := []database.FileShare{{FileID: util.ToNullUUID(&fileID)}}

	mockQ.On("ListFileShares", mock.Anything, util.ToNullUUID(&fileID)).Return(expected, nil)

	res, err := svc.ListFileShares(context.Background(), fileID)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
	mockQ.AssertExpectations(t)
}

func TestListFilesSharedWithUser(t *testing.T) {
	mockQ := new(MockQueries)
	svc := share.NewService(mockQ)

	fileID := uuid.New()
	userID := int32(1)
	expected := []database.File{{ID: fileID}}

	mockQ.On("ListFilesSharedWithUser", mock.Anything, util.ToNullInt32(&userID)).Return(expected, nil)

	res, err := svc.ListFilesSharedWithUser(context.Background(), 1)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
	mockQ.AssertExpectations(t)
}

// Folder tests

func TestCheckUserFolderAccess(t *testing.T) {
	mockQ := new(MockQueries)
	svc := share.NewService(mockQ)

	folderID := uuid.New()
	userID := int32(1)
	mockQ.On("CheckUserFolderAccess", mock.Anything, database.CheckUserFolderAccessParams{
		ID:           folderID,
		SharedUserID: util.ToNullInt32(&userID),
	}).Return(true, nil)

	res, err := svc.CheckUserFolderAccess(context.Background(), folderID, 1)
	assert.NoError(t, err)
	assert.True(t, res)
	mockQ.AssertExpectations(t)
}

func TestCreateFolderShare(t *testing.T) {
	mockQ := new(MockQueries)
	svc := share.NewService(mockQ)

	folderID := uuid.New()
	userID := int32(1)
	expected := database.FolderShare{FolderID: util.ToNullUUID(&folderID)}

	mockQ.On("CreateFolderShare", mock.Anything, database.CreateFolderShareParams{
		FolderID:     util.ToNullUUID(&folderID),
		SharedUserID: util.ToNullInt32(&userID),
	}).Return(expected, nil)

	res, err := svc.CreateFolderShare(context.Background(), folderID, 1)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
	mockQ.AssertExpectations(t)
}

func TestDeleteFolderShare(t *testing.T) {
	mockQ := new(MockQueries)
	svc := share.NewService(mockQ)

	folderID := uuid.New()
	userID := int32(1)
	mockQ.On("DeleteFolderShare", mock.Anything, database.DeleteFolderShareParams{
		FolderID:     util.ToNullUUID(&folderID),
		SharedUserID: util.ToNullInt32(&userID),
	}).Return(int64(1), nil)

	err := svc.DeleteFolderShare(context.Background(), folderID, 1)
	assert.NoError(t, err)
	mockQ.AssertExpectations(t)
}

func TestGetFolderShare(t *testing.T) {
	mockQ := new(MockQueries)
	svc := share.NewService(mockQ)

	folderID := uuid.New()
	userID := int32(1)
	expected := database.FolderShare{FolderID: util.ToNullUUID(&folderID)}

	mockQ.On("GetFolderShare", mock.Anything, database.GetFolderShareParams{
		FolderID:     util.ToNullUUID(&folderID),
		SharedUserID: util.ToNullInt32(&userID),
	}).Return(expected, nil)

	res, err := svc.GetFolderShare(context.Background(), folderID, 1)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
	mockQ.AssertExpectations(t)
}

func TestGetSharedFolderContent(t *testing.T) {
	mockQ := new(MockQueries)
	svc := share.NewService(mockQ)

	folderID := uuid.New()
	userID := int32(1)
	expectedFolders := []database.Folder{{ID: folderID}}
	expectedFiles := []database.File{{ID: uuid.New()}}

	mockQ.On("CheckUserFolderAccess", mock.Anything, database.CheckUserFolderAccessParams{
		ID:           folderID,
		SharedUserID: util.ToNullInt32(&userID),
	}).Return(true, nil)

	mockQ.On("GetSharedSubfolders", mock.Anything, util.ToNullUUID(&folderID)).Return(expectedFolders, nil)
	mockQ.On("GetFilesInSharedFolder", mock.Anything, util.ToNullUUID(&folderID)).Return(expectedFiles, nil)

	folders, files, err := svc.GetSharedFolderContent(context.Background(), folderID, 1)
	assert.NoError(t, err)
	assert.Equal(t, expectedFolders, folders)
	assert.Equal(t, expectedFiles, files)
	mockQ.AssertExpectations(t)
}

func TestListFolderShares(t *testing.T) {
	mockQ := new(MockQueries)
	svc := share.NewService(mockQ)

	folderID := uuid.New()
	expected := []database.FolderShare{{FolderID: util.ToNullUUID(&folderID)}}

	mockQ.On("ListFolderShares", mock.Anything, util.ToNullUUID(&folderID)).Return(expected, nil)

	res, err := svc.ListFolderShares(context.Background(), folderID)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
	mockQ.AssertExpectations(t)
}

func TestListFoldersSharedWithUser(t *testing.T) {
	mockQ := new(MockQueries)
	svc := share.NewService(mockQ)

	userID := int32(1)

	expected := []database.Folder{{ID: uuid.New()}}

	mockQ.On("ListFoldersSharedWithUser", mock.Anything, util.ToNullInt32(&userID)).Return(expected, nil)

	res, err := svc.ListFoldersSharedWithUser(context.Background(), 1)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
	mockQ.AssertExpectations(t)
}
