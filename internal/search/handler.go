package search

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/bellezhang119/cloud-storage/internal/middleware"
	"github.com/bellezhang119/cloud-storage/internal/util"
)

type ServiceInterface interface {
	SearchFilesAndFolders(ctx context.Context, search string, userID int32, sortBy string, asc bool, filter []string) (Result, error)
}

func SearchHandler(s ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("checking user file access")

		userID, err := util.GetUserIDFromPathAndCheck(r)
		if err != nil {
			logger.Error("invalid user ID", "error", err)
			util.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		logger = logger.With("ctx_user_id", userID)

		search := r.URL.Query().Get("query")
		sortBy := r.URL.Query().Get("sortBy")

		allowedSortBy := map[string]bool{
			"name":       true,
			"created_at": true,
			"updated_at": true,
			"mime_type":  true,
			"size_bytes": true,
		}

		if !allowedSortBy[sortBy] {
			sortBy = "name"
		}

		ascStr := r.URL.Query().Get("asc")

		asc := true
		if ascStr != "" {
			parsed, err := strconv.ParseBool(ascStr)
			if err == nil {
				asc = parsed
			}
		}

		filterStr := r.URL.Query().Get("filter")
		var filter []string
		if filterStr != "" {
			filter = strings.Split(filterStr, ",")
		}

		result, err := s.SearchFilesAndFolders(r.Context(), search, userID, sortBy, asc, filter)
		if err != nil {
			logger.Error("search files error", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logger.Info("search result", "result", result)
		util.RespondWithJSON(w, http.StatusOK, result)
	}
}
