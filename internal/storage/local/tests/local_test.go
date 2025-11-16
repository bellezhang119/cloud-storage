package tests

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/bellezhang119/cloud-storage/internal/storage/local"
	"github.com/stretchr/testify/assert"
)

func newTestStorage(t *testing.T) (*local.LocalStorage, string) {
	tmpDir := t.TempDir()
	return local.NewLocalStorage(tmpDir), tmpDir
}

func TestSaveFile(t *testing.T) {
	st, base := newTestStorage(t)
	ctx := context.Background()
	userID := int32(1)
	content := bytes.NewBufferString("hello world")
	path := "folder/test.txt"

	err := st.SaveFile(ctx, userID, path, content)
	assert.NoError(t, err)

	fullPath := filepath.Join(base, strconv.Itoa(int(userID)), path)
	data, err := os.ReadFile(fullPath)
	assert.NoError(t, err)
	assert.Equal(t, "hello world", string(data))
}

func TestReadFile(t *testing.T) {
	st, base := newTestStorage(t)
	ctx := context.Background()
	userID := int32(1)
	path := "read/test.txt"

	err := os.MkdirAll(filepath.Dir(filepath.Join(base, strconv.Itoa(int(userID)), path)), 0755)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(base, strconv.Itoa(int(userID)), path), []byte("read content"), 0644)
	assert.NoError(t, err)

	rc, err := st.ReadFile(ctx, userID, path)
	assert.NoError(t, err)
	data, err := io.ReadAll(rc)
	assert.NoError(t, err)
	rc.Close()
	assert.Equal(t, "read content", string(data))
}

func TestDeleteFile(t *testing.T) {
	st, base := newTestStorage(t)
	ctx := context.Background()
	userID := int32(1)
	path := "delete/test.txt"
	fullPath := filepath.Join(base, strconv.Itoa(int(userID)), path)

	os.MkdirAll(filepath.Dir(fullPath), 0755)
	os.WriteFile(fullPath, []byte("delete me"), 0644)

	err := st.DeleteFile(ctx, userID, path)
	assert.NoError(t, err)
	_, err = os.Stat(fullPath)
	assert.True(t, os.IsNotExist(err))
}

func TestMoveFile(t *testing.T) {
	st, base := newTestStorage(t)
	ctx := context.Background()
	userID := int32(1)
	oldPath := "move/old.txt"
	newPath := "move/new.txt"

	oldFull := filepath.Join(base, strconv.Itoa(int(userID)), oldPath)
	os.MkdirAll(filepath.Dir(oldFull), 0755)
	os.WriteFile(oldFull, []byte("move content"), 0644)

	err := st.MoveFile(ctx, userID, oldPath, newPath)
	assert.NoError(t, err)

	newFull := filepath.Join(base, strconv.Itoa(int(userID)), newPath)
	data, err := os.ReadFile(newFull)
	assert.NoError(t, err)
	assert.Equal(t, "move content", string(data))
}

func TestCreateDirectory(t *testing.T) {
	st, base := newTestStorage(t)
	ctx := context.Background()
	userID := int32(1)
	path := "newfolder/subfolder"

	err := st.CreateDirectory(ctx, userID, path)
	assert.NoError(t, err)

	fullPath := filepath.Join(base, strconv.Itoa(int(userID)), path)
	info, err := os.Stat(fullPath)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestDeleteDirectory(t *testing.T) {
	st, base := newTestStorage(t)
	ctx := context.Background()
	userID := int32(1)
	path := "delfolder/sub"
	fullPath := filepath.Join(base, strconv.Itoa(int(userID)), path)
	os.MkdirAll(fullPath, 0755)

	err := st.DeleteDirectory(ctx, userID, "delfolder")
	assert.NoError(t, err)
	_, err = os.Stat(fullPath)
	assert.True(t, os.IsNotExist(err))
}

func TestMoveDirectory(t *testing.T) {
	st, base := newTestStorage(t)
	ctx := context.Background()
	userID := int32(1)
	oldDir := "olddir"
	newDir := "newdir"

	os.MkdirAll(filepath.Join(base, strconv.Itoa(int(userID)), oldDir, "sub"), 0755)
	os.WriteFile(filepath.Join(base, strconv.Itoa(int(userID)), oldDir, "file.txt"), []byte("content"), 0644)

	err := st.MoveDirectory(ctx, userID, oldDir, newDir, true)
	assert.NoError(t, err)

	newFull := filepath.Join(base, strconv.Itoa(int(userID)), newDir, "file.txt")
	data, err := os.ReadFile(newFull)
	assert.NoError(t, err)
	assert.Equal(t, "content", string(data))
	_, err = os.Stat(filepath.Join(base, strconv.Itoa(int(userID)), oldDir))
	assert.True(t, os.IsNotExist(err))
}

func TestGetDirectorySize(t *testing.T) {
	st, base := newTestStorage(t)
	ctx := context.Background()
	userID := int32(1)
	dir := "sizefolder"
	os.MkdirAll(filepath.Join(base, strconv.Itoa(int(userID)), dir), 0755)
	os.WriteFile(filepath.Join(base, strconv.Itoa(int(userID)), dir, "f1.txt"), []byte("aaa"), 0644)
	os.WriteFile(filepath.Join(base, strconv.Itoa(int(userID)), dir, "f2.txt"), []byte("bb"), 0644)

	size, err := st.GetDirectorySize(ctx, userID, dir)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), size)
}

func TestZipMultipleFolders(t *testing.T) {
	st, base := newTestStorage(t)
	ctx := context.Background()
	userID := int32(1)
	folder := "zipfolder"
	os.MkdirAll(filepath.Join(base, strconv.Itoa(int(userID)), folder), 0755)
	os.WriteFile(filepath.Join(base, strconv.Itoa(int(userID)), folder, "file1.txt"), []byte("hello"), 0644)

	var buf bytes.Buffer
	err := st.ZipMultipleFolders(ctx, userID, []string{folder}, &buf)
	assert.NoError(t, err)

	r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(r.File))
	assert.Equal(t, folder+"/file1.txt", r.File[0].Name)
}
