// backend/config.go
package main

import (
	"fmt"
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
	LocalPath string       `mapstructure:"LocalPath"` // 修正：直接在顶层定义，而不是嵌套
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

// LoadConfig 现在可以优雅地处理配置文件不存在的情况，并返回一个可区分的错误
func LoadConfig(path string) error {
	// 1. 绑定环境变量 (需要提前，优先级最高)
	viper.SetEnvPrefix("TEMPSHARE")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	// 2. 设置默认值 (优先级最低)
	viper.SetDefault("ServerPort", "8080")
	viper.SetDefault("CORS_ALLOWED_ORIGINS", "https://localhost:5173")
	viper.SetDefault("MaxUploadSizeMB", 1024)
	viper.SetDefault("RateLimit.Enabled", true)
	viper.SetDefault("RateLimit.Requests", 30)
	viper.SetDefault("RateLimit.DurationMinutes", 10)
	viper.SetDefault("Database.Type", "sqlite")
	viper.SetDefault("Database.DSN", "data/tempshare.db")
	viper.SetDefault("Storage.Type", "local")
	viper.SetDefault("Storage.LocalPath", "data/files")
	viper.SetDefault("Storage.S3.UsePathStyle", true) // MinIO 常用
	viper.SetDefault("ClamdSocket", "")
	viper.SetDefault("Initialized", false)

	// 3. 尝试读取配置文件 (可选的，优先级居中)
	viper.SetConfigFile(path)
	viper.SetConfigType("json")

	// ReadInConfig 会首先尝试环境变量，然后是配置文件，最后是默认值
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// 文件未找到，这是 Docker 环境下的预期行为，记录信息并继续
			slog.Info("配置文件 config.json 未找到，将完全依赖环境变量和默认值。")
			// 关键：不返回错误，让程序继续
		} else {
			// 如果是其他错误 (例如 JSON 格式错误)，这是一个严重错误，必须返回它
			return fmt.Errorf("解析配置文件 %s 时出错: %w", path, err)
		}
	} else {
		slog.Info("成功从文件加载配置", "path", path)
	}

	// 4. 将所有配置源 (环境变量 > 配置文件 > 默认值) 的结果解析到结构体中
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
