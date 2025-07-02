// backend/config.go
package main

import (
	"errors" // ✨ 导入 errors 包
	"fmt"
	"log/slog"
	"os" // ✨ 导入 os 包
	"strings"
	"time"

	"github.com/spf13/viper"
)

// --- 结构体定义保持不变 ---
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
	PublicHost         string          `mapstructure:"PublicHost"`
	CORSAllowedOrigins string          `mapstructure:"CORS_ALLOWED_ORIGINS"`
	MaxUploadSizeMB    int64           `mapstructure:"MaxUploadSizeMB"`
	RateLimit          RateLimitConfig `mapstructure:"RateLimit"`
	Database           DBConfig        `mapstructure:"Database"`
	Storage            StorageConfig   `mapstructure:"Storage"`
	ClamdSocket        string          `mapstructure:"ClamdSocket"`
	Initialized        bool            `mapstructure:"Initialized"`
}

var AppConfig *Config

func LoadConfig(path string) error {
	viper.SetEnvPrefix("TEMPSHARE")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	viper.SetDefault("ServerPort", "8080")
	viper.SetDefault("PublicHost", "")
	viper.SetDefault("CORS_ALLOWED_ORIGINS", "https://localhost:5173")
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

	viper.SetConfigFile(path)
	viper.SetConfigType("json")

	if err := viper.ReadInConfig(); err != nil {
		// ✨✨✨ 核心修复点: 使用更健壮的错误检查 ✨✨✨
		// 我们不仅检查 Viper 特定的错误，还检查通用的 "文件不存在" 错误。
		// 这样无论 Viper 返回哪种错误类型，我们都能正确处理。
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) || os.IsNotExist(err) {
			slog.Info("配置文件 config.json 未找到，将完全依赖环境变量和默认值。这在 Docker 环境下是正常行为。")
		} else {
			// 如果是其他错误 (例如 JSON 格式无效)，这是一个严重错误，必须返回它
			return fmt.Errorf("解析配置文件 %s 时发生致命错误: %w", path, err)
		}
	} else {
		slog.Info("成功从文件加载配置", "path", path)
	}

	AppConfig = &Config{}
	if err := viper.Unmarshal(&AppConfig); err != nil {
		return fmt.Errorf("将配置解析到结构体时失败: %w", err)
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
