package server

import (
	"net/http"

	"github.com/bellezhang119/cloud-storage/internal/auth"
	"github.com/bellezhang119/cloud-storage/internal/middleware"
	"github.com/bellezhang119/cloud-storage/internal/search"
	"github.com/bellezhang119/cloud-storage/internal/share"
	"github.com/bellezhang119/cloud-storage/internal/storage"
	"github.com/bellezhang119/cloud-storage/internal/user"
	"github.com/bellezhang119/cloud-storage/internal/util"
)

func NewRouter(authService *auth.Service, userService *user.Service, storageService *storage.Service, shareService *share.Service, searchService *search.Service) http.Handler {
	authMiddleware := middleware.AuthMiddleware(util.VerifyAccessToken)
	mux := http.NewServeMux()
	loggedMux := middleware.LoggingMiddleware(mux)

	// Auth routes
	mux.HandleFunc("POST /auth/register", auth.RegisterHandler(authService, util.SendEmail))
	mux.HandleFunc("GET /auth/verify", auth.VerifyEmailHandler(authService))
	mux.HandleFunc("POST /auth/resend-verification", auth.SendVerificationEmailHandler(authService, util.SendEmail))
	mux.HandleFunc("POST /auth/login", auth.LoginHandler(authService))
	mux.HandleFunc("POST /auth/refresh", auth.RefreshTokenHandler(authService))

	// User routes
	mux.Handle("GET /user/{user_id}", authMiddleware(user.GetUserByIDHandler(userService)))
	mux.Handle("GET /user/email/{user_email}", authMiddleware(user.GetUserByEmailHandler(userService)))
	mux.Handle("PATCH /user/{user_id}/password", authMiddleware(user.UpdatePasswordHandler(userService)))
	mux.Handle("DELETE /user/{user_id}", authMiddleware(user.DeleteUserHandler(userService)))

	// File routes
	mux.Handle("GET /user/{user_id}/file/{file_id}", authMiddleware(storage.GetFileByIDHandler(storageService)))
	mux.Handle("GET /user/{user_id}/folder/{folder_id}/file", authMiddleware(storage.GetFileByNameInFolderHandler(storageService)))
	mux.Handle("GET /user/{user_id}/folder/{folder_id}/files", authMiddleware(storage.ListFilesInFolderHandler(storageService)))
	mux.Handle("POST /user/{user_id}/folder/{parent_id}/file/upload", authMiddleware(storage.UploadFileHandler(storageService)))
	mux.Handle("GET /user/{user_id}/files/download", authMiddleware(storage.DownloadFilesHandler(storageService)))
	mux.Handle("DELETE /user/{user_id}/files", authMiddleware(storage.DeleteFilesHandler(storageService)))
	mux.Handle("PATCH /user/{user_id}/folder/{parent_id}/files/move", authMiddleware(storage.MoveFilesHandler(storageService)))
	mux.Handle("PATCH /user/{user_id}/file/{file_id}", authMiddleware(storage.RenameFileHandler(storageService)))

	// Folder routes
	mux.Handle("GET /user/{user_id}/folder/{folder_id}", authMiddleware(storage.GetFolderByIDHandler(storageService)))
	mux.Handle("GET /user/{user_id}/folder/{parent_id}/subfolders", authMiddleware(storage.ListFoldersByParentHandler(storageService)))
	mux.Handle("GET /user/{user_id}/folder/{folder_id}/path", authMiddleware(storage.GetFolderFullPathHandler(storageService)))
	mux.Handle("GET /user/{user_id}/folder/{parent_id}/subfolder", authMiddleware(storage.GetFolderByNameInParentHandler(storageService)))
	mux.Handle("POST /user/{user_id}/folder/{parent_id}", authMiddleware(storage.CreateFolderHandler(storageService)))
	mux.Handle("GET /user/{user_id}/folders/download", authMiddleware(storage.DownloadFoldersHandler(storageService)))
	mux.Handle("POST /user/{user_id}/folder/{parent_id}/upload", authMiddleware(storage.UploadFolderHandler(storageService)))
	mux.Handle("DELETE /user/{user_id}/folders", authMiddleware(storage.DeleteFoldersHandler(storageService)))
	mux.Handle("PATCH /user/{user_id}/folder/{parent_id}/move", authMiddleware(storage.MoveFoldersHandler(storageService)))
	mux.Handle("PATCH /user/{user_id}/folder/{folder_id}", authMiddleware(storage.RenameFolderHandler(storageService)))

	// Share routes
	mux.Handle("GET /user/{user_id}/file/{file_id}/access", authMiddleware(share.CheckUserFileAccessHandler(shareService)))
	mux.Handle("POST /user/{user_id}/file/{file_id}/share", authMiddleware(share.CreateFileShareHandler(shareService)))
	mux.Handle("DELETE /user/{user_id}/file/{file_id}/share", authMiddleware(share.DeleteFileShareHandler(shareService)))
	mux.Handle("GET /user/{user_id}/file/{file_id}/share", authMiddleware(share.GetFileShareHandler(shareService)))
	mux.Handle("GET /file/{file_id}/shares", authMiddleware(share.ListFileSharesHandler(shareService)))
	mux.Handle("GET /user/{user_id}/files/shares", authMiddleware(share.ListFilesSharedWithUserHandler(shareService)))
	mux.Handle("GET /user/{user_id}/folder/{folder_id}/access", authMiddleware(share.CheckUserFolderAccessHandler(shareService)))
	mux.Handle("POST user/{user_id}/folder/{folder_id}/share", authMiddleware(share.CreateFolderShareHandler(shareService)))
	mux.Handle("DELETE /user/{user_id}/folder/{folder_id}/share", authMiddleware(share.DeleteFolderShareHandler(shareService)))
	mux.Handle("GET /user/{user_id}/folder/{folder_id}/share", authMiddleware(share.GetFolderShareHandler(shareService)))
	mux.Handle("GET /user/{user_id}/folder/{folder_id}/content", authMiddleware(share.GetSharedFolderContentHandler(shareService)))
	mux.Handle("GET /folder/{folder_id}/shares", authMiddleware(share.ListFolderSharesHandler(shareService)))
	mux.Handle("GET /user/{user_id}/folders/shares", authMiddleware(share.ListFoldersSharedWithUserHandler(shareService)))

	// Search route
	mux.Handle("GET /user/{user_id}/search", authMiddleware(search.SearchHandler(searchService)))

	// Health checks
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Ready"))
	})
	mux.HandleFunc("GET /err", HandleErr)

	return loggedMux
}
