// backend/config.go
package main

import (
	"log/slog"
	"strings"
	"time"

	"github.com/spf13/viper"
)

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

type Config struct {
	ServerPort      string          `mapstructure:"ServerPort"`
	MaxUploadSizeMB int64           `mapstructure:"MaxUploadSizeMB"`
	RateLimit       RateLimitConfig `mapstructure:"RateLimit"`
	Database        DBConfig        `mapstructure:"Database"`
	Storage         StorageConfig   `mapstructure:"Storage"`
	ClamdSocket     string          `mapstructure:"ClamdSocket"`
	Initialized     bool            `mapstructure:"Initialized"`
}

var AppConfig *Config

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
	viper.SetDefault("Database.DSN", "data/tempshare.db") // 确保路径与 docker-compose volume 对应
	viper.SetDefault("Storage.Type", "local")
	viper.SetDefault("Storage.LocalPath", "data/files") // 确保路径与 docker-compose volume 对应
	viper.SetDefault("Storage.S3.UsePathStyle", true)
	viper.SetDefault("ClamdSocket", "")
	viper.SetDefault("Initialized", false)

	// --- 核心修复点: 只在文件存在时读取，如果不存在则忽略错误 ---
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// 文件未找到，这是 Docker 环境下的预期行为，所以我们忽略这个错误
			slog.Info("配置文件 config.json 未找到，将完全依赖环境变量和默认值。")
		} else {
			// 配置文件找到了，但是解析出错了，这是一个需要报告的严重错误
			return err
		}
	}

	viper.SetEnvPrefix("TEMPSHARE")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	AppConfig = &Config{}
	if err := viper.Unmarshal(&AppConfig); err != nil {
		return err
	}

	// 敏感信息或需要覆盖的信息，再次从环境变量中强行读取
	if dsn := viper.GetString("DATABASE_DSN"); dsn != "" {
		AppConfig.Database.DSN = dsn
	}
	if localPath := viper.GetString("STORAGE_LOCALPATH"); localPath != "" {
		AppConfig.Storage.LocalPath = localPath
	}
	if port := viper.GetString("SERVERPORT"); port != "" {
		AppConfig.ServerPort = port
	}
	AppConfig.Storage.S3.AccessKeyID = viper.GetString("STORAGE_S3_ACCESSKEYID")
	AppConfig.Storage.S3.SecretAccessKey = viper.GetString("STORAGE_S3_SECRETACCESSKEY")
	AppConfig.Storage.WebDAV.Username = viper.GetString("STORAGE_WEBDAV_USERNAME")
	AppConfig.Storage.WebDAV.Password = viper.GetString("STORAGE_WEBDAV_PASSWORD")
	if clamdSocket := viper.GetString("CLAMDSOCKET"); clamdSocket != "" {
		AppConfig.ClamdSocket = clamdSocket
	}
	// `Initialized` 标志主要由环境变量控制
	AppConfig.Initialized = viper.GetBool("INITIALIZED")

	slog.Info("配置加载完成",
		slog.String("serverPort", AppConfig.ServerPort),
		slog.String("dbType", AppConfig.Database.Type),
		slog.String("storageType", AppConfig.Storage.Type),
		slog.Bool("initialized", AppConfig.Initialized),
	)

	return nil
}

func (c *Config) GetRateLimitDuration() time.Duration {
	return time.Duration(c.RateLimit.DurationMinutes) * time.Minute
}
