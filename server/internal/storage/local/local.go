package local

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/bellezhang119/cloud-storage/internal/middleware"
)

type Storage interface {
	SaveFile(ctx context.Context, userID int32, path string, content io.Reader) error
	ReadFile(ctx context.Context, userID int32, path string) (io.ReadCloser, error)
	DeleteFile(ctx context.Context, userID int32, path string) error
	CreateDirectory(ctx context.Context, userID int32, path string) error
	DeleteDirectory(ctx context.Context, userID int32, path string) error
	MoveFile(ctx context.Context, userID int32, oldPath, newPath string) error
	MoveDirectory(ctx context.Context, userID int32, oldPath, newPath string, overwriteFiles bool) error
	GetDirectorySize(ctx context.Context, userID int32, path string) (int64, error)
	ZipMultipleFolders(ctx context.Context, userID int32, folderPaths []string, w io.Writer) error
}

type LocalStorage struct {
	BasePath string
}

func NewLocalStorage(basePath string) *LocalStorage {
	return &LocalStorage{
		BasePath: basePath,
	}
}

// get the absolute safe path for a user file/folder
func (s *LocalStorage) fullPath(userID int32, path string) string {
	safePath := filepath.Clean(path)
	return filepath.Join(s.BasePath, strconv.Itoa(int(userID)), safePath)
}

// SaveFile writes content to a file, creating directories if needed
func (s *LocalStorage) SaveFile(ctx context.Context, userID int32, path string, content io.Reader) error {
	logger := middleware.GetLogger(ctx).With("path", path)

	full := s.fullPath(userID, path)
	logger.Info("saving file", "full_path", full)

	// create directories
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		logger.Error("failed to create directories", "error", err)
		return fmt.Errorf("creating directories for %s: %w", full, err)
	}

	// write to a temp file for atomic write
	temp := full + ".tmp"
	f, err := os.Create(temp)
	if err != nil {
		logger.Error("failed to create temp file", "temp_file", temp, "error", err)
		return fmt.Errorf("creating temp file %s: %w", temp, err)
	}

	if _, err := io.Copy(f, content); err != nil {
		logger.Error("failed to write to temp file", "temp_file", temp, "error", err)
		return fmt.Errorf("writing to temp file %s: %w", temp, err)
	}

	if err := f.Close(); err != nil {
		logger.Error("failed to close temp file", "temp_file", temp, "error", err)
		return fmt.Errorf("closing temp file %s: %w", temp, err)
	}

	if err := os.Rename(temp, full); err != nil {
		logger.Error("failed to rename temp file to final path", "temp_file", temp, "final_path", full, "error", err)
		return fmt.Errorf("renaming temp file %s to %s: %w", temp, full, err)
	}

	logger.Info("file saved successfully")
	return nil
}

// ReadFile opens a file for reading
func (s *LocalStorage) ReadFile(ctx context.Context, userID int32, path string) (io.ReadCloser, error) {
	logger := middleware.GetLogger(ctx).With("path", path)

	full := s.fullPath(userID, path)
	logger.Info("reading file", "full_path", full)

	f, err := os.Open(full)
	if err != nil {
		logger.Error("failed to open file", "error", err)
		return nil, fmt.Errorf("opening file %s: %w", full, err)
	}

	logger.Info("file opened successfully")
	return f, nil
}

// DeleteFile removes a file
func (s *LocalStorage) DeleteFile(ctx context.Context, userID int32, path string) error {
	logger := middleware.GetLogger(ctx).With("path", path)

	full := s.fullPath(userID, path)
	logger.Info("deleting file", "full_path", full)

	if err := os.Remove(full); err != nil {
		logger.Error("failed to delete file", "error", err)
		return fmt.Errorf("deleting file %s: %w", full, err)
	}

	logger.Info("file deleted successfully")
	return nil
}

