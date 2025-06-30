// backend/tasks.go
package main

import (
	"log/slog"
	"time"

	"gorm.io/gorm"
)

// CleanupExpiredFilesTask 接收 db 和 storage 实例
func CleanupExpiredFilesTask(db *gorm.DB, storage FileStorage) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	// 首次运行前先执行一次
	cleanup(db, storage)

	for {
		<-ticker.C
		cleanup(db, storage)
	}
}

func cleanup(db *gorm.DB, storage FileStorage) {
	slog.Info("开始执行过期文件清理任务...")

	const batchSize = 100
	var deletedCount int64

	for {
		var expiredFiles []File

		// 查询时只选择必要的字段
		result := db.Select("id", "storage_key", "access_code", "filename").
			Where("expires_at <= ?", time.Now()).Limit(batchSize).Find(&expiredFiles)

		if result.Error != nil {
			slog.Error("清理任务错误: 查询批次失败", "error", result.Error)
			break
		}

		if len(expiredFiles) == 0 {
			break
		}

		for _, file := range expiredFiles {
			// 先删除物理文件/对象
			if err := storage.Delete(file.StorageKey); err != nil {
				slog.Error("清理错误: 删除存储对象失败", "key", file.StorageKey, "error", err)
				// 即使物理文件删除失败，也继续尝试删除数据库记录，避免无限重试
			}

			// 再删除数据库记录
			if err := db.Delete(&File{}, "id = ?", file.ID).Error; err != nil {
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
