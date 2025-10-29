package util

import (
	"fmt"
	"net/http"

	"github.com/bellezhang119/cloud-storage/internal/middleware"
	"github.com/google/uuid"
)

// GetUserIDFromPathAndCheck get user id from path and compare against user id from context
func GetUserIDFromPathAndCheck(r *http.Request) (int32, error) {
	ctxUserID, _ := middleware.GetUserID(r.Context())
	pathUserID := r.PathValue("user_id")

	if string(ctxUserID) != pathUserID {
		return 0, fmt.Errorf("mismatch between context and path user ID")
	}

	return ctxUserID, nil
}

func GetFileIDFromPath(r *http.Request) (uuid.UUID, error) {
	fileIDStr := r.PathValue("file_id")
	if fileIDStr == "" {
		return uuid.Nil, fmt.Errorf("empty file id")
	}

	fileID, err := uuid.Parse(fileIDStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid file id")
	}

	return fileID, nil
}

func GetFolderIDFromPath(r *http.Request) (*uuid.UUID, error) {
	folderIDStr := r.PathValue("folder_id")
	if folderIDStr == "" {
		return nil, nil
	}

	folderID, err := uuid.Parse(folderIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid folder id")
	}

	return &folderID, nil
}

func GetParentIDFromPath(r *http.Request) (*uuid.UUID, error) {
	parentIDStr := r.PathValue("parent_id")
	if parentIDStr == "" {
		return nil, nil
	}

	parentID, err := uuid.Parse(parentIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid parent id")
	}

	return &parentID, nil
}
