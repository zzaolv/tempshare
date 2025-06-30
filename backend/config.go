// backend/config.go
package main

import (
	"log/slog"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// --- 所有结构体定义保持不变 ---
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
	ServerPort         string          `mapstructure:"ServerPort"`
	CORSAllowedOrigins string          `mapstructure:"CORS_ALLOWED_ORIGINS"`
	MaxUploadSizeMB    int64           `mapstructure:"MaxUploadSizeMB"`
	RateLimit          RateLimitConfig `mapstructure:"RateLimit"`
	Database           DBConfig        `mapstructure:"Database"`
	Storage            StorageConfig   `mapstructure:"Storage"`
	ClamdSocket        string          `mapstructure:"ClamdSocket"`
	Initialized        bool            `mapstructure:"Initialized"`
}

var AppConfig *Config

// LoadConfig 现在会处理文件不存在的错误，并成功返回
func LoadConfig(path string) error {
	// 1. 绑定环境变量 (需要提前)
	viper.SetEnvPrefix("TEMPSHARE")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	// 2. 设置默认值
	viper.SetDefault("ServerPort", "8080")
	viper.SetDefault("CORS_ALLOWED_ORIGINS", "http://localhost:5173,https://localhost:5173")
	viper.SetDefault("MaxUploadSizeMB", 1024)
	viper.SetDefault("RateLimit.Enabled", true)
	viper.SetDefault("RateLimit.Requests", 30)
	viper.SetDefault("RateLimit.DurationMinutes", 10)
	viper.SetDefault("Database.Type", "sqlite")
	viper.SetDefault("Database.DSN", "data/tempshare.db")
	viper.SetDefault("Storage.Type", "local")
	viper.SetDefault("Storage.LocalPath", "data/files")
	viper.SetDefault("Storage.S3.UsePathStyle", true)
	viper.SetDefault("ClamdSocket", "")
	viper.SetDefault("Initialized", false)

	// 3. 尝试读取配置文件 (这是可选的)
	viper.SetConfigFile(path)
	viper.SetConfigType("json")
	if err := viper.ReadInConfig(); err != nil {
		// ✨✨✨ 核心修复点 ✨✨✨
		// 我们检查错误类型。如果错误是 "ConfigFileNotFoundError"，
		// 这在 Docker 环境下是正常行为，所以我们只记录一条信息，然后继续。
		// 如果是其他错误 (例如 JSON 格式错误)，那才是真正的严重错误。
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			slog.Info("配置文件 config.json 未找到，将完全依赖环境变量和默认值。")
		} else {
			// 配置文件存在但格式错误等问题
			return err
		}
	}

	// 4. 将所有配置源 (环境变量 > 配置文件 > 默认值) 解析到结构体中
	AppConfig = &Config{}
	if err := viper.Unmarshal(&AppConfig); err != nil {
		return err // Viper 解析到结构体失败，这是严重错误
	}

	slog.Info("配置加载完成",
		slog.String("serverPort", AppConfig.ServerPort),
		slog.String("dbType", AppConfig.Database.Type),
		slog.String("storageType", AppConfig.Storage.Type),
		slog.Bool("initialized", AppConfig.Initialized),
		slog.String("allowedOrigins", AppConfig.CORSAllowedOrigins),
	)

	return nil
}

func (c *Config) GetRateLimitDuration() time.Duration {
	return time.Duration(c.RateLimit.DurationMinutes) * time.Minute
}
