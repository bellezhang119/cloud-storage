package tests

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"path/filepath"
	"testing"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/storage/services"
	"github.com/bellezhang119/cloud-storage/internal/util"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockFolderQueries struct {
	mock.Mock
}

func (m *MockFolderQueries) CreateFolder(ctx context.Context, arg database.CreateFolderParams) (database.Folder, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(database.Folder), args.Error(1)
}

func (m *MockFolderQueries) GetFolderByID(ctx context.Context, arg database.GetFolderByIDParams) (database.Folder, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(database.Folder), args.Error(1)
}

func (m *MockFolderQueries) ListFoldersByParent(ctx context.Context, arg database.ListFoldersByParentParams) ([]database.Folder, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]database.Folder), args.Error(1)
}

func (m *MockFolderQueries) GetFolderByNameInParent(ctx context.Context, arg database.GetFolderByNameInParentParams) (database.Folder, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(database.Folder), args.Error(1)
}

func (m *MockFolderQueries) DeleteFolders(ctx context.Context, arg database.DeleteFoldersParams) (int64, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockFolderQueries) ListFoldersRecursive(ctx context.Context, arg database.ListFoldersRecursiveParams) ([]database.ListFoldersRecursiveRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]database.ListFoldersRecursiveRow), args.Error(1)
}

func (m *MockFolderQueries) UpdateFolderMetadata(ctx context.Context, arg database.UpdateFolderMetadataParams) (int64, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockFolderQueries) UpdateFoldersParent(ctx context.Context, arg database.UpdateFoldersParentParams) (int64, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockFolderQueries) GetFolderFullPath(ctx context.Context, arg database.GetFolderFullPathParams) (string, error) {
	args := m.Called(ctx, arg)
	return args.String(0), args.Error(1)
}

// --- Other Mocks ---
type MockFileService struct{ mock.Mock }

func (m *MockFileService) UploadFile(ctx context.Context, folderID *uuid.UUID, userID int32, name string, sizeBytes int64, mimeType string, content io.Reader, overwrite bool) (database.File, error) {
	args := m.Called(ctx, folderID, userID, name, sizeBytes, mimeType, content, overwrite)
	return args.Get(0).(database.File), args.Error(1)
}
func (m *MockFileService) ListFilesRecursive(ctx context.Context, folderID uuid.UUID, userID int32) ([]database.ListFilesRecursiveRow, error) {
	args := m.Called(ctx, folderID, userID)
	return args.Get(0).([]database.ListFilesRecursiveRow), args.Error(1)
}
func (m *MockFileService) GetFileByNameInFolder(ctx context.Context, folderID *uuid.UUID, userID int32, name string) (database.File, error) {
	args := m.Called(ctx, folderID, userID, name)
	return args.Get(0).(database.File), args.Error(1)
}
func (m *MockFileService) DeleteFiles(ctx context.Context, filesIDs []uuid.UUID, userID int32) error {
	args := m.Called(ctx, filesIDs, userID)
	return args.Error(0)
}
func (m *MockFileService) UpdateFileNameAndPath(ctx context.Context, fileID uuid.UUID, userID int32, name, filePath string) error {
	args := m.Called(ctx, fileID, userID, name, filePath)
	return args.Error(0)
}
func (m *MockFileService) UpdateFileMetadata(ctx context.Context, fileID uuid.UUID, userID int32, sizeBytes int64, mimeType string) error {
	args := m.Called(ctx, fileID, userID, sizeBytes, mimeType)
	return args.Error(0)
}

func newTestFolderService() (
	*services.FolderServiceImpl,
	*MockFolderQueries,
	*MockFileService,
	*MockUserService,
	*MockStorage,
) {
	mockQueries := new(MockFolderQueries)
	mockFile := new(MockFileService)
	mockUser := new(MockUserService)
	mockLocal := new(MockStorage)

	svc := services.NewFolderService(mockQueries, mockUser, mockLocal)
	svc.SetFileService(mockFile)

	return svc, mockQueries, mockFile, mockUser, mockLocal
}

