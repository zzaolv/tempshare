package main

import (
	"encoding/json"
	"os"
)

// RateLimitConfig 对应 config.json 中的 "RateLimit" 部分
type RateLimitConfig struct {
	Enabled         bool `json:"Enabled"`
	Requests        int  `json:"Requests"`
	DurationMinutes int  `json:"DurationMinutes"`
}

// Config 对应整个 config.json 的结构
type Config struct {
	ServerPort      string          `json:"ServerPort"`
	DatabasePath    string          `json:"DatabasePath"`
	ClamdSocket     string          `json:"ClamdSocket"`
	MaxUploadSizeMB int64           `json:"MaxUploadSizeMB"` // 使用 int64 以便计算
	RateLimit       RateLimitConfig `json:"RateLimit"`
}

var AppConfig *Config

// LoadConfig 从指定路径加载配置
func LoadConfig(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	AppConfig = &Config{}
	err = decoder.Decode(AppConfig)
	return err
}
