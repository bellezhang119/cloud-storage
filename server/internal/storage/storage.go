package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Storage interface {
	SaveFile(path string, content io.Reader) error
	ReadFile(path string) (io.ReadCloser, error)
	DeleteFile(path string) error
	CreateDirectory(path string) error
	DeleteDirectory(path string) error
	MoveFile(oldPath, newPath string) error
}

type LocalStorage struct {
	BasePath string
}

func NewLocalStorage(basePath string) *LocalStorage {
	return &LocalStorage{BasePath: basePath}
}

func (s *LocalStorage) fullPath(path string) string {
	return filepath.Join(s.BasePath, path)
}

func (s *LocalStorage) SaveFile(path string, content io.Reader) error {
	full := s.fullPath(path)

	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		return fmt.Errorf("creating directories: %w", err)
	}

	f, err := os.Create(full)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer f.Close()

	_, err = io.Copy(f, content)
	if err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}

func (s *LocalStorage) ReadFile(path string) (io.ReadCloser, error) {
	full := s.fullPath(path)
	f, err := os.Open(full)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	return f, nil
}

func (s *LocalStorage) DeleteFile(path string) error {
	full := s.fullPath(path)
	if err := os.Remove(full); err != nil {
		return fmt.Errorf("deleting file: %w", err)
	}
	return nil
}

func (s *LocalStorage) CreateDirectory(path string) error {
	full := s.fullPath(path)
	if err := os.MkdirAll(full, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}
	return nil
}

func (s *LocalStorage) DeleteDirectory(path string) error {
	full := s.fullPath(path)
	if err := os.RemoveAll(full); err != nil {
		return fmt.Errorf("deleting directory: %w", err)
	}
	return nil
}

func (s *LocalStorage) MoveFile(oldPath, newPath string) error {
	oldFull := s.fullPath(oldPath)
	newFull := s.fullPath(newPath)

	if err := os.MkdirAll(filepath.Dir(newFull), 0755); err != nil {
		return fmt.Errorf("creating directories: %w", err)
	}

	if err := os.Rename(oldFull, newFull); err != nil {
		return fmt.Errorf("moving file: %w", err)
	}
	return nil
}