func TestGetFolderByID(t *testing.T) {
	svc, mockQ, _, _, _ := newTestFolderService()
	ctx := context.Background()
	userID := int32(1)
	folderID := uuid.New()
	expectedFolder := database.Folder{ID: folderID, Name: "Test"}

	mockQ.On("GetFolderByID", ctx, database.GetFolderByIDParams{
		ID:     folderID,
		UserID: util.ToNullInt32(&userID),
	}).Return(expectedFolder, nil)

	result, err := svc.GetFolderByID(ctx, folderID, userID)

	assert.NoError(t, err)
	assert.Equal(t, expectedFolder, result)
	mockQ.AssertExpectations(t)
}

func TestListFoldersByParent(t *testing.T) {
	svc, mockQ, _, _, _ := newTestFolderService()
	ctx := context.Background()
	userID := int32(2)
	parentID := uuid.New()
	expected := []database.Folder{
		{ID: uuid.New(), Name: "Child1"},
		{ID: uuid.New(), Name: "Child2"},
	}

	mockQ.On("ListFoldersByParent", ctx, database.ListFoldersByParentParams{
		UserID:   util.ToNullInt32(&userID),
		ParentID: util.ToNullUUID(&parentID),
	}).Return(expected, nil)

	result, err := svc.ListFoldersByParent(ctx, userID, &parentID)

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	mockQ.AssertExpectations(t)
}

func TestGetFolderFullPath(t *testing.T) {
	svc, mockQ, _, _, _ := newTestFolderService()
	ctx := context.Background()
	userID := int32(3)
	folderID := uuid.New()

	expectedPath := "root/folder/sub"

	mockQ.On("GetFolderFullPath", ctx, database.GetFolderFullPathParams{
		ID:     folderID,
		UserID: util.ToNullInt32(&userID),
	}).Return(expectedPath, nil)

	path, err := svc.GetFolderFullPath(ctx, folderID, userID)

	assert.NoError(t, err)
	assert.Equal(t, expectedPath, path)
	mockQ.AssertExpectations(t)
}

func TestGetFolderByNameInParent(t *testing.T) {
	svc, mockQ, _, _, _ := newTestFolderService()
	ctx := context.Background()
	userID := int32(4)
	parentID := (*uuid.UUID)(nil)
	expected := database.Folder{ID: uuid.New(), Name: "Docs"}

	mockQ.On("GetFolderByNameInParent", ctx, database.GetFolderByNameInParentParams{
		UserID:   util.ToNullInt32(&userID),
		Name:     "Docs",
		ParentID: util.ToNullUUID(parentID),
	}).Return(expected, nil)

	result, err := svc.GetFolderByNameInParent(ctx, userID, "Docs", parentID)

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	mockQ.AssertExpectations(t)
}

func TestCreateFolder(t *testing.T) {
	svc, mockQ, _, _, mockLocal := newTestFolderService()
	ctx := context.Background()
	userID := int32(1)
	folderID := uuid.New()
	name := "NewFolder"

	mockQ.On("CreateFolder", ctx, mock.MatchedBy(func(arg database.CreateFolderParams) bool {
		return arg.Name == name
	})).Return(database.Folder{ID: folderID, Name: name}, nil)

	mockQ.On("GetFolderFullPath", ctx, mock.MatchedBy(func(arg database.GetFolderFullPathParams) bool {
		return arg.ID == folderID && arg.UserID.Int32 == userID && arg.UserID.Valid
	})).Return("path/to/"+name, nil)

	mockLocal.On("CreateDirectory", ctx, userID, "path/to/"+name).Return(nil)

	folder, err := svc.CreateFolder(ctx, userID, name, nil)
	assert.NoError(t, err)
	assert.Equal(t, folderID, folder.ID)

	mockQ.AssertExpectations(t)
	mockLocal.AssertExpectations(t)
}