// MoveFile moves a file; supports cross-filesystem moves
func (s *LocalStorage) MoveFile(ctx context.Context, userID int32, oldPath, newPath string) error {
	logger := middleware.GetLogger(ctx).With("old_path", oldPath, "new_path", newPath)

	oldFull := s.fullPath(userID, oldPath)
	newFull := s.fullPath(userID, newPath)

	logger.Info("moving file", "old_full", oldFull, "new_full", newFull)

	if err := os.MkdirAll(filepath.Dir(newFull), 0755); err != nil {
		logger.Error("failed to create directories for new path", "error", err)
		return fmt.Errorf("creating directories for %s: %w", newFull, err)
	}

	if err := os.Rename(oldFull, newFull); err == nil {
		logger.Info("file moved successfully via rename")
		return nil
	}
	logger.Info("rename failed, falling back to copy + delete")

	// fallback: copy + delete
	src, err := os.Open(oldFull)
	if err != nil {
		logger.Error("failed to open source file", "error", err)
		return fmt.Errorf("opening source file %s: %w", oldFull, err)
	}
	defer func(src *os.File) {
		err := src.Close()
		if err != nil {

		}
	}(src)

	dst, err := os.Create(newFull)
	if err != nil {
		logger.Error("failed to create destination file", "error", err)
		return fmt.Errorf("creating destination file %s: %w", newFull, err)
	}

	// copy content
	if _, err := io.Copy(dst, src); err != nil {
		err := dst.Close()
		if err != nil {
			return err
		}
		logger.Error("failed to copy file", "error", err)
		return fmt.Errorf("copying from %s to %s: %w", oldFull, newFull, err)
	}

	// close dst before removing old file
	if err := dst.Close(); err != nil {
		logger.Error("failed to close destination file", "error", err)
		return fmt.Errorf("closing destination file %s: %w", newFull, err)
	}

	if err := os.Remove(oldFull); err != nil {
		logger.Error("failed to remove old file", "error", err)
		return fmt.Errorf("removing old file %s: %w", oldFull, err)
	}

	logger.Info("file moved successfully via copy + delete")
	return nil
}

// CreateDirectory creates a folder including parents
func (s *LocalStorage) CreateDirectory(ctx context.Context, userID int32, path string) error {
	logger := middleware.GetLogger(ctx).With("path", path)

	full := s.fullPath(userID, path)
	logger.Info("creating directory", "full_path", full)

	if err := os.MkdirAll(full, 0755); err != nil {
		logger.Error("failed to create directory", "error", err)
		return fmt.Errorf("creating directory %s: %w", full, err)
	}

	logger.Info("directory created successfully")
	return nil
}

// DeleteDirectory deletes a folder and all contents
func (s *LocalStorage) DeleteDirectory(ctx context.Context, userID int32, path string) error {
	logger := middleware.GetLogger(ctx).With("path", path)

	full := s.fullPath(userID, path)
	logger.Info("deleting directory", "full_path", full)

	if err := os.RemoveAll(full); err != nil {
		logger.Error("failed to delete directory", "error", err)
		return fmt.Errorf("deleting directory %s: %w", full, err)
	}

	logger.Info("directory deleted successfully")
	return nil
}

// MoveDirectory moves a directory
func (s *LocalStorage) MoveDirectory(ctx context.Context, userID int32, oldPath, newPath string, overwriteFiles bool) error {
	logger := middleware.GetLogger(ctx).With(
		"old_path", oldPath,
		"new_path", newPath,
		"overwrite", overwriteFiles,
	)

	oldFull := s.fullPath(userID, oldPath)
	newFull := s.fullPath(userID, newPath)

	logger.Info("starting directory move", "old_full", oldFull, "new_full", newFull)

	if err := os.MkdirAll(newFull, 0755); err != nil {
		logger.Error("failed to create new directory", "error", err)
		return fmt.Errorf("creating new directory %s: %w", newFull, err)
	}

	err := filepath.Walk(oldFull, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Error("error walking path", "path", path, "error", err)
			return err
		}

		rel, err := filepath.Rel(oldFull, path)
		if err != nil {
			logger.Error("failed to compute relative path", "path", path, "error", err)
			return err
		}
		dest := filepath.Join(newFull, rel)

		if info.IsDir() {
			logger.Info("creating subdirectory", "dest", dest)
			return os.MkdirAll(dest, info.Mode())
		}

		if _, err := os.Stat(dest); err == nil {
			if !overwriteFiles {
				logger.Info("skipping existing file", "dest", dest)
				return nil
			}
			if err := os.Remove(dest); err != nil {
				logger.Error("failed to remove existing file", "dest", dest, "error", err)
				return fmt.Errorf("removing existing file %s: %w", dest, err)
			}
		}

		srcFile, err := os.Open(path)
		if err != nil {
			logger.Error("failed to open source file", "src", path, "error", err)
			return err
		}
		defer func(srcFile *os.File) {
			err := srcFile.Close()
			if err != nil {

			}
		}(srcFile)

		dstFile, err := os.Create(dest)
		if err != nil {
			logger.Error("failed to create destination file", "dest", dest, "error", err)
			return err
		}
		defer func(dstFile *os.File) {
			err := dstFile.Close()
			if err != nil {

			}
		}(dstFile)

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			logger.Error("failed to copy file", "src", path, "dest", dest, "error", err)
			return err
		}

		logger.Info("file moved", "src", path, "dest", dest)
		return nil
	})
	if err != nil {
		logger.Error("failed to move directory contents", "error", err)
		return err
	}

	if err := os.RemoveAll(oldFull); err != nil {
		logger.Error("failed to remove old directory", "old_full", oldFull, "error", err)
		return fmt.Errorf("removing old directory %s: %w", oldFull, err)
	}

	logger.Info("directory move completed successfully")
	return nil
}

