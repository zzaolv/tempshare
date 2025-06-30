// backend/main.go
package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	// 我们不再需要在这里导入 viper 和 errors，因为 LoadConfig 已经处理了所有逻辑
)

func main() {
	InitLogger()

	// ✨✨✨ 核心修复点 1: 简化配置加载调用 ✨✨✨
	// 新的 LoadConfig 函数在文件不存在时会返回 nil，所以我们只需检查真正的错误。
	if err := LoadConfig("config.json"); err != nil {
		slog.Error("加载配置时发生严重错误，程序无法启动", "error", err)
		os.Exit(1)
	}

	if !AppConfig.Initialized {
		runInitializationGuide()
		os.Exit(1)
	}

	// ... 后续的初始化代码保持不变 ...
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
	// 默认使用 DebugMode，这在本地开发时更有用
	gin.SetMode(gin.DebugMode)
	// 如果在生产环境（例如 Docker），可以设置 GIN_MODE=release
	if os.Getenv("GIN_MODE") == "release" {
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

	// ... 路由定义保持不变 ...
	router.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	apiV1 := router.Group("/api/v1")
	// ... (省略重复的路由定义代码)
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

	// ✨✨✨ 核心修复点 2: 采用更可靠的启动逻辑 ✨✨✨
	// 我们通过检查证书文件是否存在来判断是否应该启动 HTTPS 服务器。
	// 这对于区分本地开发和 Docker 容器环境非常有效。
	certFile := "cert.pem"
	keyFile := "key.pem"
	if _, err := os.Stat(certFile); err == nil {
		if _, err := os.Stat(keyFile); err == nil {
			// 证书和密钥文件都存在，启动 HTTPS 服务器 (用于本地开发)
			slog.Info("检测到 cert.pem 和 key.pem，启动 HTTPS 服务器...", "address", "https://localhost"+serverAddr)
			if err := router.RunTLS(serverAddr, certFile, keyFile); err != nil {
				slog.Error("无法启动 HTTPS 服务器", "error", err)
				os.Exit(1)
			}
			return // 确保程序在这里结束
		}
	}

	// 如果证书文件不存在，或者检查出错，则启动标准的 HTTP 服务器 (用于 Docker 或其他生产环境)
	slog.Info("未找到证书文件，启动 HTTP 服务器...", "address", "http://localhost"+serverAddr)
	if err := router.Run(serverAddr); err != nil {
		slog.Error("无法启动 HTTP 服务器", "error", err)
		os.Exit(1)
	}
}

// runInitializationGuide 函数保持不变
func runInitializationGuide() {
	// ... (函数内容不变)
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