func TestGetZippedFoldersForDownload(t *testing.T) {
	svc, mockQ, _, _, mockLocal := newTestFolderService()
	ctx := context.Background()
	userID := int32(1)
	folderID := uuid.New()
	w := &bytes.Buffer{}

	mockQ.On("GetFolderByID", ctx, mock.MatchedBy(func(arg database.GetFolderByIDParams) bool {
		return arg.ID == folderID && arg.UserID.Int32 == userID && arg.UserID.Valid
	})).Return(database.Folder{ID: folderID, Name: "Folder1"}, nil)

	mockQ.On("GetFolderFullPath", ctx, mock.MatchedBy(func(arg database.GetFolderFullPathParams) bool {
		return arg.ID == folderID && arg.UserID.Int32 == userID && arg.UserID.Valid
	})).Return("path/to/Folder1", nil)

	mockLocal.On("ZipMultipleFolders", ctx, userID, []string{"path/to/Folder1"}, w).Return(nil)

	folders, err := svc.GetZippedFoldersForDownload(ctx, []uuid.UUID{folderID}, userID, w)
	assert.NoError(t, err)
	assert.Len(t, folders, 1)
	assert.Equal(t, folderID, folders[0].ID)

	mockQ.AssertExpectations(t)
	mockLocal.AssertExpectations(t)
}

func TestUploadFolder_FileAndFolder(t *testing.T) {
	svc, mockQ, mockFile, mockUser, mockLocal := newTestFolderService()
	ctx := context.Background()
	userID := int32(1)
	parentID := (*uuid.UUID)(nil)

	fileContent := io.NopCloser(bytes.NewReader([]byte("data")))

	items := []services.FolderUploadItem{
		{
			Name:     "FolderA",
			IsFolder: true,
			Children: []services.FolderUploadItem{
				{
					Name:      "file.txt",
					IsFolder:  false,
					SizeBytes: 100,
					Content:   fileContent,
				},
			},
		},
	}

	mockQ.On("GetFolderByNameInParent", ctx, mock.MatchedBy(func(arg database.GetFolderByNameInParentParams) bool {
		return arg.UserID.Int32 == userID && arg.UserID.Valid &&
			arg.Name == "FolderA" &&
			(!arg.ParentID.Valid && parentID == nil)
	})).Return(database.Folder{}, sql.ErrNoRows)

	folderID := uuid.New()

	mockQ.On("CreateFolder", ctx, mock.MatchedBy(func(arg database.CreateFolderParams) bool {
		return arg.UserID.Int32 == userID && arg.UserID.Valid &&
			arg.Name == "FolderA" &&
			(!arg.ParentID.Valid && parentID == nil)
	})).Return(database.Folder{ID: folderID, Name: "FolderA"}, nil)

	mockQ.On("GetFolderFullPath", ctx, mock.MatchedBy(func(arg database.GetFolderFullPathParams) bool {
		return arg.ID == folderID && arg.UserID.Int32 == userID && arg.UserID.Valid
	})).Return("path/to/FolderA", nil)

	mockFile.On("GetFileByNameInFolder", ctx, mock.MatchedBy(func(id *uuid.UUID) bool {
		return *id == folderID
	}), userID, "file.txt").Return(database.File{}, sql.ErrNoRows)

	mockFile.On("UploadFile", ctx, mock.MatchedBy(func(id *uuid.UUID) bool {
		return *id == folderID
	}), userID, "file.txt", int64(100), "", mock.Anything, false).
		Return(database.File{ID: uuid.New()}, nil)

	mockLocal.On("CreateDirectory", ctx, userID, "path/to/FolderA").Return(nil)

	mockUser.On("AdjustUsedStorage", ctx, userID, mock.Anything).Return(nil)

	result, err := svc.UploadFolder(ctx, userID, parentID, items, false, "")
	assert.NoError(t, err)
	assert.Len(t, result.Created, 2)

	mockQ.AssertExpectations(t)
	mockFile.AssertExpectations(t)
	mockLocal.AssertExpectations(t)
}

