package server

import (
	"net/http"

	"github.com/bellezhang119/cloud-storage/internal/auth"
	"github.com/bellezhang119/cloud-storage/internal/middleware"
	"github.com/bellezhang119/cloud-storage/internal/storage"
	"github.com/bellezhang119/cloud-storage/internal/util"
)

func NewRouter(authService *auth.Service, storageService *storage.Service) *http.ServeMux {
	authMiddleware := middleware.AuthMiddleware(util.VerifyAccessToken)
	mux := http.NewServeMux()

	//protectedHandler := auth.AuthMiddleware(util.VerifyAccessToken)

	// Auth routes
	mux.HandleFunc("POST /auth/register", auth.RegisterHandler(authService, util.SendEmail))
	mux.HandleFunc("GET /auth/verify", auth.VerifyEmailHandler(authService))
	mux.HandleFunc("POST /auth/resend-verification", auth.SendVerificationEmailHandler(authService, util.SendEmail))
	mux.HandleFunc("POST /auth/login", auth.LoginHandler(authService))
	mux.HandleFunc("POST /auth/refresh", auth.RefreshTokenHandler(authService))

	// File routes
	mux.Handle("GET /user/{user_id}/file/{file_id}", authMiddleware(storage.GetFileByIDHandler(storageService)))
	mux.Handle("GET /user/{user_id}/folder/{folder_id}/file", authMiddleware(storage.GetFileByNameInFolderHandler(storageService)))
	mux.Handle("GET /user/{user_id}/folder/{folder_id}/files", authMiddleware(storage.ListFilesInFolderHandler(storageService)))
	mux.Handle("POST /user/{user_id}/folder/{parent_id}/file", authMiddleware(storage.UploadFileHandler(storageService)))
	mux.Handle("GET /user/{user_id}/files", authMiddleware(storage.DownloadFilesHandler(storageService)))
	mux.Handle("DELETE /user/{user_id}/files", authMiddleware(storage.DeleteFilesHandler(storageService)))
	mux.Handle("PATCH /user/{user_id}/folder/{parent_id}/files", authMiddleware(storage.MoveFilesHandler(storageService)))
	mux.Handle("PATCH /user/{user_id}/file/{file_id}", authMiddleware(storage.RenameFileHandler(storageService)))

	// Health checks
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Ready"))
	})
	mux.HandleFunc("GET /err", HandleErr)

	return mux
}
