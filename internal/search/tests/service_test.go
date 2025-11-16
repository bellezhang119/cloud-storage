package tests

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/search"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockQueries struct {
	mock.Mock
}

func (m *MockQueries) SearchFilesAndFolders(ctx context.Context, arg database.SearchFilesAndFoldersParams) (database.SearchFilesAndFoldersRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(database.SearchFilesAndFoldersRow), args.Error(1)
}

func TestSearchFilesAndFolders(t *testing.T) {
	mockQ := new(MockQueries)
	svc := search.NewService(mockQ)

	folderID := uuid.New()
	fileID := uuid.New()

	expectedFolders := []database.Folder{{ID: folderID, Name: "Folder1"}}
	expectedFiles := []database.File{{ID: fileID, Name: "File1"}}

	foldersJSON, _ := json.Marshal(expectedFolders)
	filesJSON, _ := json.Marshal(expectedFiles)

	row := database.SearchFilesAndFoldersRow{
		Folders: foldersJSON,
		Files:   filesJSON,
	}

	mockQ.On("SearchFilesAndFolders", mock.Anything, mock.Anything).Return(row, nil)

	result, err := svc.SearchFilesAndFolders(context.Background(), "test", 1, "name", true, []string{})

	assert.NoError(t, err)
	assert.Equal(t, expectedFolders, result.Folders)
	assert.Equal(t, expectedFiles, result.Files)
	mockQ.AssertExpectations(t)
}
