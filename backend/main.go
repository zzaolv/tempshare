// backend/main.go
package main

import (
	"errors" // ✨ 导入 errors 包
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper" // ✨ 导入 viper 包
)

func main() {
	InitLogger()

	// ✨✨✨ 核心修复点 1: 对 LoadConfig 的错误进行精确判断 ✨✨✨
	if err := LoadConfig("config.json"); err != nil {
		// 只有在错误不是 "配置文件未找到" 时，才认为是严重错误并退出。
		// viper.ConfigFileNotFoundError 是一个具体的错误类型，我们用 errors.As 来检查。
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			slog.Error("加载配置时发生严重错误，程序无法启动", "error", err)
			os.Exit(1)
		}
		// 如果错误是 "文件未找到"，我们会忽略它，因为 LoadConfig 内部已经打印了日志，
		// 并且程序会继续使用环境变量和默认值运行，这正是 Docker 环境所期望的。
	}

	if !AppConfig.Initialized {
		runInitializationGuide()
		os.Exit(1)
	}

	storage, err := NewFileStorage(AppConfig.Storage)
	if err != nil {
		slog.Error("存储后端初始化失败", "error", err)
		os.Exit(1)
	}

	db, err := ConnectDatabase(AppConfig.Database)
	if err != nil {
		slog.Error("数据库初始化失败", "error", err)
		os.Exit(1)
	}

	clamdScanner, err := NewScanner(AppConfig.ClamdSocket)
	if err != nil {
		slog.Warn("Clamd 扫描器初始化失败，文件扫描功能将不可用。", "error", err)
	}

	go CleanupExpiredFilesTask(db, storage)

	// --- Gin 路由器设置 ---
	// 在生产环境中 (例如 Docker)，我们通常希望 Gin 以 ReleaseMode 运行以获得更好的性能并减少日志输出。
	// 可以通过环境变量来控制。
	if os.Getenv("GIN_MODE") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()
	router.SetTrustedProxies(nil)

	var allowedOrigins []string
	if AppConfig.CORSAllowedOrigins != "" {
		allowedOrigins = strings.Split(AppConfig.CORSAllowedOrigins, ",")
	}
	slog.Info("CORS Allowed Origins", "origins", allowedOrigins)

	corsConfig := cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "X-File-Name", "X-File-Original-Size", "X-File-Encrypted", "X-File-Salt", "X-File-Expires-In", "X-File-Download-Once", "X-Requested-With", "X-File-Verification-Hash"},
		ExposeHeaders:    []string{"Content-Length", "Content-Disposition"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	router.Use(cors.New(corsConfig))

	fileHandler := &FileHandler{
		DB:      db,
		Scanner: clamdScanner,
		Storage: storage,
	}

	// --- 路由定义 ---
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	apiV1 := router.Group("/api/v1")
	{
		if AppConfig.RateLimit.Enabled {
			limiter := NewIPRateLimiter(AppConfig.RateLimit.Requests, AppConfig.GetRateLimitDuration())
			uploadAndReportGroup := apiV1.Group("/")
			uploadAndReportGroup.Use(limiter.RateLimitMiddleware())
			{
				uploadAndReportGroup.POST("/uploads/stream-complete", fileHandler.HandleStreamUpload)
				uploadAndReportGroup.POST("/report", fileHandler.HandleReport)
			}
			slog.Info("已启用上传/举报速率限制", "requests", AppConfig.RateLimit.Requests, "durationMinutes", AppConfig.RateLimit.DurationMinutes)
		} else {
			slog.Warn("速率限制已禁用")
			apiV1.POST("/uploads/stream-complete", fileHandler.HandleStreamUpload)
			apiV1.POST("/report", fileHandler.HandleReport)
		}

		apiV1.GET("/files/meta/:code", fileHandler.HandleGetFileMeta)
		apiV1.GET("/files/public", fileHandler.HandleGetPublicFiles)
		apiV1.GET("/preview/:code", fileHandler.HandlePreviewFile)
		apiV1.GET("/preview/data-uri/:code", fileHandler.HandlePreviewDataURI)
	}

	dataGroup := router.Group("/data/:code")
	{
		dataGroup.GET("", fileHandler.HandleDownloadFile)
		dataGroup.POST("", fileHandler.HandleDownloadFile)
	}

	serverAddr := ":" + AppConfig.ServerPort
	slog.Info("后端服务准备启动...", "address", "http://localhost"+serverAddr, "storage", AppConfig.Storage.Type, "database", AppConfig.Database.Type)

	// ✨✨✨ 核心修复点 2: 区分开发和生产环境的启动方式 ✨✨✨
	// 当 GIN_MODE 为 debug 时 (通常是本地开发)，我们启动 HTTPS 服务器。
	// 否则 (在 Docker 中)，我们启动标准的 HTTP 服务器，因为 TLS 终止通常由上层的反向代理（如 Nginx Proxy Manager, Traefik）处理。
	if gin.Mode() == gin.DebugMode {
		slog.Info("在开发模式下启动 HTTPS 服务器...")
		if err := router.RunTLS(serverAddr, "cert.pem", "key.pem"); err != nil {
			slog.Error("无法启动 HTTPS 服务器", "error", err)
			slog.Warn("请确保 backend 目录下存在 cert.pem 和 key.pem 文件。")
			slog.Warn("您可以通过在 backend 目录中运行 'mkcert -install && mkcert localhost' 来生成它们。")
			os.Exit(1)
		}
	} else {
		slog.Info("在生产模式下启动 HTTP 服务器...")
		if err := router.Run(serverAddr); err != nil {
			slog.Error("无法启动 HTTP 服务器", "error", err)
			os.Exit(1)
		}
	}
}

// runInitializationGuide 函数保持不变
func runInitializationGuide() {
	fmt.Println("--- 闪传驿站 | TempShare 未初始化 ---")
	fmt.Println("检测到这是首次运行或配置尚未完成。")
	fmt.Println("\n请通过环境变量进行配置。创建一个 `.env` 文件并设置以下变量：")
	fmt.Println("-----------------------------------------------------------------")
	fmt.Println("# 基础设置")
	fmt.Println("TEMPSHARE_INITIALIZED=true                 # 完成配置后，设置为 true 来启动服务")
	fmt.Println("TEMPSHARE_SERVERPORT=8080                    # 应用监听的端口")
	fmt.Println("TEMPSHARE_CORS_ALLOWED_ORIGINS=https://your-frontend.com # 允许的前端域名, 多个用逗号隔开")

	fmt.Println("\n# 数据库配置 (选择一种)")
	fmt.Println("## SQLite (默认)")
	fmt.Println("TEMPSHARE_DATABASE_TYPE=sqlite")
	fmt.Println("TEMPSHARE_DATABASE_DSN=data/tempshare.db   # 推荐放在持久化卷中")

	fmt.Println("\n# 存储配置 (选择一种)")
	fmt.Println("## 本地存储 (默认)")
	fmt.Println("TEMPSHARE_STORAGE_TYPE=local")
	fmt.Println("TEMPSHARE_STORAGE_LOCALPATH=data/files     # 推荐放在持久化卷中")

	fmt.Println("\n# (可选) ... 其他配置项 ...")
	fmt.Println("-----------------------------------------------------------------")
	fmt.Println("\n配置完成后，请确保 TEMPSHARE_INITIALIZED=true，然后重新启动服务。")
}
