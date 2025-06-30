// backend/storage.go
// NEW FILE
package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// StorageBackend 定义了所有存储后端必须实现的接口
type StorageBackend interface {
	// Save 将数据流保存到存储后端，并返回写入的字节数
	Save(id string, reader io.Reader) (int64, error)
	// Open 返回一个可读的数据流
	Open(id string) (io.ReadCloser, error)
	// Delete 删除一个对象
	Delete(id string) error
	// ServeFile 将文件内容写入 Gin 的响应中
	ServeFile(c *gin.Context, file File)
	// IsInitialized 检查后端是否已正确配置
	IsInitialized() bool
	// GetType 返回存储后端的类型
	GetType() string
}

// --- LocalStorage 实现 ---

type LocalStorage struct {
	BasePath string
}

func NewLocalStorage(config *StorageConfig) (*LocalStorage, error) {
	if config.Local.Path == "" {
		return nil, fmt.Errorf("本地存储路径 (storage.local.path) 未配置")
	}
	if err := os.MkdirAll(config.Local.Path, os.ModePerm); err != nil {
		return nil, fmt.Errorf("无法创建本地存储目录 %s: %w", config.Local.Path, err)
	}
	return &LocalStorage{
		BasePath: config.Local.Path,
	}, nil
}

func (l *LocalStorage) getFullPath(id string) string {
	return filepath.Join(l.BasePath, id)
}

func (l *LocalStorage) Save(id string, reader io.Reader) (int64, error) {
	filePath := l.getFullPath(id)
	outFile, err := os.Create(filePath)
	if err != nil {
		return 0, fmt.Errorf("无法创建文件 %s: %w", filePath, err)
	}
	defer outFile.Close()

	writtenBytes, err := io.Copy(outFile, reader)
	if err != nil {
		// 保存失败时尝试删除不完整的文件
		os.Remove(filePath)
		return 0, fmt.Errorf("写入文件时出错 %s: %w", filePath, err)
	}
	return writtenBytes, nil
}

func (l *LocalStorage) Open(id string) (io.ReadCloser, error) {
	filePath := l.getFullPath(id)
	return os.Open(filePath)
}

func (l *LocalStorage) Delete(id string) error {
	filePath := l.getFullPath(id)
	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除本地文件失败 %s: %w", filePath, err)
	}
	return nil
}

func (l *LocalStorage) ServeFile(c *gin.Context, file File) {
	filePath := l.getFullPath(file.ID)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename*=UTF-8''%s`, url.PathEscape(file.Filename)))
	c.File(filePath)
}

func (l *LocalStorage) IsInitialized() bool {
	return l.BasePath != ""
}

func (l *LocalStorage) GetType() string {
	return "local"
}

// StorageFactory 根据配置创建对应的存储后端实例
// 目前只实现了 local, 未来可在这里扩展 webdav, s3 等
func StorageFactory(config *StorageConfig) (StorageBackend, error) {
	slog.Info("初始化存储后端", "type", config.Type)
	switch config.Type {
	case "local":
		return NewLocalStorage(config)
	case "webdav":
		// TODO: 在此实现 WebDAV 存储的初始化
		slog.Warn("WebDAV 存储尚未实现，请在 storage.go 中补充")
		return nil, fmt.Errorf("webdav 存储类型暂不支持")
	case "s3":
		// TODO: 在此实现 S3 对象存储的初始化
		slog.Warn("S3 存储尚未实现，请在 storage.go 中补充")
		return nil, fmt.Errorf("s3 存储类型暂不支持")
	default:
		return nil, fmt.Errorf("不支持的存储类型: %s", config.Type)
	}
}

// DummyStorage 用于未初始化状态
type DummyStorage struct{}

func (d *DummyStorage) Save(id string, reader io.Reader) (int64, error) {
	return 0, fmt.Errorf("系统未初始化")
}
func (d *DummyStorage) Open(id string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("系统未初始化")
}
func (d *DummyStorage) Delete(id string) error              { return fmt.Errorf("系统未初始化") }
func (d *DummyStorage) ServeFile(c *gin.Context, file File) {}
func (d *DummyStorage) IsInitialized() bool                 { return false }
func (d *DummyStorage) GetType() string                     { return "none" }
