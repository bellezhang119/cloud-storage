package tests

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/storage/services"
	"github.com/bellezhang119/cloud-storage/internal/util"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockQueries struct {
	mock.Mock
}

func (m *MockQueries) CreateFile(ctx context.Context, arg database.CreateFileParams) (database.File, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(database.File), args.Error(1)
}

func (m *MockQueries) GetFileByID(ctx context.Context, arg database.GetFileByIDParams) (database.File, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(database.File), args.Error(1)
}

func (m *MockQueries) GetFileByNameInFolder(ctx context.Context, arg database.GetFileByNameInFolderParams) (database.File, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(database.File), args.Error(1)
}

func (m *MockQueries) ListFilesInFolder(ctx context.Context, arg database.ListFilesInFolderParams) ([]database.File, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]database.File), args.Error(1)
}

func (m *MockQueries) DeleteFiles(ctx context.Context, arg database.DeleteFilesParams) (int64, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockQueries) ListFilesRecursive(ctx context.Context, arg database.ListFilesRecursiveParams) ([]database.ListFilesRecursiveRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]database.ListFilesRecursiveRow), args.Error(1)
}

func (m *MockQueries) UpdateFileMetadata(ctx context.Context, arg database.UpdateFileMetadataParams) (int64, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockQueries) UpdateFileNameAndPath(ctx context.Context, arg database.UpdateFileNameAndPathParams) (int64, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockQueries) UpdateFileParentAndPath(ctx context.Context, arg database.UpdateFileParentAndPathParams) (int64, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(int64), args.Error(1)
}

type MockFolderService struct {
	mock.Mock
}

func (m *MockFolderService) CreateFolder(ctx context.Context, userID int32, name string, parentID *uuid.UUID) (database.Folder, error) {
	args := m.Called(ctx, userID, name, parentID)
	return args.Get(0).(database.Folder), args.Error(1)
}

func (m *MockFolderService) GetFolderByID(ctx context.Context, folderID uuid.UUID, userID int32) (database.Folder, error) {
	args := m.Called(ctx, folderID, userID)
	return args.Get(0).(database.Folder), args.Error(1)
}

func (m *MockFolderService) ListFoldersByParent(ctx context.Context, userID int32, parentID *uuid.UUID) ([]database.Folder, error) {
	args := m.Called(ctx, userID, parentID)
	return args.Get(0).([]database.Folder), args.Error(1)
}

func (m *MockFolderService) GetFolderFullPath(ctx context.Context, folderID uuid.UUID, userID int32) (string, error) {
	args := m.Called(ctx, folderID, userID)
	return args.Get(0).(string), args.Error(1)
}

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) AdjustUsedStorage(ctx context.Context, userID int32, delta int64) error {
	args := m.Called(ctx, userID, delta)
	return args.Error(0)
}

type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) SaveFile(ctx context.Context, userID int32, path string, content io.Reader) error {
	args := m.Called(ctx, userID, path, content)
	return args.Error(0)
}

func (m *MockStorage) ReadFile(ctx context.Context, userID int32, path string) (io.ReadCloser, error) {
	args := m.Called(ctx, userID, path)
	rc, _ := args.Get(0).(io.ReadCloser)
	return rc, args.Error(1)
}

func (m *MockStorage) DeleteFile(ctx context.Context, userID int32, path string) error {
	args := m.Called(ctx, userID, path)
	return args.Error(0)
}

func (m *MockStorage) CreateDirectory(ctx context.Context, userID int32, path string) error {
	args := m.Called(ctx, userID, path)
	return args.Error(0)
}

func (m *MockStorage) DeleteDirectory(ctx context.Context, userID int32, path string) error {
	args := m.Called(ctx, userID, path)
	return args.Error(0)
}

func (m *MockStorage) MoveFile(ctx context.Context, userID int32, oldPath, newPath string) error {
	args := m.Called(ctx, userID, oldPath, newPath)
	return args.Error(0)
}

func (m *MockStorage) MoveDirectory(ctx context.Context, userID int32, oldPath, newPath string, overwriteFiles bool) error {
	args := m.Called(ctx, userID, oldPath, newPath, overwriteFiles)
	return args.Error(0)
}

func (m *MockStorage) GetDirectorySize(ctx context.Context, userID int32, path string) (int64, error) {
	args := m.Called(ctx, userID, path)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockStorage) ZipMultipleFolders(ctx context.Context, userID int32, folderPaths []string, w io.Writer) error {
	args := m.Called(ctx, userID, folderPaths, w)
	return args.Error(0)
}

