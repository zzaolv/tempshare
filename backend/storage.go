// backend/storage.go
package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http" // 引入 net/http 以便使用 http.StatusUnauthorized
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/studio-b12/gowebdav"
	"gorm.io/gorm"
)

// FileStorage interface (no change)
type FileStorage interface {
	Save(key string, reader io.Reader) (int64, error)
	Retrieve(key string) (io.ReadCloser, error)
	Delete(key string) error
	Exists(key string) bool
}

// --- Local Storage Implementation --- (no change)
type LocalStorage struct{ basePath string }

func NewLocalStorage(config StorageConfig) (*LocalStorage, error) {
	if err := os.MkdirAll(config.LocalPath, os.ModePerm); err != nil {
		return nil, fmt.Errorf("无法创建本地存储目录 %s: %w", config.LocalPath, err)
	}
	slog.Info("使用本地文件存储", "path", config.LocalPath)
	return &LocalStorage{basePath: config.LocalPath}, nil
}
func (l *LocalStorage) fullPath(key string) string { return filepath.Join(l.basePath, key) }
func (l *LocalStorage) Save(key string, reader io.Reader) (int64, error) {
	filePath := l.fullPath(key)
	file, err := os.Create(filePath)
	if err != nil {
		return 0, fmt.Errorf("本地存储创建文件失败: %w", err)
	}
	defer file.Close()
	return io.Copy(file, reader)
}
func (l *LocalStorage) Retrieve(key string) (io.ReadCloser, error) {
	file, err := os.Open(l.fullPath(key))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, fmt.Errorf("本地存储打开文件失败: %w", err)
	}
	return file, nil
}
func (l *LocalStorage) Delete(key string) error {
	err := os.Remove(l.fullPath(key))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("本地存储删除文件失败: %w", err)
	}
	return nil
}
func (l *LocalStorage) Exists(key string) bool {
	_, err := os.Stat(l.fullPath(key))
	return !os.IsNotExist(err)
}

// --- S3 Storage Implementation --- (no change)
type S3Storage struct {
	client *s3.Client
	bucket string
}

func NewS3Storage(config StorageConfig) (*S3Storage, error) {
	cfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithRegion(config.S3.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(config.S3.AccessKeyID, config.S3.SecretAccessKey, "")),
		awsconfig.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				if config.S3.Endpoint != "" {
					return aws.Endpoint{URL: config.S3.Endpoint}, nil
				}
				return aws.Endpoint{}, &aws.EndpointNotFoundError{}
			},
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("无法加载 S3 配置: %w", err)
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) { o.UsePathStyle = config.S3.UsePathStyle })
	slog.Info("使用 S3 对象存储", "endpoint", config.S3.Endpoint, "bucket", config.S3.Bucket)
	return &S3Storage{client: client, bucket: config.S3.Bucket}, nil
}
func (s *S3Storage) Save(key string, reader io.Reader) (int64, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return 0, fmt.Errorf("S3 存储读取数据流失败: %w", err)
	}
	contentLength := int64(len(data))
	_, err = s.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key), Body: bytes.NewReader(data), ContentLength: &contentLength,
	})
	if err != nil {
		return 0, fmt.Errorf("S3 存储上传对象失败: %w", err)
	}
	return contentLength, nil
}
func (s *S3Storage) Retrieve(key string) (io.ReadCloser, error) {
	output, err := s.client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key),
	})
	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, fmt.Errorf("S3 存储获取对象失败: %w", err)
	}
	return output.Body, nil
}
func (s *S3Storage) Delete(key string) error {
	_, err := s.client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("S3 存储删除对象失败: %w", err)
	}
	return nil
}
func (s *S3Storage) Exists(key string) bool {
	_, err := s.client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key),
	})
	return err == nil
}

// --- WebDAV Storage Implementation (Final Correction) ---

type WebDAVStorage struct {
	client *gowebdav.Client
}

func NewWebDAVStorage(config StorageConfig) (*WebDAVStorage, error) {
	client := gowebdav.NewClient(config.WebDAV.URL, config.WebDAV.Username, config.WebDAV.Password)

	// ✨✨✨ 最终修复点: 不再检查具体的 HTTPError，而是检查错误的字符串内容或使用 os.IsNotExist ✨✨✨
	if _, err := client.Stat("/"); err != nil {
		// gowebdav 在认证失败时会返回包含 "401" 的错误信息
		if strings.Contains(err.Error(), fmt.Sprintf("%d", http.StatusUnauthorized)) {
			return nil, fmt.Errorf("WebDAV 认证失败 (401 Unauthorized): 请检查用户名和密码: %w", err)
		}
		// 对于其他连接错误，直接返回
		return nil, fmt.Errorf("WebDAV 服务器连接或根目录检查失败 at %s: %w", config.WebDAV.URL, err)
	}

	slog.Info("使用 WebDAV 存储", "url", config.WebDAV.URL)
	return &WebDAVStorage{client: client}, nil
}

func (w *WebDAVStorage) Save(key string, reader io.Reader) (int64, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return 0, fmt.Errorf("WebDAV 存储读取数据流失败: %w", err)
	}
	contentLength := int64(len(data))

	err = w.client.Write(key, data, 0644)
	if err != nil {
		return 0, fmt.Errorf("WebDAV 存储写入失败: %w", err)
	}
	return contentLength, nil
}

func (w *WebDAVStorage) Retrieve(key string) (io.ReadCloser, error) {
	stream, err := w.client.ReadStream(key)
	if err != nil {
		// ✨✨✨ 最终修复点: 使用 os.IsNotExist 进行判断 ✨✨✨
		if os.IsNotExist(err) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, fmt.Errorf("WebDAV 存储读取流失败: %w", err)
	}
	return stream, nil
}

func (w *WebDAVStorage) Delete(key string) error {
	err := w.client.Remove(key)
	if err != nil {
		// ✨✨✨ 最终修复点: 使用 os.IsNotExist 进行判断 ✨✨✨
		if os.IsNotExist(err) {
			return nil // 文件本就不存在，任务完成
		}
		return fmt.Errorf("WebDAV 存储删除文件失败: %w", err)
	}
	return nil
}

func (w *WebDAVStorage) Exists(key string) bool {
	_, err := w.client.Stat(key)
	return err == nil
}

// --- Factory Function --- (no change)

func NewFileStorage(config StorageConfig) (FileStorage, error) {
	switch strings.ToLower(config.Type) {
	case "local":
		return NewLocalStorage(config)
	case "s3":
		return NewS3Storage(config)
	case "webdav":
		return NewWebDAVStorage(config)
	default:
		return nil, fmt.Errorf("不支持的存储类型: %s", config.Type)
	}
}
