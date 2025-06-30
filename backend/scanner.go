// backend/scanner.go
package main

import (
	"log/slog"
	"strings"
	"time"

	"github.com/dutchcoders/go-clamd"
)

type ClamdScanner struct {
	client *clamd.Clamd
}

// NewScanner 创建一个新的 ClamdScanner 实例。
// 它会尝试连接到 clamd 守护进程，并在连接失败时进行多次重试。
func NewScanner(clamdAddress string) (*ClamdScanner, error) {
	if clamdAddress == "" {
		slog.Warn("ClamdSocket 未在 config.json 中配置，文件扫描功能将不可用。")
		return &ClamdScanner{client: nil}, nil
	}

	const maxRetries = 5               // 最多重试5次
	const retryDelay = 5 * time.Second // 每次重试间隔5秒

	var c *clamd.Clamd
	var err error

	for i := 1; i <= maxRetries; i++ {
		c = clamd.NewClamd(clamdAddress)
		err = c.Ping()
		if err == nil {
			slog.Info("成功连接到 clamd 守护进程", "address", clamdAddress, "attempt", i)
			return &ClamdScanner{client: c}, nil
		}

		slog.Warn("无法连接到 clamd 守护进程", "attempt", i, "maxAttempts", maxRetries, "address", clamdAddress, "error", err)

		if i < maxRetries {
			slog.Info("将在指定延迟后重试", "delay", retryDelay)
			time.Sleep(retryDelay)
		}
	}

	slog.Error("最终无法连接到 clamd，所有重试均失败", "maxAttempts", maxRetries)
	slog.Warn("请确保 clamd 正在运行，并且地址配置正确。")
	slog.Warn("在Linux上, 运行 'sudo systemctl start clamav-daemon' 并使用 'systemctl status clamav-daemon' 检查状态。")
	slog.Warn("在Windows上, 启动 'ClamAV ClamD' 服务。")
	slog.Warn("文件扫描功能将在此次运行中被禁用。")

	return nil, err
}

func (s *ClamdScanner) ScanFile(filePath string) (string, string) {
	if s.client == nil {
		return ScanStatusSkipped, "扫描器未初始化"
	}

	slog.Info("开始扫描文件", "component", "clamd", "path", filePath)

	response, err := s.client.ScanFile(filePath)
	if err != nil {
		slog.Error("Clamd 扫描通信出错", "component", "clamd", "error", err)
		return ScanStatusError, "Clamd扫描通信失败"
	}

	for result := range response {
		slog.Debug("收到 Clamd 响应", "component", "clamd", "rawResponse", result.Raw)
		if result.Status == clamd.RES_FOUND {
			virusName := strings.TrimSuffix(strings.TrimPrefix(result.Raw, result.Path+": "), " FOUND")
			slog.Warn("危险! 文件发现病毒", "component", "clamd", "path", filePath, "virus", virusName)
			return ScanStatusInfected, virusName
		} else if result.Status == clamd.RES_ERROR {
			errorDetails := strings.TrimSuffix(strings.TrimPrefix(result.Raw, result.Path+": "), " ERROR")
			slog.Error("Clamd 扫描时发生错误", "component", "clamd", "details", errorDetails)
			return ScanStatusError, errorDetails
		}
	}

	slog.Info("扫描完成，文件安全", "component", "clamd", "path", filePath)
	return ScanStatusClean, "文件安全"
}
