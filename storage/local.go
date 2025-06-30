// backend/storage/local.go
package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

type LocalStorage struct {
	basePath string
}

func NewLocalStorage(path string) (*LocalStorage, error) {
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return nil, err
	}
	return &LocalStorage{basePath: path}, nil
}

func (l *LocalStorage) getFullPath(key string) string {
	return filepath.Join(l.basePath, key)
}

func (l *LocalStorage) Save(ctx context.Context, key string, reader io.Reader) (int64, error) {
	path := l.getFullPath(key)
	file, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	return io.Copy(file, reader)
}

func (l *LocalStorage) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	path := l.getFullPath(key)
	return os.Open(path)
}

func (l *LocalStorage) Delete(ctx context.Context, key string) error {
	path := l.getFullPath(key)
	err := os.Remove(path)
	// 如果文件已经不存在，我们不认为这是一个错误
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (l *LocalStorage) GetFullPath(key string) string {
	return l.getFullPath(key)
}
