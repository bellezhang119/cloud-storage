package storage

import (
	"net/http"

	"github.com/bellezhang119/cloud-storage/internal/storage/handlers"
)

// --- File handlers ---

func GetFileByIDHandler(s *Service) http.HandlerFunc {
	return handlers.GetFileByIDHandler(s.Files)
}

func GetFileByNameInFolderHandler(s *Service) http.HandlerFunc {
	return handlers.GetFileByNameInFolderHandler(s.Files)
}

func ListFilesInFolderHandler(s *Service) http.HandlerFunc {
	return handlers.ListFilesInFolderHandler(s.Files)
}

func UploadFileHandler(s *Service) http.HandlerFunc {
	return handlers.UploadFileHandler(s.Files)
}

func DownloadFilesHandler(s *Service) http.HandlerFunc {
	return handlers.DownloadFilesHandler(s.Files)
}

func DeleteFilesHandler(s *Service) http.HandlerFunc {
	return handlers.DeleteFilesHandler(s.Files)
}

func MoveFilesHandler(s *Service) http.HandlerFunc {
	return handlers.MoveFilesHandler(s.Files)
}

func RenameFileHandler(s *Service) http.HandlerFunc {
	return handlers.RenameFileHandler(s.Files)
}

// --- Folder handlers ---

func GetFolderByIDHandler(s *Service) http.HandlerFunc {
	return handlers.GetFolderByIDHandler(s.Folders)
}

func ListFoldersByParentHandler(s *Service) http.HandlerFunc {
	return handlers.ListFoldersByParentHandler(s.Folders)
}

func GetFolderFullPathHandler(s *Service) http.HandlerFunc {
	return handlers.GetFolderFullPathHandler(s.Folders)
}

func GetFolderByNameInParentHandler(s *Service) http.HandlerFunc {
	return handlers.GetFolderByNameInParentHandler(s.Folders)
}

func CreateFolderHandler(s *Service) http.HandlerFunc {
	return handlers.CreateFolderHandler(s.Folders)
}

func DownloadFoldersHandler(s *Service) http.HandlerFunc {
	return handlers.DownloadFoldersHandler(s.Folders)
}

func UploadFolderHandler(s *Service) http.HandlerFunc {
	return handlers.UploadFolderHandler(s.Folders)
}

func DeleteFoldersHandler(s *Service) http.HandlerFunc {
	return handlers.DeleteFoldersHandler(s.Folders)
}

func MoveFoldersHandler(s *Service) http.HandlerFunc {
	return handlers.MoveFoldersHandler(s.Folders)
}

func RenameFolderHandler(s *Service) http.HandlerFunc {
	return handlers.RenameFolderHandler(s.Folders)
}