func TestDeleteFolders(t *testing.T) {
	svc, mockQ, _, mockUsers, mockLocal := newTestFolderService()
	ctx := context.Background()
	userID := int32(1)
	folderID := uuid.New()

	mockQ.On("DeleteFolders", ctx, mock.Anything).Return(int64(1), nil)
	mockQ.On("GetFolderFullPath", ctx, mock.MatchedBy(func(arg database.GetFolderFullPathParams) bool {
		return arg.ID == folderID && arg.UserID.Int32 == userID && arg.UserID.Valid
	})).Return("path/to/Folder", nil)

	mockLocal.On("DeleteDirectory", ctx, userID, "path/to/Folder").Return(nil)
	mockLocal.On("GetDirectorySize", ctx, userID, "path/to/Folder").Return(int64(100), nil)
	mockUsers.On("AdjustUsedStorage", ctx, userID, -int64(100)).Return(nil)

	err := svc.DeleteFolders(ctx, []uuid.UUID{folderID}, userID)
	assert.NoError(t, err)
}

func TestUpdateFolderMetadata(t *testing.T) {
	svc, mockQ, _, _, _ := newTestFolderService()
	ctx := context.Background()
	userID := int32(1)
	folderID := uuid.New()
	newName := "Updated"

	mockQ.On("UpdateFolderMetadata", ctx, mock.Anything).Return(int64(1), nil)

	err := svc.UpdateFolderMetadata(ctx, folderID, userID, newName)
	assert.NoError(t, err)
}

func TestUpdateFoldersParent(t *testing.T) {
	svc, mockQ, _, _, _ := newTestFolderService()
	ctx := context.Background()
	userID := int32(1)
	folderID := uuid.New()
	newParentID := uuid.New()

	newParentFolder := database.Folder{ID: newParentID, UserID: util.ToNullInt32(&userID), Name: "Parent"}

	mockQ.On("GetFolderByID", ctx, database.GetFolderByIDParams{
		ID:     newParentID,
		UserID: util.ToNullInt32(&userID),
	}).Return(newParentFolder, nil)

	mockQ.On("UpdateFoldersParent", ctx, mock.Anything).Return(int64(1), nil)

	err := svc.UpdateFoldersParent(ctx, []database.Folder{{ID: folderID}}, userID, &newParentID)
	assert.NoError(t, err)

	mockQ.AssertExpectations(t)
}

