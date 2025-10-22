package storage

import (
	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/storage/local"
	"github.com/bellezhang119/cloud-storage/internal/storage/services"
)

type Service struct {
	Files   *services.FileServiceImpl
	Folders *services.FolderServiceImpl
	Local   local.Storage
}

func NewService(db *database.Queries, localStore local.Storage) *Service {
	folderSvc := services.NewFolderService(db, localStore)
	fileSvc := services.NewFileService(db, localStore)

	fileSvc.SetFolderService(folderSvc)
	folderSvc.SetFileService(fileSvc)

	return &Service{
		Files:   fileSvc,
		Folders: folderSvc,
		Local:   localStore,
	}
}
