// backend/database.go
package main

import (
	"fmt"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// --- 模型定义 ---
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
	// ✨ 核心修改点: 增加密码验证哈希字段 ✨
	VerificationHash string    `gorm:"size:64" json:"-"` // SHA-256 hex string is 64 chars
	DownloadOnce     bool      `gorm:"default:false" json:"downloadOnce"`
	StorageKey       string    `gorm:"unique" json:"-"`
	ExpiresAt        time.Time `gorm:"index" json:"expiresAt"`
	CreatedAt        time.Time `json:"createdAt"`
	ScanStatus       string    `gorm:"default:'pending';index" json:"scanStatus"`
	ScanResult       string    `gorm:"size:255" json:"scanResult"`
}

type Report struct {
	gorm.Model
	AccessCode string `json:"accessCode" binding:"required"`
	Reason     string `json:"reason"`
	ReporterIP string `json:"-"`
}

// --- 数据库连接 ---
func ConnectDatabase(dbPath string) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s?_pragma=journal_mode=WAL", dbPath)

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("无法连接数据库: %w", err)
	}

	err = db.AutoMigrate(&File{}, &Report{})
	if err != nil {
		return nil, fmt.Errorf("无法迁移数据库: %w", err)
	}

	return db, nil
}
