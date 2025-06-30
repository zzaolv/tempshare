// backend/init.go
package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type InitHandler struct{}

type InitPayload struct {
	Database DatabaseConfig `json:"database"`
	Storage  StorageConfig  `json:"storage"`
}

// GetStatus 告诉前端是否需要初始化
func (h *InitHandler) GetStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"needsInit": true})
}

// ValidateConfig 接收前端发送的配置并尝试验证它
func (h *InitHandler) ValidateConfig(c *gin.Context) {
	var payload InitPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "无效的配置数据: " + err.Error()})
		return
	}

	// 1. 验证数据库连接
	var db *gorm.DB
	var err error
	db, err = ConnectDatabase(payload.Database)
	if err != nil {
		slog.Error("初始化验证：数据库连接失败", "error", err)
		c.JSON(http.StatusConflict, gin.H{"field": "database", "message": "数据库连接失败: " + err.Error()})
		return
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	slog.Info("初始化验证：数据库连接成功")

	// 2. 验证存储配置 (这里只做简单示例)
	_, err = initStorageProvider(payload.Storage)
	if err != nil {
		slog.Error("初始化验证：存储配置失败", "error", err)
		c.JSON(http.StatusConflict, gin.H{"field": "storage", "message": "存储配置失败: " + err.Error()})
		return
	}
	slog.Info("初始化验证：存储配置成功")

	// 3. 生成推荐的环境变量
	envVars := generateEnvVars(payload)

	// 生成 docker-compose.yml 示例
	composeExample, err := generateComposeExample(payload)
	if err != nil {
		slog.Error("生成 docker-compose 示例失败", "error", err)
		// 非致命错误，继续
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "配置验证成功！请使用以下环境变量重新启动应用。",
		"envVars":        envVars,
		"composeExample": composeExample,
	})
}

// generateEnvVars 从配置生成环境变量字符串
func generateEnvVars(payload InitPayload) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("TS_DATABASE_TYPE=%s\n", payload.Database.Type))
	builder.WriteString(fmt.Sprintf("TS_DATABASE_DSN=%s\n", payload.Database.DSN))
	builder.WriteString(fmt.Sprintf("TS_STORAGE_TYPE=%s\n", payload.Storage.Type))
	if payload.Storage.Type == "local" {
		builder.WriteString(fmt.Sprintf("TS_STORAGE_LOCAL_PATH=%s\n", payload.Storage.Local.Path))
	}
	// ... 在此添加 S3 和 WebDAV 的环境变量 ...

	return builder.String()
}

func generateComposeExample(payload InitPayload) (string, error) {
	// 这里仅为示例，实际中可能需要模板引擎
	// 我们将动态构建一个 compose 结构并序列化为 YAML
	// 为了避免引入新的 YAML 库，我们用字符串拼接来演示

	var sb strings.Builder

	sb.WriteString("version: '3.8'\n\n")
	sb.WriteString("services:\n")
	sb.WriteString("  tempshare-frontend:\n")
	sb.WriteString("    build:\n")
	sb.WriteString("      context: .\n")
	sb.WriteString("      dockerfile: Dockerfile.frontend\n")
	sb.WriteString("    container_name: tempshare-frontend\n")
	sb.WriteString("    ports:\n")
	sb.WriteString("      - \"80:80\"\n")
	sb.WriteString("      - \"443:443\"\n")
	sb.WriteString("    depends_on:\n")
	sb.WriteString("      - tempshare-backend\n")
	sb.WriteString("    restart: unless-stopped\n\n")

	sb.WriteString("  tempshare-backend:\n")
	sb.WriteString("    build:\n")
	sb.WriteString("      context: .\n")
	sb.WriteString("      dockerfile: Dockerfile.backend\n")
	sb.WriteString("    container_name: tempshare-backend\n")
	// 后端不直接暴露端口给主机
	sb.WriteString("    restart: unless-stopped\n")
	sb.WriteString("    volumes:\n")
	sb.WriteString("      - tempshare_data:/app/data\n")
	sb.WriteString("    environment:\n")
	sb.WriteString(fmt.Sprintf("      - TS_DATABASE_TYPE=%s\n", payload.Database.Type))

	db_dsn := payload.Database.DSN
	db_dependency := ""

	if payload.Database.Type == "mysql" {
		db_dsn = "root:mysqlpassword@tcp(tempshare-db:3306)/tempshare?charset=utf8mb4&parseTime=True&loc=Local"
		db_dependency = "tempshare-db"
	} else if payload.Database.Type == "postgres" {
		db_dsn = "postgres://user:password@tempshare-db:5432/tempshare?sslmode=disable"
		db_dependency = "tempshare-db"
	}

	sb.WriteString(fmt.Sprintf("      - TS_DATABASE_DSN=%s\n", db_dsn))
	sb.WriteString(fmt.Sprintf("      - TS_STORAGE_TYPE=%s\n", payload.Storage.Type))
	sb.WriteString(fmt.Sprintf("      - TS_STORAGE_LOCAL_PATH=%s\n", payload.Storage.Local.Path))
	// Add other env vars here

	if db_dependency != "" {
		sb.WriteString("    depends_on:\n")
		sb.WriteString(fmt.Sprintf("      - %s\n", db_dependency))
	}

	sb.WriteString("\n")

	if payload.Database.Type == "mysql" {
		sb.WriteString("  tempshare-db:\n")
		sb.WriteString("    image: mysql:8.0\n")
		sb.WriteString("    container_name: tempshare-mysql\n")
		sb.WriteString("    command: --default-authentication-plugin=mysql_native_password\n")
		sb.WriteString("    restart: unless-stopped\n")
		sb.WriteString("    environment:\n")
		sb.WriteString("      - MYSQL_ROOT_PASSWORD=mysqlpassword\n")
		sb.WriteString("      - MYSQL_DATABASE=tempshare\n")
		sb.WriteString("    volumes:\n")
		sb.WriteString("      - tempshare_db_data:/var/lib/mysql\n\n")
	}

	if payload.Database.Type == "postgres" {
		sb.WriteString("  tempshare-db:\n")
		sb.WriteString("    image: postgres:15\n")
		sb.WriteString("    container_name: tempshare-postgres\n")
		sb.WriteString("    restart: unless-stopped\n")
		sb.WriteString("    environment:\n")
		sb.WriteString("      - POSTGRES_USER=user\n")
		sb.WriteString("      - POSTGRES_PASSWORD=password\n")
		sb.WriteString("      - POSTGRES_DB=tempshare\n")
		sb.WriteString("    volumes:\n")
		sb.WriteString("      - tempshare_db_data:/var/lib/postgresql/data\n\n")
	}

	sb.WriteString("volumes:\n")
	sb.WriteString("  tempshare_data:\n")
	if db_dependency != "" {
		sb.WriteString("  tempshare_db_data:\n")
	}

	return sb.String(), nil
}
