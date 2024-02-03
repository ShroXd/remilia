package remilia

import "os"

type FileSystemOperations interface {
	MkdirAll(path string, perm os.FileMode) error
	OpenFile(name string, flag int, perm os.FileMode) (*os.File, error)
}

type FileSystem struct{}

func (fs FileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (fs FileSystem) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

type MockFileSystem struct {
	MkdirAllErr  error
	OpenFileErr  error
	OpenFileMock *os.File
}

func (mfs MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return mfs.MkdirAllErr
}

func (mfs MockFileSystem) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	return mfs.OpenFileMock, mfs.OpenFileErr
}
