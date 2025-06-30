// backend/tasks.go
package main

import (
	"context"
	"log/slog"
	"time"

	"tempshare/storage" // 引入 storage

	"gorm.io/gorm"
)

// CleanupExpiredFilesTask 现在需要 StorageProvider
func CleanupExpiredFilesTask(db *gorm.DB, sp storage.StorageProvider) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		<-ticker.C
		slog.Info("开始执行过期文件清理任务...")

		const batchSize = 100
		var deletedCount int64

		for {
			var expiredFiles []File

			result := db.Where("expires_at <= ?", time.Now()).Limit(batchSize).Find(&expiredFiles)
			if result.Error != nil {
				slog.Error("清理任务错误: 查询批次失败", "error", result.Error)
				break
			}

			if len(expiredFiles) == 0 {
				break
			}

			for _, file := range expiredFiles {
				// 使用 StorageProvider 删除文件
				if err := sp.Delete(context.Background(), file.StorageKey); err != nil {
					slog.Error("清理错误: 删除存储文件失败", "id", file.ID, "key", file.StorageKey, "error", err)
				}

				if err := db.Delete(&file).Error; err != nil {
					slog.Error("清理错误: 删除数据库记录失败", "id", file.ID, "error", err)
				} else {
					slog.Info("已清理过期文件", "id", file.ID, "accessCode", file.AccessCode, "filename", file.Filename)
					deletedCount++
				}
			}
		}

		if deletedCount > 0 {
			slog.Info("本轮清理任务完成", "deletedCount", deletedCount)
		} else {
			slog.Info("清理完成，没有发现新的过期文件。")
		}
	}
}
