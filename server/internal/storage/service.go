package storage

import (
	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/storage/local"
	"github.com/bellezhang119/cloud-storage/internal/storage/services"
	"github.com/bellezhang119/cloud-storage/internal/user"
)

type Service struct {
	Files   *services.FileServiceImpl
	Folders *services.FolderServiceImpl
	Local   local.Storage
}

func NewService(db *database.Queries, userService *user.Service, localStore local.Storage) *Service {
	folderSvc := services.NewFolderService(db, userService, localStore)
	fileSvc := services.NewFileService(db, userService, localStore)

	fileSvc.SetFolderService(folderSvc)
	folderSvc.SetFileService(fileSvc)

	return &Service{
		Files:   fileSvc,
		Folders: folderSvc,
		Local:   localStore,
	}
}
