// backend/database.go
package main

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// --- 模型定义 (无变化) ---
const (
	ScanStatusPending  = "pending"
	ScanStatusClean    = "clean"
	ScanStatusInfected = "infected"
	ScanStatusError    = "error"
	ScanStatusSkipped  = "skipped"
)

type File struct {
	ID                string `gorm:"primaryKey" json:"-"`
	AccessCode        string `gorm:"uniqueIndex,size:6" json:"accessCode"`
	Filename          string `gorm:"size:255" json:"filename"`
	SizeBytes         int64  `gorm:"not null" json:"sizeBytes"`
	OriginalSizeBytes int64  `json:"originalSizeBytes"`
	IsEncrypted       bool   `gorm:"default:false;index" json:"isEncrypted"`
	EncryptionSalt    string `json:"encryptionSalt"`
	VerificationHash  string `gorm:"size:64" json:"-"`
	DownloadOnce      bool   `gorm:"default:false" json:"downloadOnce"`
	// ✨ 核心修改点: StorageKey 现在是一个更通用的标识符，而不是文件路径
	StorageKey string    `gorm:"unique;size:255" json:"-"`
	ExpiresAt  time.Time `gorm:"index" json:"expiresAt"`
	CreatedAt  time.Time `json:"createdAt"`
	ScanStatus string    `gorm:"default:'pending';index" json:"scanStatus"`
	ScanResult string    `gorm:"size:255" json:"scanResult"`
}

type Report struct {
	gorm.Model
	AccessCode string `json:"accessCode" binding:"required"`
	Reason     string `json:"reason"`
	ReporterIP string `json:"-"`
}

// --- 数据库连接 ---
func ConnectDatabase(config DBConfig) (*gorm.DB, error) {
	var dialector gorm.Dialector

	dbType := strings.ToLower(config.Type)
	dsn := config.DSN

	switch dbType {
	case "sqlite":
		// 为 SQLite 特殊处理 DSN，确保 WAL 模式开启
		dsnWithWAL := fmt.Sprintf("%s?_pragma=journal_mode=WAL", dsn)
		dialector = sqlite.Open(dsnWithWAL)
	case "mysql":
		// 示例 DSN: "user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local"
		dialector = mysql.Open(dsn)
	case "postgres":
		// 示例 DSN: "host=localhost user=gorm password=gorm dbname=gorm port=5432 sslmode=disable TimeZone=Asia/Shanghai"
		dialector = postgres.Open(dsn)
	default:
		return nil, fmt.Errorf("不支持的数据库类型: %s", dbType)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("无法连接数据库 (%s): %w", dbType, err)
	}

	err = db.AutoMigrate(&File{}, &Report{})
	if err != nil {
		return nil, fmt.Errorf("无法迁移数据库: %w", err)
	}

	fmt.Printf("成功连接到 %s 数据库\n", dbType)
	return db, nil
}
