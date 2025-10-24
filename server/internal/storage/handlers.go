package storage

import (
	"net/http"

	"github.com/bellezhang119/cloud-storage/internal/storage/handlers"
)

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
