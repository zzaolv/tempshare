package main

import (
	"log"
	"os"
	"time"

	"gorm.io/gorm"
)

func CleanupExpiredFilesTask(db *gorm.DB) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		<-ticker.C
		log.Println("🧹 开始执行过期文件清理任务...")

		const batchSize = 100
		var deletedCount int64

		for {
			var expiredFiles []File

			result := db.Where("expires_at <= ?", time.Now()).Limit(batchSize).Find(&expiredFiles)
			if result.Error != nil {
				log.Printf("! 清理任务错误: 查询批次失败: %v", result.Error)
				break
			}

			if len(expiredFiles) == 0 {
				break
			}

			for _, file := range expiredFiles {
				if err := os.Remove(file.StorageKey); err != nil && !os.IsNotExist(err) {
					log.Printf("! 清理错误: 删除文件 %s (ID: %s) 失败: %v", file.StorageKey, file.ID, err)
				}
				if err := db.Delete(&file).Error; err != nil {
					log.Printf("! 清理错误: 删除数据库记录 (ID: %s) 失败: %v", file.ID, err)
				} else {
					deletedCount++
				}
			}
		}

		if deletedCount > 0 {
			log.Printf("🧹 本轮清理任务完成，共清理了 %d 个文件。", deletedCount)
		} else {
			log.Println("🧹 清理完成，没有发现新的过期文件。")
		}
	}
}
