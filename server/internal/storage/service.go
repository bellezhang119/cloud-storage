package storage

import (
	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/storage/file"
	"github.com/bellezhang119/cloud-storage/internal/storage/folder"
	"github.com/bellezhang119/cloud-storage/internal/storage/local"
)

type Service struct {
	Files   *file.Service
	Folders *folder.Service
	Local   local.Storage
}

func NewService(db *database.Queries, localStore local.Storage) *Service {
	folderSvc := folder.NewService(db, localStore)
	fileSvc := file.NewService(db, localStore)

	fileSvc.SetFolderService(folderSvc)
	folderSvc.SetFileService(fileSvc)

	return &Service{
		Files:   fileSvc,
		Folders: folderSvc,
		Local:   localStore,
	}
}
