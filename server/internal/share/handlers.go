package share

import (
	"context"
	"net/http"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/middleware"
	"github.com/bellezhang119/cloud-storage/internal/util"
	"github.com/google/uuid"
)

type ServiceInterface interface {
	CheckUserFileAccess(ctx context.Context, fileID uuid.UUID, userID int32) (bool, error)
	CreateFileShare(ctx context.Context, fileID uuid.UUID, userID int32) (database.FileShare, error)
	DeleteFileShare(ctx context.Context, fileID uuid.UUID, userID int32) error
	GetFileShare(ctx context.Context, fileID uuid.UUID, userID int32) (database.FileShare, error)
	ListFileShares(ctx context.Context, fileID uuid.UUID) ([]database.FileShare, error)
	ListFilesSharedWithUser(ctx context.Context, userID int32) ([]database.File, error)
	CheckUserFolderAccess(ctx context.Context, folderID uuid.UUID, userID int32) (bool, error)
	CreateFolderShare(ctx context.Context, folderID uuid.UUID, userID int32) (database.FolderShare, error)
	DeleteFolderShare(ctx context.Context, folderID uuid.UUID, userID int32) error
	GetFolderShare(ctx context.Context, folderID uuid.UUID, userID int32) (database.FolderShare, error)
	GetSharedFolderContent(ctx context.Context, folderID uuid.UUID, userID int32) ([]database.Folder, []database.File, error)
	ListFolderShares(ctx context.Context, folderID uuid.UUID) ([]database.FolderShare, error)
	ListFoldersSharedWithUser(ctx context.Context, userID int32) ([]database.Folder, error)
}

type GetSharedFolderContentResponse struct {
	Folders []database.Folder `json:"folders"`
	Files   []database.File   `json:"files"`
}