func newTestService() (*services.FileServiceImpl, *MockQueries, *MockFolderService, *MockUserService, *MockStorage) {
	mockQ := new(MockQueries)
	mockFolderSvc := new(MockFolderService)
	mockUserSvc := new(MockUserService)
	mockStorage := new(MockStorage)
	svc := services.NewFileService(mockQ, mockUserSvc, mockStorage)
	svc.SetFolderService(mockFolderSvc)
	return svc, mockQ, mockFolderSvc, mockUserSvc, mockStorage
}

func TestGetFileByID(t *testing.T) {
	svc, mockQ, _, _, _ := newTestService()
	fileID := uuid.New()
	userID := int32(1)

	expected := database.File{ID: fileID, UserID: util.ToNullInt32(&userID)}

	mockQ.On("GetFileByID", mock.Anything, database.GetFileByIDParams{
		ID:     fileID,
		UserID: util.ToNullInt32(&userID),
	}).Return(expected, nil)

	res, err := svc.GetFileByID(context.Background(), fileID, userID)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
	mockQ.AssertExpectations(t)
}

func TestGetFileByNameInFolder(t *testing.T) {
	svc, mockQ, _, _, _ := newTestService()
	fileName := "test1.txt"
	fileID := uuid.New()
	userID := int32(1)
	folderID := uuid.New()

	expected := database.File{
		ID:     fileID,
		UserID: util.ToNullInt32(&userID),
		Name:   fileName,
	}

	mockQ.On("GetFileByNameInFolder", mock.Anything, database.GetFileByNameInFolderParams{
		FolderID: util.ToNullUUID(&folderID),
		UserID:   util.ToNullInt32(&userID),
		Name:     fileName,
	}).Return(expected, nil)

	res, err := svc.GetFileByNameInFolder(context.Background(), &folderID, userID, fileName)

	assert.NoError(t, err)
	assert.Equal(t, expected, res)
	mockQ.AssertExpectations(t)
}

func TestListFilesInFolder(t *testing.T) {
	svc, mockQ, _, _, _ := newTestService()
	userID := int32(1)
	folderID := uuid.New()

	expectedFiles := []database.File{
		{ID: uuid.New(), Name: "file1.txt", UserID: util.ToNullInt32(&userID)},
		{ID: uuid.New(), Name: "file2.txt", UserID: util.ToNullInt32(&userID)},
	}

	mockQ.On("ListFilesInFolder", mock.Anything, database.ListFilesInFolderParams{
		UserID:   util.ToNullInt32(&userID),
		FolderID: util.ToNullUUID(&folderID),
	}).Return(expectedFiles, nil)

	res, err := svc.ListFilesInFolder(context.Background(), &folderID, userID)

	assert.NoError(t, err)
	assert.Equal(t, expectedFiles, res)
	mockQ.AssertExpectations(t)
}

func TestListFilesRecursive(t *testing.T) {
	svc, mockQ, _, _, _ := newTestService()
	userID := int32(1)
	folderID := uuid.New()

	expectedRows := []database.ListFilesRecursiveRow{
		{FileID: uuid.New(), Name: "file1.txt", FilePath: "/docs/file1.txt"},
		{FileID: uuid.New(), Name: "file2.txt", FilePath: "/docs/sub/file2.txt"},
	}

	mockQ.On("ListFilesRecursive", mock.Anything, database.ListFilesRecursiveParams{
		UserID: util.ToNullInt32(&userID),
		ID:     folderID,
	}).Return(expectedRows, nil)

	res, err := svc.ListFilesRecursive(context.Background(), folderID, userID)

	assert.NoError(t, err)
	assert.Equal(t, expectedRows, res)
	mockQ.AssertExpectations(t)
}

