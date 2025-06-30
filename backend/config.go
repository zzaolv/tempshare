// backend/config.go
package main

import (
	"log/slog"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// ... 其他结构体定义不变 ...
type RateLimitConfig struct {
	Enabled         bool `mapstructure:"Enabled"`
	Requests        int  `mapstructure:"Requests"`
	DurationMinutes int  `mapstructure:"DurationMinutes"`
}

type DBConfig struct {
	Type string `mapstructure:"Type"`
	DSN  string `mapstructure:"DSN"`
}

type StorageConfig struct {
	Type      string       `mapstructure:"Type"`
	LocalPath string       `mapstructure:"LocalPath"`
	S3        S3Config     `mapstructure:"S3"`
	WebDAV    WebDAVConfig `mapstructure:"WebDAV"`
}

type S3Config struct {
	Endpoint        string `mapstructure:"Endpoint"`
	Region          string `mapstructure:"Region"`
	Bucket          string `mapstructure:"Bucket"`
	AccessKeyID     string `mapstructure:"AccessKeyID"`
	SecretAccessKey string `mapstructure:"SecretAccessKey"`
	UsePathStyle    bool   `mapstructure:"UsePathStyle"`
}

type WebDAVConfig struct {
	URL      string `mapstructure:"URL"`
	Username string `mapstructure:"Username"`
	Password string `mapstructure:"Password"`
}

// Config 对应整个应用的配置结构
type Config struct {
	ServerPort      string          `mapstructure:"ServerPort"`
	MaxUploadSizeMB int64           `mapstructure:"MaxUploadSizeMB"`
	RateLimit       RateLimitConfig `mapstructure:"RateLimit"`
	Database        DBConfig        `mapstructure:"Database"`
	Storage         StorageConfig   `mapstructure:"Storage"`
	// ✨✨✨ 修复点: 重新添加 ClamdSocket 字段 ✨✨✨
	ClamdSocket string `mapstructure:"ClamdSocket"`
	Initialized bool   `mapstructure:"Initialized"`
}

var AppConfig *Config

// LoadConfig 函数 (完全替换以包含新的默认值)
func LoadConfig(path string) error {
	viper.SetConfigFile(path)
	viper.SetConfigType("json")

	// --- 设置默认值 ---
	viper.SetDefault("ServerPort", "8080")
	viper.SetDefault("MaxUploadSizeMB", 1024)
	viper.SetDefault("RateLimit.Enabled", true)
	viper.SetDefault("RateLimit.Requests", 30)
	viper.SetDefault("RateLimit.DurationMinutes", 10)
	viper.SetDefault("Database.Type", "sqlite")
	viper.SetDefault("Database.DSN", "tempshare.db")
	viper.SetDefault("Storage.Type", "local")
	viper.SetDefault("Storage.LocalPath", "tempshare-files")
	viper.SetDefault("Storage.S3.UsePathStyle", true)
	// ✨✨✨ 修复点: 添加 ClamdSocket 的默认值 ✨✨✨
	viper.SetDefault("ClamdSocket", "") // 默认不启用
	viper.SetDefault("Initialized", false)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			slog.Warn("配置文件未找到，将使用默认值和环境变量。", "path", path)
		} else {
			return err
		}
	}

	viper.SetEnvPrefix("TEMPSHARE")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	AppConfig = &Config{}
	err := viper.Unmarshal(AppConfig)
	if err != nil {
		return err
	}

	if dsn := viper.GetString("DATABASE_DSN"); dsn != "" {
		AppConfig.Database.DSN = dsn
	}

	// 从环境变量中获取敏感信息
	AppConfig.Storage.S3.AccessKeyID = viper.GetString("STORAGE_S3_ACCESSKEYID")
	AppConfig.Storage.S3.SecretAccessKey = viper.GetString("STORAGE_S3_SECRETACCESSKEY")
	AppConfig.Storage.WebDAV.Username = viper.GetString("STORAGE_WEBDAV_USERNAME")
	AppConfig.Storage.WebDAV.Password = viper.GetString("STORAGE_WEBDAV_PASSWORD")
	// ✨✨✨ 修复点: 从环境变量读取 ClamdSocket ✨✨✨
	if clamdSocket := viper.GetString("CLAMDSOCKET"); clamdSocket != "" {
		AppConfig.ClamdSocket = clamdSocket
	}

	slog.Info("配置加载成功",
		slog.String("serverPort", AppConfig.ServerPort),
		slog.String("dbType", AppConfig.Database.Type),
		slog.String("storageType", AppConfig.Storage.Type),
	)

	return nil
}

func (c *Config) GetRateLimitDuration() time.Duration {
	return time.Duration(c.RateLimit.DurationMinutes) * time.Minute
}