func TestMoveFolders(t *testing.T) {
	svc, mockQ, mockFiles, _, mockLocal := newTestFolderService() // <-- mockFiles for FileService
	ctx := context.Background()
	userID := int32(1)
	folderID := uuid.New()
	newParentID := uuid.New()

	folder := database.Folder{ID: folderID, UserID: util.ToNullInt32(&userID), Name: "Folder"}
	newParentFolder := database.Folder{ID: newParentID, UserID: util.ToNullInt32(&userID), Name: "Parent"}

	mockQ.On("GetFolderByID", ctx, database.GetFolderByIDParams{
		ID:     folderID,
		UserID: util.ToNullInt32(&userID),
	}).Return(folder, nil)
	mockQ.On("GetFolderByID", ctx, database.GetFolderByIDParams{
		ID:     newParentID,
		UserID: util.ToNullInt32(&userID),
	}).Return(newParentFolder, nil)

	mockQ.On("ListFoldersByParent", ctx, database.ListFoldersByParentParams{
		UserID:   util.ToNullInt32(&userID),
		ParentID: util.ToNullUUID(&newParentID),
	}).Return([]database.Folder{}, nil)

	mockQ.On("GetFolderFullPath", ctx, database.GetFolderFullPathParams{
		ID:     folderID,
		UserID: util.ToNullInt32(&userID),
	}).Return("path/old", nil)
	mockQ.On("GetFolderFullPath", ctx, database.GetFolderFullPathParams{
		ID:     newParentID,
		UserID: util.ToNullInt32(&userID),
	}).Return("path/new", nil)

	mockQ.On("UpdateFoldersParent", ctx, mock.Anything).Return(int64(1), nil)

	mockFiles.On("ListFilesRecursive", ctx, folderID, userID).
		Return([]database.ListFilesRecursiveRow{
			{
				FileID:   uuid.New(),
				FilePath: "path/old/file.txt",
				Name:     "file.txt",
			},
		}, nil)
	mockFiles.On("UpdateFileNameAndPath", ctx, mock.Anything, userID, mock.Anything, mock.Anything).Return(nil)

	mockLocal.On("MoveDirectory", mock.Anything, userID, "path/old",
		mock.MatchedBy(func(dest string) bool {
			return filepath.ToSlash(dest) == filepath.ToSlash(filepath.Join("path/new", folder.Name))
		}), false).Return(nil)

	err := svc.MoveFolders(ctx, []uuid.UUID{folderID}, userID, &newParentID, false)
	assert.NoError(t, err)

	mockQ.AssertExpectations(t)
	mockFiles.AssertExpectations(t)
	mockLocal.AssertExpectations(t)
}

func TestRenameFolder(t *testing.T) {
	svc, mockQ, mockFile, _, mockLocal := newTestFolderService()
	ctx := context.Background()
	userID := int32(1)
	folderID := uuid.New()
	oldName := "OldFolder"
	newName := "NewFolder"
	oldPath := filepath.Join("path", "to", "OldFolder")
	newPath := filepath.Join("path", "to", "NewFolder")

	mockQ.On("GetFolderByID", ctx, mock.MatchedBy(func(arg database.GetFolderByIDParams) bool {
		return arg.ID == folderID
	})).Return(database.Folder{
		ID:       folderID,
		Name:     oldName,
		ParentID: uuid.NullUUID{Valid: true, UUID: uuid.New()},
	}, nil)

	mockQ.On("GetFolderFullPath", ctx, mock.MatchedBy(func(arg database.GetFolderFullPathParams) bool {
		return arg.ID == folderID && arg.UserID.Int32 == userID && arg.UserID.Valid
	})).Return(oldPath, nil)

	mockQ.On("GetFolderFullPath", ctx, mock.MatchedBy(func(arg database.GetFolderFullPathParams) bool {
		return arg.UserID.Int32 == userID && arg.UserID.Valid
	})).Return("path/to", nil)

	mockQ.On("GetFolderByNameInParent", ctx, mock.MatchedBy(func(arg database.GetFolderByNameInParentParams) bool {
		return arg.UserID.Int32 == userID &&
			arg.UserID.Valid &&
			arg.Name == newName
	})).Return(database.Folder{}, sql.ErrNoRows)

	mockQ.On("UpdateFolderMetadata", ctx, mock.MatchedBy(func(arg database.UpdateFolderMetadataParams) bool {
		return arg.ID == folderID && arg.UserID == util.ToNullInt32(&userID) && arg.Name == newName
	})).Return(int64(1), nil)

	mockFile.On("ListFilesRecursive", ctx, folderID, userID).Return([]database.ListFilesRecursiveRow{}, nil)

	mockLocal.On("MoveDirectory", ctx, userID, oldPath,
		mock.MatchedBy(func(dest string) bool {
			return filepath.ToSlash(dest) == filepath.ToSlash(newPath)
		}),
		false,
	).Return(nil)

	err := svc.RenameFolder(ctx, folderID, newName, userID, false)
	assert.NoError(t, err)

	mockQ.AssertExpectations(t)
	mockFile.AssertExpectations(t)
	mockLocal.AssertExpectations(t)
}
