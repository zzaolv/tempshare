// backend/storage/provider.go
package storage

import (
	"context"
	"io"
)

// StorageProvider 定义了所有存储后端必须实现的接口
type StorageProvider interface {
	// Save 将读取器中的数据保存到指定的 key
	Save(ctx context.Context, key string, reader io.Reader) (int64, error)

	// Open 返回一个可读取 key 对应文件内容的 io.ReadCloser
	Open(ctx context.Context, key string) (io.ReadCloser, error)

	// Delete 删除 key 对应的文件
	Delete(ctx context.Context, key string) error

	// GetFullPath 返回文件的物理路径（仅对本地存储有意义，其他类型可返回 key）
	GetFullPath(key string) string
}
