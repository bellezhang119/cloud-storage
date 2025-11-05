package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/middleware"
	"github.com/bellezhang119/cloud-storage/internal/search"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockService struct {
	mock.Mock
}

func (m *MockService) SearchFilesAndFolders(ctx context.Context, s string, userID int32, sortBy string, asc bool, filter []string) (search.Result, error) {
	args := m.Called(ctx, s, userID, sortBy, asc, filter)
	return args.Get(0).(search.Result), args.Error(1)
}

func TestSearchFilesAndFoldersHandler(t *testing.T) {
	mockSvc := &MockService{}

	folderID := uuid.New()
	fileID := uuid.New()
	expectedResult := search.Result{
		Folders: []database.Folder{{ID: folderID, Name: "report"}},
		Files:   []database.File{{ID: fileID, Name: "report.txt"}},
	}

	mockSvc.On(
		"SearchFilesAndFolders",
		mock.Anything, // context
		"report",      // search string
		int32(1),      // userID
		"name",        // sortBy
		true,          // asc
		[]string{"pdf"},
	).Return(expectedResult, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /user/{user_id}/search", search.SearchHandler(mockSvc))

	req := httptest.NewRequest("GET", "/user/1/search?query=report&sortBy=name&asc=true&filter=pdf", nil)

	ctx := context.WithValue(req.Context(), middleware.UserIDKey, int32(1))
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var result search.Result
	err := json.NewDecoder(rr.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, expectedResult, result)

	mockSvc.AssertExpectations(t)
}