func TestUploadFile(t *testing.T) {
	svc, mockQ, mockFolderSvc, mockUserSvc, mockStorage := newTestService()
	userID := int32(1)
	folderID := uuid.New()

	fileName := "test.txt"
	fileSize := int64(11)
	mimeType := "text/plain"
	folderPath := "folder"
	filePath := filepath.Join(folderPath, fileName)
	content := bytes.NewReader([]byte("hello world"))

	expectedFile := database.File{
		ID:        uuid.New(),
		Name:      fileName,
		FilePath:  filePath,
		SizeBytes: fileSize,
		MimeType:  sql.NullString{String: mimeType, Valid: true},
	}

	mockFolderSvc.On("GetFolderFullPath", mock.Anything, folderID, userID).Return(folderPath, nil)

	mockQ.On(
		"GetFileByNameInFolder",
		mock.Anything,
		database.GetFileByNameInFolderParams{
			FolderID: util.ToNullUUID(&folderID),
			UserID:   util.ToNullInt32(&userID),
			Name:     fileName,
		},
	).Return(database.File{}, sql.ErrNoRows)

	mockUserSvc.On("AdjustUsedStorage", mock.Anything, userID, fileSize).Return(nil)

	mockQ.On(
		"CreateFile",
		mock.Anything,
		mock.MatchedBy(func(p database.CreateFileParams) bool {
			return p.Name == fileName &&
				p.FilePath == filePath &&
				p.SizeBytes == fileSize &&
				p.MimeType.String == mimeType
		}),
	).Return(expectedFile, nil)

	mockStorage.On("SaveFile", mock.Anything, userID, filePath, mock.AnythingOfType("*bytes.Reader")).Return(nil)

	result, err := svc.UploadFile(context.Background(), &folderID, userID, fileName, fileSize, mimeType, content, false)

	assert.NoError(t, err)
	assert.Equal(t, expectedFile, result)
	mockQ.AssertExpectations(t)
	mockFolderSvc.AssertExpectations(t)
	mockUserSvc.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestDownloadFiles(t *testing.T) {
	svc, mockQ, _, _, mockStorage := newTestService()
	fileID := uuid.New()
	userID := int32(1)

	expectedFile := database.File{
		ID:       fileID,
		Name:     "report.pdf",
		FilePath: filepath.Join("folder", "report.pdf"),
		UserID:   util.ToNullInt32(&userID),
	}

	mockQ.On("GetFileByID", mock.Anything, database.GetFileByIDParams{
		ID:     fileID,
		UserID: util.ToNullInt32(&userID),
	}).Return(expectedFile, nil)

	fileContent := io.NopCloser(strings.NewReader("fake data"))
	mockStorage.On("ReadFile", mock.Anything, userID, expectedFile.FilePath).Return(fileContent, nil)

	res, err := svc.DownloadFiles(context.Background(), []uuid.UUID{fileID}, userID)

	assert.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, expectedFile.ID, res[0].File.ID)
	mockQ.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestDeleteFiles(t *testing.T) {
	svc, mockQ, _, mockUserSvc, mockStorage := newTestService()
	fileID1 := uuid.New()
	fileID2 := uuid.New()
	userID := int32(1)

	files := []database.File{
		{ID: fileID1, Name: "file1.txt", FilePath: "folder/file1.txt", SizeBytes: 100, UserID: util.ToNullInt32(&userID)},
		{ID: fileID2, Name: "file2.txt", FilePath: "folder/file2.txt", SizeBytes: 200, UserID: util.ToNullInt32(&userID)},
	}
	totalSize := int64(300)
	fileIDs := []uuid.UUID{fileID1, fileID2}

	mockQ.On("GetFileByID", mock.Anything, database.GetFileByIDParams{ID: fileID1, UserID: util.ToNullInt32(&userID)}).Return(files[0], nil)
	mockQ.On("GetFileByID", mock.Anything, database.GetFileByIDParams{ID: fileID2, UserID: util.ToNullInt32(&userID)}).Return(files[1], nil)
	mockQ.On("DeleteFiles", mock.Anything, database.DeleteFilesParams{Column1: fileIDs, UserID: util.ToNullInt32(&userID)}).Return(int64(2), nil)

	mockStorage.On("DeleteFile", mock.Anything, userID, files[0].FilePath).Return(nil)
	mockStorage.On("DeleteFile", mock.Anything, userID, files[1].FilePath).Return(nil)

	mockUserSvc.On("AdjustUsedStorage", mock.Anything, userID, -totalSize).Return(nil)

	err := svc.DeleteFiles(context.Background(), fileIDs, userID)

	assert.NoError(t, err)
	mockQ.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
	mockUserSvc.AssertExpectations(t)
}

func TestUpdateFileMetadata(t *testing.T) {
	svc, mockQ, _, _, _ := newTestService()
	fileID := uuid.New()
	userID := int32(1)
	sizeBytes := int64(1024)
	mimeType := "text/plain"

	mockQ.On("UpdateFileMetadata", mock.Anything, database.UpdateFileMetadataParams{
		ID:        fileID,
		UserID:    util.ToNullInt32(&userID),
		SizeBytes: sizeBytes,
		MimeType:  sql.NullString{String: mimeType, Valid: true},
	}).Return(int64(1), nil)

	err := svc.UpdateFileMetadata(context.Background(), fileID, userID, sizeBytes, mimeType)

	assert.NoError(t, err)
	mockQ.AssertExpectations(t)
}

func TestUpdateFileParentAndPath(t *testing.T) {
	svc, mockQ, _, _, _ := newTestService()
	fileID := uuid.New()
	folderID := uuid.New()
	userID := int32(1)
	filePath := filepath.Join("folder", "file1.txt")

	mockQ.On("UpdateFileParentAndPath", mock.Anything, database.UpdateFileParentAndPathParams{
		ID:       fileID,
		UserID:   util.ToNullInt32(&userID),
		FolderID: util.ToNullUUID(&folderID),
		FilePath: filePath,
	}).Return(int64(1), nil)

	err := svc.UpdateFileParentAndPath(context.Background(), fileID, userID, &folderID, filePath)
	assert.NoError(t, err)
	mockQ.AssertExpectations(t)
}

func TestUpdateFileNameAndPath(t *testing.T) {
	svc, mockQ, _, _, _ := newTestService()
	fileID := uuid.New()
	userID := int32(1)
	name := "new_name.txt"
	filePath := filepath.Join("folder", name)

	mockQ.On("UpdateFileNameAndPath", mock.Anything, database.UpdateFileNameAndPathParams{
		ID:       fileID,
		UserID:   util.ToNullInt32(&userID),
		Name:     name,
		FilePath: filePath,
	}).Return(int64(1), nil)

	err := svc.UpdateFileNameAndPath(context.Background(), fileID, userID, name, filePath)
	assert.NoError(t, err)
	mockQ.AssertExpectations(t)
}

func TestMoveFiles(t *testing.T) {
	svc, mockQ, mockFolderSvc, _, mockStorage := newTestService()
	fileID := uuid.New()
	destFolderID := uuid.New()
	userID := int32(1)

	originalFile := database.File{
		ID:       fileID,
		UserID:   util.ToNullInt32(&userID),
		Name:     "file.txt",
		FilePath: "folder/file.txt",
	}

	destFolderPath := "destFolder"
	newPath := filepath.Join(destFolderPath, originalFile.Name)

	mockFolderSvc.On("GetFolderFullPath", mock.Anything, destFolderID, userID).Return(destFolderPath, nil)

	mockQ.On("GetFileByID", mock.Anything, database.GetFileByIDParams{
		ID:     fileID,
		UserID: util.ToNullInt32(&userID),
	}).Return(originalFile, nil)

	mockQ.On("GetFileByNameInFolder", mock.Anything, database.GetFileByNameInFolderParams{
		FolderID: util.ToNullUUID(&destFolderID),
		UserID:   util.ToNullInt32(&userID),
		Name:     originalFile.Name,
	}).Return(database.File{}, sql.ErrNoRows)

	mockQ.On("UpdateFileParentAndPath", mock.Anything, database.UpdateFileParentAndPathParams{
		ID:       fileID,
		UserID:   util.ToNullInt32(&userID),
		FolderID: util.ToNullUUID(&destFolderID),
		FilePath: newPath,
	}).Return(int64(1), nil)

	mockStorage.On("MoveFile", mock.Anything, userID, originalFile.FilePath, newPath).Return(nil)

	err := svc.MoveFiles(context.Background(), []uuid.UUID{fileID}, userID, &destFolderID, false)
	assert.NoError(t, err)

	mockFolderSvc.AssertExpectations(t)
	mockQ.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestRenameFile(t *testing.T) {
	svc, mockQ, _, _, mockStorage := newTestService()
	fileID := uuid.New()
	userID := int32(1)
	folderID := uuid.New()

	oldName := "old.txt"
	newName := "new.txt"
	oldPath := filepath.Join("folder", oldName)
	newPath := filepath.Join("folder", newName)

	originalFile := database.File{
		ID:       fileID,
		UserID:   util.ToNullInt32(&userID),
		Name:     oldName,
		FilePath: oldPath,
		FolderID: util.ToNullUUID(&folderID),
	}

	mockQ.On("GetFileByID", mock.Anything, database.GetFileByIDParams{
		ID:     fileID,
		UserID: util.ToNullInt32(&userID),
	}).Return(originalFile, nil)

	mockQ.On("GetFileByNameInFolder", mock.Anything, database.GetFileByNameInFolderParams{
		FolderID: util.ToNullUUID(&folderID),
		UserID:   util.ToNullInt32(&userID),
		Name:     newName,
	}).Return(database.File{}, sql.ErrNoRows)

	mockQ.On("UpdateFileNameAndPath", mock.Anything, database.UpdateFileNameAndPathParams{
		ID:       fileID,
		UserID:   util.ToNullInt32(&userID),
		Name:     newName,
		FilePath: newPath,
	}).Return(int64(1), nil)

	mockStorage.On("MoveFile", mock.Anything, userID, oldPath, newPath).Return(nil)

	err := svc.RenameFile(context.Background(), fileID, userID, newName, false)
	assert.NoError(t, err)

	mockQ.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}
