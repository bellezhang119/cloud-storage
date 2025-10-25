package server

import (
	"net/http"

	"github.com/bellezhang119/cloud-storage/internal/auth"
	"github.com/bellezhang119/cloud-storage/internal/middleware"
	"github.com/bellezhang119/cloud-storage/internal/storage"
	"github.com/bellezhang119/cloud-storage/internal/user"
	"github.com/bellezhang119/cloud-storage/internal/util"
)

func NewRouter(authService *auth.Service, userService *user.Service, storageService *storage.Service) http.Handler {
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
	mux.Handle("PATCH /user/{user_id}/", authMiddleware(user.UpdatePasswordHandler(userService)))
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

	// Health checks
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Ready"))
	})
	mux.HandleFunc("GET /err", HandleErr)

	return loggedMux
}