func CheckUserFileAccessHandler(s ServiceInterface) http.HandlerFunc {
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

		fileID, err := util.GetFileIDFromPath(r)
		if err != nil {
			logger.Error("invalid file ID", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		logger = logger.With("file_id", fileID)

		access, err := s.CheckUserFileAccess(r.Context(), fileID, userID)
		if err != nil {
			logger.Error("failed to check file access", "error", err)
			util.RespondWithError(w, http.StatusForbidden, err.Error())
			return
		}

		logger.Info("file access checked", "file_id", fileID)
		util.RespondWithJSON(w, http.StatusOK, access)
	}
}

func CreateFileShareHandler(s ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("creating file share")

		userID, err := util.GetUserIDFromPathAndCheck(r)
		if err != nil {
			logger.Error("invalid user ID", "error", err)
			util.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		logger = logger.With("ctx_user_id", userID)

		fileID, err := util.GetFileIDFromPath(r)
		if err != nil {
			logger.Error("invalid file ID", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		logger = logger.With("file_id", fileID)

		fileShare, err := s.CreateFileShare(r.Context(), fileID, userID)
		if err != nil {
			logger.Error("failed to create file share", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logger.Info("created file share", "file_id", fileShare.FileID)
		util.RespondWithJSON(w, http.StatusCreated, fileShare)
	}
}

func DeleteFileShareHandler(s ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("creating file share")

		userID, err := util.GetUserIDFromPathAndCheck(r)
		if err != nil {
			logger.Error("invalid user ID", "error", err)
			util.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		logger = logger.With("ctx_user_id", userID)

		fileID, err := util.GetFileIDFromPath(r)
		if err != nil {
			logger.Error("invalid file ID", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		logger = logger.With("file_id", fileID)

		err = s.DeleteFileShare(r.Context(), fileID, userID)
		if err != nil {
			logger.Error("failed to delete file share", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logger.Info("deleted file share", "file_id", fileID)
		util.RespondWithJSON(w, http.StatusNoContent, nil)
	}
}

func GetFileShareHandler(s ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("getting file share")

		userID, err := util.GetUserIDFromPathAndCheck(r)
		if err != nil {
			logger.Error("invalid user ID", "error", err)
			util.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		logger = logger.With("ctx_user_id", userID)

		fileID, err := util.GetFileIDFromPath(r)
		if err != nil {
			logger.Error("invalid file ID", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		logger = logger.With("file_id", fileID)

		fileShare, err := s.GetFileShare(r.Context(), fileID, userID)
		if err != nil {
			logger.Error("failed to get file share", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logger.Info("file share", "file_id", fileShare.FileID)
		util.RespondWithJSON(w, http.StatusOK, fileShare)
	}
}

func ListFileSharesHandler(s ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("listing file shares")

		fileID, err := util.GetFileIDFromPath(r)
		if err != nil {
			logger.Error("invalid file ID", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		logger = logger.With("file_id", fileID)

		fileShares, err := s.ListFileShares(r.Context(), fileID)
		if err != nil {
			logger.Error("failed to list file shares", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logger.Info("listed file shares", "file_id", fileID)
		util.RespondWithJSON(w, http.StatusOK, fileShares)
	}
}

func ListFilesSharedWithUserHandler(s ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("listing files shared with user")

		userID, err := util.GetUserIDFromPathAndCheck(r)
		if err != nil {
			logger.Error("invalid user ID", "error", err)
			util.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		logger = logger.With("ctx_user_id", userID)

		files, err := s.ListFilesSharedWithUser(r.Context(), userID)
		if err != nil {
			logger.Error("failed to list files shared with user", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logger.Info("listed files shared with user", "file_ids", files)
		util.RespondWithJSON(w, http.StatusOK, files)
	}
}

func CheckUserFolderAccessHandler(s ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("checking user folder access")

		userID, err := util.GetUserIDFromPathAndCheck(r)
		if err != nil {
			logger.Error("invalid user ID", "error", err)
			util.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		logger = logger.With("ctx_user_id", userID)

		folderID, err := util.GetFolderIDFromPath(r)
		if err != nil {
			logger.Error("invalid folder ID", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		logger = logger.With("folder_id", folderID)

		access, err := s.CheckUserFileAccess(r.Context(), folderID, userID)
		if err != nil {
			logger.Error("failed to check folder access", "error", err)
			util.RespondWithError(w, http.StatusForbidden, err.Error())
			return
		}

		logger.Info("folder access checked", "folder_id", folderID)
		util.RespondWithJSON(w, http.StatusOK, access)
	}
}

func CreateFolderShareHandler(s ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("creating folder share")

		userID, err := util.GetUserIDFromPathAndCheck(r)
		if err != nil {
			logger.Error("invalid user ID", "error", err)
			util.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		logger = logger.With("ctx_user_id", userID)

		folderID, err := util.GetFolderIDFromPath(r)
		if err != nil {
			logger.Error("invalid folder ID", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		logger = logger.With("folder_id", folderID)

		folderShare, err := s.CreateFileShare(r.Context(), folderID, userID)
		if err != nil {
			logger.Error("failed to create folder share", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logger.Info("created folder share", "folder_id", folderShare.FileID)
		util.RespondWithJSON(w, http.StatusCreated, folderShare)
	}
}

func DeleteFolderShareHandler(s ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("deleting folder share")

		userID, err := util.GetUserIDFromPathAndCheck(r)
		if err != nil {
			logger.Error("invalid user ID", "error", err)
			util.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		logger = logger.With("ctx_user_id", userID)

		folderID, err := util.GetFolderIDFromPath(r)
		if err != nil {
			logger.Error("invalid folder ID", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		logger = logger.With("folder_id", folderID)

		err = s.DeleteFolderShare(r.Context(), folderID, userID)

		if err != nil {
			logger.Error("failed to delete folder share", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logger.Info("deleted folder share", "folder_id", folderID)
		util.RespondWithJSON(w, http.StatusNoContent, nil)
	}
}

func GetFolderShareHandler(s ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("getting folder share")

		userID, err := util.GetUserIDFromPathAndCheck(r)
		if err != nil {
			logger.Error("invalid user ID", "error", err)
			util.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		logger = logger.With("ctx_user_id", userID)

		folderID, err := util.GetFolderIDFromPath(r)
		if err != nil {
			logger.Error("invalid folder ID", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		logger = logger.With("folder_id", folderID)

		folderShare, err := s.GetFolderShare(r.Context(), folderID, userID)
		if err != nil {
			logger.Error("failed to get folder share", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logger.Info("folder share", "folder_id", folderID)
		util.RespondWithJSON(w, http.StatusOK, folderShare)
	}
}

func GetSharedFolderContentHandler(s ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("getting shared folder content")

		userID, err := util.GetUserIDFromPathAndCheck(r)
		if err != nil {
			logger.Error("invalid user ID", "error", err)
			util.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		logger = logger.With("ctx_user_id", userID)

		folderID, err := util.GetFolderIDFromPath(r)
		if err != nil {
			logger.Error("invalid folder ID", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		logger = logger.With("folder_id", folderID)

		folders, files, err := s.GetSharedFolderContent(r.Context(), folderID, userID)
		if err != nil {
			logger.Error("failed to get shared folder content", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logger.Info("shared folder content", "folder_id", folderID)
		util.RespondWithJSON(w, http.StatusOK, GetSharedFolderContentResponse{
			Folders: folders,
			Files:   files,
		})
	}
}

func ListFolderSharesHandler(s ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("listing folder shares")

		folderID, err := util.GetFolderIDFromPath(r)
		if err != nil {
			logger.Error("invalid folder ID", "error", err)
			util.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		logger = logger.With("folder_id", folderID)

		folderShares, err := s.ListFolderShares(r.Context(), folderID)
		if err != nil {
			logger.Error("failed to list folder shares", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logger.Info("folder shares", "folder_id", folderID)
		util.RespondWithJSON(w, http.StatusOK, folderShares)
	}
}

func ListFoldersSharedWithUserHandler(s ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.GetLogger(r.Context())
		logger.Info("listing folders shared with user")

		userID, err := util.GetUserIDFromPathAndCheck(r)
		if err != nil {
			logger.Error("invalid user ID", "error", err)
			util.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		logger = logger.With("ctx_user_id", userID)

		folders, err := s.ListFoldersSharedWithUser(r.Context(), userID)
		if err != nil {
			logger.Error("failed to list folder shared with user", "error", err)
			util.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logger.Info("folder shared with user", "folder_id", folders)
		util.RespondWithJSON(w, http.StatusOK, folders)
	}
}
