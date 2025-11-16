package search

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/middleware"
	"github.com/bellezhang119/cloud-storage/internal/util"
)

type Queries interface {
	SearchFilesAndFolders(ctx context.Context, arg database.SearchFilesAndFoldersParams) (database.SearchFilesAndFoldersRow, error)
}

type Service struct {
	queries Queries
}

type Result struct {
	Folders []database.Folder `json:"folders"`
	Files   []database.File   `json:"files"`
}

func NewService(q Queries) *Service {
	return &Service{queries: q}
}

func (s *Service) SearchFilesAndFolders(ctx context.Context, search string, userID int32, sortBy string, asc bool, filter []string) (Result, error) {
	logger := middleware.GetLogger(ctx)
	logger.Info("search started")

	row, err := s.queries.SearchFilesAndFolders(ctx, database.SearchFilesAndFoldersParams{
		Column1: search,
		UserID:  util.ToNullInt32(&userID),
		Column3: sortBy,
		Column4: asc,
		Column5: filter,
	})
	if err != nil {
		logger.Error("search failed", "error", err)
		return Result{}, fmt.Errorf("failed to search: %w", err)
	}

	var res Result

	// Decode folders
	if row.Folders != nil {
		data, ok := row.Folders.([]byte)
		if !ok {
			data, _ = json.Marshal(row.Folders) // fallback
		}
		if err := json.Unmarshal(data, &res.Folders); err != nil {
			logger.Error("folder json unmarshal failed", "error", err)
			return Result{}, fmt.Errorf("failed to unmarshal folders: %w", err)
		}
	}

	// Decode files
	if row.Files != nil {
		data, ok := row.Files.([]byte)
		if !ok {
			data, _ = json.Marshal(row.Files)
		}
		if err := json.Unmarshal(data, &res.Files); err != nil {
			logger.Error("file json unmarshal failed", "error", err)
			return Result{}, fmt.Errorf("failed to unmarshal files: %w", err)
		}
	}

	logger.Info("search completed successfully")
	return res, nil
}
