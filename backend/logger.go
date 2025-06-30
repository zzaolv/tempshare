// backend/logger.go
package main

import (
	"log/slog"
	"os"
)

// InitLogger 初始化一个全局的 slog JSON 格式记录器
func InitLogger() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo, // 你可以根据环境调整日志级别，例如 LevelDebug
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)
}
