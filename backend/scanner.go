package main

import (
	"log"
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
		log.Println("🟡 警告: ClamdSocket 未在 config.json 中配置，文件扫描功能将不可用。")
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
			log.Printf("🟢 成功连接到 clamd 守护进程 at %s (在第 %d 次尝试)", clamdAddress, i)
			return &ClamdScanner{client: c}, nil
		}

		log.Printf("🟠 (尝试 %d/%d) 无法连接到 clamd 守护进程 at %s: %v", i, maxRetries, clamdAddress, err)

		if i < maxRetries {
			log.Printf("   将在 %v 后重试...", retryDelay)
			time.Sleep(retryDelay)
		}
	}

	// 所有重试都失败后
	log.Printf("🔴 最终无法连接到 clamd。所有 %d 次尝试均失败。", maxRetries)
	log.Println("   请确保 clamd 正在运行，并且地址配置正确。")
	log.Println("   在Linux上, 运行 'sudo systemctl start clamav-daemon' 并使用 'systemctl status clamav-daemon' 检查状态。")
	log.Println("   在Windows上, 启动 'ClamAV ClamD' 服务。")
	log.Println("   文件扫描功能将在此次运行中被禁用。")

	// 返回 nil, error，让主程序知道初始化失败，但我们将错误处理为非致命的。
	// 主程序 `main.go` 中已经有逻辑处理这个错误，所以这里返回原始错误是正确的。
	return nil, err
}

func (s *ClamdScanner) ScanFile(filePath string) (string, string) {
	if s.client == nil {
		return ScanStatusSkipped, "扫描器未初始化"
	}

	log.Printf("🔬 (clamd) 开始扫描文件: %s", filePath)

	response, err := s.client.ScanFile(filePath)
	if err != nil {
		log.Printf("⚠️ (clamd) 扫描出错: %v", err)
		return ScanStatusError, "Clamd扫描通信失败"
	}

	// 这个通道的读取逻辑保持不变
	for result := range response {
		log.Printf("  - Clamd 响应: %s", result.Raw)
		if result.Status == clamd.RES_FOUND {
			virusName := strings.TrimSuffix(strings.TrimPrefix(result.Raw, result.Path+": "), " FOUND")
			log.Printf("🚫 (clamd) 危险! 文件 %s 发现病毒: %s", filePath, virusName)
			return ScanStatusInfected, virusName
		} else if result.Status == clamd.RES_ERROR {
			errorDetails := strings.TrimSuffix(strings.TrimPrefix(result.Raw, result.Path+": "), " ERROR")
			log.Printf("⚠️ (clamd) 扫描时发生错误: %s", errorDetails)
			return ScanStatusError, errorDetails
		}
	}

	log.Printf("✅ (clamd) 扫描完成，文件安全: %s", filePath)
	return ScanStatusClean, "文件安全"
}
