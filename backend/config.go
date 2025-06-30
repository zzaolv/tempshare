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

	// 2. 设置默认值 (这些值将被配置文件或环境变量覆盖)
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
	viper.SetDefault("Storage.S3.UsePathStyle", true)
	viper.SetDefault("ClamdSocket", "")
	viper.SetDefault("Initialized", false)

	// 3. 尝试读取配置文件 (这是可选的)
	if path != "" {
		viper.SetConfigFile(path)
		viper.SetConfigType("json")
		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				// 文件未找到，这是 Docker 环境下的预期行为，记录信息并继续
				slog.Info("配置文件未找到，将完全依赖环境变量和默认值。")
			} else {
				// 如果是其他错误 (例如 JSON 格式错误)，则这是一个严重错误，返回它
				return fmt.Errorf("解析配置文件 '%s' 失败: %w", path, err)
			}
		} else {
			slog.Info("已成功从配置文件加载配置", "path", path)
		}
	}

	// 4. 将所有配置源 (环境变量 > 配置文件 > 默认值) 解析到结构体中
	AppConfig = &Config{}
	if err := viper.Unmarshal(&AppConfig); err != nil {
		return err // Viper 解析到结构体失败，这是严重错误
	}

	// 绑定环境变量到具体字段，确保环境变量能覆盖所有设置
	// Viper 的 Unmarshal 优先级是：环境变量 > 配置文件 > 默认值
	// 但为了确保结构体字段名和环境变量名能精确对应，可以显式绑定
	bindEnvVars()

	// 再次 Unmarshal 以确保显式绑定的环境变量生效
	if err := viper.Unmarshal(&AppConfig); err != nil {
		return err
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

// bindEnvVars 显式地将环境变量绑定到配置结构体的字段
func bindEnvVars() {
	viper.BindEnv("ServerPort", "TEMPSHARE_SERVERPORT")
	viper.BindEnv("CORS_ALLOWED_ORIGINS", "TEMPSHARE_CORS_ALLOWED_ORIGINS")
	viper.BindEnv("Initialized", "TEMPSHARE_INITIALIZED")
	viper.BindEnv("Database.Type", "TEMPSHARE_DATABASE_TYPE")
	viper.BindEnv("Database.DSN", "TEMPSHARE_DATABASE_DSN")
	viper.BindEnv("Storage.Type", "TEMPSHARE_STORAGE_TYPE")
	viper.BindEnv("Storage.LocalPath", "TEMPSHARE_STORAGE_LOCALPATH")
	viper.BindEnv("Storage.S3.Endpoint", "TEMPSHARE_STORAGE_S3_ENDPOINT")
	viper.BindEnv("Storage.S3.Region", "TEMPSHARE_STORAGE_S3_REGION")
	viper.BindEnv("Storage.S3.Bucket", "TEMPSHARE_STORAGE_S3_BUCKET")
	viper.BindEnv("Storage.S3.UsePathStyle", "TEMPSHARE_STORAGE_S3_USEPATHSTYLE")
	viper.BindEnv("Storage.S3.AccessKeyID", "TEMPSHARE_STORAGE_S3_ACCESSKEYID")
	viper.BindEnv("Storage.S3.SecretAccessKey", "TEMPSHARE_STORAGE_S3_SECRETACCESSKEY")
	viper.BindEnv("Storage.WebDAV.URL", "TEMPSHARE_STORAGE_WEBDAV_URL")
	viper.BindEnv("Storage.WebDAV.Username", "TEMPSHARE_STORAGE_WEBDAV_USERNAME")
	viper.BindEnv("Storage.WebDAV.Password", "TEMPSHARE_STORAGE_WEBDAV_PASSWORD")
	viper.BindEnv("ClamdSocket", "TEMPSHARE_CLAMDSOCKET")
}

func (c *Config) GetRateLimitDuration() time.Duration {
	return time.Duration(c.RateLimit.DurationMinutes) * time.Minute
}
