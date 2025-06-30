// backend/config.go
package main

import (
	"log/slog"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// ... 其他结构体定义保持不变 ...
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

// LoadConfig 现在会处理文件不存在的错误，并成功返回
func LoadConfig(path string) error {
	viper.SetConfigFile(path)
	viper.SetConfigType("json")

	// 设置默认值
	viper.SetDefault("ServerPort", "8080")
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

	// ✨✨✨ 核心修复点 ✨✨✨
	// 尝试读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		// 检查错误是否是 "文件未找到"
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// 文件未找到，这是 Docker 环境下的预期行为，记录信息并继续
			slog.Info("配置文件 config.json 未找到，将完全依赖环境变量和默认值。")
		} else {
			// 如果是其他错误 (例如 JSON 格式错误)，则这是一个严重错误，返回它
			return err
		}
	}

	// 绑定环境变量
	viper.SetEnvPrefix("TEMPSHARE")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	AppConfig = &Config{}
	if err := viper.Unmarshal(&AppConfig); err != nil {
		return err // Viper 解析到结构体失败，这是严重错误
	}

	// 再次从环境变量读取，以确保覆盖所有值
	AppConfig.Initialized = viper.GetBool("INITIALIZED")

	slog.Info("配置加载完成",
		slog.String("serverPort", viper.GetString("SERVERPORT")), // 从viper直接获取以反映最终值
		slog.String("dbType", viper.GetString("DATABASE.TYPE")),
		slog.String("storageType", viper.GetString("STORAGE.TYPE")),
		slog.Bool("initialized", AppConfig.Initialized),
	)

	return nil // ✨ 成功完成，即使文件不存在
}

func (c *Config) GetRateLimitDuration() time.Duration {
	return time.Duration(c.RateLimit.DurationMinutes) * time.Minute
}
