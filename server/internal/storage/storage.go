package storage

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

type Storage interface {
	SaveFile(userID int32, path string, content io.Reader) error
	ReadFile(userID int32, path string) (io.ReadCloser, error)
	DeleteFile(userID int32, path string) error
	CreateDirectory(userID int32, path string) error
	DeleteDirectory(userID int32, path string) error
	MoveFile(userID int32, oldPath, newPath string) error
	MoveDirectory(userID int32, oldPath, newPath string) error
	ZipFolder(userID int32, folderPath string, w io.Writer) error
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
func (s *LocalStorage) SaveFile(userID int32, path string, content io.Reader) error {
	full := s.fullPath(userID, path)

	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		return fmt.Errorf("creating directories for %s: %w", full, err)
	}

	// write to a temp file first for atomic write
	temp := full + ".tmp"
	f, err := os.Create(temp)
	if err != nil {
		return fmt.Errorf("creating temp file %s: %w", temp, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, content); err != nil {
		return fmt.Errorf("writing to temp file %s: %w", temp, err)
	}

	if err := os.Rename(temp, full); err != nil {
		return fmt.Errorf("renaming temp file %s to %s: %w", temp, full, err)
	}

	return nil
}

// ReadFile opens a file for reading
func (s *LocalStorage) ReadFile(userID int32, path string) (io.ReadCloser, error) {
	full := s.fullPath(userID, path)
	f, err := os.Open(full)
	if err != nil {
		return nil, fmt.Errorf("opening file %s: %w", full, err)
	}
	return f, nil
}

// DeleteFile removes a file
func (s *LocalStorage) DeleteFile(userID int32, path string) error {
	full := s.fullPath(userID, path)
	if err := os.Remove(full); err != nil {
		return fmt.Errorf("deleting file %s: %w", full, err)
	}
	return nil
}

// CreateDirectory creates a folder including parents
func (s *LocalStorage) CreateDirectory(userID int32, path string) error {
	full := s.fullPath(userID, path)
	if err := os.MkdirAll(full, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", full, err)
	}
	return nil
}

// DeleteDirectory deletes a folder and all contents
func (s *LocalStorage) DeleteDirectory(userID int32, path string) error {
	full := s.fullPath(userID, path)
	if err := os.RemoveAll(full); err != nil {
		return fmt.Errorf("deleting directory %s: %w", full, err)
	}
	return nil
}

func (s *LocalStorage) ZipFolder(userID int32, folderPath string, w io.Writer) error {
	rootPath := filepath.Join(s.BasePath, strconv.Itoa(int(userID)), folderPath)

	info, err := os.Stat(rootPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("folder does not exist: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a folder")
	}

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	err = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories themselves (zip only files)
		if info.IsDir() {
			return nil
		}

		// Create relative path inside the zip
		relPath, err := filepath.Rel(filepath.Dir(rootPath), path)
		if err != nil {
			return err
		}

		zipPath := filepath.ToSlash(relPath)

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		entry, err := zipWriter.Create(zipPath)
		if err != nil {
			return err
		}

		if _, err := io.Copy(entry, file); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("zipping folder: %w", err)
	}

	return nil
}

// MoveFile moves a file; supports cross-filesystem moves
func (s *LocalStorage) MoveFile(userID int32, oldPath, newPath string) error {
	oldFull := s.fullPath(userID, oldPath)
	newFull := s.fullPath(userID, newPath)

	if err := os.MkdirAll(filepath.Dir(newFull), 0755); err != nil {
		return fmt.Errorf("creating directories for %s: %w", newFull, err)
	}

	// attempt rename
	if err := os.Rename(oldFull, newFull); err == nil {
		return nil
	}

	// fallback: copy + delete
	src, err := os.Open(oldFull)
	if err != nil {
		return fmt.Errorf("opening source file %s: %w", oldFull, err)
	}
	defer src.Close()

	dst, err := os.Create(newFull)
	if err != nil {
		return fmt.Errorf("creating destination file %s: %w", newFull, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copying from %s to %s: %w", oldFull, newFull, err)
	}

	if err := os.Remove(oldFull); err != nil {
		return fmt.Errorf("removing old file %s: %w", oldFull, err)
	}

	return nil
}

func (s *LocalStorage) MoveDirectory(userID int32, oldPath, newPath string) error {
	oldFull := s.fullPath(userID, oldPath)
	newFull := s.fullPath(userID, newPath)

	if err := os.MkdirAll(newFull, 0755); err != nil {
		return fmt.Errorf("creating new directory %s: %w", newFull, err)
	}

	err := filepath.Walk(oldFull, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(oldFull, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(newFull, rel)
		if info.IsDir() {
			return os.MkdirAll(dest, info.Mode())
		}
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()
		dstFile, err := os.Create(dest)
		if err != nil {
			return err
		}
		defer dstFile.Close()
		if _, err := io.Copy(dstFile, srcFile); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return os.RemoveAll(oldFull)
}
