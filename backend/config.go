// backend/config.go
package main

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// --- 子配置结构 ---

type DatabaseConfig struct {
	Type string `mapstructure:"Type"` // "sqlite", "mysql", "postgres"
	DSN  string `mapstructure:"DSN"`  // Data Source Name
}

type LocalStorageConfig struct {
	Path string `mapstructure:"Path"`
}

type S3StorageConfig struct {
	Endpoint        string `mapstructure:"Endpoint"`
	Region          string `mapstructure:"Region"`
	Bucket          string `mapstructure:"Bucket"`
	AccessKeyID     string `mapstructure:"AccessKeyID"`
	SecretAccessKey string `mapstructure:"SecretAccessKey"`
	UsePathStyle    bool   `mapstructure:"UsePathStyle"` // For MinIO compatibility
}

type WebDAVStorageConfig struct {
	URL      string `mapstructure:"URL"`
	Username string `mapstructure:"Username"`
	Password string `mapstructure:"Password"`
}

type StorageConfig struct {
	Type   string              `mapstructure:"Type"` // "local", "s3", "webdav"
	Local  LocalStorageConfig  `mapstructure:"Local"`
	S3     S3StorageConfig     `mapstructure:"S3"`
	WebDAV WebDAVStorageConfig `mapstructure:"WebDAV"`
}

type RateLimitConfig struct {
	Enabled         bool `mapstructure:"Enabled"`
	Requests        int  `mapstructure:"Requests"`
	DurationMinutes int  `mapstructure:"DurationMinutes"`
}

// --- 主配置结构 ---

type Config struct {
	ServerPort      string          `mapstructure:"ServerPort"`
	ClamdSocket     string          `mapstructure:"ClamdSocket"`
	MaxUploadSizeMB int64           `mapstructure:"MaxUploadSizeMB"`
	RateLimit       RateLimitConfig `mapstructure:"RateLimit"`
	Database        DatabaseConfig  `mapstructure:"Database"`
	Storage         StorageConfig   `mapstructure:"Storage"`
}

var AppConfig *Config

func LoadConfig(path string) error {
	viper.SetConfigFile(path)
	viper.SetConfigType("json")

	// 优先使用环境变量
	viper.SetEnvPrefix("TS") // TEMPSHARE_SERVERPORT
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// --- 设置默认值 ---
	// 服务器
	viper.SetDefault("ServerPort", "8080")
	viper.SetDefault("MaxUploadSizeMB", 1024)
	viper.SetDefault("ClamdSocket", "") // 默认禁用

	// 速率限制
	viper.SetDefault("RateLimit.Enabled", true)
	viper.SetDefault("RateLimit.Requests", 30)
	viper.SetDefault("RateLimit.DurationMinutes", 10)

	// 数据库 (默认SQLite)
	viper.SetDefault("Database.Type", "sqlite")
	viper.SetDefault("Database.DSN", "data/tempshare.db")

	// 存储 (默认本地)
	viper.SetDefault("Storage.Type", "local")
	viper.SetDefault("Storage.Local.Path", "data/tempshare-files")

	// --- 读取配置文件 (如果存在) ---
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			slog.Warn("配置文件未找到，将主要依赖环境变量和默认值", "path", path)
		} else {
			return err // 配置文件找到但解析错误
		}
	}

	// --- 解析到结构体 ---
	AppConfig = &Config{}
	err := viper.Unmarshal(AppConfig)
	if err != nil {
		return err
	}

	// 确保存储目录存在 (仅对本地存储有意义)
	if AppConfig.Storage.Type == "local" {
		if err := os.MkdirAll(AppConfig.Storage.Local.Path, os.ModePerm); err != nil {
			slog.Error("无法创建本地文件存储目录", "path", AppConfig.Storage.Local.Path, "error", err)
			return err
		}
	}

	slog.Info("配置加载成功",
		"serverPort", AppConfig.ServerPort,
		"dbType", AppConfig.Database.Type,
		"storageType", AppConfig.Storage.Type,
	)

	return nil
}

// IsInitialized 检查关键配置是否已设置，用于判断是否需要初始化
func (c *Config) IsInitialized() bool {
	if c.Database.Type != "sqlite" && c.Database.DSN == "" {
		return false
	}
	if c.Storage.Type == "s3" && (c.Storage.S3.Bucket == "" || c.Storage.S3.AccessKeyID == "" || c.Storage.S3.SecretAccessKey == "") {
		return false
	}
	if c.Storage.Type == "webdav" && c.Storage.WebDAV.URL == "" {
		return false
	}
	return true
}

func (c *Config) GetRateLimitDuration() time.Duration {
	return time.Duration(c.RateLimit.DurationMinutes) * time.Minute
}