func (s *LocalStorage) GetDirectorySize(ctx context.Context, userID int32, path string) (int64, error) {
	logger := middleware.GetLogger(ctx).With("user_id", userID, "path", path)
	logger.Info("starting directory size calculation")

	basePath := filepath.Join(s.BasePath, fmt.Sprint(userID), path)
	var size int64
	var fileCount int64

	err := filepath.Walk(basePath, func(fullPath string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Warn("error accessing file during walk", "file", fullPath, "error", err)
			return err // returning error stops the walk — this is intentional
		}
		if !info.IsDir() {
			size += info.Size()
			fileCount++
		}
		return nil
	})

	if err != nil {
		logger.Error("failed to calculate directory size", "error", err)
		return 0, fmt.Errorf("failed to calculate directory size for %s: %w", path, err)
	}

	logger.Info("directory size calculated successfully",
		"total_size_bytes", size,
		"file_count", fileCount,
	)

	return size, nil
}

// ZipMultipleFolders recursively zips the specified user folders, preserving their structure, and writes the archive to the provided writer
func (s *LocalStorage) ZipMultipleFolders(ctx context.Context, userID int32, folderPaths []string, w io.Writer) error {
	logger := middleware.GetLogger(ctx).With(
		"folders", folderPaths,
	)
	logger.Info("starting zip of multiple folders")

	zipWriter := zip.NewWriter(w)
	defer func() {
		if err := zipWriter.Close(); err != nil {
			logger.Error("failed to close zip writer", "error", err)
		}
	}()

	for _, folderPath := range folderPaths {
		rootPath := filepath.Join(s.BasePath, strconv.Itoa(int(userID)), folderPath)
		info, err := os.Stat(rootPath)
		if os.IsNotExist(err) {
			logger.Warn("folder does not exist, skipping", "folder", folderPath)
			return fmt.Errorf("folder does not exist: %s", folderPath)
		}
		if !info.IsDir() {
			logger.Warn("path is not a folder, skipping", "folder", folderPath)
			return fmt.Errorf("path is not a folder: %s", folderPath)
		}

		logger.Info("processing folder", "folder", folderPath)

		err = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				logger.Error("error walking path", "path", path, "error", err)
				return err
			}

			if info.IsDir() {
				return nil
			}

			relPath, err := filepath.Rel(filepath.Join(s.BasePath, strconv.Itoa(int(userID))), path)
			if err != nil {
				logger.Error("failed to compute relative path", "path", path, "error", err)
				return err
			}

			zipPath := filepath.ToSlash(relPath)
			logger.Debug("adding file to zip", "file", zipPath)

			file, err := os.Open(path)
			if err != nil {
				logger.Error("failed to open file", "file", path, "error", err)
				return err
			}
			defer func(file *os.File) {
				err := file.Close()
				if err != nil {

				}
			}(file)

			entry, err := zipWriter.Create(zipPath)
			if err != nil {
				logger.Error("failed to create zip entry", "file", zipPath, "error", err)
				return err
			}

			if _, err := io.Copy(entry, file); err != nil {
				logger.Error("failed to copy file to zip", "file", zipPath, "error", err)
				return err
			}

			return nil
		})
		if err != nil {
			logger.Error("error zipping folder", "folder", folderPath, "error", err)
			return fmt.Errorf("zipping folder %s: %w", folderPath, err)
		}
	}

	logger.Info("completed zipping all folders")
	return nil
}
