// backend/database.go
package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// --- 模型定义 (保持不变) ---
const (
	ScanStatusPending  = "pending"
	ScanStatusClean    = "clean"
	ScanStatusInfected = "infected"
	ScanStatusError    = "error"
	ScanStatusSkipped  = "skipped"
)

type File struct {
	ID                string    `gorm:"primaryKey" json:"-"`
	AccessCode        string    `gorm:"uniqueIndex,size:6" json:"accessCode"`
	Filename          string    `gorm:"size:255" json:"filename"`
	SizeBytes         int64     `gorm:"not null" json:"sizeBytes"`
	OriginalSizeBytes int64     `json:"originalSizeBytes"`
	IsEncrypted       bool      `gorm:"default:false;index" json:"isEncrypted"`
	EncryptionSalt    string    `json:"encryptionSalt"`
	VerificationHash  string    `gorm:"size:64" json:"-"`
	DownloadOnce      bool      `gorm:"default:false" json:"downloadOnce"`
	StorageKey        string    `gorm:"unique" json:"-"`
	ExpiresAt         time.Time `gorm:"index" json:"expiresAt"`
	CreatedAt         time.Time `json:"createdAt"`
	ScanStatus        string    `gorm:"default:'pending';index" json:"scanStatus"`
	ScanResult        string    `gorm:"size:255" json:"scanResult"`
}

type Report struct {
	gorm.Model
	AccessCode string `json:"accessCode" binding:"required"`
	Reason     string `json:"reason"`
	ReporterIP string `json:"-"`
}

// --- 数据库连接 (核心修改) ---
func ConnectDatabase(cfg DatabaseConfig) (*gorm.DB, error) {
	var dialector gorm.Dialector

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	switch cfg.Type {
	case "sqlite":
		// 对于SQLite，确保目录存在
		dbDir := filepath.Dir(cfg.DSN)
		if err := os.MkdirAll(dbDir, os.ModePerm); err != nil {
			return nil, fmt.Errorf("无法创建SQLite目录: %w", err)
		}
		dsn := fmt.Sprintf("%s?_pragma=journal_mode=WAL", cfg.DSN)
		dialector = sqlite.Open(dsn)
		slog.Info("使用 SQLite 数据库", "path", cfg.DSN)

	case "mysql":
		dialector = mysql.Open(cfg.DSN)
		slog.Info("连接到 MySQL 数据库")

	case "postgres":
		dialector = postgres.Open(cfg.DSN)
		slog.Info("连接到 PostgreSQL 数据库")

	default:
		return nil, fmt.Errorf("不支持的数据库类型: %s", cfg.Type)
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, fmt.Errorf("无法连接数据库 (%s): %w", cfg.Type, err)
	}

	// 自动迁移
	err = db.AutoMigrate(&File{}, &Report{})
	if err != nil {
		return nil, fmt.Errorf("无法迁移数据库: %w", err)
	}

	return db, nil
}
